package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/happy-sdk/space-cli/internal/provider"
	"github.com/happy-sdk/space-cli/pkg/config"
	"github.com/spf13/cobra"
)

// Container represents a running container
type Container struct {
	ID       string
	Name     string
	Image    string
	Status   string
	Ports    string
	Command  string
	IsDNS    bool
	Provider provider.Provider
}

// ServiceStatus represents the status of a single service
type ServiceStatus struct {
	Name      string   `json:"name"`
	State     string   `json:"state"`
	Status    string   `json:"status"`
	Ports     []string `json:"ports"`
	DNSUrls   []string `json:"dns_urls,omitempty"`
	LocalUrls []string `json:"local_urls,omitempty"`
}

// newPsCommand creates the ps command
func newPsCommand() *cobra.Command {
	var quiet bool
	var noTrunc bool
	var showAll bool
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "ps",
		Short: "List running containers",
		Long:  "Display the status of all running services and containers for the project.",
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

			// Detect provider
			detector := provider.NewDetector()
			providerType, err := detector.Detect(ctx)
			if err != nil {
				providerType = provider.ProviderGeneric
			}

			// Generate project name
			projectName := generateProjectName(cfg, workDir)

			// If quiet flag is used, run simple ps command
			if quiet {
				return runPsCommand(ctx, workDir, projectName, providerType, quiet, noTrunc)
			}

			// Otherwise run enhanced ps with DNS and URL support
			return runEnhancedPS(ctx, workDir, cfg, projectName, showAll, jsonOutput)
		},
	}

	cmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "Only display container IDs")
	cmd.Flags().BoolVar(&noTrunc, "no-trunc", false, "Don't truncate output")
	cmd.Flags().BoolVarP(&showAll, "all", "a", false, "Show all services including stopped")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")

	return cmd
}

// runEnhancedPS runs the enhanced ps command with DNS and URL support
func runEnhancedPS(ctx context.Context, workDir string, cfg *config.Config, projectName string, showAll bool, jsonOutput bool) error {
	// Check if DNS mode is active
	useDNS := isDNSServerRunning()

	// Get service status from docker-compose ps
	services, err := getDockerComposePS(ctx, workDir, cfg, projectName, showAll)
	if err != nil {
		return fmt.Errorf("failed to get service status: %w", err)
	}

	if len(services) == 0 {
		fmt.Println("No services running.")
		fmt.Println()
		fmt.Println("💡 Tip: Run 'space up' to start services")
		return nil
	}

	// Output results
	if jsonOutput {
		return outputJSON(services)
	}

	return outputTable(services, useDNS, cfg)
}

// runPsCommand executes the docker compose ps command (legacy/quiet mode)
func runPsCommand(ctx context.Context, workDir, projectName string, providerType provider.Provider, quiet, noTrunc bool) error {
	// Load config to get compose files
	loader, err := config.NewLoader(workDir)
	if err != nil {
		return fmt.Errorf("failed to create config loader: %w", err)
	}

	cfg, err := loader.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Check if docker-compose.yml exists
	composeFiles := cfg.Project.ComposeFiles
	if len(composeFiles) == 0 {
		// Default to docker-compose.yml
		composeFiles = []string{"docker-compose.yml"}
	}

	// Verify at least one compose file exists
	composeFileExists := false
	for _, file := range composeFiles {
		absPath := file
		if !filepath.IsAbs(file) {
			absPath = filepath.Join(workDir, file)
		}
		if _, err := os.Stat(absPath); err == nil {
			composeFileExists = true
			break
		}
	}

	if !composeFileExists {
		return fmt.Errorf("no docker-compose files found in %s", workDir)
	}

	// Build docker compose ps command
	composeCmd := []string{"docker", "compose"}

	// Add compose files
	for _, file := range composeFiles {
		composeCmd = append(composeCmd, "-f", file)
	}

	// Add project name
	composeCmd = append(composeCmd, "-p", projectName)

	// Add ps command
	composeCmd = append(composeCmd, "ps")

	if quiet {
		composeCmd = append(composeCmd, "-q")
	}

	if noTrunc {
		composeCmd = append(composeCmd, "--no-trunc")
	}

	// Execute docker compose
	dockerCmd := exec.Command(composeCmd[0], composeCmd[1:]...)
	dockerCmd.Dir = workDir
	dockerCmd.Stdout = os.Stdout
	dockerCmd.Stderr = os.Stderr
	dockerCmd.Stdin = os.Stdin

	if err := dockerCmd.Run(); err != nil {
		// Check if docker-compose file is not found
		if strings.Contains(err.Error(), "no such file") {
			return fmt.Errorf("docker-compose file not found in %s", workDir)
		}
		// Check if no containers are running
		if dockerCmd.ProcessState != nil && dockerCmd.ProcessState.ExitCode() == 0 {
			// Exit code 0 with no containers is not an error
			return nil
		}
		return fmt.Errorf("failed to list services: %w", err)
	}

	return nil
}

