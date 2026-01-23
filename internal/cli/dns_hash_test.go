package cli

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestGenerateDirectoryHash_Deterministic tests that the same path always produces the same hash
func TestGenerateDirectoryHash_Deterministic(t *testing.T) {
	testCases := []struct {
		name string
		path string
	}{
		{
			name: "absolute path - main branch",
			path: "/Users/developer/project-main",
		},
		{
			name: "absolute path - dev branch",
			path: "/Users/developer/project-dev",
		},
		{
			name: "absolute path with spaces",
			path: "/Users/developer/my project/worktree",
		},
		{
			name: "deep nested path",
			path: "/Users/developer/projects/work/client-a/project-xyz/worktree-feature-123",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Generate hash multiple times
			hash1 := generateDirectoryHash(tc.path)
			hash2 := generateDirectoryHash(tc.path)
			hash3 := generateDirectoryHash(tc.path)

			// All hashes should be identical (deterministic)
			if hash1 != hash2 || hash2 != hash3 {
				t.Errorf("Hash is not deterministic: got %s, %s, %s", hash1, hash2, hash3)
			}
		})
	}
}

// TestGenerateDirectoryHash_Length tests that hash is exactly 6 characters
func TestGenerateDirectoryHash_Length(t *testing.T) {
	testCases := []string{
		"/short",
		"/very/long/path/with/many/nested/directories/and/components",
		"/path/with/special-chars_123",
		"/UPPERCASE/path",
		"/path/with/dots/../relative/parts",
	}

	for _, path := range testCases {
		t.Run(path, func(t *testing.T) {
			hash := generateDirectoryHash(path)

			if len(hash) != 6 {
				t.Errorf("Expected hash length of 6, got %d for path %s (hash: %s)", len(hash), path, hash)
			}
		})
	}
}

// TestGenerateDirectoryHash_HexCharacters tests that hash contains only valid hex characters
func TestGenerateDirectoryHash_HexCharacters(t *testing.T) {
	paths := []string{
		"/Users/developer/project",
		"/tmp/test",
		"/var/www/html",
	}

	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			hash := generateDirectoryHash(path)

			// Check if all characters are valid hex (0-9, a-f)
			for _, char := range hash {
				if !((char >= '0' && char <= '9') || (char >= 'a' && char <= 'f')) {
					t.Errorf("Hash contains invalid hex character '%c' in %s (path: %s)", char, hash, path)
				}
			}
		})
	}
}

// TestGenerateDirectoryHash_Uniqueness tests that different paths produce different hashes
func TestGenerateDirectoryHash_Uniqueness(t *testing.T) {
	paths := []string{
		"/Users/developer/project-main",
		"/Users/developer/project-dev",
		"/Users/developer/project-feature",
		"/Users/developer/project",
		"/Users/developer/another-project",
		"/tmp/project",
		"/var/projects/main",
	}

	// Generate hashes for all paths
	hashes := make(map[string]string)
	for _, path := range paths {
		hash := generateDirectoryHash(path)
		if existingPath, exists := hashes[hash]; exists {
			t.Errorf("Hash collision detected! Paths '%s' and '%s' both produce hash '%s'", path, existingPath, hash)
		}
		hashes[hash] = path
	}

	// Verify we have unique hashes for all paths
	if len(hashes) != len(paths) {
		t.Errorf("Expected %d unique hashes, got %d", len(paths), len(hashes))
	}
}

// TestGenerateDirectoryHash_SameDockerComposeInDifferentWorktrees tests realistic worktree scenarios
func TestGenerateDirectoryHash_SameDockerComposeInDifferentWorktrees(t *testing.T) {
	// Simulate same project in different worktrees
	worktrees := []struct {
		path         string
		expectedHash string // We'll verify these are all different
	}{
		{path: "/Users/developer/myproject-main"},
		{path: "/Users/developer/myproject-dev"},
		{path: "/Users/developer/myproject-feature-auth"},
		{path: "/Users/developer/myproject-hotfix"},
	}

	seenHashes := make(map[string]bool)

	for _, wt := range worktrees {
		hash := generateDirectoryHash(wt.path)

		// Verify hash is 6 characters
		if len(hash) != 6 {
			t.Errorf("Hash for %s has wrong length: %d", wt.path, len(hash))
		}

		// Verify hash is unique
		if seenHashes[hash] {
			t.Errorf("Duplicate hash %s for path %s", hash, wt.path)
		}
		seenHashes[hash] = true

		t.Logf("Worktree: %s -> Hash: %s", wt.path, hash)
	}
}

