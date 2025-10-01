#!/bin/bash
set -e

echo "Seeding SafeZone default data..."

# get the system date from the cluster
SYSTEM_DATE=$(szcli -o json system time now | jq -r '.response.system_date')

# check if SYSTEM_DATE is valid
if ! date -d "$SYSTEM_DATE" >/dev/null 2>&1; then
  echo "[ERROR] Invalid system date retrieved: $SYSTEM_DATE"
  exit 1
fi

# calculate the start date (33 days before the system date)
START_DATE=$(date -d "$SYSTEM_DATE - 33 days" +%Y-%m-%d)

# simulate data from START_DATE to SYSTEM_DATE
szcli dataflow simulate "$START_DATE" --enddate="$SYSTEM_DATE"

echo "Seeding default data completed."