// getDockerComposePS executes docker-compose ps and parses the output
func getDockerComposePS(ctx context.Context, workDir string, cfg *config.Config, projectName string, showAll bool) ([]ServiceStatus, error) {
	// Build docker compose ps command
	composeCmd := []string{"docker", "compose"}

	// Add compose files
	for _, file := range cfg.Project.ComposeFiles {
		composeCmd = append(composeCmd, "-f", file)
	}

	// Add project name
	composeCmd = append(composeCmd, "-p", projectName)

	// Add ps command with --format json
	composeCmd = append(composeCmd, "ps", "--format", "json")

	// Add --all flag if requested
	if showAll {
		composeCmd = append(composeCmd, "--all")
	}

	// Execute command
	dockerCmd := exec.CommandContext(ctx, composeCmd[0], composeCmd[1:]...)
	dockerCmd.Dir = workDir

	var stdout, stderr bytes.Buffer
	dockerCmd.Stdout = &stdout
	dockerCmd.Stderr = &stderr

	if err := dockerCmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to execute docker compose ps: %w (stderr: %s)", err, stderr.String())
	}

	// Parse JSON output
	output := stdout.String()
	if output == "" {
		return []ServiceStatus{}, nil
	}

	// Docker compose ps --format json outputs one JSON object per line
	lines := strings.Split(strings.TrimSpace(output), "\n")
	services := make([]ServiceStatus, 0, len(lines))

	for _, line := range lines {
		if line == "" {
			continue
		}

		var rawService struct {
			Name       string `json:"Name"`
			Service    string `json:"Service"`
			State      string `json:"State"`
			Status     string `json:"Status"`
			Publishers []struct {
				URL           string `json:"URL"`
				TargetPort    int    `json:"TargetPort"`
				PublishedPort int    `json:"PublishedPort"`
				Protocol      string `json:"Protocol"`
			} `json:"Publishers"`
		}

		if err := json.Unmarshal([]byte(line), &rawService); err != nil {
			return nil, fmt.Errorf("failed to parse service status: %w", err)
		}

		// Extract service name (use Service field, fallback to Name)
		serviceName := rawService.Service
		if serviceName == "" {
			serviceName = rawService.Name
		}

		// Extract ports
		ports := make([]string, 0, len(rawService.Publishers))
		for _, pub := range rawService.Publishers {
			if pub.PublishedPort > 0 {
				ports = append(ports, fmt.Sprintf("%d:%d/%s", pub.PublishedPort, pub.TargetPort, pub.Protocol))
			}
		}

		// Build service status
		status := ServiceStatus{
			Name:   serviceName,
			State:  rawService.State,
			Status: rawService.Status,
			Ports:  ports,
		}

		// Add DNS URLs if DNS mode is active
		if isDNSServerRunning() {
			status.DNSUrls = generateDNSUrls(serviceName, cfg)
		}

		// Add local URLs
		status.LocalUrls = generateLocalUrls(serviceName, cfg, rawService.Publishers)

		services = append(services, status)
	}

	return services, nil
}

// generateDNSUrls generates .space.local URLs for a service
func generateDNSUrls(serviceName string, cfg *config.Config) []string {
	urls := make([]string, 0)

	// Check if service has port configured
	if svc, ok := cfg.Services[serviceName]; ok && svc.Port > 0 {
		urls = append(urls, fmt.Sprintf("http://%s.space.local:%d", serviceName, svc.Port))
	}

	return urls
}

