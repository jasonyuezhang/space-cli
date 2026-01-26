#!/bin/bash
# Connect to postgres database via psql
#
# Usage: space run db-shell [database]

DATABASE="${1:-propeldb}"

if [[ -z "$SPACE_SERVICE_POSTGRES_DNS_NAME" ]]; then
  echo "Error: No postgres service found"
  exit 1
fi

echo "Connecting to ${DATABASE} at ${SPACE_SERVICE_POSTGRES_DNS_NAME}..."
exec psql "postgres://admin:test@${SPACE_SERVICE_POSTGRES_DNS_NAME}:5432/${DATABASE}"
