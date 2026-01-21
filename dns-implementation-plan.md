# Embedded DNS Server Implementation Plan

## Problem Statement

The propel-cli sandbox currently relies on OrbStack's `.orb.local` DNS resolution, but this doesn't work reliably on all systems. When OrbStack's DNS is not configured (missing `/etc/resolver/orb.local`), containers become inaccessible from the host browser despite being fully functional.

**Current behavior:**
- Propel-cli detects OrbStack and removes port bindings
- Assumes `.orb.local` DNS will work
- If DNS fails, containers are unreachable (ERR_NAME_NOT_RESOLVED)

## Solution Overview

Embed a lightweight DNS server directly into propel-cli that:
1. Resolves `*.orb.local` queries by inspecting Docker containers
2. Starts automatically with `propel sandbox up`
3. Stops with `propel sandbox down`
4. Creates `/etc/resolver/orb.local` pointing to the embedded server
5. Falls back to upstream DNS for non-orb.local queries

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                      propel-cli                         │
│  ┌───────────────────────────────────────────────────┐  │
│  │          Embedded DNS Server (port 5353)          │  │
│  │  - Listen on 127.0.0.1:5353                       │  │
│  │  - Handle *.orb.local queries                     │  │
│  │  - Query Docker for container IPs                 │  │
│  │  - Forward other queries upstream                 │  │
│  └───────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────┘
                            │
                            ↓
            /etc/resolver/orb.local
            (nameserver 127.0.0.1)
            (port 5353)
                            │
                            ↓
                    macOS DNS Resolution
                            │
                            ↓
                Browser: http://container.orb.local:3000
```

## Implementation Steps

### Phase 1: Core DNS Server

1. **Add DNS library dependency**
   ```bash
   go get github.com/miekg/dns
   ```

2. **Create DNS server package**
   - Location: `internal/dns/server.go`
   - Responsibilities:
     - Start/stop DNS server
     - Handle DNS queries
     - Query Docker for container IPs
     - Forward non-matching queries upstream

3. **Key functions:**
   ```go
   type Server struct {
       listener   net.PacketConn
       handler    *dns.ServeMux
       docker     DockerClient
       upstream   string // e.g., "8.8.8.8:53"
       logger     *zap.Logger
   }

   func NewServer(docker DockerClient, logger *zap.Logger) *Server
   func (s *Server) Start(ctx context.Context, addr string) error
   func (s *Server) Stop() error
   func (s *Server) handleOrbLocal(w dns.ResponseWriter, r *dns.Msg)
   func (s *Server) forwardUpstream(w dns.ResponseWriter, r *dns.Msg)
   func (s *Server) resolveContainerIP(hostname string) (string, error)
   ```

### Phase 2: Docker Integration

1. **Container name parsing**
   - Extract container name from `*.orb.local` hostname
   - Handle both short names and full docker-compose names
   - Example: `api-server.orb.local` → `propel-xyz-api-server-1`

2. **Docker inspect integration**
   ```go
   func (s *Server) resolveContainerIP(hostname string) (string, error) {
       // Strip .orb.local suffix
       containerName := strings.TrimSuffix(hostname, ".orb.local")

       // Try to find matching container
       containers := s.docker.ListContainers(ctx)
       for _, c := range containers {
           if strings.Contains(c.Names[0], containerName) {
               return c.NetworkSettings.Networks["bridge"].IPAddress, nil
           }
       }
       return "", fmt.Errorf("container not found")
   }
   ```

3. **Caching layer**
   - Cache container name → IP mappings (TTL: 30 seconds)
   - Invalidate cache on container events

### Phase 3: System Integration

1. **Resolver configuration**
   - Location: `internal/dns/resolver.go`
   - Create `/etc/resolver/orb.local` (requires sudo)
   - Content:
     ```
     nameserver 127.0.0.1
     port 5353
     ```

2. **Sudo handling**
   ```go
   func (r *Resolver) Setup(ctx context.Context) error {
       // Check if /etc/resolver exists
       // Create /etc/resolver/orb.local with sudo
       // Verify it works
   }

   func (r *Resolver) Cleanup(ctx context.Context) error {
       // Remove /etc/resolver/orb.local with sudo
   }
   ```

3. **Permission handling**
   - Prompt user once for sudo password
   - Cache sudo credentials for cleanup
   - Graceful degradation if sudo denied

### Phase 4: Sandbox Integration

1. **Modify `sandbox.Start()`**
   ```go
   func (s *Sandbox) Start(ctx context.Context, services ...string) error {
       // ... existing code ...

       if s.provider.SupportsContainerDNS() {
           // Start embedded DNS server
           if err := s.startDNSServer(ctx); err != nil {
               s.logger.Warn("Failed to start DNS server, using port bindings instead",
                   zap.Error(err))
               // Fall back to port bindings
               s.provider = ProviderDocker
           }
       }

       // ... existing code ...
   }
   ```

2. **Modify `sandbox.Stop()`**
   ```go
   func (s *Sandbox) Stop(ctx context.Context) error {
       // Stop DNS server
       if s.dnsServer != nil {
           s.dnsServer.Stop()
       }

       // ... existing code ...
   }
   ```

3. **DNS server lifecycle**
   - Start before containers
   - Setup resolver after server starts
   - Cleanup resolver before stopping server
   - Handle errors gracefully

### Phase 5: Fallback Mechanism

1. **DNS health check**
   ```go
   func (s *Sandbox) testDNSResolution(hostname string) error {
       // Try to resolve a test hostname
       // Wait up to 5 seconds
       // Return error if fails
   }
   ```

2. **Automatic fallback**
   - If DNS server fails to start → keep port bindings
   - If DNS test fails → keep port bindings
   - If resolver setup fails → keep port bindings
   - Log warning but continue with localhost URLs

3. **User feedback**
   ```
   ⚠️  DNS server failed to start, using port bindings instead
   Access your app at: http://localhost:3000
   ```

## Code Structure

```
internal/dns/
├── server.go          # Core DNS server
├── resolver.go        # /etc/resolver management
├── cache.go           # IP caching layer
├── docker.go          # Docker integration
└── server_test.go     # Unit tests

