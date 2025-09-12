#!/bin/bash
set -e

# --------- Customizable Section ---------
IMAGE_REPO="ghcr.io/safezone"
IMAGE_NAME="safezone-cli-command"
INSTANCE_NAME="safezone-cli-daemon"
# ENV variables for the daemon
RELAY_URL="http://172.17.0.1:8000" # relay URL for docker compose testing
# RELAY_URL="https://safezone.omh.idv.tw/cli" # relay URL for k3s
RELAY_TIMEOUT=3600
TOKEN_FILE="/app/.temp_token.json"
# --------------------------------

# 1. Remove exited containers (to avoid zombie containers occupying the name)
if docker ps -a --filter "name=$INSTANCE_NAME" --filter "status=exited" | grep -q $INSTANCE_NAME; then
  echo "Found exited $INSTANCE_NAME, removing..."
  docker rm $INSTANCE_NAME
fi

# 2. Pull image if not found
if ! docker image inspect "$IMAGE_NAME:$IMAGE_TAG" > /dev/null 2>&1; then
  echo "Docker image $IMAGE_NAME:$IMAGE_TAG not found. Pulling..."
  docker pull "$IMAGE_REPO/$IMAGE_NAME:$IMAGE_TAG"
fi

# 3. Run daemon if not already running
if ! docker ps -q --filter "name=$INSTANCE_NAME" | grep -q .; then
  echo "Docker instance $INSTANCE_NAME not found. Running..."
  docker run --name=$INSTANCE_NAME -d \
    --env-file .secrets \
    -e "RELAY_URL=$RELAY_URL" \
    -e "RELAY_TIMEOUT=$RELAY_TIMEOUT" \
    -e "TOKEN_FILE=$TOKEN_FILE" \
    "$IMAGE_NAME:$IMAGE_TAG"
fi

# 4. Execute CLI command, aligning paths with local environment
if [ -t 0 ]; then
  # if interactive terminal 
  DOCKER_FLAGS="-it"
else
  DOCKER_FLAGS="-i"
fi

docker exec $DOCKER_FLAGS $INSTANCE_NAME szcli "$@"

