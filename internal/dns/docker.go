package dns

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// SimpleDockerClient is a simple Docker client using the docker CLI
type SimpleDockerClient struct {
	logger Logger
}

// NewSimpleDockerClient creates a new simple Docker client
func NewSimpleDockerClient(logger Logger) *SimpleDockerClient {
	return &SimpleDockerClient{
		logger: logger,
	}
}

// GetContainerIP gets the IP address of a container
func (c *SimpleDockerClient) GetContainerIP(ctx context.Context, projectName, containerName string) (string, error) {
	// Try exact match first
	ip, err := c.getIP(ctx, projectName, containerName)
	if err == nil && ip != "" {
		return ip, nil
	}

	// Try with project prefix
	ip, err = c.getIP(ctx, projectName, projectName+"-"+containerName)
	if err == nil && ip != "" {
		return ip, nil
	}

	// Try with suffix variants
	suffixes := []string{"-1", "_1"}
	for _, suffix := range suffixes {
		ip, err = c.getIP(ctx, projectName, projectName+"-"+containerName+suffix)
		if err == nil && ip != "" {
			return ip, nil
		}
	}

	return "", fmt.Errorf("container not found: %s", containerName)
}

// getIP gets the IP address of a container by exact name
func (c *SimpleDockerClient) getIP(ctx context.Context, projectName, containerName string) (string, error) {
	// Use docker inspect to get container IP
	cmd := exec.CommandContext(ctx, "docker", "inspect",
		"--format", "{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}",
		containerName)

	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	ip := strings.TrimSpace(string(output))
	if ip == "" {
		return "", fmt.Errorf("no IP address found")
	}

	return ip, nil
}

// ListProjectContainers lists all containers for a project
func (c *SimpleDockerClient) ListProjectContainers(ctx context.Context, projectName string) (map[string]string, error) {
	// Use docker ps to list containers
	cmd := exec.CommandContext(ctx, "docker", "ps",
		"--filter", fmt.Sprintf("label=com.docker.compose.project=%s", projectName),
		"--format", "{{.Names}}")

	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	containers := make(map[string]string)
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")

	for _, line := range lines {
		if line == "" {
			continue
		}

		// Get IP for this container
		ip, err := c.getIP(ctx, projectName, line)
		if err != nil {
			c.logger.Warn("Failed to get IP for container", "container", line, "error", err)
			continue
		}

		// Extract service name from container name
		// Format: projectname-servicename-1
		parts := strings.Split(line, "-")
		if len(parts) >= 2 {
			serviceName := strings.Join(parts[1:len(parts)-1], "-")
			containers[serviceName] = ip
		}
	}

	return containers, nil
}
