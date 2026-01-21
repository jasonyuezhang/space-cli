package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/happy-sdk/space-cli/pkg/config"
	"github.com/spf13/cobra"
)

func newUpCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "up [services...]",
		Short: "Start services",
		Long:  "Start all services or specific services defined in docker-compose.yml.",
		RunE: func(cmd *cobra.Command, args []string) error {
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

			// Build docker compose command
			composeCmd := []string{"docker", "compose"}

			// Add compose files
			for _, file := range cfg.Project.ComposeFiles {
				composeCmd = append(composeCmd, "-f", file)
			}

			// Add project name
			if cfg.Project.Name != "" {
				// Generate project name based on naming strategy
				projectName := generateProjectName(cfg, workDir)
				composeCmd = append(composeCmd, "-p", projectName)
				fmt.Printf("📦 Project name: %s\n", projectName)
			}

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
				return fmt.Errorf("failed to start services: %w", err)
			}

			fmt.Println()
			fmt.Println("✅ Services started successfully!")
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
