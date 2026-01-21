package provider

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// Provider represents a Docker provider
type Provider string

const (
	ProviderOrbStack      Provider = "orbstack"
	ProviderDockerDesktop Provider = "docker-desktop"
	ProviderGeneric       Provider = "generic"
)

// Detector detects the Docker provider
type Detector struct{}

// NewDetector creates a new provider detector
func NewDetector() *Detector {
	return &Detector{}
}

// Detect detects the Docker provider
func (d *Detector) Detect(ctx context.Context) (Provider, error) {
	// Check if OrbStack is running
	if d.isOrbStack(ctx) {
		return ProviderOrbStack, nil
	}

	// Check if Docker Desktop is running
	if d.isDockerDesktop(ctx) {
		return ProviderDockerDesktop, nil
	}

	// Default to generic Docker
	return ProviderGeneric, nil
}

// isOrbStack checks if OrbStack is the Docker provider
func (d *Detector) isOrbStack(ctx context.Context) bool {
	// Check docker context
	cmd := exec.CommandContext(ctx, "docker", "context", "show")
	output, err := cmd.Output()
	if err == nil && strings.Contains(strings.ToLower(string(output)), "orbstack") {
		return true
	}

	// Check docker info
	cmd = exec.CommandContext(ctx, "docker", "info", "--format", "{{.OperatingSystem}}")
	output, err = cmd.Output()
	if err == nil && strings.Contains(strings.ToLower(string(output)), "orbstack") {
		return true
	}

	// Check if orbstack command exists
	cmd = exec.CommandContext(ctx, "which", "orbstack")
	if err := cmd.Run(); err == nil {
		return true
	}

	return false
}

// isDockerDesktop checks if Docker Desktop is the Docker provider
func (d *Detector) isDockerDesktop(ctx context.Context) bool {
	// Check docker context
	cmd := exec.CommandContext(ctx, "docker", "context", "show")
	output, err := cmd.Output()
	if err == nil && strings.Contains(strings.ToLower(string(output)), "desktop") {
		return true
	}

	// Check docker info
	cmd = exec.CommandContext(ctx, "docker", "info", "--format", "{{.OperatingSystem}}")
	output, err = cmd.Output()
	if err == nil && strings.Contains(strings.ToLower(string(output)), "docker desktop") {
		return true
	}

	return false
}

// SupportsContainerDNS returns true if the provider supports container DNS
func (p Provider) SupportsContainerDNS() bool {
	return p == ProviderOrbStack
}

// String returns the string representation of the provider
func (p Provider) String() string {
	return string(p)
}

// Description returns a human-readable description of the provider
func (p Provider) Description() string {
	switch p {
	case ProviderOrbStack:
		return "OrbStack"
	case ProviderDockerDesktop:
		return "Docker Desktop"
	case ProviderGeneric:
		return "Docker"
	default:
		return fmt.Sprintf("Unknown (%s)", p)
	}
}
