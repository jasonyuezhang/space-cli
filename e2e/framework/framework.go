// Package framework provides the e2e test framework for space-cli
package framework

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// E2EFramework provides utilities for e2e testing
type E2EFramework struct {
	t          *testing.T
	spaceBin   string
	fixtureDir string
	workDir    string
	cleanup    []func()
}

// New creates a new E2EFramework instance
func New(t *testing.T) *E2EFramework {
	t.Helper()

	// Find space binary - either built or use go run
	spaceBin := os.Getenv("SPACE_BIN")
	if spaceBin == "" {
		// Try bin/space first
		projectRoot := findProjectRoot(t)
		spaceBin = filepath.Join(projectRoot, "bin", "space")
		if _, err := os.Stat(spaceBin); err != nil {
			// Fall back to building on the fly
			t.Logf("Building space binary...")
			cmd := exec.Command("go", "build", "-o", spaceBin, "./cmd/space")
			cmd.Dir = projectRoot
			if output, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("Failed to build space binary: %v\n%s", err, output)
			}
		}
	}

	return &E2EFramework{
		t:          t,
		spaceBin:   spaceBin,
		fixtureDir: filepath.Join(findProjectRoot(t), "e2e", "fixtures"),
	}
}

// WithFixture sets up a test with a specific fixture
func (f *E2EFramework) WithFixture(name string) *E2EFramework {
	f.t.Helper()

	fixturePath := filepath.Join(f.fixtureDir, name)
	if _, err := os.Stat(fixturePath); err != nil {
		f.t.Fatalf("Fixture not found: %s", name)
	}

	// Create a temporary working directory
	tmpDir, err := os.MkdirTemp("", fmt.Sprintf("space-e2e-%s-", name))
	if err != nil {
		f.t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Copy fixture to temp dir
	if err := copyDir(fixturePath, tmpDir); err != nil {
		os.RemoveAll(tmpDir)
		f.t.Fatalf("Failed to copy fixture: %v", err)
	}

	f.workDir = tmpDir
	f.cleanup = append(f.cleanup, func() {
		os.RemoveAll(tmpDir)
	})

	return f
}

// Cleanup runs all cleanup functions
func (f *E2EFramework) Cleanup() {
	// Always try to bring down containers first
	if f.workDir != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		f.RunSpaceCmd(ctx, "down")
	}

	for i := len(f.cleanup) - 1; i >= 0; i-- {
		f.cleanup[i]()
	}
}

// WorkDir returns the working directory for the current test
func (f *E2EFramework) WorkDir() string {
	return f.workDir
}

// RunSpaceCmd executes a space command and returns the result
func (f *E2EFramework) RunSpaceCmd(ctx context.Context, args ...string) *CmdResult {
	f.t.Helper()

	var cmdArgs []string
	if f.workDir != "" {
		cmdArgs = append([]string{"-w", f.workDir}, args...)
	} else {
		cmdArgs = args
	}
	cmd := exec.CommandContext(ctx, f.spaceBin, cmdArgs...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	return &CmdResult{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: cmd.ProcessState.ExitCode(),
		Err:      err,
	}
}

// RunSpaceCmdInteractive runs a space command with stdin/stdout attached
func (f *E2EFramework) RunSpaceCmdInteractive(ctx context.Context, args ...string) error {
	f.t.Helper()

	var cmdArgs []string
	if f.workDir != "" {
		cmdArgs = append([]string{"-w", f.workDir}, args...)
	} else {
		cmdArgs = args
	}
	cmd := exec.CommandContext(ctx, f.spaceBin, cmdArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// WaitForService waits for a service to become healthy
func (f *E2EFramework) WaitForService(ctx context.Context, url string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 5 * time.Second}

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		resp, err := client.Get(url)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode >= 200 && resp.StatusCode < 400 {
				return nil
			}
		}

		time.Sleep(1 * time.Second)
	}

	return fmt.Errorf("service at %s not healthy after %v", url, timeout)
}

