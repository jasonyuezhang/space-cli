package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/happy-sdk/space-cli/internal/hooks"
	"github.com/spf13/cobra"
)

func newHooksCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "hooks",
		Short: "Manage space hooks",
		Long:  "Commands for managing lifecycle hooks that run during space operations.",
	}

	cmd.AddCommand(newHooksInitCommand())
	cmd.AddCommand(newHooksListCommand())

	return cmd
}

func newHooksInitCommand() *cobra.Command {
	var withTemplates bool

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize hooks directory structure",
		Long:  "Creates the .space/hooks directory with event subdirectories and documentation.",
		RunE: func(cmd *cobra.Command, args []string) error {
			workDir := Workdir
			if workDir == "." {
				var err error
				workDir, err = os.Getwd()
				if err != nil {
					return fmt.Errorf("failed to get working directory: %w", err)
				}
			}

			// Make absolute
			workDir, err := filepath.Abs(workDir)
			if err != nil {
				return fmt.Errorf("failed to resolve working directory: %w", err)
			}

			fmt.Printf("🪝 Initializing hooks in %s/.space/hooks/\n", workDir)

			if err := hooks.InitHooksDir(workDir); err != nil {
				return fmt.Errorf("failed to initialize hooks: %w", err)
			}

			fmt.Println("✅ Hooks directory initialized!")
			fmt.Println()
			fmt.Println("Created directories:")
			fmt.Println("   .space/hooks/pre-up.d/      - Before docker compose up")
			fmt.Println("   .space/hooks/post-up.d/     - After services are running")
			fmt.Println("   .space/hooks/pre-down.d/    - Before docker compose down")
			fmt.Println("   .space/hooks/post-down.d/   - After services are stopped")
			fmt.Println("   .space/hooks/on-dns-ready.d/ - When DNS is configured")
			fmt.Println()
			fmt.Println("📖 See .space/hooks/README.md for documentation and examples")

			if withTemplates {
				if err := createTemplateHooks(workDir); err != nil {
					fmt.Printf("⚠️  Failed to create templates: %v\n", err)
				} else {
					fmt.Println()
					fmt.Println("📝 Created template hooks:")
					fmt.Println("   .space/hooks/post-up.d/10-vite-env.sh.template")
					fmt.Println("   .space/hooks/post-up.d/20-river-db.sh.template")
					fmt.Println()
					fmt.Println("💡 To use a template, rename it (remove .template) and make executable:")
					fmt.Println("   mv .space/hooks/post-up.d/10-vite-env.sh.template .space/hooks/post-up.d/10-vite-env.sh")
					fmt.Println("   chmod +x .space/hooks/post-up.d/10-vite-env.sh")
				}
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&withTemplates, "templates", false, "Create template hook scripts")

	return cmd
}

func newHooksListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List configured hooks",
		Long:  "Lists all hook scripts that will be executed during space operations.",
		RunE: func(cmd *cobra.Command, args []string) error {
			workDir := Workdir
			if workDir == "." {
				var err error
				workDir, err = os.Getwd()
				if err != nil {
					return fmt.Errorf("failed to get working directory: %w", err)
				}
			}

			workDir, _ = filepath.Abs(workDir)
			hooksDir := filepath.Join(workDir, ".space", "hooks")

			if _, err := os.Stat(hooksDir); os.IsNotExist(err) {
				fmt.Println("No hooks configured.")
				fmt.Println("Run 'space hooks init' to create the hooks directory.")
				return nil
			}

			events := []string{"pre-up", "post-up", "pre-down", "post-down", "on-dns-ready"}
			hasHooks := false

			for _, event := range events {
				eventDir := filepath.Join(hooksDir, event+".d")
				entries, err := os.ReadDir(eventDir)
				if err != nil {
					continue
				}

				var scripts []string
				for _, entry := range entries {
					name := entry.Name()
					if entry.IsDir() || name == ".gitkeep" {
						continue
					}
					// Skip templates
					if filepath.Ext(name) == ".template" {
						continue
					}
					info, _ := entry.Info()
					if info != nil && info.Mode()&0111 != 0 {
						scripts = append(scripts, name)
					}
				}

				if len(scripts) > 0 {
					hasHooks = true
					fmt.Printf("📁 %s:\n", event)
					for _, script := range scripts {
						fmt.Printf("   • %s\n", script)
					}
					fmt.Println()
				}
			}

			if !hasHooks {
				fmt.Println("No executable hooks found.")
				fmt.Println("Add scripts to .space/hooks/{event}.d/ and make them executable.")
			}

			return nil
		},
	}

	return cmd
}

