# Space CLI

A development environment orchestration tool for Docker Compose with intelligent provider detection (OrbStack, Docker Desktop).

## Features

- **Zero-config mode**: Works with just a `docker-compose.yml` present
- **Provider-aware networking**: OrbStack DNS (`*.space.local`) or Docker Desktop port mapping
- **Lifecycle hooks**: Automation via `.space/hooks/` scripts
- **Custom commands**: Project-specific commands via `.space/commands/`
- **DNS collision prevention**: Directory-based hashing for multi-project support

## Installation

```bash
# Build from source
make build

# Or install directly
go install github.com/happy-sdk/space-cli/cmd/space@latest
```

## Quick Start

```bash
# Start services (auto-detects provider)
space up

# List running containers with URLs
space ps

# Stop services
space down
```

## Commands

| Command | Description |
|---------|-------------|
| `space up` | Start services with DNS (OrbStack) or port mapping (Docker Desktop) |
| `space down` | Stop services and cleanup DNS |
| `space ps` | List containers with service URLs |
| `space config show` | Display merged configuration |
| `space config validate` | Validate configuration |
| `space dns status` | Check DNS daemon status |
| `space hooks list` | List available hooks |
| `space run <cmd>` | Run custom command from `.space/commands/` |

## Configuration

Create `.space.yaml` in your project root (optional - works without it):

```yaml
project:
  name: myapp
  naming_strategy: git-branch  # or "directory", "static"
  compose_files:
    - docker-compose.yml

services:
  api:
    port: 6060
    shell: /bin/bash
```

### Configuration Priority

1. Project config (`.space.yaml`) - Highest priority
2. Global config (`~/.config/space/config.yaml`)
3. Defaults

## Provider Detection

Space CLI automatically detects your Docker provider:

| Provider | DNS Mode | Port Mapping | Container DNS |
|----------|----------|--------------|---------------|
| OrbStack | Yes (`*.space.local`) | No | Yes |
| Docker Desktop | No | Yes | No |
| Generic Docker | No | Yes | No |

## Hooks

Create executable scripts in `.space/hooks/` to run at lifecycle events:

```
.space/
└── hooks/
    ├── pre-up.d/       # Before services start
    ├── post-up.d/      # After services running
    ├── pre-down.d/     # Before services stop
    ├── post-down.d/    # After services stopped
    └── on-dns-ready.d/ # When DNS configured
```

Hooks receive context as JSON on stdin with project info, services, and DNS details.

## Custom Commands

Create scripts in `.space/commands/` to add project-specific commands:

```
.space/
└── commands/
    ├── deploy.sh    # Shell
    ├── migrate.py   # Python
    └── test.ts      # TypeScript
```

Run with: `space run deploy` or `space deploy` (if no conflict)

Supported languages: Shell, Python, Node.js, TypeScript, Go, Ruby, Perl

## DNS Architecture (OrbStack)

With OrbStack, services are accessible via DNS names:

```
service-<hash>.space.local
```

The 6-character hash is derived from the project directory path, preventing collisions when multiple projects have services with the same name.

## Development

```bash
# Build
make build

# Run tests
make test

# Run linter
make lint

# Show version
space --version
```

### Version Management

Versions are automatically bumped based on commit message keywords:

- `[major]` or `[breaking]` - Major version (x.0.0)
- `[minor]` or `[feature]` or `[feat]` - Minor version (0.x.0)
- `[patch]` or default - Patch version (0.0.x)

## Architecture

```
space-cli/
├── cmd/space/           # CLI entry point
├── internal/
│   ├── cli/             # Command implementations
│   ├── dns/             # Embedded DNS server
│   ├── hooks/           # Hook execution system
│   └── provider/        # Docker provider detection
├── pkg/
│   └── config/          # Configuration system
└── examples/            # Example configurations
```

## License

MIT
