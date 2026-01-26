package hooks

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// ScriptExecutor executes external hook scripts
type ScriptExecutor struct {
	// HooksDir is the base directory for hooks (default: .space/hooks)
	HooksDir string

	// Timeout for script execution (default: 5 minutes)
	Timeout time.Duration

	// Logger for output
	Logger ScriptLogger
}

// ScriptLogger interface for script execution logging
type ScriptLogger interface {
	Info(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Error(msg string, args ...interface{})
}

// DefaultScriptLogger logs to stdout
type DefaultScriptLogger struct{}

func (l *DefaultScriptLogger) Info(msg string, args ...interface{})  { fmt.Printf("   "+msg+"\n", args...) }
func (l *DefaultScriptLogger) Warn(msg string, args ...interface{})  { fmt.Printf("   ⚠️  "+msg+"\n", args...) }
func (l *DefaultScriptLogger) Error(msg string, args ...interface{}) { fmt.Printf("   ❌ "+msg+"\n", args...) }

// NewScriptExecutor creates a new script executor
func NewScriptExecutor(workDir string) *ScriptExecutor {
	return &ScriptExecutor{
		HooksDir: filepath.Join(workDir, ".space", "hooks"),
		Timeout:  5 * time.Minute,
		Logger:   &DefaultScriptLogger{},
	}
}

// HookContextJSON is the JSON structure passed to hook scripts
type HookContextJSON struct {
	Event       string                     `json:"event"`
	WorkDir     string                     `json:"work_dir"`
	ProjectName string                     `json:"project_name"`
	Hash        string                     `json:"hash"`
	BaseDomain  string                     `json:"base_domain"`
	DNSEnabled  bool                       `json:"dns_enabled"`
	DNSAddress  string                     `json:"dns_address,omitempty"`
	Services    map[string]ServiceInfoJSON `json:"services"`
	Metadata    map[string]interface{}     `json:"metadata,omitempty"`
}

// ServiceInfoJSON is the JSON structure for service info
type ServiceInfoJSON struct {
	Name         string `json:"name"`
	DNSName      string `json:"dns_name,omitempty"`
	InternalPort int    `json:"internal_port,omitempty"`
	ExternalPort int    `json:"external_port,omitempty"`
	URL          string `json:"url,omitempty"`
	Status       string `json:"status,omitempty"`
}

// Execute runs all scripts for a given event
func (e *ScriptExecutor) Execute(ctx context.Context, event EventType, hookCtx *HookContext) error {
	// Build event directory path (e.g., .space/hooks/post-up.d/)
	eventDir := filepath.Join(e.HooksDir, string(event)+".d")

	// Check if directory exists
	if _, err := os.Stat(eventDir); os.IsNotExist(err) {
		return nil // No hooks for this event
	}

	// Find all executable scripts
	scripts, err := e.findScripts(eventDir)
	if err != nil {
		return fmt.Errorf("failed to find scripts: %w", err)
	}

	if len(scripts) == 0 {
		return nil
	}

	e.Logger.Info("Running %d %s hook(s)...", len(scripts), event)

	// Build context JSON
	contextJSON, err := e.buildContextJSON(event, hookCtx)
	if err != nil {
		return fmt.Errorf("failed to build context: %w", err)
	}

	// Build environment variables
	env := e.buildEnvironment(hookCtx)

	// Execute each script in order
	for _, script := range scripts {
		scriptName := filepath.Base(script)
		e.Logger.Info("Running %s...", scriptName)

		if err := e.runScript(ctx, script, contextJSON, env, hookCtx.WorkDir); err != nil {
			e.Logger.Error("%s failed: %v", scriptName, err)
			// Continue with other scripts unless it's critical
			// Could add a "fail-fast" option later
		}
	}

	return nil
}

// findScripts finds all executable scripts in a directory, sorted by name
func (e *ScriptExecutor) findScripts(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var scripts []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()

		// Skip templates and non-executable files
		if strings.HasSuffix(name, ".template") ||
			strings.HasSuffix(name, ".md") ||
			strings.HasSuffix(name, ".txt") {
			continue
		}

		path := filepath.Join(dir, name)

		// Check if file is executable
		info, err := entry.Info()
		if err != nil {
			continue
		}

		// On Unix, check executable bit
		if info.Mode()&0111 != 0 {
			scripts = append(scripts, path)
		}
	}

	// Sort by name (allows ordering like 10-first.sh, 20-second.sh)
	sort.Strings(scripts)

	return scripts, nil
}

