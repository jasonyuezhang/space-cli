package dns

import (
	"path/filepath"
	"testing"
)

func TestGenerateDirectoryHash(t *testing.T) {
	tests := []struct {
		name     string
		dirPath  string
		wantLen  int
		wantHex  bool
	}{
		{
			name:    "simple path",
			dirPath: "/home/user/project",
			wantLen: 6,
			wantHex: true,
		},
		{
			name:    "path with spaces",
			dirPath: "/home/user/my project",
			wantLen: 6,
			wantHex: true,
		},
		{
			name:    "relative path",
			dirPath: "./project",
			wantLen: 6,
			wantHex: true,
		},
		{
			name:    "path with special chars",
			dirPath: "/home/user/my-project_v2",
			wantLen: 6,
			wantHex: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash := GenerateDirectoryHash(tt.dirPath)

			// Check length
			if len(hash) != tt.wantLen {
				t.Errorf("GenerateDirectoryHash() length = %d, want %d", len(hash), tt.wantLen)
			}

			// Check if hexadecimal
			if tt.wantHex && !isHexString(hash) {
				t.Errorf("GenerateDirectoryHash() = %s, want hexadecimal string", hash)
			}
		})
	}
}

func TestGenerateDirectoryHash_Deterministic(t *testing.T) {
	// Same path should always produce the same hash
	path := "/home/user/project"
	hash1 := GenerateDirectoryHash(path)
	hash2 := GenerateDirectoryHash(path)

	if hash1 != hash2 {
		t.Errorf("GenerateDirectoryHash() is not deterministic: %s != %s", hash1, hash2)
	}
}

func TestGenerateDirectoryHash_Collision(t *testing.T) {
	// Different paths should produce different hashes
	paths := []string{
		"/home/user/project1",
		"/home/user/project2",
		"/home/user/other-project",
		"/var/www/project",
	}

	hashes := make(map[string]string)
	for _, path := range paths {
		hash := GenerateDirectoryHash(path)
		if existingPath, exists := hashes[hash]; exists {
			t.Errorf("Hash collision detected: %s and %s both produce hash %s", path, existingPath, hash)
		}
		hashes[hash] = path
	}
}

func TestGenerateHashedDomainName(t *testing.T) {
	tests := []struct {
		name        string
		serviceName string
		dirPath     string
		domain      string
		wantPattern string // Pattern to check (e.g., "web-*.space.local")
	}{
		{
			name:        "simple service",
			serviceName: "web",
			dirPath:     "/home/user/project",
			domain:      "space.local",
			wantPattern: "web-*.space.local",
		},
		{
			name:        "api service",
			serviceName: "api",
			dirPath:     "/var/www/myapp",
			domain:      "space.local",
			wantPattern: "api-*.space.local",
		},
		{
			name:        "postgres service",
			serviceName: "postgres",
			dirPath:     "/opt/services/db",
			domain:      "orb.local",
			wantPattern: "postgres-*.orb.local",
		},
		{
			name:        "service with dash",
			serviceName: "web-frontend",
			dirPath:     "/home/user/project",
			domain:      "space.local",
			wantPattern: "web-frontend-*.space.local",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateHashedDomainName(tt.serviceName, tt.dirPath, tt.domain)

			// Check that domain ends with the base domain
			expectedSuffix := "." + tt.domain
			if !hasSuffix(got, expectedSuffix) {
				t.Errorf("GenerateHashedDomainName() = %s, want suffix %s", got, expectedSuffix)
			}

			// Check that domain starts with service name
			if !hasPrefix(got, tt.serviceName+"-") {
				t.Errorf("GenerateHashedDomainName() = %s, want prefix %s-", got, tt.serviceName)
			}

			// Check format: service-hash.domain
			// Extract the hash part
			withoutDomain := got[:len(got)-len(expectedSuffix)]
			parts := splitLast(withoutDomain, "-")
			if len(parts) != 2 {
				t.Errorf("GenerateHashedDomainName() = %s, want format service-hash.domain", got)
				return
			}

			hash := parts[1]
			if len(hash) != 6 || !isHexString(hash) {
				t.Errorf("GenerateHashedDomainName() hash = %s, want 6-char hex", hash)
			}
		})
	}
}

