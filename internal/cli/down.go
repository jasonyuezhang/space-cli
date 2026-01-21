package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/happy-sdk/space-cli/pkg/config"
	"github.com/spf13/cobra"
)

func newDownCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "down",
		Short: "Stop and remove services",
		Long:  "Stop all running services and remove containers, networks, and volumes.",
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

			fmt.Printf("🛑 Stopping services for project: %s\n", cfg.Project.Name)
			fmt.Printf("📁 Working directory: %s\n", workDir)
			fmt.Println()

			// Generate project name
			projectName := generateProjectName(cfg, workDir)
			fmt.Printf("📦 Project name: %s\n", projectName)
			fmt.Println()

			// Clean up DNS server if running
			if globalDNSServer != nil && globalDNSServer.IsRunning() {
				cleanupDNSServer(ctx)
				fmt.Println()
			}

			// Build docker compose command
			composeCmd := []string{"docker", "compose"}

			// Add compose files
			for _, file := range cfg.Project.ComposeFiles {
				composeCmd = append(composeCmd, "-f", file)
			}

			// Add project name
			composeCmd = append(composeCmd, "-p", projectName)

			// Add down command
			composeCmd = append(composeCmd, "down")

			// Execute docker compose
			dockerCmd := exec.Command(composeCmd[0], composeCmd[1:]...)
			dockerCmd.Dir = workDir
			dockerCmd.Stdout = os.Stdout
			dockerCmd.Stderr = os.Stderr
			dockerCmd.Stdin = os.Stdin

			fmt.Printf("🔧 Running: %s\n", strings.Join(composeCmd, " "))
			fmt.Println()

			if err := dockerCmd.Run(); err != nil {
				return fmt.Errorf("failed to stop services: %w", err)
			}

			fmt.Println()
			fmt.Println("✅ Services stopped successfully!")

			return nil
		},
	}
}