// buildContextJSON creates the JSON context for hook scripts
func (e *ScriptExecutor) buildContextJSON(event EventType, hookCtx *HookContext) ([]byte, error) {
	ctx := HookContextJSON{
		Event:       string(event),
		WorkDir:     hookCtx.WorkDir,
		ProjectName: hookCtx.ProjectName,
		Hash:        hookCtx.Hash,
		BaseDomain:  hookCtx.BaseDomain,
		DNSEnabled:  hookCtx.DNSEnabled,
		DNSAddress:  hookCtx.DNSAddress,
		Services:    make(map[string]ServiceInfoJSON),
		Metadata:    hookCtx.Metadata,
	}

	for name, svc := range hookCtx.Services {
		ctx.Services[name] = ServiceInfoJSON{
			Name:         svc.Name,
			DNSName:      svc.DNSName,
			InternalPort: svc.InternalPort,
			ExternalPort: svc.ExternalPort,
			URL:          svc.URL,
			Status:       svc.Status,
		}
	}

	return json.MarshalIndent(ctx, "", "  ")
}

// buildEnvironment creates environment variables for hook scripts
func (e *ScriptExecutor) buildEnvironment(hookCtx *HookContext) []string {
	env := os.Environ()

	// Add space-specific variables
	env = append(env,
		"SPACE_WORKDIR="+hookCtx.WorkDir,
		"SPACE_PROJECT_NAME="+hookCtx.ProjectName,
		"SPACE_HASH="+hookCtx.Hash,
		"SPACE_BASE_DOMAIN="+hookCtx.BaseDomain,
		fmt.Sprintf("SPACE_DNS_ENABLED=%t", hookCtx.DNSEnabled),
	)

	if hookCtx.DNSAddress != "" {
		env = append(env, "SPACE_DNS_ADDRESS="+hookCtx.DNSAddress)
	}

	// Add service-specific variables
	for name, svc := range hookCtx.Services {
		prefix := "SPACE_SERVICE_" + strings.ToUpper(strings.ReplaceAll(name, "-", "_"))

		if svc.DNSName != "" {
			env = append(env, prefix+"_DNS_NAME="+svc.DNSName)
		}
		if svc.InternalPort > 0 {
			env = append(env, fmt.Sprintf("%s_PORT=%d", prefix, svc.InternalPort))
		}
		if svc.URL != "" {
			env = append(env, prefix+"_URL="+svc.URL)
		}
	}

	return env
}

// runScript executes a single script
func (e *ScriptExecutor) runScript(ctx context.Context, script string, contextJSON []byte, env []string, workDir string) error {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, e.Timeout)
	defer cancel()

	// Determine interpreter based on shebang or extension
	interpreter, args := e.getInterpreter(script)

	cmdArgs := append(args, script)
	cmd := exec.CommandContext(ctx, interpreter, cmdArgs...)
	cmd.Dir = workDir
	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Pass context JSON via stdin
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start script: %w", err)
	}

	// Write context JSON to stdin
	if _, err := stdin.Write(contextJSON); err != nil {
		stdin.Close()
		return fmt.Errorf("failed to write context: %w", err)
	}
	stdin.Close()

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("script failed: %w", err)
	}

	return nil
}

// getInterpreter determines the interpreter for a script
func (e *ScriptExecutor) getInterpreter(script string) (string, []string) {
	ext := filepath.Ext(script)

	switch ext {
	case ".py":
		return "python3", nil
	case ".rb":
		return "ruby", nil
	case ".js":
		return "node", nil
	case ".ts":
		return "npx", []string{"ts-node"}
	default:
		// Default to sh for .sh or unknown
		return "sh", nil
	}
}

// InitHooksDir creates the hooks directory structure with README
func InitHooksDir(workDir string) error {
	hooksDir := filepath.Join(workDir, ".space", "hooks")

	// Create event directories
	events := []string{"pre-up", "post-up", "pre-down", "post-down", "on-dns-ready"}
	for _, event := range events {
		eventDir := filepath.Join(hooksDir, event+".d")
		if err := os.MkdirAll(eventDir, 0755); err != nil {
			return fmt.Errorf("failed to create %s: %w", eventDir, err)
		}

		// Create .gitkeep
		gitkeep := filepath.Join(eventDir, ".gitkeep")
		if _, err := os.Stat(gitkeep); os.IsNotExist(err) {
			if err := os.WriteFile(gitkeep, []byte(""), 0644); err != nil {
				return err
			}
		}
	}

	// Create README
	readme := filepath.Join(hooksDir, "README.md")
	if _, err := os.Stat(readme); os.IsNotExist(err) {
		if err := os.WriteFile(readme, []byte(hooksReadme), 0644); err != nil {
			return err
		}
	}

	return nil
}

