package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/happy-sdk/space-cli/internal/dns"
	"github.com/happy-sdk/space-cli/internal/hooks"
	"github.com/happy-sdk/space-cli/internal/provider"
	"github.com/happy-sdk/space-cli/pkg/config"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	// Global DNS server for cleanup
	globalDNSServer   *dns.Server
	globalDNSResolver *dns.ResolverManager
)

func newUpCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "up [services...]",
		Short: "Start services",
		Long:  "Start all services or specific services defined in docker-compose.yml.",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			// Get verbose flag
			verbose, _ := cmd.Flags().GetBool("verbose")

			// Get working directory
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

			// Create loader
			loader, err := config.NewLoader(workDir)
			if err != nil {
				return fmt.Errorf("failed to create config loader: %w", err)
			}

			// Load configuration
			cfg, err := loader.Load()
			if err != nil {
				return fmt.Errorf("failed to load configuration: %w", err)
			}

			// Validate
			if err := cfg.Validate(); err != nil {
				return fmt.Errorf("invalid configuration: %w", err)
			}

			fmt.Printf("🚀 Starting services for project: %s\n", cfg.Project.Name)
			fmt.Printf("📁 Working directory: %s\n", workDir)
			fmt.Println()

			// Detect provider
			detector := provider.NewDetector()
			providerType, err := detector.Detect(ctx)
			if err != nil {
				fmt.Printf("⚠️  Failed to detect provider: %v\n", err)
				providerType = provider.ProviderGeneric
			}
			fmt.Printf("🔍 Detected provider: %s\n", providerType.Description())

			// Generate project name
			projectName := generateProjectName(cfg, workDir)
			fmt.Printf("📦 Project name: %s\n", projectName)

			// Try to start DNS server if using OrbStack
			useDNS := false
			var overrideFile string
			if providerType.SupportsContainerDNS() {
				fmt.Println()

				// Check if DNS daemon is already running
				if isDNSServerRunning() {
					state, _ := loadDNSState()
					fmt.Printf("✅ Using existing space-dns-daemon on %s\n", state.Address)
					useDNS = true
				} else {
					// Start DNS daemon as background process
					fmt.Println("🌐 Starting space-dns-daemon in background...")
					if err := spawnDNSDaemon(); err != nil {
						fmt.Printf("⚠️  Failed to start DNS daemon: %v\n", err)
						fmt.Println("⚠️  Falling back to port bindings")
					} else {
						// Wait a moment for daemon to start
						time.Sleep(500 * time.Millisecond)

						if isDNSServerRunning() {
							state, _ := loadDNSState()
							fmt.Printf("✅ DNS daemon started on %s\n", state.Address)
							useDNS = true
						} else {
							fmt.Println("⚠️  DNS daemon failed to start, falling back to port bindings")
						}
					}
				}

				if useDNS {
					fmt.Printf("   Containers will be accessible at: *.space.local\n")

					// Create modified compose file without port bindings
					overrideFile, err = createDNSModeCompose(workDir, cfg)
					if err != nil {
						fmt.Printf("⚠️  Failed to create DNS mode compose file: %v\n", err)
						// Continue anyway - docker-compose will use original ports
					}
				}
			}

			fmt.Println()

			// Build docker compose command
			composeCmd := []string{"docker", "compose"}

			// Use DNS mode compose file if available, otherwise use original files
			if overrideFile != "" {
				composeCmd = append(composeCmd, "-f", overrideFile)
				fmt.Printf("📝 Using DNS mode compose file: %s\n", overrideFile)
			} else {
				// Add compose files
				for _, file := range cfg.Project.ComposeFiles {
					composeCmd = append(composeCmd, "-f", file)
				}
			}

			// Add project name
			composeCmd = append(composeCmd, "-p", projectName)

			// Add up command
			composeCmd = append(composeCmd, "up", "-d")

			// Add services if specified
			if len(args) > 0 {
				composeCmd = append(composeCmd, args...)
				fmt.Printf("📋 Starting services: %s\n", strings.Join(args, ", "))
			} else {
				fmt.Println("📋 Starting all services")
			}

			fmt.Println()

			// Execute docker compose
			dockerCmd := exec.Command(composeCmd[0], composeCmd[1:]...)
			dockerCmd.Dir = workDir
			dockerCmd.Stdout = os.Stdout
			dockerCmd.Stderr = os.Stderr
			dockerCmd.Stdin = os.Stdin

			fmt.Printf("🔧 Running: %s\n", strings.Join(composeCmd, " "))
			fmt.Println()

			if err := dockerCmd.Run(); err != nil {
				// Stop DNS server on failure (but keep resolver configured)
				if useDNS && globalDNSServer != nil {
					fmt.Println("🛑 Stopping space-dns-daemon...")
					if err := globalDNSServer.Stop(); err != nil {
						fmt.Printf("⚠️  Failed to stop DNS daemon: %v\n", err)
					}
					globalDNSServer = nil
					// Remove DNS state file on failure
					if err := removeDNSState(); err != nil {
						fmt.Printf("⚠️  Failed to remove DNS state: %v\n", err)
					}
					// Note: We intentionally do NOT remove the resolver file
					// It's meant to be permanent once configured
				}
				// Don't remove DNS mode compose file on failure so user can inspect it
				if overrideFile != "" {
					fmt.Printf("💡 DNS mode compose file preserved for debugging: %s\n", overrideFile)
				}
				return fmt.Errorf("failed to start services: %w", err)
			}

			// Clean up DNS mode compose file on success
			if overrideFile != "" {
				if err := os.Remove(overrideFile); err != nil {
					fmt.Printf("⚠️  Failed to cleanup DNS mode compose file: %v\n", err)
				}
			}

			fmt.Println()
			fmt.Println("✅ Services started successfully!")

			// Show DNS daemon status
			if useDNS {
				fmt.Println("🔄 space-dns-daemon is running in the background")
				fmt.Println("   Use 'space dns status' to check status")
				fmt.Println("   Use 'space dns stop' to stop the daemon")
			}
			fmt.Println()

			// Run post-up hooks (external scripts) - always run regardless of DNS mode
			runScriptHooks(ctx, hooks.PostUp, workDir, projectName, cfg, useDNS, verbose)

			// Show access information
			if useDNS {
				fmt.Println("🌍 Access your services at:")
				for serviceName, service := range cfg.Services {
					if service.Port > 0 {
						fmt.Printf("   • %s: http://%s.space.local:%d\n", serviceName, serviceName, service.Port)
					}
				}
			} else {
				fmt.Println("🌍 Access your services at:")
				for serviceName, service := range cfg.Services {
					if service.ExternalPort > 0 {
						fmt.Printf("   • %s: http://localhost:%d\n", serviceName, service.ExternalPort)
					} else if service.Port > 0 {
						fmt.Printf("   • %s: http://localhost:%d\n", serviceName, service.Port)
					}
				}
			}

			fmt.Println()
			fmt.Println("💡 Tip: Run 'space config show' to see your configuration")
			fmt.Println("💡 Tip: Run 'space status' to check service status")
			fmt.Println("💡 Tip: Run 'space logs <service>' to view logs")

			return nil
		},
	}

	cmd.Flags().BoolP("detach", "d", true, "Run services in detached mode")
	cmd.Flags().Bool("build", false, "Build images before starting")
	cmd.Flags().Bool("force-recreate", false, "Recreate containers even if config hasn't changed")
	cmd.Flags().BoolP("verbose", "v", false, "Verbose output for debugging hooks and execution")

	return cmd
}

