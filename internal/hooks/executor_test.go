package hooks

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestNewScriptExecutor(t *testing.T) {
	executor := NewScriptExecutor("/tmp/test")

	if executor.HooksDir != "/tmp/test/.space/hooks" {
		t.Errorf("Expected HooksDir '/tmp/test/.space/hooks', got %s", executor.HooksDir)
	}

	if executor.Timeout != 5*60*1000000000 { // 5 minutes in nanoseconds
		t.Errorf("Expected 5 minute timeout, got %v", executor.Timeout)
	}
}

func TestScriptExecutor_NoHooksDir(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "hooks-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	executor := NewScriptExecutor(tmpDir)
	hookCtx := NewHookContext()
	hookCtx.WorkDir = tmpDir

	// Should not error when hooks dir doesn't exist
	err = executor.Execute(context.Background(), PostUp, hookCtx)
	if err != nil {
		t.Errorf("Expected no error when hooks dir doesn't exist, got: %v", err)
	}
}

func TestScriptExecutor_EmptyHooksDir(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "hooks-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create empty hooks directory
	eventDir := filepath.Join(tmpDir, ".space", "hooks", "post-up.d")
	if err := os.MkdirAll(eventDir, 0755); err != nil {
		t.Fatalf("Failed to create hooks dir: %v", err)
	}

	executor := NewScriptExecutor(tmpDir)
	hookCtx := NewHookContext()
	hookCtx.WorkDir = tmpDir

	err = executor.Execute(context.Background(), PostUp, hookCtx)
	if err != nil {
		t.Errorf("Expected no error for empty hooks dir, got: %v", err)
	}
}

func TestScriptExecutor_SkipsNonExecutable(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "hooks-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create hooks directory with non-executable file
	eventDir := filepath.Join(tmpDir, ".space", "hooks", "post-up.d")
	if err := os.MkdirAll(eventDir, 0755); err != nil {
		t.Fatalf("Failed to create hooks dir: %v", err)
	}

	// Create non-executable script
	scriptPath := filepath.Join(eventDir, "10-test.sh")
	if err := os.WriteFile(scriptPath, []byte("#!/bin/bash\necho test"), 0644); err != nil {
		t.Fatalf("Failed to create script: %v", err)
	}

	executor := NewScriptExecutor(tmpDir)
	scripts, err := executor.findScripts(eventDir)
	if err != nil {
		t.Fatalf("findScripts failed: %v", err)
	}

	if len(scripts) != 0 {
		t.Errorf("Expected 0 scripts (non-executable), got %d", len(scripts))
	}
}

func TestScriptExecutor_FindsExecutableScripts(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "hooks-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create hooks directory
	eventDir := filepath.Join(tmpDir, ".space", "hooks", "post-up.d")
	if err := os.MkdirAll(eventDir, 0755); err != nil {
		t.Fatalf("Failed to create hooks dir: %v", err)
	}

	// Create executable scripts
	scripts := []string{"10-first.sh", "20-second.sh", "05-before.sh"}
	for _, name := range scripts {
		path := filepath.Join(eventDir, name)
		if err := os.WriteFile(path, []byte("#!/bin/bash\necho "+name), 0755); err != nil {
			t.Fatalf("Failed to create script: %v", err)
		}
	}

	// Create template (should be skipped)
	templatePath := filepath.Join(eventDir, "30-template.sh.template")
	if err := os.WriteFile(templatePath, []byte("#!/bin/bash"), 0755); err != nil {
		t.Fatalf("Failed to create template: %v", err)
	}

	executor := NewScriptExecutor(tmpDir)
	found, err := executor.findScripts(eventDir)
	if err != nil {
		t.Fatalf("findScripts failed: %v", err)
	}

	if len(found) != 3 {
		t.Errorf("Expected 3 scripts, got %d", len(found))
	}

	// Should be sorted by name
	expectedOrder := []string{"05-before.sh", "10-first.sh", "20-second.sh"}
	for i, expected := range expectedOrder {
		actual := filepath.Base(found[i])
		if actual != expected {
			t.Errorf("Script %d: expected %s, got %s", i, expected, actual)
		}
	}
}

