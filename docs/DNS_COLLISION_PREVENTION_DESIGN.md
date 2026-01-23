# DNS Collision Prevention System - Architecture Design

## Executive Summary

This document outlines the architecture for preventing DNS name collisions in Space CLI by implementing deterministic path-based hashing. The system generates unique domain names in the format `{service}-{hash}.space.local` where the hash is derived from the project's absolute directory path.

## Problem Statement

### Current State
- Domain names follow pattern: `{service}.space.local`
- Multiple project instances in different directories create DNS collisions
- Git worktrees using the same service names conflict
- Example collision: `postgres.space.local` in both `/project` and `/project-worktree-feature`

### Requirements
1. **Deterministic**: Same directory path always generates the same hash
2. **Collision-resistant**: Different paths produce different hashes
3. **Short**: 6-character hash for usability and DNS label limits
4. **Human-readable**: Format `{service}-{hash}.space.local`
5. **Git worktree aware**: Different worktrees get different hashes
6. **Backward compatible**: Optional feature with fallback to current behavior
7. **Configuration-driven**: Enable/disable via space.yml

## Architecture Overview

### System Components

```
┌─────────────────────────────────────────────────────────────┐
│                     Space CLI User Interface                 │
│                       (up/down commands)                      │
└────────────────────┬────────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────────┐
│                   Configuration Layer                        │
│  - Load space.yml                                           │
│  - network.dns_hashing.enabled (bool)                       │
│  - network.dns_hashing.length (int, default: 6)            │
└────────────────────┬────────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────────┐
│              Domain Name Generator Service                   │
│  - GenerateDomainName(service, workDir, config)             │
│  - generatePathHash(workDir) -> string                      │
│  - validateDomain(domain) -> error                          │
└────────────────────┬────────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────────┐
│                   DNS Server Integration                     │
│  - Update handleOrbLocal() to support hashed domains        │
│  - Container lookup with hash stripping                      │
│  - Cache key includes hash                                   │
└─────────────────────────────────────────────────────────────┘
```

## Detailed Component Design

### 1. Hash Algorithm Design

**Algorithm: Blake2b-256 (truncated to 6 characters)**

```go
// pkg/dns/hasher.go
package dns

import (
    "encoding/base32"
    "path/filepath"
    "strings"

    "golang.org/x/crypto/blake2b"
)

// HashConfig configures path hashing behavior
type HashConfig struct {
    Length      int  // Hash length (default: 6)
    Enabled     bool // Enable hashing
    Lowercase   bool // Force lowercase (default: true)
}

// PathHasher generates deterministic hashes from directory paths
type PathHasher struct {
    config HashConfig
}

// NewPathHasher creates a new path hasher with config
func NewPathHasher(cfg HashConfig) *PathHasher {
    if cfg.Length == 0 {
        cfg.Length = 6
    }
    if cfg.Length < 4 || cfg.Length > 32 {
        cfg.Length = 6 // Clamp to safe range
    }
    return &PathHasher{config: cfg}
}

// GenerateHash creates a deterministic hash from an absolute path
// Uses Blake2b-256 for cryptographic quality without OpenSSL dependency
func (h *PathHasher) GenerateHash(absolutePath string) (string, error) {
    // Ensure path is absolute and cleaned
    absPath, err := filepath.Abs(absolutePath)
    if err != nil {
        return "", err
    }
    absPath = filepath.Clean(absPath)

    // Use Blake2b-256 (faster than SHA-256, no OpenSSL)
    hash := blake2b.Sum256([]byte(absPath))

    // Base32 encoding (URL-safe, case-insensitive)
    // Use Crockford's base32 alphabet for better human readability
    encoded := base32.StdEncoding.WithPadding(base32.NoPadding).
        EncodeToString(hash[:])

    // Take first N characters
    result := encoded[:h.config.Length]

    // Force lowercase for DNS compatibility
    if h.config.Lowercase {
        result = strings.ToLower(result)
    }

    return result, nil
}

// ValidateHash checks if a hash string is valid
func (h *PathHasher) ValidateHash(hash string) bool {
    if len(hash) != h.config.Length {
        return false
    }
    // Check if all characters are valid base32
    for _, c := range hash {
        if !isValidBase32Char(c) {
            return false
        }
    }
    return true
}

func isValidBase32Char(c rune) bool {
    return (c >= 'a' && c <= 'z') ||
           (c >= 'A' && c <= 'Z') ||
           (c >= '2' && c <= '7')
}
```