// generateProjectName generates a project name based on the configured strategy
func generateProjectName(cfg *config.Config, workDir string) string {
	baseName := cfg.Project.Name
	if baseName == "" {
		baseName = filepath.Base(workDir)
	}

	prefix := cfg.Project.Prefix
	if prefix == "" {
		prefix = baseName + "-"
	}

	var projectName string
	switch cfg.Project.NamingStrategy {
	case "git-branch":
		// Try to get git branch
		branch := getGitBranch(workDir)
		if branch != "" {
			// Clean branch name (remove special chars)
			branch = strings.ReplaceAll(branch, "/", "-")
			branch = strings.ReplaceAll(branch, "_", "-")
			projectName = prefix + branch
		} else {
			// Fallback to directory
			projectName = prefix + filepath.Base(workDir)
		}

	case "directory":
		// Use directory name
		projectName = prefix + filepath.Base(workDir)

	case "static":
		fallthrough

	default:
		// Use configured name
		projectName = baseName
	}

	// Normalize project name to meet Docker requirements:
	// - Must start with letter or number
	// - Only lowercase alphanumeric, hyphens, and underscores
	return normalizeProjectName(projectName)
}

// normalizeProjectName ensures project name meets Docker Compose requirements:
// - Must start with a letter or number
// - Only lowercase alphanumeric characters, hyphens, and underscores
func normalizeProjectName(name string) string {
	// Convert to lowercase
	name = strings.ToLower(name)

	// Remove leading underscores and hyphens
	name = strings.TrimLeft(name, "_-")

	// If name is now empty or doesn't start with alphanumeric, prefix with 'p'
	if len(name) == 0 || !isAlphanumeric(name[0]) {
		name = "p" + name
	}

	// Replace any remaining invalid characters with hyphens
	var result strings.Builder
	for i, c := range name {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' || c == '_' {
			result.WriteRune(c)
		} else if i > 0 {
			// Replace invalid chars with hyphen (but not at start)
			result.WriteRune('-')
		}
	}

	return result.String()
}

