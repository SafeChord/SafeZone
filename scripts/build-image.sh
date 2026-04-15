#!/bin/bash
set -e

echo "Building Docker image..."
docker buildx build -t "$IMAGE_NAME":"$IMAGE_TAG" -f "$BUILD_PATH/Dockerfile" .