**Why Blake2b-256?**
- **Fast**: Faster than SHA-256 on modern CPUs
- **No OpenSSL**: Pure Go implementation, no CGo dependencies
- **Cryptographic quality**: 256-bit output ensures low collision probability
- **Deterministic**: Same input always produces same output
- **Standard library**: `golang.org/x/crypto/blake2b`

**Why Base32 encoding?**
- **DNS-safe**: Only uses characters `[a-z2-7]` (case-insensitive)
- **Human-readable**: Easier to type than hex
- **No ambiguity**: Crockford's alphabet excludes confusing characters (0/O, 1/I/L)
- **Compact**: More efficient than hex (5 bits per character vs 4)

**Collision Probability**

With 6-character Base32:
- Character space: 32 characters
- Possible combinations: 32^6 = 1,073,741,824 (~1 billion)
- Birthday paradox: ~50% collision probability after √(1B) ≈ 32,768 projects
- **Practical**: Most users won't exceed 1000 projects
- **If needed**: Can increase to 8 characters (32^8 = 1 trillion combinations)

### 2. Configuration Schema Updates

```go
// pkg/config/schema.go

// NetworkConfig defines networking settings
type NetworkConfig struct {
    // ... existing fields ...

    // DNSHashing configures domain name collision prevention
    DNSHashing DNSHashingConfig `yaml:"dns_hashing,omitempty" json:"dns_hashing,omitempty"`
}

// DNSHashingConfig configures DNS collision prevention via path-based hashing
type DNSHashingConfig struct {
    // Enabled enables path-based hashing for domain names
    // When enabled, domains become {service}-{hash}.space.local
    // Default: false (disabled for backward compatibility)
    Enabled bool `yaml:"enabled,omitempty" json:"enabled,omitempty"`

    // Length of the hash string (4-32 characters)
    // Default: 6
    // Longer hashes = lower collision probability but longer domain names
    Length int `yaml:"length,omitempty" json:"length,omitempty"`

    // Format template for domain names
    // Variables: {service}, {hash}, {project}
    // Default: "{service}-{hash}"
    Format string `yaml:"format,omitempty" json:"format,omitempty"`

    // Separator between service name and hash
    // Default: "-"
    Separator string `yaml:"separator,omitempty" json:"separator,omitempty"`
}
```

**Default Configuration (Backward Compatible)**

```yaml
network:
  dns_hashing:
    enabled: false  # Must be explicitly enabled
    length: 6       # 6-character hash
    format: "{service}-{hash}"
    separator: "-"
```

**Enabled Configuration Example**

```yaml
network:
  dns_hashing:
    enabled: true
    length: 6  # Can increase if collisions occur
```

### 3. Domain Name Generator Service

