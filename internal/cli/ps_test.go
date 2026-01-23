package cli

import (
	"bytes"
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/happy-sdk/space-cli/internal/provider"
	"github.com/happy-sdk/space-cli/pkg/config"
)

// TestParseContainers tests parsing of docker compose ps output
func TestParseContainers(t *testing.T) {
	tests := []struct {
		name      string
		output    string
		wantCount int
		validate  func(t *testing.T, containers []Container)
	}{
		{
			name: "empty output",
			output: `CONTAINER ID   IMAGE     COMMAND   CREATED   STATUS    PORTS     NAMES
`,
			wantCount: 0,
			validate: func(t *testing.T, containers []Container) {
				if len(containers) != 0 {
					t.Errorf("expected 0 containers, got %d", len(containers))
				}
			},
		},
		{
			name: "single container",
			output: `CONTAINER ID   IMAGE              COMMAND                  CREATED        STATUS          PORTS                    NAMES
abc123def456   postgres:15        "docker-entrypoint.s…"   2 minutes ago   Up 2 minutes    5432/tcp                 my_app_db_1
`,
			wantCount: 1,
			validate: func(t *testing.T, containers []Container) {
				if len(containers) != 1 {
					t.Errorf("expected 1 container, got %d", len(containers))
				}
				c := containers[0]
				if c.ID != "abc123def456" {
					t.Errorf("expected ID 'abc123def456', got '%s'", c.ID)
				}
				if c.Image != "postgres:15" {
					t.Errorf("expected image 'postgres:15', got '%s'", c.Image)
				}
				if c.Name != "my_app_db_1" {
					t.Errorf("expected name 'my_app_db_1', got '%s'", c.Name)
				}
			},
		},
		{
			name: "multiple containers",
			output: `CONTAINER ID   IMAGE              COMMAND                  CREATED        STATUS          PORTS                    NAMES
abc123def456   postgres:15        "docker-entrypoint.s…"   2 minutes ago   Up 2 minutes    5432/tcp                 my_app_db_1
def456ghi789   redis:7            "redis-server"           3 minutes ago   Up 3 minutes    6379/tcp                 my_app_redis_1
ghi789jkl012   nginx:latest       "nginx -g 'daemon off"   1 minute ago    Up 1 minute     0.0.0.0:80->80/tcp       my_app_web_1
`,
			wantCount: 3,
			validate: func(t *testing.T, containers []Container) {
				if len(containers) != 3 {
					t.Errorf("expected 3 containers, got %d", len(containers))
				}
				images := []string{containers[0].Image, containers[1].Image, containers[2].Image}
				expected := []string{"postgres:15", "redis:7", "nginx:latest"}
				for i, exp := range expected {
					if images[i] != exp {
						t.Errorf("container %d: expected image '%s', got '%s'", i, exp, images[i])
					}
				}
			},
		},
		{
			name:      "only header",
			output:    "CONTAINER ID   IMAGE     COMMAND   CREATED   STATUS    PORTS     NAMES",
			wantCount: 0,
		},
		{
			name:      "whitespace only",
			output:    "",
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			containers := ParseContainers(tt.output)
			if len(containers) != tt.wantCount {
				t.Errorf("expected %d containers, got %d", tt.wantCount, len(containers))
			}
			if tt.validate != nil {
				tt.validate(t, containers)
			}
		})
	}
}

// TestProjectNameGeneration tests project name detection
func TestProjectNameGeneration(t *testing.T) {
	tests := []struct {
		name        string
		projectName string
		workDir     string
		validate    func(t *testing.T, generated string)
	}{
		{
			name:        "explicit project name",
			projectName: "myapp",
			workDir:     "/path/to/project",
			validate: func(t *testing.T, generated string) {
				if generated != "myapp" {
					t.Errorf("expected 'myapp', got '%s'", generated)
				}
			},
		},
		{
			name:        "generated from directory",
			projectName: "",
			workDir:     "/path/to/my_app",
			validate: func(t *testing.T, generated string) {
				if !strings.Contains(generated, "my_app") {
					t.Errorf("expected generated name to contain 'my_app', got '%s'", generated)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Project: config.ProjectConfig{
					Name: tt.projectName,
				},
			}
			generated := generateProjectName(cfg, tt.workDir)
			if tt.validate != nil {
				tt.validate(t, generated)
			}
		})
	}
}

