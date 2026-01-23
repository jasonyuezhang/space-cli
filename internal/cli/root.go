package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	// Version is set at build time
	Version = "dev"

	// Workdir is the working directory
	Workdir string
)

// rootCmd represents the base command
var rootCmd = &cobra.Command{
	Use:   "space",
	Short: "Space CLI - Generic development environment orchestration",
	Long: `Space CLI is a generic tool for managing Docker Compose and VM-based
development environments with support for Docker providers (OrbStack, Docker Desktop)
and VM providers (OrbStack, Lima).

Features:
  • Zero-config mode with auto-detection
  • Provider-aware networking (OrbStack DNS, Docker port mapping)
  • Smart port allocation with persistence
  • Database operations (create, migrate, seed)
  • VM management (OrbStack VM, Lima)
  • Framework presets (Rails, Node.js, Go)`,
	Version: Version,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVarP(&Workdir, "workdir", "w", ".", "working directory")

	// Add subcommands
	rootCmd.AddCommand(newInitCommand())
	rootCmd.AddCommand(newUpCommand())
	rootCmd.AddCommand(newDownCommand())
	rootCmd.AddCommand(newStatusCommand())
	rootCmd.AddCommand(newRestartCommand())
	rootCmd.AddCommand(newLogsCommand())
	rootCmd.AddCommand(newShellCommand())
	rootCmd.AddCommand(newLinksCommand())
	rootCmd.AddCommand(newBuildCommand())
	rootCmd.AddCommand(newConfigCommand())
	rootCmd.AddCommand(newDBCommand())
	rootCmd.AddCommand(newVMCommand())
	rootCmd.AddCommand(newMigrateCommand())
	rootCmd.AddCommand(newDNSCommand())
}

// Placeholder commands - to be implemented

func newInitCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize space-cli configuration",
		Long:  "Create a .space.yaml configuration file with auto-detected settings or a preset.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("not yet implemented")
		},
	}

	cmd.Flags().String("preset", "", "Use a preset: rails, nodejs, go, propel")

	return cmd
}

// Up command is now in up.go
// Down command is now in down.go

func newStatusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show service status",
		Long:  "Display the status of all services (running, stopped, health).",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("not yet implemented")
		},
	}
}

func newRestartCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "restart [services...]",
		Short: "Restart services",
		Long:  "Restart all services or specific services.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("not yet implemented")
		},
	}
}

func newLogsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logs [service]",
		Short: "View service logs",
		Long:  "Stream logs from a specific service or all services.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("not yet implemented")
		},
	}

	cmd.Flags().BoolP("follow", "f", false, "Follow log output")
	cmd.Flags().IntP("tail", "n", 100, "Number of lines to show from the end of the logs")

	return cmd
}

func newShellCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "shell [service]",
		Short: "Open a shell in a service container",
		Long:  "Open an interactive shell in the specified service container.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("not yet implemented")
		},
	}

	cmd.Flags().String("user", "", "User to run the shell as")
	cmd.Flags().String("shell", "", "Shell to use (default: /bin/sh)")

	return cmd
}

func newLinksCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "links",
		Short: "Display service URLs",
		Long:  "Show URLs for accessing services (provider-aware).",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("not yet implemented")
		},
	}
}

func newBuildCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "build [services...]",
		Short: "Build service images",
		Long:  "Build or rebuild service images.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("not yet implemented")
		},
	}

	cmd.Flags().Bool("no-cache", false, "Do not use cache when building")
	cmd.Flags().Bool("pull", false, "Always attempt to pull newer version of base images")

	return cmd
}

// Config command is now in config.go

func newDBCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "db",
		Short: "Database operations",
		Long:  "Manage databases (create, drop, migrate, seed).",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "shell [database]",
		Short: "Open database shell",
		Long:  "Open an interactive database shell.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("not yet implemented")
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "create <database>",
		Short: "Create database",
		Long:  "Create a new database.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("not yet implemented")
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "drop <database>",
		Short: "Drop database",
		Long:  "Drop an existing database.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("not yet implemented")
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "migrate [database]",
		Short: "Run database migrations",
		Long:  "Run pending database migrations.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("not yet implemented")
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "seed [database]",
		Short: "Seed database",
		Long:  "Populate database with seed data.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("not yet implemented")
		},
	})

	return cmd
}

func newVMCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vm",
		Short: "VM management",
		Long:  "Manage development VMs (Lima, OrbStack VM).",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "start",
		Short: "Start VM",
		Long:  "Create and start the development VM.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("not yet implemented")
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "stop",
		Short: "Stop VM",
		Long:  "Stop the running VM.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("not yet implemented")
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "restart",
		Short: "Restart VM",
		Long:  "Restart the VM.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("not yet implemented")
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "status",
		Short: "VM status",
		Long:  "Show VM status and resource usage.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("not yet implemented")
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "shell",
		Short: "VM shell",
		Long:  "Open a shell in the VM.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("not yet implemented")
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "delete",
		Short: "Delete VM",
		Long:  "Delete the VM and all its data.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("not yet implemented")
		},
	})

	return cmd
}

func newMigrateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Migrate from other tools",
		Long:  "Migrate configuration from propel-cli or other development tools.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("not yet implemented")
		},
	}

	cmd.Flags().String("from", "", "Tool to migrate from: propel-cli")
	cmd.MarkFlagRequired("from")

	return cmd
}