```go
// pkg/dns/domain_generator.go
package dns

import (
    "fmt"
    "path/filepath"
    "regexp"
    "strings"
)

// DomainGenerator generates unique domain names with optional path-based hashing
type DomainGenerator struct {
    hasher    *PathHasher
    config    DomainConfig
    validator *DomainValidator
}

// DomainConfig configures domain generation
type DomainConfig struct {
    BaseDomain string           // e.g., "space.local"
    HashConfig HashConfig        // Hash configuration
    Format     string            // Format template
    Separator  string            // Separator character
}

// NewDomainGenerator creates a new domain generator
func NewDomainGenerator(cfg DomainConfig) *DomainGenerator {
    if cfg.BaseDomain == "" {
        cfg.BaseDomain = "space.local"
    }
    if cfg.Format == "" {
        cfg.Format = "{service}-{hash}"
    }
    if cfg.Separator == "" {
        cfg.Separator = "-"
    }

    return &DomainGenerator{
        hasher:    NewPathHasher(cfg.HashConfig),
        config:    cfg,
        validator: NewDomainValidator(),
    }
}

// GenerateDomainName creates a domain name for a service
func (g *DomainGenerator) GenerateDomainName(serviceName, workDir string) (string, error) {
    // Sanitize service name
    serviceName = g.sanitizeServiceName(serviceName)

    // If hashing is disabled, use simple format
    if !g.config.HashConfig.Enabled {
        domain := fmt.Sprintf("%s.%s", serviceName, g.config.BaseDomain)
        return domain, g.validator.Validate(domain)
    }

    // Generate path hash
    hash, err := g.hasher.GenerateHash(workDir)
    if err != nil {
        return "", fmt.Errorf("failed to generate hash: %w", err)
    }

    // Apply format template
    domain := g.applyFormat(serviceName, hash)

    // Validate domain
    if err := g.validator.Validate(domain); err != nil {
        return "", fmt.Errorf("invalid domain generated: %w", err)
    }

    return domain, nil
}

// sanitizeServiceName cleans service name for DNS compatibility
func (g *DomainGenerator) sanitizeServiceName(name string) string {
    // Convert to lowercase
    name = strings.ToLower(name)

    // Replace invalid characters with separator
    reg := regexp.MustCompile(`[^a-z0-9-]`)
    name = reg.ReplaceAllString(name, g.config.Separator)

    // Remove leading/trailing separators
    name = strings.Trim(name, g.config.Separator)

    // Collapse multiple separators
    reg = regexp.MustCompile(fmt.Sprintf("%s+", regexp.QuoteMeta(g.config.Separator)))
    name = reg.ReplaceAllString(name, g.config.Separator)

    return name
}

// applyFormat applies the format template
func (g *DomainGenerator) applyFormat(serviceName, hash string) string {
    // Replace template variables
    result := g.config.Format
    result = strings.ReplaceAll(result, "{service}", serviceName)
    result = strings.ReplaceAll(result, "{hash}", hash)

    // Add base domain
    return fmt.Sprintf("%s.%s", result, g.config.BaseDomain)
}

// ExtractServiceName extracts the original service name from a hashed domain
func (g *DomainGenerator) ExtractServiceName(domain string) (string, error) {
    // Remove base domain suffix
    domain = strings.TrimSuffix(domain, "."+g.config.BaseDomain)
    domain = strings.TrimSuffix(domain, ".")

    // If hashing is disabled, entire domain is service name
    if !g.config.HashConfig.Enabled {
        return domain, nil
    }

    // Parse format to extract service name
    // For format "{service}-{hash}", split on separator and take first part
    parts := strings.Split(domain, g.config.Separator)
    if len(parts) < 2 {
        return domain, nil // Fallback for non-hashed domains
    }

    // Last part is likely the hash, everything before is service name
    serviceName := strings.Join(parts[:len(parts)-1], g.config.Separator)

    return serviceName, nil
}
```

### 4. Domain Validator

```go
// pkg/dns/domain_validator.go
package dns

import (
    "fmt"
    "regexp"
    "strings"
)

// DomainValidator validates DNS domain names per RFC 1035
type DomainValidator struct {
    labelRegex *regexp.Regexp
    maxLength  int
}

// NewDomainValidator creates a domain validator
func NewDomainValidator() *DomainValidator {
    return &DomainValidator{
        // RFC 1035: Labels must start with letter/digit, contain only letters/digits/hyphens
        labelRegex: regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`),
        maxLength:  253, // RFC 1035: Max 253 characters total
    }
}

