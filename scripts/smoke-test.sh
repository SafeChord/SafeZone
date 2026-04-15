#!/usr/bin/env bash

# ==============================================================================
# Container-Native Smoke Test Runner
#
# Starts the local environment via dev-up, then runs the ops container
# (CSV-driven smoke test engine) on the same Docker network.
#
# Usage:
#   ./scripts/ops-smoke-test.sh
#   VERSION=0.3.0 ./scripts/ops-smoke-test.sh
# ==============================================================================

source "$(dirname "$0")/lib/common.sh"

check_dependencies docker

# --- Configuration ---
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
DEFAULT_COMPOSE_FILE="$SCRIPT_DIR/../docker-compose/local-compose-all.yml"
COMPOSE_FILE=${COMPOSE_FILE:-"$DEFAULT_COMPOSE_FILE"}
VERSION=${VERSION:-latest}
OPS_IMAGE="safezone-cli-ops:${VERSION}"

# --- Resolve compose network name ---
# Docker Compose names the default network as <project>_default.
# Project name defaults to the compose file's parent directory name.
COMPOSE_PROJECT=$(docker compose -f "$COMPOSE_FILE" config --format json | python3 -c "import sys,json; print(json.load(sys.stdin).get('name','smoke-test'))" 2>/dev/null || echo "smoke-test")
NETWORK_NAME="${COMPOSE_PROJECT}_default"

# --- Cleanup ---
trap cleanup EXIT

cleanup() {
    log_info "--- Triggering environment teardown ---"
    COMPOSE_FILE="$COMPOSE_FILE" bash "$SCRIPT_DIR/dev-down.sh"
}

# --- Main ---
main() {
    log_info "========== Container-Native Smoke Test =========="

    # Phase 0: Environment setup
    log_info "--- Phase 0: Starting environment ---"
    COMPOSE_FILE="$COMPOSE_FILE" bash "$SCRIPT_DIR/dev-up.sh"

    # Verify network exists
    if ! docker network inspect "$NETWORK_NAME" &>/dev/null; then
        log_error "Network '$NETWORK_NAME' not found. Available networks:"
        docker network ls --filter "driver=bridge" --format "  {{.Name}}"
        exit 1
    fi
    log_success "Network '$NETWORK_NAME' confirmed."

    # Verify ops image exists
    if ! docker image inspect "$OPS_IMAGE" &>/dev/null; then
        log_error "Image '$OPS_IMAGE' not found. Run 'make build-tool-cli' first."
        exit 1
    fi

    # Phase 1: Run ops container on compose network
    log_info "--- Phase 1: Running ops smoke test ---"
    docker run --rm \
        --network "$NETWORK_NAME" \
        --env-file "$SCRIPT_DIR/../.env.secret" \
        -e RELAY_URL=http://cli-relay:8000 \
        "$OPS_IMAGE"

    log_success "========== Container-Native Smoke Test Passed =========="
}

main
