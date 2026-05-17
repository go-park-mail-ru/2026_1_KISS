import base64
import logging
import os
import threading
import time
from queue import Empty

from jupyter_client import KernelManager

from .config import KERNEL_NAME
from .models import ExecuteResponse, OutputItem, SnapshotResponse


logger = logging.getLogger(__name__)

# Каталог, куда pip --target ставит пользовательские пакеты (см. PIP_TARGET в Dockerfile)
# Должен существовать до старта kernel'а: иначе Python закеширует NullImporter
# для отсутствующего пути и далее импорты установленных пакетов будут падать
USER_SITE_PACKAGES = os.environ.get("PIP_TARGET", "/home/runner/.local/site-packages")


class KernelBridge:
    def __init__(self):
        self._km = None
        self._kc = None
        self._lock = threading.RLock()

    def start(self):
        if self._km is not None:
            return

        os.makedirs(USER_SITE_PACKAGES, exist_ok=True)

        logger.info("Starting Jupyter kernel")
        km = KernelManager(kernel_name=KERNEL_NAME)
        km.start_kernel()
        kc = km.blocking_client()
        kc.start_channels()
        kc.wait_for_ready(timeout=30)

        self._km = km
        self._kc = kc
        logger.info("Kernel is ready")

    def stop(self):
        logger.info("Stopping Jupyter kernel")
        if self._kc is not None:
            try:
                self._kc.stop_channels()
            except Exception:
                logger.exception("Failed to stop kernel channels")
            finally:
                self._kc = None

        if self._km is not None:
            try:
                self._km.shutdown_kernel(now=True)
            except Exception:
                logger.exception("Failed to shutdown kernel")
            finally:
                self._km = None

    def is_ready(self):
        return self._km is not None and self._kc is not None

    def _recover_after_timeout(self):
        if self._km is None:
            return
        try:
            logger.warning("Execution timed out, interrupting kernel")
            self._km.interrupt_kernel()
        except Exception:
            logger.exception("Failed to interrupt kernel after timeout")

    def execute(self, code, timeout):
        if self._kc is None:
            raise RuntimeError("Kernel is not initialized")

        with self._lock:
            started_at = time.monotonic()
            msg_id = self._kc.execute(code)
            stdout_chunks = []
            stderr_chunks = []
            result_text = ""
            rich_outputs: list[OutputItem] = []
            output_bytes = 0

            deadline = time.monotonic() + timeout
            while True:
                remaining = deadline - time.monotonic()
                if remaining <= 0:
                    self._recover_after_timeout()
                    raise TimeoutError(f"Execution timeout after {timeout} seconds")

                try:
                    msg = self._kc.get_iopub_msg(timeout=min(1.0, remaining))
                except Empty:
                    continue

                parent_header = msg.get("parent_header", {})
                if parent_header.get("msg_id") != msg_id:
                    continue

                msg_type = msg.get("msg_type")
                content = msg.get("content", {})

                if msg_type == "stream":
                    text = content.get("text", "")
                    output_bytes += len(text.encode("utf-8"))
                    if output_bytes > self.MAX_OUTPUT_BYTES:
                        self._recover_after_timeout()
                        raise TimeoutError(f"Output limit exceeded ({self.MAX_OUTPUT_BYTES // (1024*1024)} MB)")
                    if content.get("name") == "stderr":
                        stderr_chunks.append(text)
                    else:
                        stdout_chunks.append(text)
                elif msg_type in {"execute_result", "display_data"}:
                    data = content.get("data", {})
                    # Collect all MIME types as rich outputs
                    for mime in ("image/png", "image/jpeg", "text/html", "text/plain"):
                        if mime in data:
                            rich_outputs.append(OutputItem(mime_type=mime, data=data[mime]))
                    # Keep result as text/plain for backward compat
                    if "text/plain" in data:
                        result_text = data["text/plain"]
                elif msg_type == "error":
                    tb = content.get("traceback") or []
                    if tb:
                        stderr_chunks.append("\n".join(tb) + "\n")
                    else:
                        ename = content.get("ename", "Error")
                        evalue = content.get("evalue", "")
                        stderr_chunks.append(f"{ename}: {evalue}\n")
                elif msg_type == "status" and content.get("execution_state") == "idle":
                    break

            elapsed = time.monotonic() - started_at
            logger.info("Execution finished in %.3fs", elapsed)
            return ExecuteResponse(
                stdout="".join(stdout_chunks),
                stderr="".join(stderr_chunks),
                result=result_text,
                outputs=rich_outputs,
            )

    MAX_OUTPUT_BYTES = 5 * 1024 * 1024

    # Код, выполняемый в ядре для создания снапшота.
    # Сериализует пользовательские переменные через dill в /tmp/_snapshot.dill.
    _SNAPSHOT_CODE = r"""
import dill as _dill, io as _io, types as _types
_skip = []
_save = {}
for _k, _v in list(globals().items()):
    if _k.startswith('_') or isinstance(_v, _types.ModuleType):
        continue
    try:
        _dill.dumps(_v)
        _save[_k] = _v
    except Exception:
        _skip.append(_k)
_buf = _io.BytesIO()
_dill.dump(_save, _buf)
with open('/tmp/_snapshot.dill', 'wb') as _f:
    _f.write(_buf.getvalue())
_snapshot_var_count = len(_save)
_snapshot_skipped = list(_skip)
del _buf, _save, _k, _v, _skip
"""

    _RESTORE_CODE = r"""
import dill as _dill
try:
    with open('/tmp/_snapshot.dill', 'rb') as _f:
        _saved = _dill.load(_f)
    for _k, _v in _saved.items():
        try:
            globals()[_k] = _v
        except Exception:
            pass
    del _saved, _k, _v
except Exception as _e:
    print(f"[runner] snapshot restore failed: {_e}")
"""

    def take_snapshot(self) -> SnapshotResponse:
        if self._kc is None:
            raise RuntimeError("Kernel is not initialized")
        with self._lock:
            self.execute(self._SNAPSHOT_CODE, timeout=30)
            try:
                result = self.execute("print(_snapshot_var_count, '|', ','.join(_snapshot_skipped))", timeout=5)
                parts = result.stdout.strip().split("|", 1)
                var_count = int(parts[0].strip()) if parts else 0
                skipped = [s.strip() for s in parts[1].split(",") if s.strip()] if len(parts) > 1 else []
            except Exception:
                var_count = 0
                skipped = []

        with open("/tmp/_snapshot.dill", "rb") as f:
            raw = f.read()

        return SnapshotResponse(
            data=base64.b64encode(raw).decode(),
            size_bytes=len(raw),
            var_count=var_count,
            skipped_vars=skipped,
        )

    def restore_snapshot(self, data_b64: str) -> None:
        if self._kc is None:
            raise RuntimeError("Kernel is not initialized")
        raw = base64.b64decode(data_b64)
        with open("/tmp/_snapshot.dill", "wb") as f:
            f.write(raw)
        with self._lock:
            self.execute(self._RESTORE_CODE, timeout=30)

    def restart_kernel(self) -> None:
        logger.info("Restarting Jupyter kernel")
        if self._kc is not None:
            try:
                self._kc.stop_channels()
            except Exception:
                logger.exception("Failed to stop channels before restart")
            self._kc = None
        if self._km is not None:
            try:
                self._km.restart_kernel(now=True)
            except Exception:
                logger.exception("Failed to restart kernel")
        kc = self._km.blocking_client()
        kc.start_channels()
        kc.wait_for_ready(timeout=30)
        self._kc = kc
        logger.info("Kernel restarted and ready")

    def execute_streaming(self, code, timeout):
        if self._kc is None:
            raise RuntimeError("Kernel is not initialized")

        with self._lock:
            msg_id = self._kc.execute(code)
            result_text = ""
            rich_outputs: list[OutputItem] = []
            output_bytes = 0

            deadline = time.monotonic() + timeout
            while True:
                remaining = deadline - time.monotonic()
                if remaining <= 0:
                    self._recover_after_timeout()
                    yield {"type": "error", "data": f"Execution timeout after {timeout} seconds"}
                    return

                try:
                    msg = self._kc.get_iopub_msg(timeout=min(1.0, remaining))
                except Empty:
                    continue

                parent_header = msg.get("parent_header", {})
                if parent_header.get("msg_id") != msg_id:
                    continue

                msg_type = msg.get("msg_type")
                content = msg.get("content", {})

                if msg_type == "stream":
                    text = content.get("text", "")
                    output_bytes += len(text.encode("utf-8"))
                    if output_bytes > self.MAX_OUTPUT_BYTES:
                        self._recover_after_timeout()
                        yield {"type": "error", "data": f"Output limit exceeded ({self.MAX_OUTPUT_BYTES // (1024*1024)} MB). Execution interrupted."}
                        return
                    name = "stderr" if content.get("name") == "stderr" else "stdout"
                    yield {"type": name, "data": text}
                elif msg_type in {"execute_result", "display_data"}:
                    data = content.get("data", {})
                    for mime in ("image/png", "image/jpeg", "text/html", "text/plain"):
                        if mime in data:
                            rich_outputs.append(OutputItem(mime_type=mime, data=data[mime]))
                    if "text/plain" in data:
                        result_text = data["text/plain"]
                elif msg_type == "error":
                    tb = content.get("traceback") or []
                    if tb:
                        yield {"type": "stderr", "data": "\n".join(tb) + "\n"}
                    else:
                        ename = content.get("ename", "Error")
                        evalue = content.get("evalue", "")
                        yield {"type": "stderr", "data": f"{ename}: {evalue}\n"}
                elif msg_type == "status" and content.get("execution_state") == "idle":
                    break

            if result_text:
                yield {"type": "result", "data": result_text}
            for output in rich_outputs:
                yield {"type": "output", "data": output.data, "mime_type": output.mime_type}