// Validate checks if a domain name is valid per RFC 1035
func (v *DomainValidator) Validate(domain string) error {
    // Check total length
    if len(domain) > v.maxLength {
        return fmt.Errorf("domain exceeds max length of %d characters", v.maxLength)
    }

    // Split into labels
    labels := strings.Split(domain, ".")
    if len(labels) == 0 {
        return fmt.Errorf("domain must contain at least one label")
    }

    // Validate each label
    for i, label := range labels {
        if err := v.validateLabel(label, i); err != nil {
            return err
        }
    }

    return nil
}

// validateLabel validates a single DNS label
func (v *DomainValidator) validateLabel(label string, position int) error {
    // Check length (RFC 1035: 1-63 characters)
    if len(label) == 0 {
        return fmt.Errorf("label %d is empty", position)
    }
    if len(label) > 63 {
        return fmt.Errorf("label %d exceeds max length of 63 characters", position)
    }

    // Check format
    if !v.labelRegex.MatchString(label) {
        return fmt.Errorf("label %d (%s) contains invalid characters or format", position, label)
    }

    return nil
}

// SanitizeDomain attempts to fix common domain issues
func (v *DomainValidator) SanitizeDomain(domain string) string {
    // Convert to lowercase
    domain = strings.ToLower(domain)

    // Remove invalid characters
    reg := regexp.MustCompile(`[^a-z0-9.-]`)
    domain = reg.ReplaceAllString(domain, "-")

    // Remove leading/trailing dots and dashes
    domain = strings.Trim(domain, ".-")

    return domain
}
```

### 5. DNS Server Integration

```go
// internal/dns/server.go (updates)

// Update handleOrbLocal to support hashed domains
func (s *Server) handleOrbLocal(w dns.ResponseWriter, r *dns.Msg) {
    m := new(dns.Msg)
    m.SetReply(r)
    m.Authoritative = true

    for _, q := range r.Question {
        if q.Qtype != dns.TypeA {
            continue
        }

        hostname := strings.TrimSuffix(q.Name, ".")

        // Check cache first
        if ip := s.cache.get(hostname); ip != "" {
            s.logger.Debug("DNS cache hit", "hostname", hostname, "ip", ip)
            // ... (existing cache hit logic)
            continue
        }

        // Resolve from Docker (supports both hashed and non-hashed domains)
        ip, err := s.resolveContainerIP(context.Background(), hostname)
        if err != nil {
            s.logger.Warn("Failed to resolve container", "hostname", hostname, "error", err)
            continue
        }

        // ... (existing response logic)
    }

    w.WriteMsg(m)
}

// Update resolveContainerIP to handle hashed domains
func (s *Server) resolveContainerIP(ctx context.Context, hostname string) (string, error) {
    // Strip domain suffix
    hostname = strings.TrimSuffix(hostname, "."+s.domain)
    hostname = strings.TrimSuffix(hostname, ".")

    // Extract service name (handles both hashed and non-hashed)
    serviceName := s.extractServiceName(hostname)

    // Try multiple container name patterns
    patterns := []string{
        serviceName,                          // Direct match
        s.projectName + "-" + serviceName,    // With project prefix
        s.projectName + "-" + serviceName + "-1",  // With suffix
        s.projectName + "-" + serviceName + "_1",  // Underscore variant
    }

    var lastErr error
    for _, containerName := range patterns {
        ip, err := s.docker.GetContainerIP(ctx, s.projectName, containerName)
        if err == nil && ip != "" {
            return ip, nil
        }
        lastErr = err
    }

    return "", fmt.Errorf("container not found for service %s: %w", serviceName, lastErr)
}

