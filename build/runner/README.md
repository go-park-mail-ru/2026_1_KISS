# Python Runner

FastAPI bridge over a single long-lived Jupyter kernel (`ipykernel`) using `jupyter_client`.

## Endpoints

- `GET /health` -> `{"status":"ok"}` (or `503` when kernel is not ready)
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

## Local run

```bash
python3 -m venv .venv
source .venv/bin/activate
pip install -r requirements.txt
uvicorn agent:app --host 0.0.0.0 --port 8080
```

## Quick smoke test

```bash
curl -sS http://localhost:8080/health
curl -sS -X POST http://localhost:8080/execute -H 'Content-Type: application/json' -d '{"code":"x = 10"}'
curl -sS -X POST http://localhost:8080/execute -H 'Content-Type: application/json' -d '{"code":"print(x)"}'
curl -sS -X POST http://localhost:8080/execute -H 'Content-Type: application/json' -d '{"code":"1/0"}'
curl -sS -X POST http://localhost:8080/execute -H 'Content-Type: application/json' -d '{"code":"import time; time.sleep(3)", "timeout": 1}'
```

## Container build

```bash
docker build -t kiss-python-runner -f Dockerfile .
docker run --rm -p 8080:8080 kiss-python-runner
```
