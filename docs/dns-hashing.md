# DNS Collision Prevention with Directory-Based Hashing

## Overview

Space CLI implements DNS collision prevention using directory-based hashing. This feature prevents naming conflicts when running multiple projects with the same service names from different directories.

## The Problem

When running multiple projects simultaneously, services with the same names (e.g., `web`, `api`, `postgres`) can collide in DNS resolution:

```
Project A: /home/user/project-a
  - web.space.local → Container A

Project B: /home/user/project-b
  - web.space.local → Container A or B? ❌ Collision!
```

## The Solution

Space CLI generates a unique 6-character hash based on the project's directory path and appends it to service names:

```
Project A: /home/user/project-a (hash: a1b2c3)
  - web-a1b2c3.space.local → Container A ✅

Project B: /home/user/project-b (hash: d4e5f6)
  - web-d4e5f6.space.local → Container B ✅
```

## How It Works

### 1. Hash Generation

The hash is generated using SHA-256 of the absolute directory path:

```go
hash := sha256.Sum256([]byte("/absolute/path/to/project"))
shortHash := hex.Encode(hash)[:6] // First 6 characters
```

### 2. DNS Domain Format

Services are accessible at: `{service}-{hash}.space.local`

Examples:
- `web-a1b2c3.space.local:3000`
- `api-a1b2c3.space.local:8080`
- `postgres-a1b2c3.space.local:5432`

### 3. DNS Server Resolution

The DNS server automatically:
1. Extracts the service name from the hashed domain
2. Resolves the container IP for that service
3. Returns the IP address

## Configuration

### Enable/Disable Hashing

Add to your `.space.yml` file:

```yaml
network:
  # Enable DNS hashing (default: true)
  dns_hashing: true
```

To disable hashing and use simple domain names:

```yaml
network:
  dns_hashing: false
```

### Default Behavior

- **Enabled by default**: DNS hashing is ON when using DNS mode
- **Automatic**: No manual configuration needed
- **Transparent**: Works seamlessly with existing workflows

## Benefits

### 1. Multi-Project Development

Run multiple projects simultaneously without DNS conflicts:

```bash
# Terminal 1
cd ~/project-a
space up
# Services: web-a1b2c3.space.local, api-a1b2c3.space.local

# Terminal 2
cd ~/project-b
space up
# Services: web-d4e5f6.space.local, api-d4e5f6.space.local

# Both work! No conflicts! ✅
```

### 2. Deterministic Hashing

- Same directory → same hash (always)
- Predictable and reliable
- Easy to remember for a specific project

### 3. Backward Compatible

- Legacy domain names still work (without hash)
- Graceful fallback if hash extraction fails
- No breaking changes to existing workflows

## Technical Details

### Hash Properties

- **Length**: 6 hexadecimal characters
- **Algorithm**: SHA-256
- **Deterministic**: Same input → same output
- **Collision Resistance**: Extremely low probability of collisions

### Domain Pattern Recognition

The DNS server recognizes hashed domains by:
1. Checking for format: `{name}-{6-hex-chars}.{domain}`
2. Validating the hash is 6 characters of hex
3. Extracting the service name before the hash

### Service Name Extraction

```
Input:  web-frontend-a1b2c3.space.local
Output: web-frontend

Input:  api-service-123abc.space.local
Output: api-service
```

## Usage Examples

### Check Current DNS URLs

```bash
space ps
```

Output with hashing enabled:
```
SERVICE   STATE     PORTS          DNS URL                          LOCAL URL
-------   -----     -----          -------                          ---------
web       running   3000/tcp       http://web-a1b2c3.space.local:3000   http://localhost:3000
api       running   8080/tcp       http://api-a1b2c3.space.local:8080   http://localhost:8080
postgres  running   5432/tcp       http://postgres-a1b2c3.space.local:5432   http://localhost:5432
```

### Testing DNS Resolution

```bash
# Get the DNS server address
space dns status

# Test resolution
dig @127.0.0.1 -p 5353 web-a1b2c3.space.local

# Or use curl
curl http://web-a1b2c3.space.local:3000
```

### Disabling Hashing

If you don't need collision prevention:

1. Create `.space.yml`:
```yaml
network:
  dns_hashing: false
```

2. Restart services:
```bash
space down
space up
```

3. Services now use simple names:
```
http://web.space.local:3000
http://api.space.local:8080
```

## Troubleshooting

### Hash Not Matching Expected

The hash is based on the **absolute path**, so:

```bash
# These produce DIFFERENT hashes
cd /home/user/project && space up  # hash: abc123
cd ~/project && space up             # hash: def456 (~ expanded differently)

# Solution: Always use absolute paths or cd to directory first
```

### DNS Not Resolving

1. Check DNS server is running:
```bash
space dns status
```

2. Verify hashing is enabled:
```bash
grep dns_hashing .space.yml
```

3. Restart DNS server:
```bash
space dns restart
```

### Multiple Projects Showing Same Hash

If two projects in the same directory show the same hash, that's expected!
The hash is directory-based, not project-name-based.

To run multiple configurations from the same directory:
1. Use different compose files
2. Use different project names in `.space.yml`
3. Or place projects in different directories

## Implementation Details

### Files Modified

- `internal/dns/hash.go` - Hash generation and domain utilities
- `internal/dns/hash_test.go` - Comprehensive test suite
- `internal/dns/server.go` - DNS server hashing support
- `internal/cli/ps.go` - URL generation with hashing
- `internal/cli/up.go` - DNS server initialization with workDir
- `pkg/config/schema.go` - Configuration schema for dns_hashing

### Key Functions

- `GenerateDirectoryHash(dirPath)` - Creates 6-char hash
- `GenerateHashedDomainName(service, path, domain)` - Full domain with hash
- `ExtractServiceNameFromHashedDomain(domain, base)` - Reverse lookup
- `ValidateHashedDomain(domain, base)` - Pattern validation

## Best Practices

1. **Enable hashing** for development environments with multiple projects
2. **Disable hashing** for production or single-project setups
3. **Use absolute paths** in scripts to ensure consistent hashing
4. **Document custom domains** if you modify the hashing behavior
5. **Test DNS resolution** after enabling/disabling hashing

## Future Enhancements

Potential improvements being considered:

- Custom hash length configuration
- Alternative hashing algorithms (MD5, CityHash)
- Hash caching for performance
- DNS dashboard showing all hashed domains
- Collision detection and warnings

## References

- [DNS Server Implementation](../internal/dns/server.go)
- [Configuration Schema](../pkg/config/schema.go)
- [Example Configuration](../examples/space.example.yml)