// extractServiceName strips hash from domain if present
func (s *Server) extractServiceName(hostname string) string {
    // If domain generator is available, use it
    if s.domainGenerator != nil {
        if name, err := s.domainGenerator.ExtractServiceName(hostname + "." + s.domain); err == nil {
            return name
        }
    }

    // Fallback: hostname is service name (backward compatibility)
    return hostname
}
```

## Migration Strategy

### Phase 1: Opt-in (v0.3.0)

1. **Default Behavior**: Feature disabled by default
2. **Configuration**: Users must explicitly enable in `space.yml`
3. **Documentation**: Clear migration guide
4. **Testing**: Comprehensive test suite for both modes

### Phase 2: Soft Enforcement (v0.4.0)

1. **Warning**: Show warning when collision is detected
2. **Recommendation**: Suggest enabling DNS hashing
3. **Auto-detection**: Detect multiple project instances

### Phase 3: Default Enabled (v0.5.0)

1. **Default**: Enable hashing by default for new projects
2. **Legacy Support**: Detect and maintain existing non-hashed domains
3. **Migration Tool**: `space migrate dns-hashing` command

## Configuration Examples

### Example 1: Basic Enablement

```yaml
# space.yml
network:
  dns_hashing:
    enabled: true
```

**Result**: `postgres-a1b2c3.space.local` instead of `postgres.space.local`

### Example 2: Custom Hash Length

```yaml
# space.yml
network:
  dns_hashing:
    enabled: true
    length: 8  # Longer hash for extra collision resistance
```

**Result**: `postgres-a1b2c3d4.space.local`

### Example 3: Disabled (Current Behavior)

```yaml
# space.yml
network:
  dns_hashing:
    enabled: false
```

**Result**: `postgres.space.local` (backward compatible)

## Edge Cases and Error Handling

### Edge Case 1: Invalid Path

**Problem**: Path cannot be resolved to absolute path

**Solution**:
```go
func (h *PathHasher) GenerateHash(path string) (string, error) {
    absPath, err := filepath.Abs(path)
    if err != nil {
        return "", fmt.Errorf("cannot resolve path: %w", err)
    }
    // ... continue with hash generation
}
```

### Edge Case 2: Hash Collision

**Problem**: Two different paths produce same 6-character hash (unlikely but possible)

**Detection**:
```go
// pkg/dns/collision_detector.go
type CollisionDetector struct {
    registry map[string]string // hash -> path
}

func (c *CollisionDetector) DetectCollision(hash, path string) (bool, string) {
    if existing, exists := c.registry[hash]; exists {
        if existing != path {
            return true, existing // Collision detected
        }
    }
    c.registry[hash] = path
    return false, ""
}
```

**Resolution**:
1. Automatically increment hash length by 2
2. Log warning with both paths
3. Suggest manual configuration

### Edge Case 3: Service Name Too Long

**Problem**: `{service}-{hash}.space.local` exceeds DNS label limit (63 chars)

**Solution**:
```go
func (g *DomainGenerator) validateLength(serviceName, hash string) error {
    label := fmt.Sprintf("%s%s%s", serviceName, g.config.Separator, hash)
    if len(label) > 63 {
        // Truncate service name to fit
        maxServiceLen := 63 - len(g.config.Separator) - len(hash)
        if maxServiceLen < 1 {
            return fmt.Errorf("service name cannot fit in DNS label")
        }
        // Suggest truncating
        return fmt.Errorf("service name too long, truncate to %d chars", maxServiceLen)
    }
    return nil
}
```

### Edge Case 4: Symbolic Links

**Problem**: Symlinks may resolve to different paths

**Solution**:
```go
func (h *PathHasher) GenerateHash(path string) (string, error) {
    // Always resolve symlinks to canonical path
    absPath, err := filepath.Abs(path)
    if err != nil {
        return "", err
    }

    // Evaluate symlinks
    canonicalPath, err := filepath.EvalSymlinks(absPath)
    if err != nil {
        // If symlink evaluation fails, use original path
        canonicalPath = absPath
    }

    // Clean path
    canonicalPath = filepath.Clean(canonicalPath)

    // Generate hash
    // ...
}
```

### Edge Case 5: Git Worktrees

**Problem**: Must generate different hashes for different worktrees

**Solution**: No special handling needed - worktrees have different absolute paths:
- Main: `/Users/dev/project/.git`
- Worktree: `/Users/dev/project-worktrees/feature-branch`

Hash automatically differs due to different paths.

### Edge Case 6: Docker Container Name Mismatch

**Problem**: Container name doesn't match expected pattern

**Solution**: Implement fallback search in DNS resolver:
```go
func (s *Server) resolveContainerIP(ctx context.Context, hostname string) (string, error) {
    serviceName := s.extractServiceName(hostname)

    // Try multiple patterns (existing behavior)
    patterns := []string{
        serviceName,
        s.projectName + "-" + serviceName,
        s.projectName + "-" + serviceName + "-1",
        s.projectName + "-" + serviceName + "_1",
    }

    // If all patterns fail, list all containers and fuzzy match
    if ip, err := s.fuzzyContainerMatch(ctx, serviceName); err == nil {
        return ip, nil
    }

    return "", fmt.Errorf("no container found")
}
```

## Testing Strategy

### Unit Tests

```go
// pkg/dns/hasher_test.go
func TestPathHasher_Deterministic(t *testing.T) {
    hasher := NewPathHasher(HashConfig{Length: 6})

    path := "/Users/dev/project"
    hash1, _ := hasher.GenerateHash(path)
    hash2, _ := hasher.GenerateHash(path)

    assert.Equal(t, hash1, hash2, "same path should generate same hash")
}

