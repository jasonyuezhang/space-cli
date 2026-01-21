# Embedded DNS Server Implementation

## Overview

Space CLI now includes an embedded DNS server that enables `*.orb.local` domain resolution for Docker containers when using OrbStack. This eliminates the need for localhost port bindings and provides a more native container networking experience.

## Features

- **Automatic Provider Detection**: Detects OrbStack, Docker Desktop, or generic Docker
- **Embedded DNS Server**: Runs on localhost (ports 5353-5356, tries alternatives if in use)
- **Container IP Resolution**: Resolves `*.orb.local` queries by inspecting Docker containers
- **Upstream Forwarding**: Forwards non-orb.local queries to Google DNS (8.8.8.8)
- **DNS Caching**: Caches container IP resolutions for 30 seconds
- **Resolver Setup**: Creates `/etc/resolver/orb.local` pointing to embedded server
- **Graceful Fallback**: Falls back to port bindings if DNS setup fails
- **Port Conflict Resolution**: Automatically removes port bindings when DNS is enabled

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                     space-cli                           │
│  ┌───────────────────────────────────────────────────┐  │
│  │     Embedded DNS Server (port 5353-5356)          │  │
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
            (port 5354)
                            │
                            ↓
                    macOS DNS Resolution
                            │
                            ↓
                Browser: http://api-server.orb.local:6060
```

## Implementation Components

### Provider Detection (`internal/provider/provider.go`)
- Detects OrbStack by checking docker context and docker info
- Detects Docker Desktop similarly
- Falls back to generic Docker if neither is detected
- Determines if provider supports container DNS

### DNS Server (`internal/dns/`)
- **server.go**: Core DNS server using miekg/dns library
  - Handles *.orb.local A record queries
  - Forwards other queries to upstream DNS
  - Manages DNS server lifecycle

- **docker.go**: Docker integration
  - Resolves container names to IP addresses
  - Supports multiple naming patterns (with/without project prefix)
  - Lists all project containers

- **cache.go**: DNS response caching
  - LRU cache with TTL (30 seconds)
  - Automatic expiration
  - Thread-safe

- **resolver.go**: `/etc/resolver/orb.local` management
  - Creates resolver configuration (requires sudo)
  - Cleans up on `space down`
  - Flushes macOS DNS cache

- **logger.go**: Simple logging interface

### CLI Integration (`internal/cli/up.go`)
1. Detects provider
2. Starts DNS server if OrbStack is detected
3. Creates docker-compose override to remove port bindings
4. Runs docker-compose with override
5. Shows service URLs (*.orb.local or localhost depending on DNS status)

### Down Command (`internal/cli/down.go`)
- Stops and removes containers
- Cleans up DNS server and resolver automatically

## Usage

### Starting Services with DNS

```bash
cd /path/to/project
space up
```

Output:
```
🚀 Starting services for project: propel-gtm
📁 Working directory: /path/to/project

🔍 Detected provider: OrbStack
📦 Project name: propel-gtm-main

🌐 Starting embedded DNS server for container access...
ℹ️  Starting DNS server
ℹ️  DNS server started successfully
📝 Setting up DNS resolver (may require sudo password)...
[sudo password prompt]
✅ DNS server started successfully
   Containers will be accessible at: *.orb.local

📋 Starting all services

🌍 Access your services at:
   • postgres: http://postgres.orb.local:5432
   • api-server: http://api-server.orb.local:6060
   • app: http://app.orb.local:3000
   • worker: http://worker.orb.local:50000
   • workerui: http://workerui.orb.local:8080
```

### Stopping Services

```bash
space down
```

Output:
```
🛑 Stopping services for project: propel-gtm
📦 Project name: propel-gtm-main

🧹 Cleaning up DNS resolver...
🛑 Stopping DNS server...

✅ Services stopped successfully!
```

## Fallback Behavior

If DNS server fails to start (e.g., port in use, sudo denied), space-cli automatically falls back to traditional port bindings:

```
⚠️  Failed to start DNS server: ...
⚠️  Falling back to port bindings

🌍 Access your services at:
   • app: http://localhost:4000
   • api-server: http://localhost:6060
```

## Configuration

DNS settings can be configured in `.space.yaml`:

```yaml
provider:
  type: auto  # auto, orbstack, docker-desktop, generic

network:
  allowed_hosts: ".orb.local,localhost,127.0.0.1"
  network_mode: bridge
```

## Requirements

- macOS (for `/etc/resolver/` support)
- OrbStack or Docker Desktop
- Sudo access (for resolver setup, prompted once)

## Known Limitations

1. **Sudo Required**: First run requires sudo password to create `/etc/resolver/orb.local`
2. **macOS Only**: `/etc/resolver/` is a macOS feature
3. **Port Conflicts**: If ports 5353-5356 are all in use, DNS will fail
4. **Container Must Be Running**: DNS only resolves running containers
5. **Interactive Shell Required**: Sudo password prompt requires interactive terminal

## Future Enhancements

- [ ] Support for other TLDs (`.docker.local`, `.dev.local`)
- [ ] SRV records for service discovery
- [ ] IPv6 support (AAAA records)
- [ ] Linux support (systemd-resolved integration)
- [ ] Windows support (hosts file fallback)
- [ ] Web UI for DNS management
- [ ] Automatic certificate generation with mkcert

## Testing

The implementation includes:
- Provider detection (OrbStack vs Docker Desktop)
- DNS server with multiple port fallback (5353→5354→5355→5356)
- Container IP resolution with name pattern matching
- Docker compose override generation
- Automatic cleanup on failure
- Graceful fallback to port bindings

Tested with Propel GTM project (9 services, 2 databases).

## Dependencies

- `github.com/miekg/dns` - DNS server library
- `github.com/docker/docker` - Docker client (for future use)
- `github.com/spf13/cobra` - CLI framework
- `gopkg.in/yaml.v3` - YAML parsing

## Files Changed/Added

- `internal/provider/provider.go` (new)
- `internal/dns/server.go` (new)
- `internal/dns/cache.go` (new)
- `internal/dns/docker.go` (new)
- `internal/dns/resolver.go` (new)
- `internal/dns/logger.go` (new)
- `internal/cli/up.go` (modified)
- `internal/cli/down.go` (new)
- `internal/cli/root.go` (modified)
- `go.mod` (updated with dns dependency)

## References

- DNS Implementation Plan: `dns-implementation-plan.md`
- Postman Collection Format: Collection v2.1.0 schema
- Docker Compose: CLI integration
- OrbStack Documentation: Provider-specific networking
