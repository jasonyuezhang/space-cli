package dns

import (
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"
	"strings"
)

// GenerateDirectoryHash creates a 6-character hash from a directory path.
// This hash is deterministic and helps prevent DNS collisions when multiple
// projects with the same service names are running from different directories.
//
// Example:
//   /path/to/project -> "a1b2c3"
//   /another/path    -> "d4e5f6"
func GenerateDirectoryHash(dirPath string) string {
	// Clean and normalize the path
	cleanPath := filepath.Clean(dirPath)

	// Convert to absolute path for consistency
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		// Fallback to cleaned path if absolute path fails
		absPath = cleanPath
	}

	// Create SHA256 hash
	hasher := sha256.New()
	hasher.Write([]byte(absPath))
	hashBytes := hasher.Sum(nil)

	// Convert to hex and take first 6 characters
	hexHash := hex.EncodeToString(hashBytes)
	return hexHash[:6]
}

// GenerateHashedDomainName creates a DNS domain name with directory hash.
// Format: {serviceName}-{hash}.space.local
//
// Parameters:
//   - serviceName: The name of the service (e.g., "web", "api", "postgres")
//   - dirPath: The absolute directory path of the project
//   - domain: The base domain (e.g., "space.local", "orb.local")
//
// Example:
//   GenerateHashedDomainName("web", "/home/user/project", "space.local")
//   -> "web-a1b2c3.space.local"
func GenerateHashedDomainName(serviceName, dirPath, domain string) string {
	hash := GenerateDirectoryHash(dirPath)
	return serviceName + "-" + hash + "." + domain
}

// ExtractServiceNameFromHashedDomain extracts the service name from a hashed domain.
// This is useful for reverse lookups in the DNS server.
//
// Example:
//   ExtractServiceNameFromHashedDomain("web-a1b2c3.space.local", "space.local")
//   -> "web"
func ExtractServiceNameFromHashedDomain(domain, baseDomain string) string {
	// Remove trailing dot first if present
	domain = strings.TrimSuffix(domain, ".")

	// Remove the base domain suffix
	domain = strings.TrimSuffix(domain, "."+baseDomain)

	// Find the last dash (before the hash)
	lastDash := strings.LastIndex(domain, "-")
	if lastDash == -1 {
		// No hash present, return as-is
		return domain
	}

	// Check if what follows the dash looks like a 6-char hex hash
	potentialHash := domain[lastDash+1:]
	if len(potentialHash) == 6 && isHexString(potentialHash) {
		// This looks like a hash, return the service name part
		return domain[:lastDash]
	}

	// No hash pattern detected, return full domain
	return domain
}

// ExtractHashFromHashedDomain extracts the hash from a hashed domain.
// This is useful for matching containers by their directory hash.
//
// Example:
//
//	ExtractHashFromHashedDomain("web-a1b2c3.space.local", "space.local")
//	-> "a1b2c3"
func ExtractHashFromHashedDomain(domain, baseDomain string) string {
	// Remove trailing dot first if present
	domain = strings.TrimSuffix(domain, ".")

	// Remove the base domain suffix
	domain = strings.TrimSuffix(domain, "."+baseDomain)

	// Find the last dash (before the hash)
	lastDash := strings.LastIndex(domain, "-")
	if lastDash == -1 {
		// No hash present
		return ""
	}

	// Check if what follows the dash looks like a 6-char hex hash
	potentialHash := domain[lastDash+1:]
	if len(potentialHash) == 6 && isHexString(potentialHash) {
		return potentialHash
	}

	// No hash pattern detected
	return ""
}

// isHexString checks if a string contains only hexadecimal characters
func isHexString(s string) bool {
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

// ValidateHashedDomain validates if a domain has the correct hashed format.
// Returns true if the domain matches the pattern: {service}-{hash}.{baseDomain}
func ValidateHashedDomain(domain, baseDomain string) bool {
	// Remove the base domain suffix
	domain = strings.TrimSuffix(domain, "."+baseDomain)
	domain = strings.TrimSuffix(domain, ".")

	// Find the last dash
	lastDash := strings.LastIndex(domain, "-")
	if lastDash == -1 {
		return false
	}

	// Check if what follows is a 6-char hex hash
	potentialHash := domain[lastDash+1:]
	return len(potentialHash) == 6 && isHexString(potentialHash)
}