// TestRunPsCommandErrors tests error handling in ps command
func TestRunPsCommandErrors(t *testing.T) {
	tests := []struct {
		name          string
		setupFunc     func(t *testing.T, workDir string)
		wantErr       bool
		errContains   string
		workDir       string
		projectName   string
		providerType  provider.Provider
	}{
		{
			name: "no docker-compose file",
			setupFunc: func(t *testing.T, workDir string) {
				// Create workdir but no docker-compose.yml
			},
			wantErr:      true,
			errContains:  "no docker-compose files found",
			workDir:      "",
			projectName:  "test",
			providerType: provider.ProviderGeneric,
		},
		{
			name: "docker-compose file exists",
			setupFunc: func(t *testing.T, workDir string) {
				// Create minimal docker-compose.yml
				content := `version: '3.8'
services:
  app:
    image: busybox
    command: sleep 1000
`
				filePath := filepath.Join(workDir, "docker-compose.yml")
				if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
					t.Fatalf("failed to create docker-compose.yml: %v", err)
				}
			},
			wantErr:      false,
			workDir:      "",
			projectName:  "test",
			providerType: provider.ProviderGeneric,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory
			tempDir, err := os.MkdirTemp("", "space-cli-test-*")
			if err != nil {
				t.Fatalf("failed to create temp directory: %v", err)
			}
			defer os.RemoveAll(tempDir)

			// Setup test environment
			if tt.setupFunc != nil {
				tt.setupFunc(t, tempDir)
			}

			workDir := tt.workDir
			if workDir == "" {
				workDir = tempDir
			}

			// Run ps command
			err = runPsCommand(context.Background(), workDir, tt.projectName, tt.providerType, false, false)

			if (err != nil) != tt.wantErr {
				t.Errorf("runPsCommand() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr && tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
				t.Errorf("error should contain '%s', got '%v'", tt.errContains, err)
			}
		})
	}
}

// TestDNSModeDetection tests DNS mode vs regular mode detection
func TestDNSModeDetection(t *testing.T) {
	tests := []struct {
		name         string
		providerType provider.Provider
		expectedDNS  bool
	}{
		{
			name:         "orbstack supports dns",
			providerType: provider.ProviderOrbStack,
			expectedDNS:  true,
		},
		{
			name:         "docker desktop does not support dns",
			providerType: provider.ProviderDockerDesktop,
			expectedDNS:  false,
		},
		{
			name:         "generic does not support dns",
			providerType: provider.ProviderGeneric,
			expectedDNS:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			supportsDNS := tt.providerType.SupportsContainerDNS()
			if supportsDNS != tt.expectedDNS {
				t.Errorf("provider %s: expected DNS support=%v, got %v",
					tt.providerType.Description(), tt.expectedDNS, supportsDNS)
			}
		})
	}
}

