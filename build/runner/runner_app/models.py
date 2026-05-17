from pydantic import BaseModel, Field

from .config import DEFAULT_TIMEOUT, MAX_TIMEOUT


class ExecuteRequest(BaseModel):
    code: str = Field(..., min_length=1)
    timeout: float = Field(default=DEFAULT_TIMEOUT, gt=0, le=MAX_TIMEOUT)


class OutputItem(BaseModel):
    mime_type: str   # "text/plain", "image/png", "text/html", etc.
    data: str        # text or base64-encoded string


class ExecuteResponse(BaseModel):
    stdout: str
    stderr: str
    result: str                        # text/plain result (backward compat)
    outputs: list[OutputItem] = []     # rich outputs (images, html, etc.)


class SnapshotResponse(BaseModel):
    data: str              # base64-encoded dill bytes
    size_bytes: int
    var_count: int
    skipped_vars: list[str] = []


class RestoreRequest(BaseModel):
    data: str              # base64-encoded dill bytes
