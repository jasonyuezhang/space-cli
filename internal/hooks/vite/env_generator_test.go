package vite

import (
	"os"
	"strings"
	"testing"

	"github.com/happy-sdk/space-cli/pkg/config"
)

func TestEnvGenerator_Generate(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "vite-env-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	generator, err := NewEnvGenerator(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create generator: %v", err)
	}

	cfg := &config.Config{
		Services: map[string]config.ServiceConfig{
			"api-server": {Port: 6060},
			"app":        {Port: 3000},
		},
	}

	result, err := generator.Generate(cfg)
	if err != nil {
		t.Fatalf("Generation failed: %v", err)
	}

	if !result.Generated {
		t.Error("Expected Generated to be true")
	}

	// Check that env file was created
	if _, err := os.Stat(result.FilePath); os.IsNotExist(err) {
		t.Errorf("Env file was not created at %s", result.FilePath)
	}

	// Read and verify content
	content, err := os.ReadFile(result.FilePath)
	if err != nil {
		t.Fatalf("Failed to read env file: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "VITE_API_BASE_URL=") {
		t.Error("Expected VITE_API_BASE_URL in env file")
	}
	if !strings.Contains(contentStr, "space.local") {
		t.Error("Expected space.local domain in env file")
	}
}

func TestEnvGenerator_GenerateWithServices(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "vite-env-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	generator, err := NewEnvGenerator(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create generator: %v", err)
	}

	services := []ServiceEnvConfig{
		{ServiceName: "api-server", Port: 6060, EnvVarName: "VITE_API_BASE_URL"},
		{ServiceName: "app", Port: 3000, EnvVarName: "VITE_APP_BASE_URL"},
	}

	result, err := generator.GenerateWithServices(services)
	if err != nil {
		t.Fatalf("Generation failed: %v", err)
	}

	if !result.Generated {
		t.Error("Expected Generated to be true")
	}

	// Verify specific env vars
	if _, ok := result.Variables["VITE_API_BASE_URL"]; !ok {
		t.Error("Expected VITE_API_BASE_URL in variables")
	}
	if _, ok := result.Variables["VITE_APP_BASE_URL"]; !ok {
		t.Error("Expected VITE_APP_BASE_URL in variables")
	}

	// Check URL format
	apiURL := result.Variables["VITE_API_BASE_URL"]
	if !strings.Contains(apiURL, "api-server") || !strings.Contains(apiURL, ":6060") {
		t.Errorf("Unexpected API URL format: %s", apiURL)
	}
}

func TestEnvGenerator_Hash(t *testing.T) {
	tmpDir1, _ := os.MkdirTemp("", "vite-hash-test1-*")
	tmpDir2, _ := os.MkdirTemp("", "vite-hash-test2-*")
	defer os.RemoveAll(tmpDir1)
	defer os.RemoveAll(tmpDir2)

	gen1, _ := NewEnvGenerator(tmpDir1)
	gen2, _ := NewEnvGenerator(tmpDir2)

	// Different directories should produce different hashes
	if gen1.Hash() == gen2.Hash() {
		t.Error("Different directories should have different hashes")
	}

	// Same directory should produce same hash
	gen1Again, _ := NewEnvGenerator(tmpDir1)
	if gen1.Hash() != gen1Again.Hash() {
		t.Error("Same directory should have same hash")
	}

	// Hash should be 6 characters
	if len(gen1.Hash()) != 6 {
		t.Errorf("Expected hash length 6, got %d", len(gen1.Hash()))
	}
}

func TestEnvGenerator_URLFormat(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "vite-url-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	generator, err := NewEnvGeneratorWithHash(tmpDir, "abc123")
	if err != nil {
		t.Fatalf("Failed to create generator: %v", err)
	}

	services := []ServiceEnvConfig{
		{ServiceName: "api-server", Port: 6060, EnvVarName: "VITE_API_BASE_URL"},
	}

	result, err := generator.GenerateWithServices(services)
	if err != nil {
		t.Fatalf("Generation failed: %v", err)
	}

	expectedURL := "http://api-server-abc123.space.local:6060"
	if result.Variables["VITE_API_BASE_URL"] != expectedURL {
		t.Errorf("Expected URL %s, got %s", expectedURL, result.Variables["VITE_API_BASE_URL"])
	}
}
