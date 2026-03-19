#!/usr/bin/env bash

# ==============================================================================
# SafeZone Environment Teardown Script
#
# This script stops and removes all containers, networks, and volumes
# associated with the test environment.
# ==============================================================================

# Source the common library
source "$(dirname "$0")/lib/common.sh"

# Find the compose file in the parent directory structure
DEFAULT_COMPOSE_FILE=$(dirname "$0")/../docker-compose/smoke-test/release_0.2.0.yml
COMPOSE_FILE=${COMPOSE_FILE:-$DEFAULT_COMPOSE_FILE}

if [ ! -f "$COMPOSE_FILE" ]; then
    log_error "Docker compose file not found at $COMPOSE_FILE"
    exit 1
fi

log_info "--- Tearing down environment ---"
docker compose -f "$COMPOSE_FILE" --profile=infra --profile=toolkit --profile=core --profile=ui down --remove-orphans
log_success "Environment cleaned up."
