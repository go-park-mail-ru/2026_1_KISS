# Build context: build/
FROM python:3.13-slim

ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1 \
    KERNEL_NAME=python3

WORKDIR /app

COPY py-runner/requirements.txt /app/requirements.txt
RUN pip install --no-cache-dir -r /app/requirements.txt
RUN python -m ipykernel install --user --name python3 --display-name "Python 3"

RUN useradd --create-home --shell /usr/sbin/nologin runner

COPY runner/agent.py /app/agent.py
COPY runner/runner_app /app/runner_app
RUN chown -R runner:runner /app

USER runner

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=20s --retries=3 \
  CMD python -c "import urllib.request; urllib.request.urlopen('http://127.0.0.1:8080/health', timeout=3)" || exit 1

CMD ["uvicorn", "agent:app", "--host", "0.0.0.0", "--port", "8080"]
