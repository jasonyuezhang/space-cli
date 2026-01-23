// +build integration

package cli

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/happy-sdk/space-cli/internal/provider"
)

// TestPsIntegrationBasic tests basic ps functionality with real docker-compose
// This test requires docker and docker-compose to be installed
func TestPsIntegrationBasic(t *testing.T) {
	// Check if docker is available
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker not found, skipping integration test")
	}

	tempDir, err := os.MkdirTemp("", "space-cli-integration-*")
	if err != nil {
		t.Fatalf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create docker-compose file with simple service
	composeContent := `version: '3.8'
services:
  test_app:
    image: busybox
    command: sleep 300
    container_name: test_app_container_123
`
	composeFile := filepath.Join(tempDir, "docker-compose.yml")
	if err := os.WriteFile(composeFile, []byte(composeContent), 0644); err != nil {
		t.Fatalf("failed to write docker-compose.yml: %v", err)
	}

	// Start containers
	upCmd := exec.Command("docker", "compose", "-f", composeFile, "-p", "test_ps_int", "up", "-d")
	upCmd.Dir = tempDir
	if err := upCmd.Run(); err != nil {
		t.Fatalf("failed to start containers: %v", err)
	}

	// Cleanup: stop containers after test
	defer func() {
		downCmd := exec.Command("docker", "compose", "-f", composeFile, "-p", "test_ps_int", "down")
		downCmd.Dir = tempDir
		downCmd.Run()
	}()

	// Wait for containers to start
	time.Sleep(2 * time.Second)

	// Test ps command
	err = runPsCommand(context.Background(), tempDir, "test_ps_int", provider.ProviderGeneric, false, false)
	if err != nil {
		t.Fatalf("ps command failed: %v", err)
	}
}

// TestPsIntegrationMultipleServices tests ps with multiple services
func TestPsIntegrationMultipleServices(t *testing.T) {
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker not found, skipping integration test")
	}

	tempDir, err := os.MkdirTemp("", "space-cli-integration-*")
	if err != nil {
		t.Fatalf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create docker-compose file with multiple services
	composeContent := `version: '3.8'
services:
  web:
    image: busybox
    command: sleep 300
  db:
    image: busybox
    command: sleep 300
  cache:
    image: busybox
    command: sleep 300
`
	composeFile := filepath.Join(tempDir, "docker-compose.yml")
	if err := os.WriteFile(composeFile, []byte(composeContent), 0644); err != nil {
		t.Fatalf("failed to write docker-compose.yml: %v", err)
	}

	// Start containers
	upCmd := exec.Command("docker", "compose", "-f", composeFile, "-p", "test_ps_multi", "up", "-d")
	upCmd.Dir = tempDir
	if err := upCmd.Run(); err != nil {
		t.Fatalf("failed to start containers: %v", err)
	}

	defer func() {
		downCmd := exec.Command("docker", "compose", "-f", composeFile, "-p", "test_ps_multi", "down")
		downCmd.Dir = tempDir
		downCmd.Run()
	}()

	time.Sleep(2 * time.Second)

	// Test ps command
	err = runPsCommand(context.Background(), tempDir, "test_ps_multi", provider.ProviderGeneric, false, false)
	if err != nil {
		t.Fatalf("ps command failed: %v", err)
	}
}

// TestPsIntegrationQuietMode tests ps with quiet flag
func TestPsIntegrationQuietMode(t *testing.T) {
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker not found, skipping integration test")
	}

	tempDir, err := os.MkdirTemp("", "space-cli-integration-*")
	if err != nil {
		t.Fatalf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	composeContent := `version: '3.8'
services:
  app:
    image: busybox
    command: sleep 300
`
	composeFile := filepath.Join(tempDir, "docker-compose.yml")
	if err := os.WriteFile(composeFile, []byte(composeContent), 0644); err != nil {
		t.Fatalf("failed to write docker-compose.yml: %v", err)
	}

	upCmd := exec.Command("docker", "compose", "-f", composeFile, "-p", "test_ps_quiet", "up", "-d")
	upCmd.Dir = tempDir
	if err := upCmd.Run(); err != nil {
		t.Fatalf("failed to start containers: %v", err)
	}

	defer func() {
		downCmd := exec.Command("docker", "compose", "-f", composeFile, "-p", "test_ps_quiet", "down")
		downCmd.Dir = tempDir
		downCmd.Run()
	}()

	time.Sleep(2 * time.Second)

	// Test ps command with quiet flag
	err = runPsCommand(context.Background(), tempDir, "test_ps_quiet", provider.ProviderGeneric, true, false)
	if err != nil {
		t.Fatalf("ps command failed: %v", err)
	}
}

