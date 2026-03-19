#!/usr/bin/env bash

# ==============================================================================
# SafeZone Smoke Test Script (Refactored)
#
# This script performs an end-to-end smoke test of the SafeZone application.
# It uses the `szcli` tool to interact with the system, verifying the core
# dataflow from simulation to verification.
#
# Usage:
#   ./smoke-test.sh
# ==============================================================================

# --- Configuration ---
# Source the common library
source "$(dirname "$0")/lib/common.sh"

# Check for required dependencies
check_dependencies docker jq

# Define paths for compose and test data files
# If NO ENV is set, use the default values
DEFAULT_COMPOSE_FILE=$(dirname "$0")/../docker-compose/smoke-test/release_0.2.0.yml
COMPOSE_FILE=${COMPOSE_FILE:-"$DEFAULT_COMPOSE_FILE"}

TEST_CASE_FILE_PHASE2=${TEST_CASE_FILE_PHASE2:-"data/smoke-test/smoke-test-phase-2.csv"}
TEST_CASE_FILE_PHASE3=${TEST_CASE_FILE_PHASE3:-"data/smoke-test/smoke-test-phase-3.csv"}

# Ensure test data files exist
if [ ! -f "$TEST_CASE_FILE_PHASE2" ] || [ ! -f "$TEST_CASE_FILE_PHASE3" ]; then
    log_error "Test case files not found. Searched in 'data/smoke-test/'"
    exit 1
fi

# --- Helper Functions ---
log_trace_id() {
    if [[ -z "$1" ]]; then
        log_error "Trace ID is missing."
        exit 1
    fi
    log_info "Trace ID: $1"
}

# --- Cleanup ---
# Set a trap to ensure cleanup runs regardless of script exit status
trap cleanup EXIT

cleanup() {
    log_info "--- Triggering environment teardown ---"
    COMPOSE_FILE="$COMPOSE_FILE" "$(dirname "$0")/dev-down.sh"
}

# --- Test Logic ---
locate_log_by_trace_id() {
    local trace_id_1=$1
    local trace_id_2=$2

    docker compose -f "$COMPOSE_FILE" logs analytics-api | grep "$trace_id_1" | grep "$trace_id_2"
}

test_cache_invalidation_flow() {
    local test_date="1970-01-01" 
    
    log_info "Step 1/6: Simulating initial data for $test_date..."
    local simulate_trace_id_1
    simulate_trace_id_1=$(szcli -o json dataflow simulate "$test_date" | jq -r '.task.trace_id')
    log_trace_id "$simulate_trace_id_1"
    sleep 5

    log_info "Step 2/6: Verifying data (expecting CACHE MISS)..."
    local verify_trace_id_1
    verify_trace_id_1=$(szcli -o json dataflow verify "$test_date" | jq -r '.task.trace_id')
    log_trace_id "$verify_trace_id_1"

    if locate_log_by_trace_id "$simulate_trace_id_1" "$verify_trace_id_1" | grep -q "Cache miss"; then
        log_success "  -> PASSED: Cache MISS confirmed."
    else
        log_error "  -> FAILED: Expected cache MISS, but was not found."
        exit 1
    fi

    log_info "Step 3/6: Verifying data again (expecting CACHE HIT)..."
    local verify_trace_id_2
    verify_trace_id_2=$(szcli -o json dataflow verify "$test_date" | jq -r '.task.trace_id')
    log_trace_id "$verify_trace_id_2"

    if locate_log_by_trace_id "$simulate_trace_id_1" "$verify_trace_id_2" | grep -q "Cache hit"; then
        log_success "  -> PASSED: Cache HIT confirmed."
    else
        log_error "  -> FAILED: Expected cache HIT, but was not found."
        exit 1
    fi

    log_info "Step 4/6: Re-Simulating to trigger cache invalidation..."
    local simulate_trace_id_2
    simulate_trace_id_2=$(szcli -o json dataflow simulate "$test_date" | jq -r '.task.trace_id')
    log_trace_id "$simulate_trace_id_2"
    sleep 5

    log_info "Step 5/6: Verifying data after invalidation (expecting new CACHE MISS)..."
    local verify_trace_id_3
    verify_trace_id_3=$(szcli -o json dataflow verify "$test_date" | jq -r '.task.trace_id')
    log_trace_id "$verify_trace_id_3"

    if locate_log_by_trace_id "$simulate_trace_id_2" "$verify_trace_id_3" | grep -q "Cache miss"; then
        log_success "  -> PASSED: New Cache MISS confirmed after invalidation."
    else
        log_error "  -> FAILED: Expected new cache MISS, but was not found."
        exit 1
    fi

    log_info "Step 6/6: Verifying data one last time (expecting new CACHE HIT)..."
    local verify_trace_id_4
    verify_trace_id_4=$(szcli -o json dataflow verify "$test_date" | jq -r '.task.trace_id')
    log_trace_id "$verify_trace_id_4"

    if locate_log_by_trace_id "$simulate_trace_id_2" "$verify_trace_id_4" | grep -q "Cache hit"; then
        log_success "  -> PASSED: New Cache HIT confirmed after invalidation."
    else
        log_error "  -> FAILED: Expected new cache HIT, but was not found."
        exit 1
    fi
}

run_test_cases_from_file() {
    local test_file=$1
    log_info "--- Running test cases from: $test_file ---"

    tail -n +2 "$test_file" | tr -d '\r' | while IFS=, read -r szcli_command jq_path expected_value || [[ -n "$expected_value" ]]
    do
        szcli_command=$(echo "$szcli_command" | tr -d '"')
        jq_path=$(echo "$jq_path" | xargs | tr -d '"')
        expected_value=$(echo "$expected_value" | xargs | tr -d '"')

        log_info "Executing: $szcli_command"
        
        local output
        output=$(eval "$szcli_command" < /dev/null)
        echo "$output"

        local actual_value
        actual_value=$(echo "$output" | jq -r "$jq_path")
        
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

# --- Script Main Execution ---
main() {
    log_info "========== Starting SafeZone Smoke Test =========="
    
    log_info "--- Phase 0: Setting up environment ---"
    COMPOSE_FILE="$COMPOSE_FILE" "$(dirname "$0")/dev-up.sh"

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