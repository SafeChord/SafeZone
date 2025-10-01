#!/bin/bash

# ==============================================================================
# SafeZone Smoke Test Script
#
# This script performs an end-to-end smoke test of the SafeZone application.
# It uses the `szcli` tool to interact with the system, verifying the core
# dataflow from simulation to verification.
#
# Usage:
#   ./run_smoke_test.sh
# ==============================================================================

# --- Configuration ---
set -e
set -o pipefail # Exit script if any command in a pipeline fails

# Ensure jq is installed
if ! command -v jq &> /dev/null
then
    echo "jq could not be found. Please install jq to run this script."
    exit 1
fi

# --- Helper functions for colored logging ---
Color_Off='\033[0m'
BGreen='\033[1;32m'
BRed='\033[1;31m'
BYellow='\033[1;33m'

log_info() {
    echo -e "${BYellow}[INFO] $1${Color_Off}"
}

log_success() {
    echo -e "${BGreen}[SUCCESS] $1${Color_Off}"
}

log_error() {
    echo -e "${BRed}[ERROR] $1${Color_Off}"
}

log_trace_id() {
    if [[ -z "$1" ]]; then
        log_error "Trace ID is missing."
        exit 1
    fi
    log_info "Trace ID: $1"
}
# --- Main Functions ---
# 
szcli() {
    local instance_name="cli-daemon"
    local container_id
    container_id=$(docker ps -q --filter "name=${instance_name}")
    
    if [[ -z "$container_id" ]]; then
        log_error "CI MODE ERROR: CLI daemon container ('${instance_name}') is not running!"
        return 1
    fi
    # Execute the command inside the container
    docker exec "$container_id" szcli "$@"
}

# Cleanup function that runs on script exit
cleanup() {
    log_info "--- Tearing down test environment ---"
    docker compose -f "$COMPOSE_FILE" --profile=ui down -v --remove-orphans
    docker compose -f "$COMPOSE_FILE" --profile=core down -v --remove-orphans
    docker compose -f "$COMPOSE_FILE" --profile=toolkit down -v --remove-orphans
    docker compose -f "$COMPOSE_FILE" --profile=infra down -v --remove-orphans
    log_success "Test environment cleaned up."
}

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
            local status=$(docker compose -f "$COMPOSE_FILE" ps -q "$service" | xargs docker inspect -f '{{.State.Health.Status}}' 2>/dev/null || echo "starting")
            
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

# Set a trap to ensure cleanup runs regardless of script exit status (success or failure)
trap cleanup EXIT

# Phase 0: Setup and start the environment
setup_environment() {
    log_info "Starting infrastructure services (db, redis)..."
    docker compose -f "$COMPOSE_FILE" --profile=infra up -d
    sleep 5

    # robust wait for infra services to be healthy
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
}

# Generic test case runner function
run_test_cases_from_file() {
    local test_file=$1
    log_info "--- Running test cases from: $test_file ---"

    # Read CSV file, skip header, and handle potential carriage returns
    tail -n +2 "$test_file" | tr -d '\r' | while IFS=, read -r szcli_command jq_path expected_value || [[ -n "$expected_value" ]]
    do
        # Remove quotes that might be wrapping the command
        szcli_command=$(echo "$szcli_command" | tr -d '"')
        jq_path=$(echo "$jq_path" | xargs | tr -d '"')
        expected_value=$(echo "$expected_value" | xargs | tr -d '"')

        log_info "Executing: $szcli_command"
        
        # Use eval to correctly handle arguments with spaces
        local output=$(eval "$szcli_command" < /dev/null)
        echo "$output"

        # Use jq to extract the actual value from the JSON output
        actual_value=$(echo "$output" | jq -r "$jq_path")

        # Compare actual value with expected value
        if [[ "$actual_value" == "$expected_value" ]]; then
            log_success "PASSED: Command '$szcli_command' -> Expected '$expected_value', Got '$actual_value'"
        else
            log_error "FAILED: Command '$szcli_command'"
            log_error "  --> Expected: '$expected_value'"
            log_error "  --> Got:      '$actual_value'"
            log_error "  --> Full output:"
            echo "$output"
            exit 1
        fi
        sleep 5
    done
}

