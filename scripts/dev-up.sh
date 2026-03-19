#!/usr/bin/env bash

# ==============================================================================
# SafeZone Environment Setup Script
#
# This script starts the full SafeZone stack, waits for all services to be
# healthy, and initializes the database.
# ==============================================================================

# Source the common library
source "$(dirname "$0")/lib/common.sh"

# --- Configuration ---
# Check for required dependencies
check_dependencies docker jq

# Find the compose file
DEFAULT_COMPOSE_FILE=$(dirname "$0")/../docker-compose/smoke-test/release_0.2.0.yml
COMPOSE_FILE=${COMPOSE_FILE:-$DEFAULT_COMPOSE_FILE}

if [ ! -f "$COMPOSE_FILE" ]; then
    log_error "Docker compose file not found at $COMPOSE_FILE"
    exit 1
fi

# Function to wait for all infra services to be healthy
wait_for_infra_services() {
    log_info "Waiting for all infra services to become healthy..."
    
    local services=("db" "redis-state" "redis-cache" "kafka")
    local all_healthy=false
    local attempt=0
    local max_attempts=30
    local sleep_interval=5

    while [ "$all_healthy" = false ] && [ $attempt -lt $max_attempts ]; do
        all_healthy=true
        for service in "${services[@]}"; do
            local status
            status=$(docker compose -f "$COMPOSE_FILE" ps -q "$service" | xargs docker inspect -f '{{.State.Health.Status}}' 2>/dev/null || echo "starting")
            
            if [ "$status" != "healthy" ]; then
                all_healthy=false
                echo -n "." 
                break 
            fi
        done

        if [ "$all_healthy" = false ]; then
            sleep $sleep_interval 
            ((attempt++))
        fi
    done

    echo ""

    if [ "$all_healthy" = true ]; then
        log_success "All infra services are healthy and ready!"
    else
        log_error "Timeout: Not all infra services became healthy after 150 seconds."
        docker compose -f "$COMPOSE_FILE" logs 
        exit 1
    fi
}

# Main setup function
setup_environment() {
    log_info "Starting infrastructure services (db, redis)..."
    docker compose -f "$COMPOSE_FILE" --profile=infra up -d
    sleep 5

    wait_for_infra_services

    log_info "Starting toolkit services (cli-relay)..."
    docker compose -f "$COMPOSE_FILE" --profile=toolkit up -d
    sleep 5

    log_info "Initializing database with base data..."
    if ! szcli db init; then
        log_error "Database initialization failed!"
        exit 1
    fi
    log_success "Database initialized."

    log_info "Starting core application services..."
    docker compose -f "$COMPOSE_FILE" --profile=core up -d
    sleep 5

    log_info "Starting ui services..."
    docker compose -f "$COMPOSE_FILE" --profile=ui up -d
    sleep 5

    log_info "Performing final health check on all services..."
    health_output=$(szcli -o json health all)
    if echo "$health_output" | grep -q "unhealthy"; then
        log_error "Some services are not healthy!"
        echo "$health_output"
        exit 1
    else
        log_success "All services are healthy."
        echo "$health_output"
    fi
    
    log_success "Environment is up and running!"
    log_info "Dashboard available at: http://localhost:8080/dashboard/"
}

# Execute if the script is run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    setup_environment
fi