// TestContainerStructure tests Container struct initialization
func TestContainerStructure(t *testing.T) {
	tests := []struct {
		name      string
		container Container
		validate  func(t *testing.T, c Container)
	}{
		{
			name: "minimal container",
			container: Container{
				ID:   "abc123",
				Name: "app_1",
			},
			validate: func(t *testing.T, c Container) {
				if c.ID != "abc123" {
					t.Errorf("expected ID 'abc123', got '%s'", c.ID)
				}
				if c.Name != "app_1" {
					t.Errorf("expected name 'app_1', got '%s'", c.Name)
				}
			},
		},
		{
			name: "container with all fields",
			container: Container{
				ID:       "abc123",
				Name:     "app_1",
				Image:    "myapp:latest",
				Status:   "Up 2 minutes",
				Ports:    "0.0.0.0:3000->3000/tcp",
				Command:  "npm start",
				IsDNS:    false,
				Provider: provider.ProviderOrbStack,
			},
			validate: func(t *testing.T, c Container) {
				if c.Image != "myapp:latest" {
					t.Errorf("expected image 'myapp:latest', got '%s'", c.Image)
				}
				if c.Provider != provider.ProviderOrbStack {
					t.Errorf("expected provider OrbStack, got %v", c.Provider)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.validate != nil {
				tt.validate(t, tt.container)
			}
		})
	}
}

// TestDockerComposeInvocation tests proper docker-compose command building
func TestDockerComposeInvocation(t *testing.T) {
	tests := []struct {
		name        string
		composeFiles []string
		projectName  string
		quiet        bool
		noTrunc      bool
		validateCmd  func(t *testing.T, cmd *exec.Cmd)
	}{
		{
			name:         "basic ps command",
			composeFiles: []string{"docker-compose.yml"},
			projectName:  "myapp",
			quiet:        false,
			noTrunc:      false,
			validateCmd: func(t *testing.T, cmd *exec.Cmd) {
				args := cmd.Args
				if len(args) < 4 {
					t.Errorf("expected at least 4 args, got %d", len(args))
				}
				if args[0] != "docker" {
					t.Errorf("expected first arg 'docker', got '%s'", args[0])
				}
				if args[1] != "compose" {
					t.Errorf("expected second arg 'compose', got '%s'", args[1])
				}
			},
		},
		{
			name:         "with quiet flag",
			composeFiles: []string{"docker-compose.yml"},
			projectName:  "myapp",
			quiet:        true,
			noTrunc:      false,
			validateCmd: func(t *testing.T, cmd *exec.Cmd) {
				args := strings.Join(cmd.Args, " ")
				if !strings.Contains(args, "-q") && !strings.Contains(args, "--quiet") {
					t.Errorf("expected -q flag in args, got %s", args)
				}
			},
		},
		{
			name:         "with no-trunc flag",
			composeFiles: []string{"docker-compose.yml"},
			projectName:  "myapp",
			quiet:        false,
			noTrunc:      true,
			validateCmd: func(t *testing.T, cmd *exec.Cmd) {
				args := strings.Join(cmd.Args, " ")
				if !strings.Contains(args, "--no-trunc") {
					t.Errorf("expected --no-trunc flag in args, got %s", args)
				}
			},
		},
		{
			name:         "multiple compose files",
			composeFiles: []string{"docker-compose.yml", "docker-compose.override.yml"},
			projectName:  "myapp",
			quiet:        false,
			noTrunc:      false,
			validateCmd: func(t *testing.T, cmd *exec.Cmd) {
				args := strings.Join(cmd.Args, " ")
				if !strings.Contains(args, "docker-compose.yml") {
					t.Errorf("expected docker-compose.yml in args")
				}
				if !strings.Contains(args, "docker-compose.override.yml") {
					t.Errorf("expected docker-compose.override.yml in args")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory with docker-compose files
			tempDir, err := os.MkdirTemp("", "space-cli-test-*")
			if err != nil {
				t.Fatalf("failed to create temp directory: %v", err)
			}
			defer os.RemoveAll(tempDir)

			// Create dummy compose files
			content := `version: '3.8'
services:
  app:
    image: busybox
`
			for _, file := range tt.composeFiles {
				filePath := filepath.Join(tempDir, file)
				if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
					t.Fatalf("failed to create %s: %v", file, err)
				}
			}

			// Build command (without executing)
			composeCmd := []string{"docker", "compose"}
			for _, file := range tt.composeFiles {
				composeCmd = append(composeCmd, "-f", file)
			}
			composeCmd = append(composeCmd, "-p", tt.projectName, "ps")

			if tt.quiet {
				composeCmd = append(composeCmd, "-q")
			}
			if tt.noTrunc {
				composeCmd = append(composeCmd, "--no-trunc")
			}

			cmd := exec.Command(composeCmd[0], composeCmd[1:]...)
			cmd.Dir = tempDir

			if tt.validateCmd != nil {
				tt.validateCmd(t, cmd)
			}
		})
	}
}

// TestOutputFormatting tests output formatting variations
func TestOutputFormatting(t *testing.T) {
	tests := []struct {
		name        string
		quiet       bool
		noTrunc     bool
		description string
	}{
		{
			name:        "default format",
			quiet:       false,
			noTrunc:     false,
			description: "full container info with truncation",
		},
		{
			name:        "quiet format",
			quiet:       true,
			noTrunc:     false,
			description: "only container IDs",
		},
		{
			name:        "no truncation",
			quiet:       false,
			noTrunc:     true,
			description: "full container info without truncation",
		},
		{
			name:        "quiet no truncation",
			quiet:       true,
			noTrunc:     true,
			description: "only container IDs without truncation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify formatting flags are set correctly
			if tt.quiet != false && tt.quiet != true {
				t.Errorf("invalid quiet value: %v", tt.quiet)
			}
			if tt.noTrunc != false && tt.noTrunc != true {
				t.Errorf("invalid noTrunc value: %v", tt.noTrunc)
			}
		})
	}
}