// WaitForContainer waits for a container to be running
func (f *E2EFramework) WaitForContainer(ctx context.Context, containerName string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		cmd := exec.CommandContext(ctx, "docker", "inspect", "-f", "{{.State.Running}}", containerName)
		output, err := cmd.Output()
		if err == nil && strings.TrimSpace(string(output)) == "true" {
			return nil
		}

		time.Sleep(1 * time.Second)
	}

	return fmt.Errorf("container %s not running after %v", containerName, timeout)
}

// DockerComposePS runs docker compose ps and returns the output
func (f *E2EFramework) DockerComposePS(ctx context.Context, projectName string) (string, error) {
	cmd := exec.CommandContext(ctx, "docker", "compose", "-p", projectName, "ps")
	cmd.Dir = f.workDir
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// ServiceURL represents a service URL configuration
type ServiceURL struct {
	LocalhostURL   string // e.g., http://localhost:8080
	ContainerURL   string // e.g., http://e2e-microservices-frontend-1.orb.local
	HealthEndpoint string // e.g., /health
}

// WaitForServiceWithFallback tries multiple URLs to reach a service
// This handles both localhost (port bindings) and container DNS modes
func (f *E2EFramework) WaitForServiceWithFallback(ctx context.Context, svc ServiceURL, timeout time.Duration) (string, error) {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 5 * time.Second}

	urls := []string{}
	if svc.LocalhostURL != "" {
		urls = append(urls, svc.LocalhostURL+svc.HealthEndpoint)
	}
	if svc.ContainerURL != "" {
		urls = append(urls, svc.ContainerURL+svc.HealthEndpoint)
	}

	if len(urls) == 0 {
		return "", fmt.Errorf("no URLs provided for service")
	}

	var lastErr error
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		for _, url := range urls {
			resp, err := client.Get(url)
			if err == nil {
				resp.Body.Close()
				if resp.StatusCode >= 200 && resp.StatusCode < 400 {
					// Return the base URL (without health endpoint)
					baseURL := strings.TrimSuffix(url, svc.HealthEndpoint)
					return baseURL, nil
				}
			}
			lastErr = err
		}

		time.Sleep(1 * time.Second)
	}

	return "", fmt.Errorf("service not healthy after %v (tried %v): %v", timeout, urls, lastErr)
}

// GetContainerName returns the full container name for a service
func (f *E2EFramework) GetContainerName(projectName, serviceName string) string {
	return fmt.Sprintf("%s-%s-1", projectName, serviceName)
}

// GetOrbstackURL returns the OrbStack DNS URL for a container
func (f *E2EFramework) GetOrbstackURL(containerName string, port int) string {
	if port == 80 || port == 0 {
		return fmt.Sprintf("http://%s.orb.local", containerName)
	}
	return fmt.Sprintf("http://%s.orb.local:%d", containerName, port)
}

// CmdResult holds the result of a command execution
type CmdResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
	Err      error
}

// Success returns true if the command succeeded
func (r *CmdResult) Success() bool {
	return r.ExitCode == 0 && r.Err == nil
}

// Contains checks if stdout contains a substring
func (r *CmdResult) Contains(s string) bool {
	return strings.Contains(r.Stdout, s) || strings.Contains(r.Stderr, s)
}

// findProjectRoot finds the root of the space-cli project
func findProjectRoot(t *testing.T) string {
	t.Helper()

	// Start from the current directory and walk up
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	for {
		// Check for go.mod with the right module name
		gomod := filepath.Join(dir, "go.mod")
		if data, err := os.ReadFile(gomod); err == nil {
			if strings.Contains(string(data), "space-cli") {
				return dir
			}
		}

		// Move up one directory
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	t.Fatalf("Could not find project root")
	return ""
}

// copyDir recursively copies a directory
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Get relative path
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		// Copy file
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		return os.WriteFile(dstPath, data, info.Mode())
	})
}