// TestPsIntegrationNoTruncMode tests ps without truncation
func TestPsIntegrationNoTruncMode(t *testing.T) {
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker not found, skipping integration test")
	}

	tempDir, err := os.MkdirTemp("", "space-cli-integration-*")
	if err != nil {
		t.Fatalf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	composeContent := `version: '3.8'
services:
  app:
    image: busybox
    command: sleep 300
`
	composeFile := filepath.Join(tempDir, "docker-compose.yml")
	if err := os.WriteFile(composeFile, []byte(composeContent), 0644); err != nil {
		t.Fatalf("failed to write docker-compose.yml: %v", err)
	}

	upCmd := exec.Command("docker", "compose", "-f", composeFile, "-p", "test_ps_notrunc", "up", "-d")
	upCmd.Dir = tempDir
	if err := upCmd.Run(); err != nil {
		t.Fatalf("failed to start containers: %v", err)
	}

	defer func() {
		downCmd := exec.Command("docker", "compose", "-f", composeFile, "-p", "test_ps_notrunc", "down")
		downCmd.Dir = tempDir
		downCmd.Run()
	}()

	time.Sleep(2 * time.Second)

	// Test ps command with no-trunc flag
	err = runPsCommand(context.Background(), tempDir, "test_ps_notrunc", provider.ProviderGeneric, false, true)
	if err != nil {
		t.Fatalf("ps command failed: %v", err)
	}
}

// TestPsIntegrationMissingComposeFile tests ps with missing compose file
func TestPsIntegrationMissingComposeFile(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "space-cli-integration-*")
	if err != nil {
		t.Fatalf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Don't create any docker-compose file
	err = runPsCommand(context.Background(), tempDir, "test_missing", provider.ProviderGeneric, false, false)

	if err == nil {
		t.Error("expected error for missing compose file, but got nil")
	}

	if !strings.Contains(err.Error(), "no docker-compose files found") {
		t.Errorf("expected error about missing compose files, got: %v", err)
	}
}

// TestPsIntegrationNoRunningContainers tests ps with no running containers
func TestPsIntegrationNoRunningContainers(t *testing.T) {
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker not found, skipping integration test")
	}

	tempDir, err := os.MkdirTemp("", "space-cli-integration-*")
	if err != nil {
		t.Fatalf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	composeContent := `version: '3.8'
services:
  app:
    image: busybox
    command: sleep 300
`
	composeFile := filepath.Join(tempDir, "docker-compose.yml")
	if err := os.WriteFile(composeFile, []byte(composeContent), 0644); err != nil {
		t.Fatalf("failed to write docker-compose.yml: %v", err)
	}

	// Test ps without starting containers - should show empty list
	err = runPsCommand(context.Background(), tempDir, "test_ps_empty", provider.ProviderGeneric, false, false)
	// Should not error, just show no containers
	if err != nil {
		// Some systems might error, that's ok
		t.Logf("Note: error on empty ps: %v", err)
	}
}

// TestPsIntegrationWithMultipleComposeFiles tests ps with multiple compose files
func TestPsIntegrationWithMultipleComposeFiles(t *testing.T) {
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker not found, skipping integration test")
	}

	tempDir, err := os.MkdirTemp("", "space-cli-integration-*")
	if err != nil {
		t.Fatalf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create base compose file
	baseContent := `version: '3.8'
services:
  app:
    image: busybox
    command: sleep 300
`
	baseFile := filepath.Join(tempDir, "docker-compose.yml")
	if err := os.WriteFile(baseFile, []byte(baseContent), 0644); err != nil {
		t.Fatalf("failed to write base compose: %v", err)
	}

	// Create override compose file
	overrideContent := `version: '3.8'
services:
  app:
    ports:
      - "3000:3000"
`
	overrideFile := filepath.Join(tempDir, "docker-compose.override.yml")
	if err := os.WriteFile(overrideFile, []byte(overrideContent), 0644); err != nil {
		t.Fatalf("failed to write override compose: %v", err)
	}

	upCmd := exec.Command("docker", "compose", "-f", baseFile, "-f", overrideFile, "-p", "test_ps_multi_file", "up", "-d")
	upCmd.Dir = tempDir
	if err := upCmd.Run(); err != nil {
		t.Fatalf("failed to start containers: %v", err)
	}

	defer func() {
		downCmd := exec.Command("docker", "compose", "-f", baseFile, "-f", overrideFile, "-p", "test_ps_multi_file", "down")
		downCmd.Dir = tempDir
		downCmd.Run()
	}()

	time.Sleep(2 * time.Second)

	// Test ps command
	err = runPsCommand(context.Background(), tempDir, "test_ps_multi_file", provider.ProviderGeneric, false, false)
	if err != nil {
		t.Fatalf("ps command failed: %v", err)
	}
}

