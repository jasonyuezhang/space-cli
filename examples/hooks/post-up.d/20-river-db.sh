#!/bin/bash
# .space/hooks/post-up.d/20-river-db.sh
# Sets up River queue database with migrations
#
# Prerequisites:
#   - psql (PostgreSQL client)
#   - river CLI: go install github.com/riverqueue/river/cmd/river@latest

set -e

# Configuration (can be overridden via environment)
DB_USER="${RIVER_DB_USER:-admin}"
DB_PASS="${RIVER_DB_PASS:-test}"
DB_NAME="${RIVER_DB_NAME:-river}"
DB_PORT="${RIVER_DB_PORT:-5432}"

# Read JSON context from stdin
CONTEXT=$(cat)

# Check if river CLI is available
if ! command -v river &> /dev/null; then
  echo "   ⏭️  river CLI not found, skipping"
  echo "   💡 Install: go install github.com/riverqueue/river/cmd/river@latest"
  exit 0
fi

# Check if psql is available
if ! command -v psql &> /dev/null; then
  echo "   ⚠️  psql not found, cannot setup database"
  exit 1
fi

# Get postgres host from environment or context
PG_HOST="${SPACE_SERVICE_POSTGRES_DNS_NAME:-}"

if [[ -z "$PG_HOST" ]] && command -v jq &> /dev/null; then
  PG_HOST=$(echo "$CONTEXT" | jq -r '.services["postgres"].dns_name // empty')
fi

if [[ -z "$PG_HOST" ]]; then
  echo "   ⏭️  No postgres service found, skipping"
  exit 0
fi

echo "🗄️  Setting up River database..."

ADMIN_URL="postgres://${DB_USER}:${DB_PASS}@${PG_HOST}:${DB_PORT}/postgres"
RIVER_URL="postgres://${DB_USER}:${DB_PASS}@${PG_HOST}:${DB_PORT}/${DB_NAME}"

# Wait for postgres to be ready
echo "   ⏳ Waiting for postgres at ${PG_HOST}:${DB_PORT}..."
for i in {1..30}; do
  if psql "$ADMIN_URL" -c "SELECT 1" &>/dev/null; then
    echo "   ✅ Postgres is ready"
    break
  fi
  if [[ $i -eq 30 ]]; then
    echo "   ❌ Postgres not ready after 30 seconds"
    exit 1
  fi
  sleep 1
done

# Create database if it doesn't exist
echo "   📦 Creating database '${DB_NAME}' if not exists..."
psql "$ADMIN_URL" -c "CREATE DATABASE ${DB_NAME}" 2>/dev/null || echo "   ℹ️  Database already exists"

# Run river migrations
echo "   🔄 Running river migrations..."
river migrate-up --database-url "$RIVER_URL"

echo "   ✅ River database ready at ${PG_HOST}:${DB_PORT}/${DB_NAME}"