// generateLocalUrls generates localhost URLs based on published ports
func generateLocalUrls(serviceName string, cfg *config.Config, publishers []struct {
	URL           string `json:"URL"`
	TargetPort    int    `json:"TargetPort"`
	PublishedPort int    `json:"PublishedPort"`
	Protocol      string `json:"Protocol"`
}) []string {
	urls := make([]string, 0)

	// Use published ports from docker-compose ps
	for _, pub := range publishers {
		if pub.PublishedPort > 0 && pub.Protocol == "tcp" {
			urls = append(urls, fmt.Sprintf("http://localhost:%d", pub.PublishedPort))
		}
	}

	// Fallback to configured ports if no published ports
	if len(urls) == 0 {
		if svc, ok := cfg.Services[serviceName]; ok {
			if svc.ExternalPort > 0 {
				urls = append(urls, fmt.Sprintf("http://localhost:%d", svc.ExternalPort))
			} else if svc.Port > 0 {
				urls = append(urls, fmt.Sprintf("http://localhost:%d", svc.Port))
			}
		}
	}

	return urls
}

// outputJSON outputs service status as JSON
func outputJSON(services []ServiceStatus) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(services)
}

// outputTable outputs service status as a formatted table
func outputTable(services []ServiceStatus, useDNS bool, cfg *config.Config) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	defer w.Flush()

	// Print header
	fmt.Println()
	if useDNS {
		fmt.Fprintln(w, "SERVICE\tSTATE\tPORTS\tDNS URL\tLOCAL URL")
		fmt.Fprintln(w, "-------\t-----\t-----\t-------\t---------")
	} else {
		fmt.Fprintln(w, "SERVICE\tSTATE\tPORTS\tLOCAL URL")
		fmt.Fprintln(w, "-------\t-----\t-----\t---------")
	}

	// Print services
	for _, svc := range services {
		ports := "-"
		if len(svc.Ports) > 0 {
			ports = strings.Join(svc.Ports, ", ")
		}

		dnsUrl := "-"
		if len(svc.DNSUrls) > 0 {
			dnsUrl = svc.DNSUrls[0]
		}

		localUrl := "-"
		if len(svc.LocalUrls) > 0 {
			localUrl = svc.LocalUrls[0]
		}

		// Format state with color indicators
		stateDisplay := svc.State
		switch strings.ToLower(svc.State) {
		case "running":
			stateDisplay = "✅ " + svc.State
		case "exited":
			stateDisplay = "🛑 " + svc.State
		case "restarting":
			stateDisplay = "🔄 " + svc.State
		case "paused":
			stateDisplay = "⏸️  " + svc.State
		}

		if useDNS {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
				svc.Name,
				stateDisplay,
				ports,
				dnsUrl,
				localUrl,
			)
		} else {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
				svc.Name,
				stateDisplay,
				ports,
				localUrl,
			)
		}
	}

	w.Flush()
	fmt.Println()

	// Print DNS mode status
	if useDNS {
		state, _ := loadDNSState()
		fmt.Printf("🌐 DNS Mode: Active (daemon running on %s)\n", state.Address)
		fmt.Println("   Services are accessible via .space.local domains")
	} else {
		fmt.Println("🔌 DNS Mode: Inactive (using port bindings)")
		fmt.Println("   Use 'space up' to enable DNS mode on supported providers")
	}

	fmt.Println()
	fmt.Println("💡 Tip: Run 'space logs <service>' to view service logs")
	fmt.Println("💡 Tip: Run 'space shell <service>' to access a service shell")

	return nil
}

// ParseContainers parses docker compose ps output into Container structs
// This is a utility function for testing
func ParseContainers(output string) []Container {
	containers := []Container{}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) <= 1 {
		// Header only or empty
		return containers
	}

	// Skip header line
	for _, line := range lines[1:] {
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Parse container line
		// Format: CONTAINER ID | IMAGE | COMMAND | CREATED | STATUS | PORTS | NAMES
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		container := Container{
			ID:   parts[0],
			Name: parts[len(parts)-1], // Last part is typically the name
		}

		if len(parts) > 1 {
			container.Image = parts[1]
		}
		if len(parts) > 2 {
			container.Command = parts[2]
		}
		if len(parts) > 4 {
			container.Status = parts[4]
		}

		containers = append(containers, container)
	}

	return containers
}
