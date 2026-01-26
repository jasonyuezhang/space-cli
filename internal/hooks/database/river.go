package database

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/happy-sdk/space-cli/internal/dns"
	"github.com/happy-sdk/space-cli/internal/hooks"
)

// RiverHook handles River queue database setup
type RiverHook struct {
	// PostgresService is the name of the postgres service (default: "postgres")
	PostgresService string

	// DatabaseName is the River database name (default: "river")
	DatabaseName string

	// Username for postgres connection
	Username string

	// Password for postgres connection
	Password string

	// Port for postgres (default: 5432)
	Port int

	// MaxRetries for database connection (default: 10)
	MaxRetries int

	// RetryDelay between connection attempts (default: 2s)
	RetryDelay time.Duration
}

// NewRiverHook creates a new River database hook with defaults
func NewRiverHook() *RiverHook {
	return &RiverHook{
		PostgresService: "postgres",
		DatabaseName:    "river",
		Username:        "admin",
		Password:        "test",
		Port:            5432,
		MaxRetries:      10,
		RetryDelay:      2 * time.Second,
	}
}

// Name returns the hook name
func (h *RiverHook) Name() string {
	return "river-database"
}

// Description returns the hook description
func (h *RiverHook) Description() string {
	return "Creates River queue database and runs migrations"
}

// Events returns the events this hook handles
func (h *RiverHook) Events() []hooks.EventType {
	return []hooks.EventType{hooks.PostUp}
}

// Priority returns the hook priority (run after services are up but early in post-up)
func (h *RiverHook) Priority() hooks.Priority {
	return hooks.PriorityHigh
}

// ShouldExecute checks if this hook should run
func (h *RiverHook) ShouldExecute(ctx context.Context, event hooks.EventType, hookCtx *hooks.HookContext) bool {
	// Only run if DNS is enabled
	if !hookCtx.DNSEnabled {
		return false
	}

	// Check if postgres service exists
	_, hasPostgres := hookCtx.Services[h.PostgresService]
	return hasPostgres
}

// Execute runs the River database setup
func (h *RiverHook) Execute(ctx context.Context, event hooks.EventType, hookCtx *hooks.HookContext) error {
	// Get postgres service info
	pgService, ok := hookCtx.Services[h.PostgresService]
	if !ok {
		return fmt.Errorf("postgres service %q not found", h.PostgresService)
	}

	// Build postgres host from DNS name or generate it
	pgHost := pgService.DNSName
	if pgHost == "" {
		// Generate DNS name using hash
		hash := hookCtx.Hash
		if hash == "" {
			hash = dns.GenerateDirectoryHash(hookCtx.WorkDir)
		}
		pgHost = fmt.Sprintf("%s-%s.%s", h.PostgresService, hash, hookCtx.BaseDomain)
	}

	// Determine port
	port := h.Port
	if pgService.InternalPort > 0 {
		port = pgService.InternalPort
	}

	// Build connection URLs
	adminURL := fmt.Sprintf("postgres://%s:%s@%s:%d/postgres",
		h.Username, h.Password, pgHost, port)
	riverURL := fmt.Sprintf("postgres://%s:%s@%s:%d/%s",
		h.Username, h.Password, pgHost, port, h.DatabaseName)

	fmt.Printf("🗄️  Setting up River database on %s:%d...\n", pgHost, port)

	// Wait for postgres to be ready
	if err := h.waitForPostgres(ctx, adminURL); err != nil {
		return fmt.Errorf("postgres not ready: %w", err)
	}

	// Create river database if it doesn't exist
	if err := h.createDatabase(ctx, adminURL); err != nil {
		return fmt.Errorf("failed to create database: %w", err)
	}

	// Run river migrations
	if err := h.runMigrations(ctx, riverURL); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	fmt.Printf("   ✅ River database ready at %s:%d/%s\n", pgHost, port, h.DatabaseName)

	// Store connection info in metadata for other hooks
	hookCtx.SetMetadata("river.database_url", riverURL)
	hookCtx.SetMetadata("river.host", pgHost)
	hookCtx.SetMetadata("river.port", port)

	return nil
}

// waitForPostgres waits for postgres to accept connections
func (h *RiverHook) waitForPostgres(ctx context.Context, connURL string) error {
	fmt.Printf("   ⏳ Waiting for postgres to be ready...\n")

	for i := 0; i < h.MaxRetries; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Try to connect using psql
		cmd := exec.CommandContext(ctx, "psql", connURL, "-c", "SELECT 1")
		if err := cmd.Run(); err == nil {
			fmt.Printf("   ✅ Postgres is ready\n")
			return nil
		}

		fmt.Printf("   ⏳ Postgres not ready, retrying in %v... (%d/%d)\n",
			h.RetryDelay, i+1, h.MaxRetries)
		time.Sleep(h.RetryDelay)
	}

	return fmt.Errorf("postgres not ready after %d attempts", h.MaxRetries)
}

// createDatabase creates the river database if it doesn't exist
func (h *RiverHook) createDatabase(ctx context.Context, adminURL string) error {
	fmt.Printf("   📦 Creating database '%s' if not exists...\n", h.DatabaseName)

	// Check if database exists
	checkCmd := exec.CommandContext(ctx, "psql", adminURL, "-tAc",
		fmt.Sprintf("SELECT 1 FROM pg_database WHERE datname='%s'", h.DatabaseName))
	output, err := checkCmd.Output()
	if err == nil && strings.TrimSpace(string(output)) == "1" {
		fmt.Printf("   ℹ️  Database '%s' already exists\n", h.DatabaseName)
		return nil
	}

	// Create database
	createCmd := exec.CommandContext(ctx, "psql", adminURL, "-c",
		fmt.Sprintf("CREATE DATABASE %s", h.DatabaseName))
	if output, err := createCmd.CombinedOutput(); err != nil {
		// Check if it's just "already exists" error
		if strings.Contains(string(output), "already exists") {
			fmt.Printf("   ℹ️  Database '%s' already exists\n", h.DatabaseName)
			return nil
		}
		return fmt.Errorf("create database failed: %s", string(output))
	}

	fmt.Printf("   ✅ Database '%s' created\n", h.DatabaseName)
	return nil
}

// runMigrations runs river migrate-up
func (h *RiverHook) runMigrations(ctx context.Context, riverURL string) error {
	fmt.Printf("   🔄 Running river migrations...\n")

	// Check if river CLI is available
	if _, err := exec.LookPath("river"); err != nil {
		return fmt.Errorf("river CLI not found in PATH - install with: go install github.com/riverqueue/river/cmd/river@latest")
	}

	// Run migrations
	cmd := exec.CommandContext(ctx, "river", "migrate-up", "--database-url", riverURL)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("river migrate-up failed: %s", string(output))
	}

	fmt.Printf("   ✅ River migrations completed\n")
	return nil
}

// GetDatabaseURL returns the River database URL for a given context
func GetRiverDatabaseURL(hookCtx *hooks.HookContext, postgresService, username, password, database string, port int) string {
	hash := hookCtx.Hash
	if hash == "" {
		hash = dns.GenerateDirectoryHash(hookCtx.WorkDir)
	}

	pgHost := fmt.Sprintf("%s-%s.%s", postgresService, hash, hookCtx.BaseDomain)
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s", username, password, pgHost, port, database)
}
