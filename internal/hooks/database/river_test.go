package database

import (
	"context"
	"testing"

	"github.com/happy-sdk/space-cli/internal/hooks"
)

func TestRiverHook_Name(t *testing.T) {
	hook := NewRiverHook()
	if hook.Name() != "river-database" {
		t.Errorf("Expected name 'river-database', got %s", hook.Name())
	}
}

func TestRiverHook_Events(t *testing.T) {
	hook := NewRiverHook()
	events := hook.Events()

	if len(events) != 1 {
		t.Errorf("Expected 1 event, got %d", len(events))
	}
	if events[0] != hooks.PostUp {
		t.Errorf("Expected PostUp event, got %s", events[0])
	}
}

func TestRiverHook_Priority(t *testing.T) {
	hook := NewRiverHook()
	if hook.Priority() != hooks.PriorityHigh {
		t.Errorf("Expected PriorityHigh, got %d", hook.Priority())
	}
}

func TestRiverHook_ShouldExecute_DNSDisabled(t *testing.T) {
	hook := NewRiverHook()
	ctx := context.Background()

	hookCtx := &hooks.HookContext{
		DNSEnabled: false,
		Services:   make(map[string]*hooks.ServiceInfo),
	}
	hookCtx.Services["postgres"] = &hooks.ServiceInfo{
		Name:         "postgres",
		InternalPort: 5432,
	}

	if hook.ShouldExecute(ctx, hooks.PostUp, hookCtx) {
		t.Error("Expected ShouldExecute to return false when DNS is disabled")
	}
}

func TestRiverHook_ShouldExecute_NoPostgres(t *testing.T) {
	hook := NewRiverHook()
	ctx := context.Background()

	hookCtx := &hooks.HookContext{
		DNSEnabled: true,
		Services:   make(map[string]*hooks.ServiceInfo),
	}
	hookCtx.Services["web"] = &hooks.ServiceInfo{
		Name:         "web",
		InternalPort: 8080,
	}

	if hook.ShouldExecute(ctx, hooks.PostUp, hookCtx) {
		t.Error("Expected ShouldExecute to return false when postgres is not present")
	}
}

func TestRiverHook_ShouldExecute_WithPostgres(t *testing.T) {
	hook := NewRiverHook()
	ctx := context.Background()

	hookCtx := &hooks.HookContext{
		DNSEnabled: true,
		Services:   make(map[string]*hooks.ServiceInfo),
	}
	hookCtx.Services["postgres"] = &hooks.ServiceInfo{
		Name:         "postgres",
		InternalPort: 5432,
	}

	if !hook.ShouldExecute(ctx, hooks.PostUp, hookCtx) {
		t.Error("Expected ShouldExecute to return true when postgres is present and DNS enabled")
	}
}

func TestRiverHook_Defaults(t *testing.T) {
	hook := NewRiverHook()

	if hook.PostgresService != "postgres" {
		t.Errorf("Expected PostgresService 'postgres', got %s", hook.PostgresService)
	}
	if hook.DatabaseName != "river" {
		t.Errorf("Expected DatabaseName 'river', got %s", hook.DatabaseName)
	}
	if hook.Username != "admin" {
		t.Errorf("Expected Username 'admin', got %s", hook.Username)
	}
	if hook.Password != "test" {
		t.Errorf("Expected Password 'test', got %s", hook.Password)
	}
	if hook.Port != 5432 {
		t.Errorf("Expected Port 5432, got %d", hook.Port)
	}
	if hook.MaxRetries != 10 {
		t.Errorf("Expected MaxRetries 10, got %d", hook.MaxRetries)
	}
}

func TestGetRiverDatabaseURL(t *testing.T) {
	hookCtx := &hooks.HookContext{
		WorkDir:    "/tmp/test-project",
		BaseDomain: "space.local",
		Hash:       "abc123",
	}

	url := GetRiverDatabaseURL(hookCtx, "postgres", "admin", "test", "river", 5432)
	expected := "postgres://admin:test@postgres-abc123.space.local:5432/river"

	if url != expected {
		t.Errorf("Expected URL %s, got %s", expected, url)
	}
}

func TestGetRiverDatabaseURL_GeneratesHash(t *testing.T) {
	hookCtx := &hooks.HookContext{
		WorkDir:    "/tmp/unique-project-path",
		BaseDomain: "space.local",
		Hash:       "", // Empty hash should be generated
	}

	url := GetRiverDatabaseURL(hookCtx, "postgres", "admin", "test", "river", 5432)

	// Should contain generated hash
	if url == "postgres://admin:test@postgres-.space.local:5432/river" {
		t.Error("Expected hash to be generated when empty")
	}

	// Should match expected format
	if len(url) < 50 {
		t.Errorf("URL seems too short, got: %s", url)
	}
}