// isAlphanumeric checks if a byte is a lowercase letter or digit
func isAlphanumeric(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= '0' && b <= '9')
}

// getGitBranch returns the current git branch name
func getGitBranch(workDir string) string {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = workDir
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

// startDNSServer starts the embedded DNS server as a persistent daemon
func startDNSServer(ctx context.Context, projectName string) error {
	// Get working directory for hash generation
	workDir := Workdir
	if workDir == "." {
		var err error
		workDir, err = os.Getwd()
		if err != nil {
			fmt.Printf("⚠️  Could not get working directory: %v\n", err)
			workDir = ""
		}
	}

	// Make absolute for consistent hashing
	if workDir != "" {
		absPath, err := filepath.Abs(workDir)
		if err == nil {
			workDir = absPath
		}
	}

	// Check if DNS server is already running
	if isDNSServerRunning() {
		dnsState, err := loadDNSState()
		if err == nil {
			fmt.Printf("✅ DNS server already running on %s\n", dnsState.Address)
			return nil
		}
	}

	// Create logger
	logger := dns.NewStdLogger()

	// Create Docker client
	dockerClient := dns.NewSimpleDockerClient(logger)

	// Try alternative ports if 5353 is in use
	ports := []int{5353, 5354, 5355, 5356}
	var server *dns.Server
	var dnsAddr string
	var lastErr error

	for _, port := range ports {
		dnsAddr = fmt.Sprintf("127.0.0.1:%d", port)

		// Create DNS server with hashing enabled
		var err error
		server, err = dns.NewServer(dns.Config{
			Addr:        dnsAddr,
			Upstream:    "8.8.8.8:53",
			ProjectName: projectName,
			Domain:      "space.local",
			WorkDir:     workDir,     // Enable directory-based hashing
			UseHashing:  true,        // Enable hashing by default
			CacheTTL:    30 * time.Second,
			Docker:      dockerClient,
			Logger:      logger,
		})
		if err != nil {
			lastErr = err
			continue
		}

		// Start DNS server with background context so it persists
		if err := server.Start(context.Background()); err != nil {
			lastErr = err
			continue
		}

		// Success!
		break
	}

	if server == nil || !server.IsRunning() {
		if lastErr != nil {
			return fmt.Errorf("failed to start DNS server on any port: %w", lastErr)
		}
		return fmt.Errorf("failed to start DNS server")
	}

	// Store global reference
	globalDNSServer = server

	// Create resolver manager
	resolver := dns.NewResolverManager("space.local", dnsAddr, logger)
	globalDNSResolver = resolver

	// Setup resolver (requires sudo)
	fmt.Println("📝 Setting up DNS resolver (may require sudo password)...")
	if err := resolver.Setup(ctx); err != nil {
		// Clean up server if resolver setup fails
		_ = server.Stop()
		return fmt.Errorf("failed to setup resolver: %w", err)
	}

	// Save DNS server state for persistence
	if err := saveDNSState(dnsAddr, projectName); err != nil {
		fmt.Printf("⚠️  Failed to save DNS state: %v\n", err)
		// Don't fail - DNS server is running even if state save failed
	}

	return nil
}

// createDNSModeCompose creates a modified docker-compose file without port bindings for DNS mode
func createDNSModeCompose(workDir string, cfg *config.Config) (string, error) {
	// Get the original compose file
	composeFile := filepath.Join(workDir, "docker-compose.yml")
	if len(cfg.Project.ComposeFiles) > 0 {
		composeFile = filepath.Join(workDir, cfg.Project.ComposeFiles[0])
	}

	// Read docker-compose.yml
	composeData, err := os.ReadFile(composeFile)
	if err != nil {
		return "", fmt.Errorf("failed to read docker-compose.yml: %w", err)
	}

	// Parse YAML
	var composeConfig map[string]interface{}
	if err := yaml.Unmarshal(composeData, &composeConfig); err != nil {
		return "", fmt.Errorf("failed to parse docker-compose.yml: %w", err)
	}

	// Process services to remove port bindings
	removedPorts := []string{}
	if composeServices, ok := composeConfig["services"].(map[string]interface{}); ok {
		for serviceName, serviceConfig := range composeServices {
			if svc, ok := serviceConfig.(map[string]interface{}); ok {
				// Check if service has ports defined
				if portsArray, hasPort := svc["ports"]; hasPort {
					// Extract container ports for expose directive
					containerPorts := []interface{}{}
					if ports, ok := portsArray.([]interface{}); ok {
						for _, portDef := range ports {
							if portStr, ok := portDef.(string); ok {
								// Parse port mapping (can be "5432:5432" or just "5432")
								parts := strings.Split(portStr, ":")
								containerPort := parts[len(parts)-1] // Get the container port (last part)
								containerPorts = append(containerPorts, containerPort)
							} else if portNum, ok := portDef.(int); ok {
								containerPorts = append(containerPorts, portNum)
							}
						}
					}

					// Remove ports binding
					delete(svc, "ports")

					// Add expose if not already present
					if _, hasExpose := svc["expose"]; !hasExpose && len(containerPorts) > 0 {
						svc["expose"] = containerPorts
					}

					removedPorts = append(removedPorts, serviceName)
				}
			}
		}
	}

	if len(removedPorts) > 0 {
		fmt.Printf("🔧 Removing host port bindings for: %s\n", strings.Join(removedPorts, ", "))
		fmt.Printf("   Ports will be accessible via DNS at: *.space.local\n")
	}

	// Write modified compose file
	dnsComposeFile := filepath.Join(workDir, ".space-dns-compose.yml")

	// Marshal back to YAML
	modifiedData, err := yaml.Marshal(composeConfig)
	if err != nil {
		return "", fmt.Errorf("failed to marshal modified compose: %w", err)
	}

	// Add header comment
	header := "# Auto-generated DNS mode compose file\n"
	header += "# This file has all port bindings removed - services accessible via DNS at *.space.local\n"
	header += "# Generated from: " + filepath.Base(composeFile) + "\n\n"

	finalContent := header + string(modifiedData)

	if err := os.WriteFile(dnsComposeFile, []byte(finalContent), 0644); err != nil {
		return "", fmt.Errorf("failed to write DNS mode compose file: %w", err)
	}

	return dnsComposeFile, nil
}

// DNSState represents the state of the running DNS daemon
type DNSState struct {
	Address     string    `json:"address"`
	ProjectName string    `json:"project_name"`
	StartTime   time.Time `json:"start_time"`
	PID         int       `json:"pid"`
}

// spawnDNSDaemon spawns the DNS daemon as a detached background process
func spawnDNSDaemon() error {
	// Get the path to the current executable
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// Create log file for DNS daemon output
	logPath := filepath.Join(os.TempDir(), "space-dns-daemon.log")
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to create log file: %w", err)
	}

	// Spawn "space dns start" as a background process
	cmd := exec.Command(execPath, "dns", "start")

	// Redirect output to log file
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.Stdin = nil

	// Set process group to detach from parent
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	// Start the process in the background
	if err := cmd.Start(); err != nil {
		logFile.Close()
		return fmt.Errorf("failed to spawn DNS daemon: %w", err)
	}

	// Don't wait for the process - let it run independently
	// Note: We don't close logFile here - the child process needs it

	fmt.Printf("   DNS daemon log: %s\n", logPath)

	return nil
}