func TestExtractServiceNameFromHashedDomain(t *testing.T) {
	tests := []struct {
		name       string
		domain     string
		baseDomain string
		want       string
	}{
		{
			name:       "simple hashed domain",
			domain:     "web-a1b2c3.space.local",
			baseDomain: "space.local",
			want:       "web",
		},
		{
			name:       "service with dash",
			domain:     "web-frontend-a1b2c3.space.local",
			baseDomain: "space.local",
			want:       "web-frontend",
		},
		{
			name:       "non-hashed domain",
			domain:     "web.space.local",
			baseDomain: "space.local",
			want:       "web",
		},
		{
			name:       "domain with trailing dot",
			domain:     "api-abc123.space.local.",
			baseDomain: "space.local",
			want:       "api",
		},
		{
			name:       "service with multiple dashes",
			domain:     "my-web-service-abc123.space.local",
			baseDomain: "space.local",
			want:       "my-web-service",
		},
		{
			name:       "invalid hash (not hex)",
			domain:     "web-notahx.space.local",
			baseDomain: "space.local",
			want:       "web-notahx", // Returns full name if hash pattern not detected
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractServiceNameFromHashedDomain(tt.domain, tt.baseDomain)
			if got != tt.want {
				t.Errorf("ExtractServiceNameFromHashedDomain() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestValidateHashedDomain(t *testing.T) {
	tests := []struct {
		name       string
		domain     string
		baseDomain string
		want       bool
	}{
		{
			name:       "valid hashed domain",
			domain:     "web-a1b2c3.space.local",
			baseDomain: "space.local",
			want:       true,
		},
		{
			name:       "valid with uppercase hex",
			domain:     "api-ABC123.space.local",
			baseDomain: "space.local",
			want:       true,
		},
		{
			name:       "invalid - no hash",
			domain:     "web.space.local",
			baseDomain: "space.local",
			want:       false,
		},
		{
			name:       "invalid - hash too short",
			domain:     "web-abc.space.local",
			baseDomain: "space.local",
			want:       false,
		},
		{
			name:       "invalid - hash too long",
			domain:     "web-abc1234.space.local",
			baseDomain: "space.local",
			want:       false,
		},
		{
			name:       "invalid - non-hex hash",
			domain:     "web-notahx.space.local",
			baseDomain: "space.local",
			want:       false,
		},
		{
			name:       "valid - service with dashes",
			domain:     "my-service-abc123.space.local",
			baseDomain: "space.local",
			want:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidateHashedDomain(tt.domain, tt.baseDomain)
			if got != tt.want {
				t.Errorf("ValidateHashedDomain() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsHexString(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{
			name:  "valid lowercase hex",
			input: "abc123",
			want:  true,
		},
		{
			name:  "valid uppercase hex",
			input: "ABC123",
			want:  true,
		},
		{
			name:  "valid mixed case hex",
			input: "AbC123",
			want:  true,
		},
		{
			name:  "invalid - contains non-hex",
			input: "abc12g",
			want:  false,
		},
		{
			name:  "invalid - special chars",
			input: "abc-12",
			want:  false,
		},
		{
			name:  "empty string",
			input: "",
			want:  true, // Empty string is technically valid hex (no invalid chars)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isHexString(tt.input)
			if got != tt.want {
				t.Errorf("isHexString() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Helper functions for tests

func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

func hasSuffix(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}

func splitLast(s, sep string) []string {
	idx := -1
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == sep[0] && (len(sep) == 1 || s[i:i+len(sep)] == sep) {
			idx = i
			break
		}
	}
	if idx == -1 {
		return []string{s}
	}
	return []string{s[:idx], s[idx+len(sep):]}
}

// TestHashedDomainRoundTrip tests that we can generate and extract service names correctly
func TestHashedDomainRoundTrip(t *testing.T) {
	services := []string{"web", "api", "postgres", "redis", "web-frontend"}
	paths := []string{
		"/home/user/project1",
		"/var/www/app",
		"/opt/services/backend",
	}
	domains := []string{"space.local", "orb.local"}

	for _, service := range services {
		for _, path := range paths {
			for _, domain := range domains {
				t.Run(service+"-"+filepath.Base(path)+"-"+domain, func(t *testing.T) {
					// Generate hashed domain
					hashed := GenerateHashedDomainName(service, path, domain)

					// Extract service name
					extracted := ExtractServiceNameFromHashedDomain(hashed, domain)

					// Should match original service name
					if extracted != service {
						t.Errorf("Round trip failed: got %s, want %s (hashed: %s)", extracted, service, hashed)
					}

					// Should be valid
					if !ValidateHashedDomain(hashed, domain) {
						t.Errorf("Generated domain %s is not valid", hashed)
					}
				})
			}
		}
	}
}