// createTemplateHooks creates template hook scripts
func createTemplateHooks(workDir string) error {
	hooksDir := filepath.Join(workDir, ".space", "hooks")

	// Vite env template
	viteTemplate := `#!/bin/bash
# .space/hooks/post-up.d/10-vite-env.sh
# Auto-generates Vite environment file with space.local URLs
#
# To enable: rename this file (remove .template) and make executable:
#   mv 10-vite-env.sh.template 10-vite-env.sh && chmod +x 10-vite-env.sh

set -e

# Read JSON context from stdin
CONTEXT=$(cat)

# Check if this is a Vite project
if [[ ! -f "vite.config.ts" ]] && [[ ! -f "vite.config.js" ]]; then
  echo "   Not a Vite project, skipping"
  exit 0
fi

echo "🎨 Configuring Vite environment..."

# Check if jq is available
if ! command -v jq &> /dev/null; then
  echo "   ⚠️  jq not found, using environment variables"

  # Fallback to environment variables
  if [[ -n "$SPACE_SERVICE_API_SERVER_DNS_NAME" ]]; then
    cat > .env.development.local << EOF
# Auto-generated by space hooks
VITE_API_BASE_URL=http://${SPACE_SERVICE_API_SERVER_DNS_NAME}:${SPACE_SERVICE_API_SERVER_PORT:-6060}
EOF
    echo "   ✅ Generated .env.development.local"
  fi
  exit 0
fi

# Extract service info using jq
API_HOST=$(echo "$CONTEXT" | jq -r '.services["api-server"].dns_name // empty')
API_PORT=$(echo "$CONTEXT" | jq -r '.services["api-server"].internal_port // empty')
APP_HOST=$(echo "$CONTEXT" | jq -r '.services["app"].dns_name // empty')
APP_PORT=$(echo "$CONTEXT" | jq -r '.services["app"].internal_port // empty')

# Generate .env.development.local
{
  echo "# Auto-generated by space hooks"
  echo "# $(date)"

  if [[ -n "$API_HOST" ]] && [[ -n "$API_PORT" ]]; then
    echo "VITE_API_BASE_URL=http://${API_HOST}:${API_PORT}"
  fi

  if [[ -n "$APP_HOST" ]] && [[ -n "$APP_PORT" ]]; then
    echo "VITE_APP_BASE_URL=http://${APP_HOST}:${APP_PORT}"
  fi
} > .env.development.local

echo "   ✅ Generated .env.development.local"
cat .env.development.local | sed 's/^/      /'
`

	// River database template
	riverTemplate := `#!/bin/bash
# .space/hooks/post-up.d/20-river-db.sh
# Sets up River queue database with migrations
#
# To enable: rename this file (remove .template) and make executable:
#   mv 20-river-db.sh.template 20-river-db.sh && chmod +x 20-river-db.sh
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
echo "   ⏳ Waiting for postgres..."
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
`

	// Write templates
	viteFile := filepath.Join(hooksDir, "post-up.d", "10-vite-env.sh.template")
	if err := os.WriteFile(viteFile, []byte(viteTemplate), 0644); err != nil {
		return fmt.Errorf("failed to write vite template: %w", err)
	}

	riverFile := filepath.Join(hooksDir, "post-up.d", "20-river-db.sh.template")
	if err := os.WriteFile(riverFile, []byte(riverTemplate), 0644); err != nil {
		return fmt.Errorf("failed to write river template: %w", err)
	}

	return nil
}