internal/sandbox/
├── sandbox.go         # Modified to use DNS server
├── dns.go             # DNS server lifecycle management
└── provider.go        # Enhanced provider detection
```

## Configuration

Add to sandbox configuration:

```go
type Config struct {
    // ... existing fields ...

    // DNS server configuration
    DNSEnabled     bool   // Enable embedded DNS server
    DNSAddr        string // DNS server address (default: "127.0.0.1:5353")
    DNSUpstream    string // Upstream DNS server (default: "8.8.8.8:53")
    DNSCacheTTL    int    // Cache TTL in seconds (default: 30)
    DNSResolverDir string // Resolver directory (default: "/etc/resolver")
}
```

## Testing Approach

### Unit Tests

1. **DNS server tests**
   - Mock Docker client
   - Test A record resolution
   - Test forwarding to upstream
   - Test error handling

2. **Resolver tests**
   - Mock filesystem operations
   - Test resolver file creation
   - Test cleanup

### Integration Tests

1. **End-to-end test**
   ```bash
   # Start sandbox with DNS
   propel sandbox up

   # Verify DNS resolution
   dig @127.0.0.1 -p 5353 api-server.orb.local

   # Verify browser access
   curl http://api-server.orb.local:6060

   # Stop sandbox
   propel sandbox down

   # Verify cleanup
   ls /etc/resolver/orb.local  # Should not exist
   ```

2. **Failure scenarios**
   - DNS server port already in use
   - Sudo password denied
   - Container not found
   - Network connectivity issues

## Edge Cases

1. **Port 5353 already in use**
   - Try alternative ports (5354, 5355, etc.)
   - Fall back to port bindings if all fail

2. **Multiple sandbox instances**
   - Share single DNS server across instances
   - Reference count for cleanup
   - Handle concurrent access

3. **Container IP changes**
   - Invalidate cache on Docker events
   - Re-resolve on NXDOMAIN

4. **Upstream DNS failure**
   - Retry with exponential backoff
   - Cache last known good result
   - Return SERVFAIL if truly unavailable

5. **macOS DNS cache**
   - Flush DNS cache after resolver setup: `sudo dscacheutil -flushcache`
   - Handle Ventura+ changes to DNS resolution

## Security Considerations

1. **Sudo access**
   - Only request sudo when necessary
   - Clear explanation of why sudo is needed
   - Fallback if denied

2. **DNS spoofing protection**
   - Only resolve `*.orb.local` domains
   - Validate container names before resolution
   - Don't expose internal Docker network

3. **Port binding**
   - Bind to localhost only (127.0.0.1)
   - Don't expose DNS server to LAN

## Performance

1. **Caching strategy**
   - Cache DNS responses for 30 seconds
   - Invalidate on Docker events
   - LRU cache with max 1000 entries

2. **Query forwarding**
   - Reuse upstream connections
   - Timeout after 2 seconds
   - Concurrent query handling

3. **Memory usage**
   - Expected: ~5-10MB for DNS server
   - Cache size limit: 1000 entries * ~100 bytes = 100KB

## Migration Path

1. **Phase 1: Add DNS server (optional)**
   - DNS server disabled by default
   - Users opt-in with `--enable-dns` flag
   - Keep existing OrbStack DNS behavior

2. **Phase 2: Enable by default**
   - DNS server enabled by default
   - Automatic fallback to port bindings
   - Deprecation warning for OrbStack DNS

3. **Phase 3: Remove OrbStack dependency**
   - Remove OrbStack DNS detection
   - Always use embedded DNS server
   - Remove port binding fallback

## Future Enhancements

1. **Support for other TLDs**
   - `.docker.local`
   - `.dev.local`
   - User-configurable

2. **Advanced DNS features**
   - SRV records for service discovery
   - AAAA records for IPv6
   - MX records for email testing

3. **Web UI**
   - View active DNS mappings
   - Test DNS resolution
   - Clear cache manually

4. **Integration with other tools**
   - Export DNS mappings for /etc/hosts
   - Generate mkcert certificates
   - Browser extension for auto-configuration

## Dependencies

- `github.com/miekg/dns` - DNS library
- `github.com/docker/docker` - Docker client (already present)
- `go.uber.org/zap` - Logging (already present)

## Estimated Effort

- Phase 1 (Core DNS): 2-3 days
- Phase 2 (Docker Integration): 1 day
- Phase 3 (System Integration): 1-2 days
- Phase 4 (Sandbox Integration): 1 day
- Phase 5 (Fallback): 1 day
- Testing & documentation: 2 days

**Total: ~8-10 days**

## Success Criteria

- ✅ DNS server resolves `*.orb.local` correctly
- ✅ Browser can access containers via DNS names
- ✅ Works without OrbStack DNS configured
- ✅ Graceful fallback to port bindings on failure
- ✅ No sudo required if resolver already exists
- ✅ Clean up on sandbox stop
- ✅ Zero configuration for end users
- ✅ Performance: DNS queries < 10ms
- ✅ Reliability: 99.9% uptime during sandbox session
