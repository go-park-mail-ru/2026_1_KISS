import uvicorn

from runner_app import create_app


app = create_app()

if __name__ == "__main__":
    uvicorn.run("agent:app", host="0.0.0.0", port=8080)