// getDNSStateFile returns the path to the DNS state file
func getDNSStateFile() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ".space-dns-daemon.json"
	}
	return filepath.Join(homeDir, ".space-dns-daemon.json")
}

// saveDNSState saves the DNS daemon state to a file
func saveDNSState(address, projectName string) error {
	state := DNSState{
		Address:     address,
		ProjectName: projectName,
		StartTime:   time.Now(),
		PID:         os.Getpid(),
	}

	data, err := yaml.Marshal(state)
	if err != nil {
		return err
	}

	return os.WriteFile(getDNSStateFile(), data, 0644)
}

// loadDNSState loads the DNS daemon state from file
func loadDNSState() (*DNSState, error) {
	data, err := os.ReadFile(getDNSStateFile())
	if err != nil {
		return nil, err
	}

	var state DNSState
	if err := yaml.Unmarshal(data, &state); err != nil {
		return nil, err
	}

	return &state, nil
}

// isDNSServerRunning checks if the DNS daemon is running
func isDNSServerRunning() bool {
	state, err := loadDNSState()
	if err != nil {
		return false
	}

	// Check if process is still running (simple check)
	// Note: This is a basic check - could be improved with actual process verification
	_, err = os.ReadFile(getDNSStateFile())
	return err == nil && state.Address != ""
}

