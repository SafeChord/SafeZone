#!/bin/bash
set -e

echo "Daily simulation: fetching current system date..."
SYSTEM_DATE=$(szcli -o json system time now | jq -r '.response.system_date')

if ! date -d "$SYSTEM_DATE" >/dev/null 2>&1; then
  echo "[ERROR] Invalid system date retrieved: $SYSTEM_DATE"
  exit 1
fi

echo "Simulating data for $SYSTEM_DATE..."
szcli dataflow simulate "$SYSTEM_DATE"

echo "Daily simulation completed for $SYSTEM_DATE."
