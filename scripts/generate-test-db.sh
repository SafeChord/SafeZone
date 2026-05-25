#!/usr/bin/env bash
# SafeZone/scripts/generate-test-db.sh

# Source the common library
source "$(dirname "$0")/lib/common.sh"

# --- Configuration ---
PROJECT_ROOT="$(dirname "$0")/.."
COMPOSE_FILE="$PROJECT_ROOT/docker-compose/provision-test-db.yml"
OUTPUT_PATH="$PROJECT_ROOT/services/analytics-api/test/db/test.db"
VERSION=${VERSION:-latest}

# Check for required dependencies
check_dependencies docker

# Ensure the output directory exists
mkdir -p "$(dirname "$OUTPUT_PATH")"

setup_provision_env() {
    log_info "Starting lightweight provision environment (cli-relay, cli-daemon)..."
    
    # Export VERSION for docker-compose
    export VERSION=$VERSION

    # Start services
    docker compose -f "$COMPOSE_FILE" --profile provision up -d
    
    log_info "Waiting for cli-relay to initialize..."
    sleep 5 # Simple wait as relay is very fast
}

run_provision() {
    log_info "Initializing base database (Admin + Population data) via szcli..."
    
    # Use the szcli wrapper from common.sh
    # Note: szcli wrapper expects container name 'cli-daemon'
    if ! szcli db init; then
        log_error "Database initialization failed!"
        cleanup
        exit 1
    fi
    log_success "Database initialization completed inside container."
}

extract_db_file() {
    log_info "Extracting generated test.db to $OUTPUT_PATH..."
    
    # Get the container ID of cli-relay
    local relay_container
    relay_container=$(docker compose -f "$COMPOSE_FILE" ps -q cli-relay)

    if [ -z "$relay_container" ]; then
        log_error "Could not find cli-relay container!"
        exit 1
    fi

    # Copy the file from the container's /app/test.db to the host
    if docker cp "$relay_container:/app/test.db" "$OUTPUT_PATH"; then
        log_success "Successfully extracted test.db to $OUTPUT_PATH"
    else
        log_error "Failed to copy test.db from container!"
        exit 1
    fi
}

cleanup() {
    log_info "Cleaning up provision environment..."
    docker compose -f "$COMPOSE_FILE" --profile provision down
    log_info "Environment cleaned."
}

# --- Main Execution ---
main() {
    log_info "====== Starting Base DB Generation ======"
    
    setup_provision_env
    run_provision
    extract_db_file
    cleanup
    
    log_success "====== Base DB Generation Finished! ======"
    log_info "You can now verify 'test_new.db' at: $OUTPUT_PATH"
}

main
