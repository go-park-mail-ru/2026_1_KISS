# Runners

FastAPI bridge over a single long-lived Jupyter kernel using `jupyter_client`.

```
build/
  runner/      # shared app code (agent.py, runner_app/)
  py-runner/   # Python-specific Dockerfile + requirements.txt
```

The kernel is selected via the `KERNEL_NAME` env var (set in each Dockerfile).

## Endpoints

- `GET /health` → `{"status":"ok"}` (or `503` when kernel is not ready)
- `POST /execute`

Request body:

```json
{
  "code": "print('hello')",
  "timeout": 15
}
```

Response body:

```json
{
  "stdout": "",
  "stderr": "",
  "result": ""
}
```

## Container build

Build context must be `build/` so both Dockerfiles can access shared `runner/` code.

```shell
# Python runner
docker build -t kiss-python-runner -f py-runner/Dockerfile .
```

Run:

```shell
docker run --rm -p 8080:8080 kiss-python-runner
```

## Local run (Python runner)

```shell
cd runner
python3 -m venv .venv
source .venv/bin/activate
pip install -r ../py-runner/requirements.txt
uvicorn agent:app --host 0.0.0.0 --port 8080
```

## Test code execution

```shell
curl -sS http://localhost:8080/health
curl -sS -X POST http://localhost:8080/execute -H 'Content-Type: application/json' -d '{"code":"x = 10"}'
curl -sS -X POST http://localhost:8080/execute -H 'Content-Type: application/json' -d '{"code":"print(x)"}'
curl -sS -X POST http://localhost:8080/execute -H 'Content-Type: application/json' -d '{"code":"1/0"}'
curl -sS -X POST http://localhost:8080/execute -H 'Content-Type: application/json' -d '{"code":"import time; time.sleep(3)", "timeout": 1}'
```
