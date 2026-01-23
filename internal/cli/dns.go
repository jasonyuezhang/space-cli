package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

func newDNSCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dns",
		Short: "Manage DNS daemon",
		Long:  "Manage the space-dns-daemon for container DNS resolution.",
	}

	cmd.AddCommand(newDNSStatusCommand())
	cmd.AddCommand(newDNSStopCommand())
	cmd.AddCommand(newDNSStartCommand())
	cmd.AddCommand(newDNSRestartCommand())

	return cmd
}

func newDNSStatusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show DNS daemon status",
		RunE: func(cmd *cobra.Command, args []string) error {
			state, err := loadDNSState()
			if err != nil {
				fmt.Println("❌ DNS daemon is not running")
				fmt.Printf("   State file: %s\n", getDNSStateFile())
				return nil
			}

			fmt.Println("✅ space-dns-daemon is running")
			fmt.Printf("   Address:      %s\n", state.Address)
			fmt.Printf("   Project:      %s\n", state.ProjectName)
			fmt.Printf("   Started:      %s\n", state.StartTime.Format(time.RFC3339))
			fmt.Printf("   Uptime:       %s\n", time.Since(state.StartTime).Round(time.Second))
			fmt.Printf("   State file:   %s\n", getDNSStateFile())
			fmt.Println()
			fmt.Println("📡 DNS Configuration:")
			fmt.Printf("   Domain:       *.space.local\n")
			fmt.Printf("   Resolver:     /etc/resolver/space.local\n")
			fmt.Println()
			fmt.Println("💡 Test DNS resolution:")
			fmt.Printf("   dig @%s postgres.space.local\n", state.Address)

			return nil
		},
	}
}

func newDNSStopCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Stop DNS daemon",
		RunE: func(cmd *cobra.Command, args []string) error {
			state, err := loadDNSState()
			if err != nil {
				fmt.Println("ℹ️  DNS daemon is not running")
				return nil
			}

			fmt.Printf("🛑 Stopping space-dns-daemon (%s)...\n", state.Address)

			// Note: We can't actually stop a server running in another process
			// This just removes the state file as the server runs in the parent process
			if err := removeDNSState(); err != nil {
				return fmt.Errorf("failed to remove DNS state: %w", err)
			}

			fmt.Println("✅ DNS daemon stopped")
			fmt.Println()
			fmt.Println("💡 Note: DNS resolver configuration at /etc/resolver/space.local is preserved")
			fmt.Println("   Run 'space dns start' to start the daemon again")

			return nil
		},
	}
}

func newDNSStartCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start DNS daemon",
		Long: `Start the space-dns-daemon in the foreground.

The DNS daemon will continue running until stopped with Ctrl+C or 'space dns stop'.
To run in the background, use: space dns start &`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Check if already running
			if isDNSServerRunning() {
				state, _ := loadDNSState()
				fmt.Printf("ℹ️  DNS daemon is already running on %s\n", state.Address)
				return nil
			}

			ctx := context.Background()
			// Empty project name means search all projects
			projectName := ""

			fmt.Println("🌐 Starting space-dns-daemon...")

			if err := startDNSServer(ctx, projectName); err != nil {
				return fmt.Errorf("failed to start DNS daemon: %w", err)
			}

			state, _ := loadDNSState()
			fmt.Printf("✅ DNS daemon started on %s\n", state.Address)
			fmt.Println("🔄 DNS daemon is running... (Press Ctrl+C to stop)")
			fmt.Println()
			fmt.Println("💡 Containers will be accessible at: *.space.local")
			fmt.Println("💡 To run in background: space dns start &")
			fmt.Println()

			// Block forever to keep the DNS server running
			select {}
		},
	}

	return cmd
}

func newDNSRestartCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "restart",
		Short: "Restart DNS daemon",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Stop if running
			if isDNSServerRunning() {
				fmt.Println("🛑 Stopping DNS daemon...")
				if err := removeDNSState(); err != nil {
					return fmt.Errorf("failed to stop DNS daemon: %w", err)
				}
				time.Sleep(500 * time.Millisecond)
			}

			// Start
			ctx := context.Background()
			projectName := "space"

			fmt.Println("🌐 Starting space-dns-daemon...")

			if err := startDNSServer(ctx, projectName); err != nil {
				return fmt.Errorf("failed to start DNS daemon: %w", err)
			}

			state, _ := loadDNSState()
			fmt.Printf("✅ DNS daemon restarted on %s\n", state.Address)

			return nil
		},
	}
}
