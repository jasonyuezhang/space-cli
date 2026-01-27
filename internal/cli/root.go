package cli

import (
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
	Short: "Space CLI - Development environment orchestration",
	Long: `Space CLI is a tool for managing Docker Compose development environments
with intelligent provider detection (OrbStack, Docker Desktop).

Features:
  • Zero-config mode with auto-detection
  • Provider-aware networking (OrbStack DNS, Docker port mapping)
  • Lifecycle hooks for automation
  • Custom commands support`,
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
	rootCmd.AddCommand(newUpCommand())
	rootCmd.AddCommand(newDownCommand())
	rootCmd.AddCommand(newPsCommand())
	rootCmd.AddCommand(newConfigCommand())
	rootCmd.AddCommand(newDNSCommand())
	rootCmd.AddCommand(newHooksCommand())
	rootCmd.AddCommand(newRunCommand())
}
