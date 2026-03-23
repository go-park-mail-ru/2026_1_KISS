import logging
import threading
import time
from queue import Empty

from jupyter_client import KernelManager

from .models import ExecuteResponse


logger = logging.getLogger(__name__)


class KernelBridge:
    def __init__(self):
        self._km = None
        self._kc = None
        self._lock = threading.Lock()

    def start(self):
        if self._km is not None:
            return

        logger.info("Starting Jupyter kernel")
        km = KernelManager(kernel_name="python3")
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
                    if "text/plain" in data:
                        result_text = data["text/plain"]
                    else:
                        result_text = str(data)
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
            )