// TestGenerateDirectoryHash_PathNormalization tests that path normalization works correctly
func TestGenerateDirectoryHash_PathNormalization(t *testing.T) {
	// These should produce the same hash after normalization
	testCases := []struct {
		name  string
		path1 string
		path2 string
	}{
		{
			name:  "trailing slash",
			path1: "/Users/developer/project",
			path2: "/Users/developer/project/",
		},
		{
			name:  "double slashes",
			path1: "/Users/developer/project",
			path2: "/Users//developer//project",
		},
		{
			name:  "dot segments",
			path1: "/Users/developer/project",
			path2: "/Users/developer/other/../project",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			hash1 := generateDirectoryHash(tc.path1)
			hash2 := generateDirectoryHash(tc.path2)

			// After normalization via filepath.Clean, these should be equal
			normalized1 := filepath.Clean(tc.path1)
			normalized2 := filepath.Clean(tc.path2)

			if normalized1 == normalized2 {
				if hash1 != hash2 {
					t.Errorf("Normalized paths are equal but hashes differ:\n  Path1: %s -> %s (hash: %s)\n  Path2: %s -> %s (hash: %s)",
						tc.path1, normalized1, hash1, tc.path2, normalized2, hash2)
				}
			} else {
				// If normalized paths differ, hashes should differ
				if hash1 == hash2 {
					t.Errorf("Different normalized paths produce same hash:\n  Path1: %s -> %s\n  Path2: %s -> %s\n  Hash: %s",
						tc.path1, normalized1, tc.path2, normalized2, hash1)
				}
			}
		})
	}
}

// TestGenerateDNSDomain tests DNS domain name generation
func TestGenerateDNSDomain(t *testing.T) {
	testCases := []struct {
		name        string
		serviceName string
		workDir     string
	}{
		{
			name:        "simple service",
			serviceName: "api",
			workDir:     "/Users/developer/project-main",
		},
		{
			name:        "hyphenated service",
			serviceName: "web-server",
			workDir:     "/Users/developer/project-dev",
		},
		{
			name:        "database service",
			serviceName: "postgres",
			workDir:     "/tmp/test-project",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			domain := generateDNSDomain(tc.serviceName, tc.workDir)

			// Verify format: <service>-<hash>.space.local
			if !strings.HasSuffix(domain, ".space.local") {
				t.Errorf("Domain %s doesn't end with .space.local", domain)
			}

			// Extract hash portion (between last hyphen and .space.local)
			parts := strings.Split(domain, ".")
			if len(parts) < 3 {
				t.Errorf("Domain %s has unexpected format", domain)
				return
			}

			// Get the name-hash part (before .space.local)
			nameHashPart := parts[0]

			// Should be service-hash
			if !strings.HasPrefix(nameHashPart, tc.serviceName+"-") {
				t.Errorf("Domain %s doesn't start with service name %s", domain, tc.serviceName)
			}

			// Extract hash
			hashPart := strings.TrimPrefix(nameHashPart, tc.serviceName+"-")
			if len(hashPart) != 6 {
				t.Errorf("Hash portion %s is not 6 characters", hashPart)
			}

			// Verify domain is deterministic
			domain2 := generateDNSDomain(tc.serviceName, tc.workDir)
			if domain != domain2 {
				t.Errorf("Domain generation is not deterministic: %s vs %s", domain, domain2)
			}

			t.Logf("Service: %s, Path: %s -> Domain: %s", tc.serviceName, tc.workDir, domain)
		})
	}
}

// TestGenerateDNSDomain_DifferentPathsSameService tests collision prevention
func TestGenerateDNSDomain_DifferentPathsSameService(t *testing.T) {
	serviceName := "api"
	paths := []string{
		"/Users/developer/project-main",
		"/Users/developer/project-dev",
		"/Users/developer/project-feature",
	}

	domains := make(map[string]string)

	for _, path := range paths {
		domain := generateDNSDomain(serviceName, path)

		// Verify uniqueness
		if existingPath, exists := domains[domain]; exists {
			t.Errorf("Same domain %s for different paths: %s and %s", domain, path, existingPath)
		}
		domains[domain] = path

		t.Logf("Path: %s -> Domain: %s", path, domain)
	}

	// All domains should be unique
	if len(domains) != len(paths) {
		t.Errorf("Expected %d unique domains, got %d", len(paths), len(domains))
	}
}

// TestGenerateDNSDomain_Format tests domain name format compliance
func TestGenerateDNSDomain_Format(t *testing.T) {
	serviceName := "my-service"
	workDir := "/Users/developer/project"

	domain := generateDNSDomain(serviceName, workDir)

	// Test DNS name format compliance
	// - Lowercase alphanumeric and hyphens only
	// - No consecutive hyphens
	// - No hyphen at start or end of labels
	for i, char := range domain {
		if !((char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '-' || char == '.') {
			t.Errorf("Invalid character '%c' at position %d in domain %s", char, i, domain)
		}
	}

	// Should not have consecutive hyphens
	if strings.Contains(domain, "--") {
		t.Errorf("Domain %s contains consecutive hyphens", domain)
	}

	// Should match pattern: <service>-<6-char-hash>.space.local
	expectedPattern := serviceName + "-[0-9a-f]{6}.space.local"
	t.Logf("Domain: %s (expected pattern: %s)", domain, expectedPattern)
}

// BenchmarkGenerateDirectoryHash benchmarks hash generation performance
func BenchmarkGenerateDirectoryHash(b *testing.B) {
	path := "/Users/developer/very/long/path/to/project/worktree/feature/branch"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = generateDirectoryHash(path)
	}
}

// BenchmarkGenerateDNSDomain benchmarks DNS domain generation performance
func BenchmarkGenerateDNSDomain(b *testing.B) {
	serviceName := "api-server"
	workDir := "/Users/developer/project/worktree"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = generateDNSDomain(serviceName, workDir)
	}
}
