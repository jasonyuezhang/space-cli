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
	// If projectName is empty, search all containers
	if projectName == "" {
		return c.findContainerIPAcrossProjects(ctx, containerName)
	}

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

// findContainerIPAcrossProjects searches all containers for a matching service name
func (c *SimpleDockerClient) findContainerIPAcrossProjects(ctx context.Context, serviceName string) (string, error) {
	// List all running containers
	cmd := exec.CommandContext(ctx, "docker", "ps", "--format", "{{.Names}}")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to list containers: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")

	// Try to find container matching the service name
	// Container name format: projectname-servicename-1 or projectname-servicename_1
	for _, containerName := range lines {
		if containerName == "" {
			continue
		}

		// Check if container name contains the service name
		// Match patterns like: *-servicename-1, *-servicename_1, *-servicename
		if c.matchesServiceName(containerName, serviceName) {
			ip, err := c.getIP(ctx, "", containerName)
			if err == nil && ip != "" {
				return ip, nil
			}
		}
	}

	return "", fmt.Errorf("container not found for service: %s", serviceName)
}

// matchesServiceName checks if a container name matches the service name pattern
func (c *SimpleDockerClient) matchesServiceName(containerName, serviceName string) bool {
	// Container format: projectname-servicename-1 or projectname-servicename_1
	// Service format: servicename (without hash, hash is only in DNS domain)

	// Check if container name ends with -servicename-1
	if strings.HasSuffix(containerName, "-"+serviceName+"-1") {
		return true
	}

	// Check if container name ends with -servicename_1
	if strings.HasSuffix(containerName, "-"+serviceName+"_1") {
		return true
	}

	// Check if container name ends with _servicename-1 (less common but possible)
	if strings.HasSuffix(containerName, "_"+serviceName+"-1") {
		return true
	}

	// Check if container name ends with _servicename_1
	if strings.HasSuffix(containerName, "_"+serviceName+"_1") {
		return true
	}

	return false
}
