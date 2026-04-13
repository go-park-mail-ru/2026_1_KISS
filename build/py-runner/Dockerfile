# Build context: build/
FROM python:3.13-slim

ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1 \
    KERNEL_NAME=python3 \
    VIRTUAL_ENV=/opt/venv \
    PATH=/opt/venv/bin:$PATH

RUN useradd --create-home --shell /usr/sbin/nologin runner \
 && python -m venv /opt/venv \
 && chown -R runner:runner /opt/venv

WORKDIR /app

COPY py-runner/requirements.txt /app/requirements.txt
RUN pip install --no-cache-dir -r /app/requirements.txt \
 && python -m ipykernel install --sys-prefix --name python3 --display-name "Python 3"

COPY runner/agent.py /app/agent.py
COPY runner/runner_app /app/runner_app
RUN chown -R runner:runner /app

# Редирект пользовательских pip install в tmpfs/quota-layer,
# PIP_TARGET - куда пипу ставить пакеты
# PYTHONPATH - где интерпретатору искать пакеты
ENV PIP_TARGET=/home/runner/.local/site-packages \
    PYTHONPATH=/home/runner/.local/site-packages

USER runner

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=20s --retries=3 \
  CMD python -c "import urllib.request; urllib.request.urlopen('http://127.0.0.1:8080/health', timeout=3)" || exit 1

CMD ["uvicorn", "agent:app", "--host", "0.0.0.0", "--port", "8080"]
