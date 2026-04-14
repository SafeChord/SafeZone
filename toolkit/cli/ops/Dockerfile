ARG BASE_IMAGE_TAG=latest
FROM safezone-cli-command:${BASE_IMAGE_TAG}

COPY toolkit/cli/ops/smoke_test.py /app/smoke_test.py
COPY toolkit/cli/ops/routines /app/routines

WORKDIR /app
ENTRYPOINT ["python", "smoke_test.py"]
