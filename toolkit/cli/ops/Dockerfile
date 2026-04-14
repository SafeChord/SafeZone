ARG BASE_IMAGE_TAG=latest
FROM safezone-cli-command:${BASE_IMAGE_TAG}

COPY toolkit/cli/ops/requirements.txt /tmp/requirements.ops.txt
RUN pip install --no-cache-dir -r /tmp/requirements.ops.txt --root-user-action=ignore \
    && rm /tmp/requirements.ops.txt

COPY toolkit/cli/ops/assertions /app/assertions
COPY toolkit/cli/ops/routines /app/routines

WORKDIR /app
ENTRYPOINT ["pytest", "assertions/", "-v"]