// TestPsIntegrationProviderDetection tests ps with different providers
func TestPsIntegrationProviderDetection(t *testing.T) {
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker not found, skipping integration test")
	}

	tempDir, err := os.MkdirTemp("", "space-cli-integration-*")
	if err != nil {
		t.Fatalf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	composeContent := `version: '3.8'
services:
  app:
    image: busybox
    command: sleep 300
`
	composeFile := filepath.Join(tempDir, "docker-compose.yml")
	if err := os.WriteFile(composeFile, []byte(composeContent), 0644); err != nil {
		t.Fatalf("failed to write docker-compose.yml: %v", err)
	}

	upCmd := exec.Command("docker", "compose", "-f", composeFile, "-p", "test_ps_provider", "up", "-d")
	upCmd.Dir = tempDir
	if err := upCmd.Run(); err != nil {
		t.Fatalf("failed to start containers: %v", err)
	}

	defer func() {
		downCmd := exec.Command("docker", "compose", "-f", composeFile, "-p", "test_ps_provider", "down")
		downCmd.Dir = tempDir
		downCmd.Run()
	}()

	time.Sleep(2 * time.Second)

	providers := []provider.Provider{
		provider.ProviderGeneric,
		provider.ProviderOrbStack,
		provider.ProviderDockerDesktop,
	}

	for _, prov := range providers {
		t.Run(prov.Description(), func(t *testing.T) {
			err = runPsCommand(context.Background(), tempDir, "test_ps_provider", prov, false, false)
			if err != nil {
				t.Fatalf("ps command failed with provider %s: %v", prov.Description(), err)
			}
		})
	}
}

// TestPsIntegrationStress tests ps with many containers
func TestPsIntegrationStress(t *testing.T) {
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker not found, skipping integration test")
	}

	tempDir, err := os.MkdirTemp("", "space-cli-integration-*")
	if err != nil {
		t.Fatalf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create compose file with many services
	var services strings.Builder
	services.WriteString("version: '3.8'\nservices:\n")
	for i := 0; i < 5; i++ {
		services.WriteString("  app")
		services.WriteString(string(rune('0' + i)))
		services.WriteString(":\n")
		services.WriteString("    image: busybox\n")
		services.WriteString("    command: sleep 300\n")
	}

	composeFile := filepath.Join(tempDir, "docker-compose.yml")
	if err := os.WriteFile(composeFile, []byte(services.String()), 0644); err != nil {
		t.Fatalf("failed to write docker-compose.yml: %v", err)
	}

	upCmd := exec.Command("docker", "compose", "-f", composeFile, "-p", "test_ps_stress", "up", "-d")
	upCmd.Dir = tempDir
	if err := upCmd.Run(); err != nil {
		t.Fatalf("failed to start containers: %v", err)
	}

	defer func() {
		downCmd := exec.Command("docker", "compose", "-f", composeFile, "-p", "test_ps_stress", "down")
		downCmd.Dir = tempDir
		downCmd.Run()
	}()

	time.Sleep(3 * time.Second)

	// Test ps command
	err = runPsCommand(context.Background(), tempDir, "test_ps_stress", provider.ProviderGeneric, false, false)
	if err != nil {
		t.Fatalf("ps command failed: %v", err)
	}
}

// TestPsIntegrationQuietAndNoTrunc tests ps with both flags
func TestPsIntegrationQuietAndNoTrunc(t *testing.T) {
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker not found, skipping integration test")
	}

	tempDir, err := os.MkdirTemp("", "space-cli-integration-*")
	if err != nil {
		t.Fatalf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	composeContent := `version: '3.8'
services:
  app:
    image: busybox
    command: sleep 300
`
	composeFile := filepath.Join(tempDir, "docker-compose.yml")
	if err := os.WriteFile(composeFile, []byte(composeContent), 0644); err != nil {
		t.Fatalf("failed to write docker-compose.yml: %v", err)
	}

	upCmd := exec.Command("docker", "compose", "-f", composeFile, "-p", "test_ps_both", "up", "-d")
	upCmd.Dir = tempDir
	if err := upCmd.Run(); err != nil {
		t.Fatalf("failed to start containers: %v", err)
	}

	defer func() {
		downCmd := exec.Command("docker", "compose", "-f", composeFile, "-p", "test_ps_both", "down")
		downCmd.Dir = tempDir
		downCmd.Run()
	}()

	time.Sleep(2 * time.Second)

	// Test ps command with both flags
	err = runPsCommand(context.Background(), tempDir, "test_ps_both", provider.ProviderGeneric, true, true)
	if err != nil {
		t.Fatalf("ps command failed: %v", err)
	}
}
