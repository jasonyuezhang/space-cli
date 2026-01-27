package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/happy-sdk/space-cli/pkg/config"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func newConfigCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Configuration management",
		Long:  "Manage space-cli configuration.",
	}

	cmd.AddCommand(newConfigShowCommand())
	cmd.AddCommand(newConfigValidateCommand())

	return cmd
}

func newConfigShowCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Show merged configuration",
		Long:  "Display the final merged configuration from all sources.",
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

			// Marshal to YAML for display
			data, err := yaml.Marshal(cfg)
			if err != nil {
				return fmt.Errorf("failed to marshal configuration: %w", err)
			}

			// Print configuration
			fmt.Println("# Merged Configuration")
			fmt.Println("# Sources: defaults → global (~/.config/space/config.yaml) → project (.space.yaml)")
			fmt.Println("# Working directory:", workDir)
			fmt.Println()
			fmt.Print(string(data))

			return nil
		},
	}
}

func newConfigValidateCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "Validate configuration",
		Long:  "Check configuration for errors and warnings.",
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
				fmt.Println("❌ Configuration validation failed:")
				fmt.Println(err)
				return err
			}

			fmt.Println("✅ Configuration is valid")
			fmt.Println("Working directory:", workDir)
			fmt.Println("Project:", cfg.Project.Name)
			fmt.Println("Services:", len(cfg.Services))

			return nil
		},
	}
}
