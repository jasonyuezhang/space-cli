package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/happy-sdk/space-cli/internal/dns"
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
				fmt.Println("🌐 Starting embedded DNS server for container access...")

				if err := startDNSServer(ctx, projectName); err != nil {
					fmt.Printf("⚠️  Failed to start DNS server: %v\n", err)
					fmt.Println("⚠️  Falling back to port bindings")
				} else {
					useDNS = true
					fmt.Println("✅ DNS server started successfully")
					fmt.Printf("   Containers will be accessible at: *.orb.local\n")

					// Create override file to remove port bindings
					overrideFile, err = createNoPortsOverride(workDir, cfg)
					if err != nil {
						fmt.Printf("⚠️  Failed to create override file: %v\n", err)
						// Continue anyway - docker-compose will use original ports
					}
				}
			}

			fmt.Println()

			// Build docker compose command
			composeCmd := []string{"docker", "compose"}

			// Add compose files
			for _, file := range cfg.Project.ComposeFiles {
				composeCmd = append(composeCmd, "-f", file)
			}

			// Add override file if using DNS
			if overrideFile != "" {
				composeCmd = append(composeCmd, "-f", overrideFile)
				defer os.Remove(overrideFile) // Clean up override file
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
				// Clean up DNS server on failure
				if useDNS {
					cleanupDNSServer(ctx)
				}
				return fmt.Errorf("failed to start services: %w", err)
			}

			fmt.Println()
			fmt.Println("✅ Services started successfully!")
			fmt.Println()

			// Show access information
			if useDNS {
				fmt.Println("🌍 Access your services at:")
				for serviceName, service := range cfg.Services {
					if service.Port > 0 {
						fmt.Printf("   • %s: http://%s.orb.local:%d\n", serviceName, serviceName, service.Port)
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

	switch cfg.Project.NamingStrategy {
	case "git-branch":
		// Try to get git branch
		branch := getGitBranch(workDir)
		if branch != "" {
			// Clean branch name (remove special chars)
			branch = strings.ReplaceAll(branch, "/", "-")
			branch = strings.ReplaceAll(branch, "_", "-")
			return prefix + branch
		}
		fallthrough

	case "directory":
		// Use directory name
		return prefix + filepath.Base(workDir)

	case "static":
		fallthrough

	default:
		// Use configured name
		return baseName
	}
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

// startDNSServer starts the embedded DNS server
func startDNSServer(ctx context.Context, projectName string) error {
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

		// Create DNS server
		var err error
		server, err = dns.NewServer(dns.Config{
			Addr:        dnsAddr,
			Upstream:    "8.8.8.8:53",
			ProjectName: projectName,
			Domain:      "orb.local",
			CacheTTL:    30 * time.Second,
			Docker:      dockerClient,
			Logger:      logger,
		})
		if err != nil {
			lastErr = err
			continue
		}

		// Start DNS server
		if err := server.Start(ctx); err != nil {
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

	// Store global reference for cleanup
	globalDNSServer = server

	// Create resolver manager
	resolver := dns.NewResolverManager("orb.local", dnsAddr, logger)
	globalDNSResolver = resolver

	// Setup resolver (requires sudo)
	fmt.Println("📝 Setting up DNS resolver (may require sudo password)...")
	if err := resolver.Setup(ctx); err != nil {
		// Clean up server if resolver setup fails
		server.Stop()
		return fmt.Errorf("failed to setup resolver: %w", err)
	}

	return nil
}

// createNoPortsOverride creates a docker-compose override file that removes port bindings
func createNoPortsOverride(workDir string, cfg *config.Config) (string, error) {
	// Create override content
	override := map[string]interface{}{
		"services": make(map[string]interface{}),
	}

	services := override["services"].(map[string]interface{})
	for serviceName := range cfg.Services {
		services[serviceName] = map[string]interface{}{
			"ports": []string{}, // Remove all port bindings
		}
	}

	// Write to temporary file
	overrideFile := filepath.Join(workDir, ".space-override.yml")
	data, err := yaml.Marshal(override)
	if err != nil {
		return "", err
	}

	if err := os.WriteFile(overrideFile, data, 0644); err != nil {
		return "", err
	}

	return overrideFile, nil
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
		fmt.Println("🛑 Stopping DNS server...")
		if err := globalDNSServer.Stop(); err != nil {
			fmt.Printf("⚠️  Failed to stop DNS server: %v\n", err)
		}
		globalDNSServer = nil
	}
}