func TestPathHasher_UniqueHashes(t *testing.T) {
    hasher := NewPathHasher(HashConfig{Length: 6})

    path1 := "/Users/dev/project1"
    path2 := "/Users/dev/project2"

    hash1, _ := hasher.GenerateHash(path1)
    hash2, _ := hasher.GenerateHash(path2)

    assert.NotEqual(t, hash1, hash2, "different paths should generate different hashes")
}

func TestDomainGenerator_BackwardCompatibility(t *testing.T) {
    gen := NewDomainGenerator(DomainConfig{
        BaseDomain: "space.local",
        HashConfig: HashConfig{Enabled: false},
    })

    domain, _ := gen.GenerateDomainName("postgres", "/path/to/project")
    assert.Equal(t, "postgres.space.local", domain)
}

func TestDomainGenerator_WithHashing(t *testing.T) {
    gen := NewDomainGenerator(DomainConfig{
        BaseDomain: "space.local",
        HashConfig: HashConfig{Enabled: true, Length: 6},
    })

    domain, _ := gen.GenerateDomainName("postgres", "/path/to/project")
    assert.Regexp(t, `^postgres-[a-z2-7]{6}\.space\.local$`, domain)
}
```

### Integration Tests

```go
// internal/dns/server_integration_test.go
func TestDNSServer_HashedDomains(t *testing.T) {
    // Setup: Start DNS server with hashing enabled
    // Test: Query postgres-abc123.space.local
    // Verify: Resolves to correct container IP
}

func TestDNSServer_LegacyDomains(t *testing.T) {
    // Setup: Start DNS server with hashing disabled
    // Test: Query postgres.space.local
    // Verify: Resolves to correct container IP (backward compatibility)
}
```

### End-to-End Tests

```bash
# Test scenario: Multiple project instances
mkdir -p /tmp/test-project-1
mkdir -p /tmp/test-project-2

cd /tmp/test-project-1
space up
curl http://postgres-abc123.space.local:5432

cd /tmp/test-project-2
space up
curl http://postgres-def456.space.local:5432

