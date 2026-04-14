FROM python:3.13-slim

COPY toolkit/cli/command/requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt --root-user-action=ignore

COPY toolkit/cli/command /app
COPY utils /app/utils

ENV PYTHONPATH=/app
WORKDIR /app

RUN chmod +x /app/main.py && ln -s /app/main.py /usr/local/bin/szcli

# Install jq for JSON processing
RUN set -eux; \
    apt-get update; \
    DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends jq; \
    apt-get clean; \
    rm -rf /var/lib/apt/lists/*

CMD ["tail", "-f", "/dev/null"]