const hooksReadme = `# Space Hooks

Scripts in these directories are executed during space lifecycle events.

## Directory Structure

` + "```" + `
.space/hooks/
├── pre-up.d/      # Before docker compose up
├── post-up.d/     # After services are running
├── pre-down.d/    # Before docker compose down
├── post-down.d/   # After services are stopped
└── on-dns-ready.d/ # When DNS is configured
` + "```" + `

## Writing Hooks

1. Create an executable script in the appropriate directory
2. Name it with a numeric prefix for ordering (e.g., ` + "`10-setup.sh`" + `)
3. Make it executable: ` + "`chmod +x your-script.sh`" + `

## Context

Scripts receive context in two ways:

### Environment Variables

` + "```bash" + `
SPACE_WORKDIR=/path/to/project
SPACE_PROJECT_NAME=myproject-main
SPACE_HASH=6a9c8c
SPACE_BASE_DOMAIN=space.local
SPACE_DNS_ENABLED=true

# Per-service variables
SPACE_SERVICE_POSTGRES_DNS_NAME=postgres-6a9c8c.space.local
SPACE_SERVICE_POSTGRES_PORT=5432
SPACE_SERVICE_API_SERVER_DNS_NAME=api-server-6a9c8c.space.local
SPACE_SERVICE_API_SERVER_PORT=6060
` + "```" + `

### JSON on stdin

` + "```json" + `
{
  "event": "post-up",
  "work_dir": "/path/to/project",
  "project_name": "myproject-main",
  "hash": "6a9c8c",
  "base_domain": "space.local",
  "dns_enabled": true,
  "services": {
    "postgres": {
      "name": "postgres",
      "dns_name": "postgres-6a9c8c.space.local",
      "internal_port": 5432
    }
  }
}
` + "```" + `

## Example: Vite Environment Setup

` + "```bash" + `
#!/bin/bash
# .space/hooks/post-up.d/10-vite-env.sh

# Read JSON context
CONTEXT=$(cat)

# Check if this is a Vite project
if [[ ! -f "vite.config.ts" ]] && [[ ! -f "vite.config.js" ]]; then
  exit 0
fi

echo "🎨 Configuring Vite environment..."

# Extract service info using jq
API_HOST=$(echo "$CONTEXT" | jq -r '.services["api-server"].dns_name // empty')
API_PORT=$(echo "$CONTEXT" | jq -r '.services["api-server"].internal_port // empty')

if [[ -n "$API_HOST" ]] && [[ -n "$API_PORT" ]]; then
  cat > .env.development.local << EOF
# Auto-generated by space hooks
VITE_API_BASE_URL=http://${API_HOST}:${API_PORT}
EOF
  echo "✅ Generated .env.development.local"
fi
` + "```" + `

## Example: River Database Setup

` + "```bash" + `
#!/bin/bash
# .space/hooks/post-up.d/20-river-db.sh

CONTEXT=$(cat)

# Check if river CLI is available
if ! command -v river &> /dev/null; then
  exit 0
fi

# Check if postgres service exists
PG_HOST=$(echo "$CONTEXT" | jq -r '.services["postgres"].dns_name // empty')
if [[ -z "$PG_HOST" ]]; then
  exit 0
fi

echo "🗄️ Setting up River database..."

DATABASE_URL="postgres://admin:test@${PG_HOST}:5432/river"

# Wait for postgres
for i in {1..30}; do
  if psql "$DATABASE_URL" -c "SELECT 1" &>/dev/null; then
    break
  fi
  sleep 1
done

# Create database if needed
psql "postgres://admin:test@${PG_HOST}:5432/postgres" -c "CREATE DATABASE river" 2>/dev/null || true

# Run migrations
river migrate-up --database-url "$DATABASE_URL"
echo "✅ River database ready"
` + "```" + `
`
