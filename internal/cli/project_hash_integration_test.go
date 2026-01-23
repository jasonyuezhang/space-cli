package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/happy-sdk/space-cli/pkg/config"
)

// TestDNSCollisionPrevention_MultipleWorktrees tests the complete DNS collision prevention
// scenario with multiple worktrees pointing to the same docker-compose.yml
func TestDNSCollisionPrevention_MultipleWorktrees(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create temporary directories to simulate worktrees
	tmpDir := t.TempDir()

	worktrees := []struct {
		name string
		path string
	}{
		{name: "main", path: filepath.Join(tmpDir, "project-main")},
		{name: "dev", path: filepath.Join(tmpDir, "project-dev")},
		{name: "feature-auth", path: filepath.Join(tmpDir, "project-feature-auth")},
	}

	// Create worktree directories
	for _, wt := range worktrees {
		if err := os.MkdirAll(wt.path, 0755); err != nil {
			t.Fatalf("Failed to create worktree directory %s: %v", wt.path, err)
		}

		// Create a dummy docker-compose.yml (same content in all worktrees)
		dockerCompose := `version: '3.8'
services:
  api:
    image: nginx:latest
    ports:
      - "8080:8080"
  web:
    image: nginx:latest
    ports:
      - "3000:3000"
`
		composePath := filepath.Join(wt.path, "docker-compose.yml")
		if err := os.WriteFile(composePath, []byte(dockerCompose), 0644); err != nil {
			t.Fatalf("Failed to write docker-compose.yml to %s: %v", wt.path, err)
		}
	}

	// Generate DNS domains for each worktree
	serviceName := "api"
	domains := make(map[string]string)

	for _, wt := range worktrees {
		domain := generateDNSDomain(serviceName, wt.path)
		domains[wt.name] = domain

		t.Logf("Worktree %s (%s) -> DNS: %s", wt.name, wt.path, domain)
	}

	// Verify all domains are unique
	seen := make(map[string]string)
	for name, domain := range domains {
		if existingName, exists := seen[domain]; exists {
			t.Errorf("DNS collision detected! Worktrees '%s' and '%s' have same domain: %s", name, existingName, domain)
		}
		seen[domain] = name
	}

	// Verify domain format
	for name, domain := range domains {
		if len(domain) < len(serviceName)+8 { // service-XXXXXX.space.local
			t.Errorf("Domain for worktree %s is too short: %s", name, domain)
		}
	}
}

// TestGenerateDNSUrls_WithHash tests DNS URL generation with collision prevention
func TestGenerateDNSUrls_WithHash(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()
	serviceName := "api"

	// Mock config with DNSHashing enabled
	cfg := &config.Config{
		Services: map[string]config.ServiceConfig{
			serviceName: {Port: 8080},
		},
		Network: config.NetworkConfig{
			DNSHashing: true, // Enable DNS hashing
		},
	}

	// Mock publishers (from docker-compose ps output)
	publishers := []struct {
		URL           string `json:"URL"`
		TargetPort    int    `json:"TargetPort"`
		PublishedPort int    `json:"PublishedPort"`
		Protocol      string `json:"Protocol"`
	}{
		{TargetPort: 8080, PublishedPort: 0, Protocol: "tcp"},
	}

	// Save current Workdir and restore after test
	oldWorkdir := Workdir
	defer func() { Workdir = oldWorkdir }()

	// Set Workdir to temp directory
	Workdir = tmpDir

	// Generate URLs
	urls := generateDNSUrls(serviceName, cfg, publishers)

	if len(urls) == 0 {
		t.Fatal("Expected at least one DNS URL")
	}

	url := urls[0]

	// Verify URL contains hash
	expectedHash := generateDirectoryHash(tmpDir)
	expectedDomain := serviceName + "-" + expectedHash + ".space.local"

	if !contains(url, expectedDomain) {
		t.Errorf("URL %s doesn't contain expected domain %s", url, expectedDomain)
	}

	// Verify URL format
	expectedPrefix := "http://" + expectedDomain + ":"
	if !contains(url, expectedPrefix) {
		t.Errorf("URL %s doesn't start with expected prefix %s", url, expectedPrefix)
	}

	t.Logf("Generated DNS URL: %s", url)
}

// TestGenerateDNSUrlsLegacy_NoHash tests backward compatibility mode
func TestGenerateDNSUrlsLegacy_NoHash(t *testing.T) {
	serviceName := "api"

	cfg := &config.Config{
		Services: map[string]config.ServiceConfig{
			serviceName: {Port: 8080},
		},
	}

	publishers := []struct {
		URL           string `json:"URL"`
		TargetPort    int    `json:"TargetPort"`
		PublishedPort int    `json:"PublishedPort"`
		Protocol      string `json:"Protocol"`
	}{
		{TargetPort: 8080, PublishedPort: 0, Protocol: "tcp"},
	}

	// Generate URLs using legacy function
	urls := generateDNSUrlsLegacy(serviceName, cfg, publishers)

	if len(urls) == 0 {
		t.Fatal("Expected at least one DNS URL")
	}

	url := urls[0]

	// Verify URL uses simple domain without hash
	expectedDomain := serviceName + ".space.local"
	if !contains(url, expectedDomain) {
		t.Errorf("Legacy URL %s doesn't contain expected domain %s", url, expectedDomain)
	}

	// Verify no hash in URL
	if len(url) > len("http://"+serviceName+".space.local:8080")+1 {
		t.Errorf("Legacy URL %s appears to contain extra characters (possibly a hash)", url)
	}

	t.Logf("Generated legacy DNS URL: %s", url)
}