// removeDNSState removes the DNS state file
func removeDNSState() error {
	return os.Remove(getDNSStateFile())
}

// cleanupDNSServer stops the DNS server and cleans up resolver
func cleanupDNSServer(ctx context.Context) {
	if globalDNSResolver != nil {
		fmt.Println("🧹 Cleaning up DNS resolver...")
		if err := globalDNSResolver.Cleanup(ctx); err != nil {
			fmt.Printf("⚠️  Failed to cleanup resolver: %v\n", err)
		}
		globalDNSResolver = nil
	}

	if globalDNSServer != nil {
		fmt.Println("🛑 Stopping space-dns-daemon...")
		if err := globalDNSServer.Stop(); err != nil {
			fmt.Printf("⚠️  Failed to stop DNS daemon: %v\n", err)
		}
		globalDNSServer = nil

		// Remove state file
		if err := removeDNSState(); err != nil {
			fmt.Printf("⚠️  Failed to remove DNS state: %v\n", err)
		}
	}
}

// runScriptHooks runs external hook scripts for a given event
func runScriptHooks(ctx context.Context, event hooks.EventType, workDir, projectName string, cfg *config.Config, dnsEnabled, verbose bool) {
	// Check if hooks directory exists
	hooksDir := filepath.Join(workDir, ".space", "hooks", string(event)+".d")
	if _, err := os.Stat(hooksDir); os.IsNotExist(err) {
		if verbose {
			fmt.Printf("   [verbose] No hooks directory found: %s\n", hooksDir)
		}
		return // No hooks directory for this event
	}

	// Create script executor
	executor := hooks.NewScriptExecutor(workDir)

	fmt.Println()
	fmt.Printf("🪝 Running %s hooks...\n", event)

	if verbose {
		fmt.Printf("   [verbose] Hooks directory: %s\n", hooksDir)
	}

	// Build hook context
	hookCtx := &hooks.HookContext{
		WorkDir:     workDir,
		ProjectName: projectName,
		DNSEnabled:  dnsEnabled,
		BaseDomain:  "space.local",
		Hash:        dns.GenerateDirectoryHash(workDir),
		Services:    make(map[string]*hooks.ServiceInfo),
		Metadata:    make(map[string]interface{}),
	}

	if verbose {
		fmt.Printf("   [verbose] Hook context:\n")
		fmt.Printf("             WorkDir: %s\n", hookCtx.WorkDir)
		fmt.Printf("             ProjectName: %s\n", hookCtx.ProjectName)
		fmt.Printf("             DNSEnabled: %t\n", hookCtx.DNSEnabled)
		fmt.Printf("             Hash: %s\n", hookCtx.Hash)
	}

	// Add services from config with DNS names or localhost
	for name, svc := range cfg.Services {
		var serviceHost string
		var serviceURL string
		if dnsEnabled {
			dnsName := fmt.Sprintf("%s-%s.%s", name, hookCtx.Hash, hookCtx.BaseDomain)
			serviceHost = dnsName
			serviceURL = fmt.Sprintf("http://%s:%d", dnsName, svc.Port)
		} else {
			// Non-DNS mode: use localhost with external port
			serviceHost = "localhost"
			port := svc.ExternalPort
			if port == 0 {
				port = svc.Port
			}
			serviceURL = fmt.Sprintf("http://localhost:%d", port)
		}

		hookCtx.Services[name] = &hooks.ServiceInfo{
			Name:         name,
			DNSName:      serviceHost,
			InternalPort: svc.Port,
			ExternalPort: svc.ExternalPort,
			URL:          serviceURL,
		}

		if verbose {
			fmt.Printf("   [verbose] Service '%s': host=%s, port=%d, url=%s\n",
				name, serviceHost, svc.Port, serviceURL)
		}
	}

	if verbose && len(cfg.Services) == 0 {
		fmt.Printf("   [verbose] No services defined in config - hooks may not have service info\n")
		fmt.Printf("   [verbose] Consider creating a .space.yaml with service definitions\n")
	}

	// List scripts that will be executed
	if verbose {
		entries, err := os.ReadDir(hooksDir)
		if err == nil {
			fmt.Printf("   [verbose] Scripts in directory:\n")
			for _, entry := range entries {
				if !entry.IsDir() {
					info, _ := entry.Info()
					executable := info != nil && info.Mode()&0111 != 0
					status := "skipped (not executable)"
					if executable {
						status = "will execute"
					}
					if strings.HasSuffix(entry.Name(), ".md") || strings.HasSuffix(entry.Name(), ".txt") {
						status = "skipped (documentation)"
					}
					if entry.Name() == ".gitkeep" {
						status = "skipped (gitkeep)"
					}
					fmt.Printf("             %s - %s\n", entry.Name(), status)
				}
			}
		}
	}

	// Execute scripts
	if err := executor.Execute(ctx, event, hookCtx); err != nil {
		fmt.Printf("   ⚠️  Hook execution failed: %v\n", err)
	}
}
