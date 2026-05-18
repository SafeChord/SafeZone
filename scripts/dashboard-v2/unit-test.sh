#!/bin/bash
set -e

CONTAINER_NAME=test-${IMAGE_NAME}-${IMAGE_TAG}

# Step 1: build testing image
echo "Building Docker image..."
docker buildx build --build-arg BASE_IMAGE_TAG=$VERSION -t "$IMAGE_NAME:$IMAGE_TAG" -f "$BUILD_PATH/Dockerfile.test" .

# Step 2: run the tests in the testing container
echo "Running unit tests..."
docker run --rm \
  --name "$CONTAINER_NAME" "$IMAGE_NAME:$IMAGE_TAG"

# Step 3: clean up
echo "Cleaning up..."
docker rmi "$IMAGE_NAME":"$IMAGE_TAG"|| true