// TestDNSUrlGeneration_MultipleServices tests URL generation for multiple services
func TestDNSUrlGeneration_MultipleServices(t *testing.T) {
	tmpDir := t.TempDir()

	services := []string{"api", "web", "worker"}
	cfg := &config.Config{
		Services: map[string]config.ServiceConfig{
			"api":    {Port: 8080},
			"web":    {Port: 3000},
			"worker": {Port: 9000},
		},
		Network: config.NetworkConfig{
			DNSHashing: true, // Enable DNS hashing
		},
	}

	// Save and restore Workdir
	oldWorkdir := Workdir
	defer func() { Workdir = oldWorkdir }()
	Workdir = tmpDir

	// Generate URLs for each service
	expectedHash := generateDirectoryHash(tmpDir)
	seenDomains := make(map[string]bool)

	for _, serviceName := range services {
		publishers := []struct {
			URL           string `json:"URL"`
			TargetPort    int    `json:"TargetPort"`
			PublishedPort int    `json:"PublishedPort"`
			Protocol      string `json:"Protocol"`
		}{
			{TargetPort: cfg.Services[serviceName].Port, PublishedPort: 0, Protocol: "tcp"},
		}

		urls := generateDNSUrls(serviceName, cfg, publishers)

		if len(urls) == 0 {
			t.Errorf("No URLs generated for service %s", serviceName)
			continue
		}

		url := urls[0]

		// All services should have same hash (same directory)
		expectedDomain := serviceName + "-" + expectedHash + ".space.local"
		if !contains(url, expectedDomain) {
			t.Errorf("URL for %s doesn't contain expected domain %s: %s", serviceName, expectedDomain, url)
		}

		// Domains should be unique per service
		if seenDomains[expectedDomain] {
			t.Errorf("Duplicate domain %s", expectedDomain)
		}
		seenDomains[expectedDomain] = true

		t.Logf("Service %s -> URL: %s", serviceName, url)
	}

	// Verify all services have unique domains
	if len(seenDomains) != len(services) {
		t.Errorf("Expected %d unique domains, got %d", len(services), len(seenDomains))
	}
}

// TestDNSCollisionPrevention_RealWorldScenario simulates a realistic git worktree scenario
func TestDNSCollisionPrevention_RealWorldScenario(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Simulate a real project structure with multiple worktrees
	tmpDir := t.TempDir()
	projectRoot := filepath.Join(tmpDir, "myproject")

	worktrees := []struct {
		branch string
		path   string
	}{
		{branch: "main", path: filepath.Join(projectRoot, "main")},
		{branch: "develop", path: filepath.Join(projectRoot, "develop")},
		{branch: "feature/user-auth", path: filepath.Join(projectRoot, "feature-user-auth")},
		{branch: "hotfix/security-patch", path: filepath.Join(projectRoot, "hotfix-security-patch")},
	}

	// Create worktree directories
	for _, wt := range worktrees {
		if err := os.MkdirAll(wt.path, 0755); err != nil {
			t.Fatalf("Failed to create worktree %s: %v", wt.branch, err)
		}
	}

	// Generate DNS domains for the same service in different worktrees
	serviceName := "api-server"
	domainsByBranch := make(map[string]string)

	for _, wt := range worktrees {
		domain := generateDNSDomain(serviceName, wt.path)
		domainsByBranch[wt.branch] = domain

		t.Logf("Branch: %-25s Path: %-50s Domain: %s", wt.branch, wt.path, domain)
	}

	// Verify all DNS domains are unique (no collisions)
	seen := make(map[string]string)
	for branch, domain := range domainsByBranch {
		if existingBranch, exists := seen[domain]; exists {
			t.Errorf("DNS COLLISION! Branches '%s' and '%s' have the same domain: %s", branch, existingBranch, domain)
		}
		seen[domain] = branch
	}

	// Verify we can derive the worktree from the domain by extracting the hash
	for branch, domain := range domainsByBranch {
		// Extract hash from domain (format: servicename-HASH.space.local)
		parts := splitDomain(domain)
		if len(parts) < 1 {
			t.Errorf("Invalid domain format for branch %s: %s", branch, domain)
			continue
		}

		// Get hash from first part (service-hash)
		nameHash := parts[0]
		hashStart := len(serviceName) + 1 // +1 for hyphen
		if len(nameHash) <= hashStart {
			t.Errorf("Domain %s for branch %s is too short", domain, branch)
			continue
		}

		extractedHash := nameHash[hashStart:]

		// Verify hash matches the path
		expectedHash := generateDirectoryHash(worktreePathByBranch(worktrees, branch))
		if extractedHash != expectedHash {
			t.Errorf("Extracted hash %s doesn't match expected hash %s for branch %s", extractedHash, expectedHash, branch)
		}
	}

	t.Logf("✓ All %d worktrees have unique DNS domains", len(worktrees))
}

// Helper functions

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func splitDomain(domain string) []string {
	var parts []string
	current := ""
	for _, char := range domain {
		if char == '.' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(char)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}

func worktreePathByBranch(worktrees []struct {
	branch string
	path   string
}, branch string) string {
	for _, wt := range worktrees {
		if wt.branch == branch {
			return wt.path
		}
	}
	return ""
}