func TestScriptExecutor_BuildEnvironment(t *testing.T) {
	executor := NewScriptExecutor("/tmp/test")

	hookCtx := &HookContext{
		WorkDir:     "/path/to/project",
		ProjectName: "myproject",
		Hash:        "abc123",
		BaseDomain:  "space.local",
		DNSEnabled:  true,
		DNSAddress:  "127.0.0.1:5353",
		Services: map[string]*ServiceInfo{
			"api-server": {
				Name:         "api-server",
				DNSName:      "api-server-abc123.space.local",
				InternalPort: 6060,
				URL:          "http://api-server-abc123.space.local:6060",
			},
		},
	}

	env := executor.buildEnvironment(hookCtx)

	// Convert to map for easier checking
	envMap := make(map[string]string)
	for _, e := range env {
		parts := filepath.SplitList(e)
		if len(parts) >= 1 {
			// Find the = sign
			for i, c := range e {
				if c == '=' {
					envMap[e[:i]] = e[i+1:]
					break
				}
			}
		}
	}

	// Check expected variables
	checks := map[string]string{
		"SPACE_WORKDIR":                    "/path/to/project",
		"SPACE_PROJECT_NAME":               "myproject",
		"SPACE_HASH":                       "abc123",
		"SPACE_BASE_DOMAIN":                "space.local",
		"SPACE_DNS_ENABLED":                "true",
		"SPACE_DNS_ADDRESS":                "127.0.0.1:5353",
		"SPACE_SERVICE_API_SERVER_DNS_NAME": "api-server-abc123.space.local",
		"SPACE_SERVICE_API_SERVER_PORT":    "6060",
		"SPACE_SERVICE_API_SERVER_URL":     "http://api-server-abc123.space.local:6060",
	}

	for key, expected := range checks {
		actual, ok := envMap[key]
		if !ok {
			t.Errorf("Expected environment variable %s not found", key)
			continue
		}
		if actual != expected {
			t.Errorf("Environment variable %s: expected %q, got %q", key, expected, actual)
		}
	}
}

func TestScriptExecutor_GetInterpreter(t *testing.T) {
	executor := NewScriptExecutor("/tmp/test")

	tests := []struct {
		script      string
		interpreter string
	}{
		{"test.sh", "sh"},
		{"test.py", "python3"},
		{"test.rb", "ruby"},
		{"test.js", "node"},
		{"test", "sh"}, // default
	}

	for _, tc := range tests {
		interpreter, _ := executor.getInterpreter(tc.script)
		if interpreter != tc.interpreter {
			t.Errorf("Script %s: expected interpreter %s, got %s", tc.script, tc.interpreter, interpreter)
		}
	}
}

func TestInitHooksDir(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "hooks-init-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	err = InitHooksDir(tmpDir)
	if err != nil {
		t.Fatalf("InitHooksDir failed: %v", err)
	}

	// Check directories were created
	events := []string{"pre-up", "post-up", "pre-down", "post-down", "on-dns-ready"}
	for _, event := range events {
		dir := filepath.Join(tmpDir, ".space", "hooks", event+".d")
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			t.Errorf("Expected directory %s to exist", dir)
		}

		// Check .gitkeep exists
		gitkeep := filepath.Join(dir, ".gitkeep")
		if _, err := os.Stat(gitkeep); os.IsNotExist(err) {
			t.Errorf("Expected .gitkeep in %s", dir)
		}
	}

	// Check README exists
	readme := filepath.Join(tmpDir, ".space", "hooks", "README.md")
	if _, err := os.Stat(readme); os.IsNotExist(err) {
		t.Errorf("Expected README.md to exist")
	}
}

func TestBuildContextJSON(t *testing.T) {
	executor := NewScriptExecutor("/tmp/test")

	hookCtx := &HookContext{
		WorkDir:     "/path/to/project",
		ProjectName: "myproject",
		Hash:        "abc123",
		BaseDomain:  "space.local",
		DNSEnabled:  true,
		Services: map[string]*ServiceInfo{
			"postgres": {
				Name:         "postgres",
				DNSName:      "postgres-abc123.space.local",
				InternalPort: 5432,
			},
		},
		Metadata: map[string]interface{}{
			"custom_key": "custom_value",
		},
	}

	jsonData, err := executor.buildContextJSON(PostUp, hookCtx)
	if err != nil {
		t.Fatalf("buildContextJSON failed: %v", err)
	}

	// Basic validation that JSON is valid and contains expected fields
	jsonStr := string(jsonData)
	expectedFields := []string{
		`"event": "post-up"`,
		`"project_name": "myproject"`,
		`"hash": "abc123"`,
		`"base_domain": "space.local"`,
		`"dns_enabled": true`,
		`"postgres"`,
		`"internal_port": 5432`,
	}

	for _, field := range expectedFields {
		if !contains(jsonStr, field) {
			t.Errorf("Expected JSON to contain %s", field)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
