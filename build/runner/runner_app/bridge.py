import logging
import os
import threading
import time
from queue import Empty

from jupyter_client import KernelManager

from .config import KERNEL_NAME
from .models import ExecuteResponse, OutputItem


logger = logging.getLogger(__name__)

# Каталог, куда pip --target ставит пользовательские пакеты (см. PIP_TARGET в Dockerfile)
# Должен существовать до старта kernel'а: иначе Python закеширует NullImporter
# для отсутствующего пути и далее импорты установленных пакетов будут падать
USER_SITE_PACKAGES = os.environ.get("PIP_TARGET", "/home/runner/.local/site-packages")


class KernelBridge:
    def __init__(self):
        self._km = None
        self._kc = None
        self._lock = threading.Lock()

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

    def execute_streaming(self, code, timeout):
        if self._kc is None:
            raise RuntimeError("Kernel is not initialized")

        with self._lock:
            msg_id = self._kc.execute(code)
            result_text = ""
            rich_outputs: list[OutputItem] = []

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