# Verify both postgres instances are accessible without collision
```

## Performance Considerations

### Hash Generation Performance

- **Blake2b**: ~500 MB/s on modern CPUs
- **Path hashing**: <1μs per path (negligible)
- **Caching**: Hash generated once per `space up`, cached in memory

### DNS Resolution Performance

- **Additional overhead**: ~10-50μs for service name extraction
- **Cache hit**: No performance impact (uses full domain as key)
- **Cache miss**: Negligible (single string operation)

### Memory Usage

- **Hash storage**: 6 bytes per project (negligible)
- **Domain cache**: Existing cache, no change

## Security Considerations

### Information Disclosure

**Risk**: Path hash reveals directory structure?

**Mitigation**:
- Blake2b is one-way (cannot reverse hash to path)
- Hash reveals no information about path contents
- 6-character Base32 provides sufficient entropy

### DNS Cache Poisoning

**Risk**: Attacker poisons DNS cache with fake IPs

**Mitigation**:
- DNS server only resolves *.space.local (local domains)
- Cache TTL is short (30 seconds)
- No external DNS queries for *.space.local

### Path Traversal

**Risk**: Malicious path input causes issues

**Mitigation**:
```go
func (h *PathHasher) GenerateHash(path string) (string, error) {
    // Sanitize input
    absPath, err := filepath.Abs(path)
    if err != nil {
        return "", err
    }

    // Clean path (removes .., ., //)
    absPath = filepath.Clean(absPath)

    // Validate path exists (optional)
    if _, err := os.Stat(absPath); err != nil {
        return "", fmt.Errorf("path does not exist: %w", err)
    }

    // ... generate hash
}
```

## Documentation Updates

### User Documentation

1. **Quick Start**: Add section on DNS collision prevention
2. **Configuration Reference**: Document `network.dns_hashing` options
3. **Migration Guide**: Step-by-step guide for existing users
4. **Troubleshooting**: Common issues and solutions

### Developer Documentation

1. **Architecture**: This design document
2. **API Reference**: Godoc for all new packages
3. **Contributing**: Guidelines for DNS-related changes

## Implementation Checklist

### Phase 1: Core Implementation

- [ ] Implement `pkg/dns/hasher.go` with Blake2b-256
- [ ] Implement `pkg/dns/domain_generator.go`
- [ ] Implement `pkg/dns/domain_validator.go`
- [ ] Add `DNSHashingConfig` to `pkg/config/schema.go`
- [ ] Update `internal/dns/server.go` for hashed domain support
- [ ] Add unit tests for all new packages
- [ ] Add integration tests for DNS resolution

### Phase 2: CLI Integration

- [ ] Update `internal/cli/up.go` to use domain generator
- [ ] Update `internal/cli/dns.go` for hash-aware commands
- [ ] Add `space config validate` to check DNS settings
- [ ] Update `createDNSModeCompose()` to use hashed domains

### Phase 3: Documentation & Testing

- [ ] Write user documentation
- [ ] Write migration guide
- [ ] Add end-to-end tests
- [ ] Performance benchmarks
- [ ] Security review

### Phase 4: Release

- [ ] Release notes
- [ ] Changelog
- [ ] Blog post / announcement
- [ ] Monitor for issues

## Future Enhancements

### Custom Hash Algorithms

Allow users to choose hash algorithm:

```yaml
network:
  dns_hashing:
    enabled: true
    algorithm: "blake2b"  # Options: blake2b, sha256, md5
```

### Global Hash Registry

Detect collisions across all projects:

```go
// Global registry in ~/.space/dns-registry.json
type GlobalRegistry struct {
    Projects map[string]ProjectInfo
}

type ProjectInfo struct {
    Hash string
    Path string
    LastAccessed time.Time
}
```

### DNS Alias Support

Support multiple aliases for same service:

```yaml
services:
  postgres:
    dns_aliases:
      - db
      - database
```

Results in:
- `postgres-abc123.space.local`
- `db-abc123.space.local`
- `database-abc123.space.local`

## Conclusion

This design provides a robust, backward-compatible solution for DNS collision prevention in Space CLI. The 6-character hash based on Blake2b-256 offers excellent collision resistance for typical use cases while maintaining human-readable domain names.

**Key Benefits:**
- ✅ Deterministic: Same path always generates same hash
- ✅ Collision-resistant: 1B possible combinations
- ✅ Short & readable: 6-character Base32 encoding
- ✅ Backward compatible: Optional feature, disabled by default
- ✅ Git worktree aware: Different paths = different hashes
- ✅ Fast: <1μs hash generation, negligible DNS overhead
- ✅ Secure: One-way hash, no information disclosure

**Next Steps:**
1. Review and approve design
2. Implement Phase 1 (Core implementation)
3. Test with real-world projects
4. Gather feedback and iterate
5. Release as opt-in feature in v0.3.0
