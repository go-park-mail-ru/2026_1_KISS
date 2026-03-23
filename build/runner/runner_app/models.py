from pydantic import BaseModel, Field

from .config import DEFAULT_TIMEOUT, MAX_TIMEOUT


class ExecuteRequest(BaseModel):
    code: str = Field(..., min_length=1)
    timeout: float = Field(default=DEFAULT_TIMEOUT, gt=0, le=MAX_TIMEOUT)


class ExecuteResponse(BaseModel):
    stdout: str
    stderr: str
    result: str
