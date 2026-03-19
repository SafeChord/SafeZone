#!/usr/bin/env bash

# ==============================================================================
# Common Library for SafeZone Scripts
#
# This script provides shared functions and variables for logging, dependency
# checking, and interacting with the SafeZone environment.
# ==============================================================================

# --- Shell Configuration ---
set -e
set -o pipefail

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

# --- Dependency Checking ---
check_dependencies() {
    local missing_deps=0
    for dep in "$@"; do
        if ! command -v "$dep" &> /dev/null; then
            log_error "Dependency '$dep' could not be found. Please install it."
            missing_deps=$((missing_deps + 1))
        fi
    done

    if [ "$missing_deps" -gt 0 ]; then
        log_error "Please install missing dependencies to continue."
        exit 1
    fi
}

# --- CLI Wrapper ---
szcli() {
    local instance_name="cli-daemon"
    local container_id
    container_id=$(docker ps -q --filter "name=${instance_name}")

    if [[ -z "$container_id" ]]; then
        log_error "CLI daemon container ('${instance_name}') is not running!"
        return 1
    fi
    # Execute the command inside the container
    docker exec "$container_id" szcli "$@"
}