locate_log_by_trace_id() {
    local trace_id_1=$1
    local trace_id_2=$2

    docker compose -f "$COMPOSE_FILE" logs analytics-api | grep "$trace_id_1" | grep "$trace_id_2"
}

# Test Cache Invalidation Mechanism (Corrected Logic)
test_cache_invalidation_flow() {
    local test_date="1970-01-01" 
    
    # ... (Step 1: Simulate) ...
    log_info "Step 1/6: Simulating initial data for $test_date..."
    local simulate_trace_id_1=$(szcli -o json dataflow simulate "$test_date" | jq -r '.task.trace_id')
    log_trace_id "$simulate_trace_id_1"
    sleep 5 # Wait for worker

    # ... (Step 2: Verify Miss) ...
    log_info "Step 2/6: Verifying data (expecting CACHE MISS)..."
    local verify_trace_id_1=$(szcli -o json dataflow verify "$test_date" | jq -r '.task.trace_id')
    log_trace_id "$verify_trace_id_1"

    if locate_log_by_trace_id "$simulate_trace_id_1" "$verify_trace_id_1" | grep -q "Cache miss"; then
        log_success "  -> PASSED: Cache MISS confirmed."
    else
        log_error "  -> FAILED: Expected cache MISS, but was not found."
        exit 1
    fi

    # ... (Step 3: Verify Hit) ...
    log_info "Step 3/6: Verifying data again (expecting CACHE HIT)..."
    local verify_trace_id_2=$(szcli -o json dataflow verify "$test_date" | jq -r '.task.trace_id')
    log_trace_id "$verify_trace_id_2"

    if locate_log_by_trace_id "$simulate_trace_id_1" "$verify_trace_id_2" | grep -q "Cache hit"; then
        log_success "  -> PASSED: Cache HIT confirmed."
    else
        log_error "  -> FAILED: Expected cache HIT, but was not found."
        exit 1
    fi

    # ... (Step 4: Cache Invalidation) ...
    log_info "Step 4/6: Re-Simulating to trigger cache invalidation..."
    local simulate_output_2=$(szcli -o json dataflow simulate "$test_date" | jq -r '.task.trace_id')
    log_trace_id "$simulate_output_2"
    sleep 5 # Wait for worker

    # ... (Step 5: Verify Miss) ...
    log_info "Step 5/6: Verifying data after invalidation (expecting new CACHE MISS)..."
    local verify_trace_id_3=$(szcli -o json dataflow verify "$test_date" | jq -r '.task.trace_id')
    log_trace_id "$verify_trace_id_3"

    if locate_log_by_trace_id "$simulate_trace_id_2" "$verify_trace_id_3" | grep -q "Cache miss"; then
        log_success "  -> PASSED: New Cache MISS confirmed after invalidation."
    else
        log_error "  -> FAILED: Expected new cache MISS, but was not found."
        exit 1
    fi

    # ... (Step 6: Verify Hit) ...
    log_info "Step 6/6: Verifying data one last time (expecting new CACHE HIT)..."
    local verify_trace_id_4=$(szcli -o json dataflow verify "$test_date" | jq -r '.task.trace_id')
    log_trace_id "$verify_trace_id_4"

    if locate_log_by_trace_id "$simulate_trace_id_2" "$verify_trace_id_4" | grep -q "Cache hit"; then
        log_success "  -> PASSED: New Cache HIT confirmed after invalidation."
    else
        log_error "  -> FAILED: Expected new cache HIT, but was not found."
        exit 1
    fi
}

# --- Script Main Execution ---
main() {
    log_info "========== Starting SafeZone Smoke Test =========="
    
    log_info "--- Phase 0: Setting up environment ---"
    setup_environment

    log_info "--- Phase 1: Testing Cache Invalidation Mechanism ---"
    test_cache_invalidation_flow
    
    log_info "--- Phase 2: Testing dataflow-1 (with smaller cases) ---"
    run_test_cases_from_file "$TEST_CASE_FILE_PHASE2"

    log_info "--- Phase 3: Testing dataflow-1 (with whole cases) ---"
    run_test_cases_from_file "$TEST_CASE_FILE_PHASE3"

    log_success "========== Smoke Test Completed Successfully =========="
}

# Execute the main function
main
