package dns

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ResolverManager manages /etc/resolver configuration
type ResolverManager struct {
	domain      string
	resolverDir string
	dnsAddr     string
	logger      Logger
}

// NewResolverManager creates a new resolver manager
func NewResolverManager(domain, dnsAddr string, logger Logger) *ResolverManager {
	return &ResolverManager{
		domain:      domain,
		resolverDir: "/etc/resolver",
		dnsAddr:     dnsAddr,
		logger:      logger,
	}
}

// Setup creates the resolver configuration
func (r *ResolverManager) Setup(ctx context.Context) error {
	resolverFile := filepath.Join(r.resolverDir, r.domain)

	// Check if resolver directory exists
	if _, err := os.Stat(r.resolverDir); os.IsNotExist(err) {
		r.logger.Info("Creating resolver directory", "dir", r.resolverDir)
		if err := r.runSudo(ctx, "mkdir", "-p", r.resolverDir); err != nil {
			return fmt.Errorf("failed to create resolver directory: %w", err)
		}
	}

	// Check if resolver file already exists
	if _, err := os.Stat(resolverFile); err == nil {
		r.logger.Info("Resolver file already exists", "file", resolverFile)
		// Check if it points to our DNS server
		content, err := os.ReadFile(resolverFile)
		if err == nil && strings.Contains(string(content), r.extractHost(r.dnsAddr)) {
			r.logger.Info("Resolver already configured correctly")
			return nil
		}
	}

	// Extract host and port from dnsAddr
	host := r.extractHost(r.dnsAddr)
	port := r.extractPort(r.dnsAddr)

	// Create resolver content
	content := fmt.Sprintf("nameserver %s\nport %s\n", host, port)

	// Write to temporary file first
	tmpFile := filepath.Join(os.TempDir(), r.domain)
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write temp resolver file: %w", err)
	}
	defer os.Remove(tmpFile)

	r.logger.Info("Creating resolver configuration", "file", resolverFile)

	// Copy to /etc/resolver with sudo
	if err := r.runSudo(ctx, "cp", tmpFile, resolverFile); err != nil {
		return fmt.Errorf("failed to create resolver file: %w", err)
	}

	// Flush DNS cache
	r.logger.Info("Flushing DNS cache")
	if err := r.flushDNSCache(ctx); err != nil {
		r.logger.Warn("Failed to flush DNS cache", "error", err)
		// Don't fail on this
	}

	r.logger.Info("Resolver configured successfully")
	return nil
}

// Cleanup removes the resolver configuration
func (r *ResolverManager) Cleanup(ctx context.Context) error {
	resolverFile := filepath.Join(r.resolverDir, r.domain)

	// Check if file exists
	if _, err := os.Stat(resolverFile); os.IsNotExist(err) {
		r.logger.Info("Resolver file does not exist, nothing to clean up")
		return nil
	}

	r.logger.Info("Removing resolver configuration", "file", resolverFile)

	// Remove with sudo
	if err := r.runSudo(ctx, "rm", "-f", resolverFile); err != nil {
		return fmt.Errorf("failed to remove resolver file: %w", err)
	}

	// Flush DNS cache
	r.logger.Info("Flushing DNS cache")
	if err := r.flushDNSCache(ctx); err != nil {
		r.logger.Warn("Failed to flush DNS cache", "error", err)
		// Don't fail on this
	}

	r.logger.Info("Resolver cleaned up successfully")
	return nil
}

// IsConfigured checks if the resolver is already configured
func (r *ResolverManager) IsConfigured() bool {
	resolverFile := filepath.Join(r.resolverDir, r.domain)
	_, err := os.Stat(resolverFile)
	return err == nil
}

// runSudo runs a command with sudo
func (r *ResolverManager) runSudo(ctx context.Context, command string, args ...string) error {
	cmdArgs := append([]string{command}, args...)
	cmd := exec.CommandContext(ctx, "sudo", cmdArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// flushDNSCache flushes the macOS DNS cache
func (r *ResolverManager) flushDNSCache(ctx context.Context) error {
	// Different commands for different macOS versions
	commands := [][]string{
		{"sudo", "dscacheutil", "-flushcache"},
		{"sudo", "killall", "-HUP", "mDNSResponder"},
	}

	for _, cmd := range commands {
		if err := exec.CommandContext(ctx, cmd[0], cmd[1:]...).Run(); err != nil {
			r.logger.Debug("Command failed", "command", strings.Join(cmd, " "), "error", err)
		}
	}

	return nil
}

// extractHost extracts the host from an address (e.g., "127.0.0.1:5353" -> "127.0.0.1")
func (r *ResolverManager) extractHost(addr string) string {
	parts := strings.Split(addr, ":")
	if len(parts) > 0 {
		return parts[0]
	}
	return addr
}

// extractPort extracts the port from an address (e.g., "127.0.0.1:5353" -> "5353")
func (r *ResolverManager) extractPort(addr string) string {
	parts := strings.Split(addr, ":")
	if len(parts) > 1 {
		return parts[1]
	}
	return "53"
}
