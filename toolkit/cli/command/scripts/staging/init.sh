#!/bin/bash
set -e

# init the database for preview environment
echo "Seeding SafeZone database..."
szcli -o json db init
echo "Seeding database completed."

# init the system time for preview environment
echo "Setting system time to 2023-04-20..."
szcli -o json system time set --mockdate=2023-04-20
echo "System time set completed."