from contextlib import asynccontextmanager
import json
import logging

from fastapi import FastAPI, HTTPException
from fastapi.responses import StreamingResponse

from .bridge import KernelBridge
from .models import ExecuteRequest, ExecuteResponse, RestoreRequest, SnapshotResponse


logger = logging.getLogger(__name__)


def create_app() -> FastAPI:
    bridge = KernelBridge()

    @asynccontextmanager
    async def lifespan(_: FastAPI):
        bridge.start()
        try:
            yield
        finally:
            bridge.stop()

    app = FastAPI(title="Python Runner", version="1.0.0", lifespan=lifespan)

    @app.get("/health")
    def health() -> dict[str, str]:
        if not bridge.is_ready():
            raise HTTPException(status_code=503, detail="Kernel is not ready")
        return {"status": "ok"}

    @app.post("/execute", response_model=ExecuteResponse)
    def execute(payload: ExecuteRequest) -> ExecuteResponse:
        try:
            return bridge.execute(code=payload.code, timeout=payload.timeout)
        except TimeoutError as exc:
            raise HTTPException(status_code=504, detail=str(exc)) from exc
        except RuntimeError as exc:
            raise HTTPException(status_code=503, detail=str(exc)) from exc
        except Exception as exc:
            logger.exception("Execution failed")
            raise HTTPException(status_code=500, detail="Internal execution error") from exc

    @app.post("/execute/stream")
    def execute_stream(payload: ExecuteRequest):
        def generate():
            try:
                for chunk in bridge.execute_streaming(code=payload.code, timeout=payload.timeout):
                    yield json.dumps(chunk, ensure_ascii=False) + "\n"
            except RuntimeError as exc:
                yield json.dumps({"type": "error", "data": str(exc)}) + "\n"
            except Exception as exc:
                logger.exception("Streaming execution failed")
                yield json.dumps({"type": "error", "data": str(exc)}) + "\n"

        return StreamingResponse(generate(), media_type="application/x-ndjson")

    @app.post("/snapshot", response_model=SnapshotResponse)
    def snapshot():
        try:
            return bridge.take_snapshot()
        except RuntimeError as exc:
            raise HTTPException(status_code=503, detail=str(exc)) from exc
        except Exception as exc:
            logger.exception("Snapshot failed")
            raise HTTPException(status_code=500, detail="snapshot failed") from exc

    @app.post("/restore")
    def restore(req: RestoreRequest):
        try:
            bridge.restore_snapshot(req.data)
            return {"status": "ok"}
        except RuntimeError as exc:
            raise HTTPException(status_code=503, detail=str(exc)) from exc
        except Exception as exc:
            logger.exception("Restore failed")
            raise HTTPException(status_code=500, detail="restore failed") from exc

    @app.post("/restart")
    def restart():
        try:
            bridge.restart_kernel()
            return {"status": "ok"}
        except Exception as exc:
            logger.exception("Kernel restart failed")
            raise HTTPException(status_code=500, detail="restart failed") from exc

    return app
