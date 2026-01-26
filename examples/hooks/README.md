# Example Hook Scripts

Ready-to-use hook scripts for common development tasks.

## Quick Setup

Copy these scripts to your project:

```bash
# Create hooks directory
mkdir -p .space/hooks/post-up.d

# Copy scripts
cp /path/to/space-cli/examples/hooks/post-up.d/*.sh .space/hooks/post-up.d/

# Make executable
chmod +x .space/hooks/post-up.d/*.sh
```

Or use the CLI:

```bash
space hooks init --templates
```

## Available Scripts

### `10-vite-env.sh`

Automatically generates `.env.development.local` for Vite projects with space.local URLs.

**What it does:**
- Detects Vite projects (vite.config.ts/js)
- Creates `.env.development.local` with service URLs
- Uses environment variables or JSON context (with jq)

**Example output:**
```
VITE_API_BASE_URL=http://api-server-6a9c8c.space.local:6060
```

### `20-river-db.sh`

Sets up River queue database with migrations.

**What it does:**
- Waits for postgres to be ready
- Creates `river` database if not exists
- Runs `river migrate-up`

**Prerequisites:**
- `psql` - PostgreSQL client
- `river` CLI - `go install github.com/riverqueue/river/cmd/river@latest`

**Configuration (via environment):**
```bash
RIVER_DB_USER=admin    # default: admin
RIVER_DB_PASS=test     # default: test
RIVER_DB_NAME=river    # default: river
RIVER_DB_PORT=5432     # default: 5432
```

## Writing Custom Hooks

Scripts receive context two ways:

### Environment Variables

```bash
SPACE_WORKDIR=/path/to/project
SPACE_PROJECT_NAME=myproject-main
SPACE_HASH=6a9c8c
SPACE_BASE_DOMAIN=space.local
SPACE_DNS_ENABLED=true
SPACE_SERVICE_POSTGRES_DNS_NAME=postgres-6a9c8c.space.local
SPACE_SERVICE_POSTGRES_PORT=5432
```

### JSON on stdin

```bash
CONTEXT=$(cat)
echo "$CONTEXT" | jq '.services["postgres"].dns_name'
```

## Execution Order

Scripts are executed in sorted order by filename:
- `10-vite-env.sh` runs before `20-river-db.sh`

Use numeric prefixes to control order.
