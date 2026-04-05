# Build context: build/
FROM r-base:4.5.3

ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1 \
    KERNEL_NAME=ir

RUN apt-get update && apt-get install -y --no-install-recommends \
    python3 python3-pip python3-venv \
    libzmq3-dev \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

RUN python3 -m venv /app/.venv
ENV PATH="/app/.venv/bin:$PATH"

COPY r-runner/requirements.txt /app/requirements.txt
RUN pip install --no-cache-dir -r /app/requirements.txt

RUN useradd --create-home --shell /usr/sbin/nologin runner

COPY runner/agent.py /app/agent.py
COPY runner/runner_app /app/runner_app
RUN chown -R runner:runner /app

RUN Rscript -e "install.packages('IRkernel', repos='https://cloud.r-project.org')"
RUN Rscript -e "IRkernel::installspec(user=FALSE, name='ir', displayname='R')"

USER runner

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=60s --retries=3 \
  CMD python -c "import urllib.request; urllib.request.urlopen('http://127.0.0.1:8080/health', timeout=3)" || exit 1

CMD ["uvicorn", "agent:app", "--host", "0.0.0.0", "--port", "8080"]
