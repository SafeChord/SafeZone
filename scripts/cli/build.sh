#!/bin/bash
set -e

# ================== Build Environment Script ==================
# This script builds both the CLI command image and the CLI relay image.


COMMAND_IMAGE_NAME="safezone-cli-command"
RELAY_IMAGE_NAME="safezone-cli-relay"
COMMAND_DOCKERFILE_PATH="./toolkit/cli/command"
RELAY_DOCKERFILE_PATH="./toolkit/cli/relay"

# check if IMAGE_TAG is set, if not, default to 'latest'
if [ -z "$IMAGE_TAG" ]; then
  echo "[INFO] IMAGE_TAG is not set. Defaulting to 'latest'."
  IMAGE_TAG="latest"
fi

# ----------- PHASE 1: Build CLI Command Image ----------------
echo "[PHASE 1] Building CLI command image..."

docker buildx build -t "$COMMAND_IMAGE_NAME:$IMAGE_TAG" -f "$COMMAND_DOCKERFILE_PATH/Dockerfile" .

# ----------- PHASE 2: Build CLI Relay Image ------------------
echo "[PHASE 2] Building CLI relay image..."

docker buildx build -t "$RELAY_IMAGE_NAME:$IMAGE_TAG" -f "$RELAY_DOCKERFILE_PATH/Dockerfile" .