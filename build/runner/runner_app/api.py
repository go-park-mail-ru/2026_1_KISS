from contextlib import asynccontextmanager
import logging

from fastapi import FastAPI, HTTPException

from .bridge import KernelBridge
from .models import ExecuteRequest, ExecuteResponse


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

    return app
