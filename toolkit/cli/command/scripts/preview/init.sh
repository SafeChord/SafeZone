#!/bin/bash
set -e

# init the database for preview environment
echo "Seeding SafeZone database..."
szcli -o json db init
echo "Seeding database completed."

# init the system time for preview environment
echo "Setting system time to 1970-02-01..."
szcli -o json system time set --mockdate=1970-02-01
echo "System time set completed."