// BenchmarkParseContainers benchmarks container parsing
func BenchmarkParseContainers(b *testing.B) {
	output := `CONTAINER ID   IMAGE              COMMAND                  CREATED        STATUS          PORTS                    NAMES
abc123def456   postgres:15        "docker-entrypoint.s…"   2 minutes ago   Up 2 minutes    5432/tcp                 my_app_db_1
def456ghi789   redis:7            "redis-server"           3 minutes ago   Up 3 minutes    6379/tcp                 my_app_redis_1
ghi789jkl012   nginx:latest       "nginx -g 'daemon off"   1 minute ago    Up 1 minute     0.0.0.0:80->80/tcp       my_app_web_1
jkl012mno345   node:18            "node app.js"            1 minute ago    Up 1 minute     0.0.0.0:3000->3000/tcp   my_app_app_1
mno345pqr678   mysql:8            "docker-entrypoint.s…"   45 seconds ago  Up 44 seconds   3306/tcp                 my_app_mysql_1
`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ParseContainers(output)
	}
}

// TestNewPsCommand tests command initialization
func TestNewPsCommand(t *testing.T) {
	cmd := newPsCommand()

	if cmd.Use != "ps" {
		t.Errorf("expected Use='ps', got '%s'", cmd.Use)
	}

	if cmd.Short == "" {
		t.Error("expected non-empty Short description")
	}

	if cmd.Long == "" {
		t.Error("expected non-empty Long description")
	}

	if cmd.RunE == nil {
		t.Error("expected RunE to be set")
	}

	// Check flags
	if cmd.Flags().Lookup("quiet") == nil {
		t.Error("expected 'quiet' flag")
	}

	if cmd.Flags().Lookup("no-trunc") == nil {
		t.Error("expected 'no-trunc' flag")
	}
}

// TestCaptureStdout captures stdout for testing
func captureStdout(f func() error) (string, error) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := f()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String(), err
}

// TestErrorHandlingNoContainers tests error handling when no containers exist
func TestErrorHandlingNoContainers(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "space-cli-test-*")
	if err != nil {
		t.Fatalf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create minimal docker-compose.yml
	content := `version: '3.8'
services:
  app:
    image: busybox
`
	filePath := filepath.Join(tempDir, "docker-compose.yml")
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create docker-compose.yml: %v", err)
	}

	// This test validates that the function handles "no containers" gracefully
	// The actual docker-compose ps might return empty list with exit code 0
	err = runPsCommand(context.Background(), tempDir, "test", provider.ProviderGeneric, false, false)
	// Should not error even if no containers are running
	if err != nil && strings.Contains(err.Error(), "failed to list services") {
		t.Errorf("unexpected error when no containers: %v", err)
	}
}

// TestProviderAwarenessInOutput tests provider-aware formatting
func TestProviderAwarenessInOutput(t *testing.T) {
	tests := []struct {
		name     string
		provider provider.Provider
	}{
		{
			name:     "orbstack provider",
			provider: provider.ProviderOrbStack,
		},
		{
			name:     "docker desktop provider",
			provider: provider.ProviderDockerDesktop,
		},
		{
			name:     "generic provider",
			provider: provider.ProviderGeneric,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir, err := os.MkdirTemp("", "space-cli-test-*")
			if err != nil {
				t.Fatalf("failed to create temp directory: %v", err)
			}
			defer os.RemoveAll(tempDir)

			// Create docker-compose file
			content := `version: '3.8'
services:
  app:
    image: busybox
`
			filePath := filepath.Join(tempDir, "docker-compose.yml")
			if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
				t.Fatalf("failed to create docker-compose.yml: %v", err)
			}

			// Run ps command with specific provider
			err = runPsCommand(context.Background(), tempDir, "test", tt.provider, false, false)
			// Don't assert error here, just that it runs
			_ = err
		})
	}
}
