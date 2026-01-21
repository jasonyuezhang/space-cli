# Space CLI Prototypes

This directory contains prototypes and planning documents for extracting generic Docker Compose and VM management functionality from propel-cli into a separate `space-cli` repository.

## Contents

### Planning Documents

- **`../MIGRATION_PLAN.md`** - Complete 4-week migration plan with phases and tasks
- **`EXTRACTION_CHECKLIST.md`** - Detailed checklist of files to extract and changes needed
- **`ARCHITECTURE.md`** - Architecture diagrams and component breakdown

### Prototypes

- **`space-cli/pkg/config/`** - Configuration system prototypes
  - `schema.go` - Complete configuration schema with all settings (including VM support)
  - `loader.go` - Multi-level config loading and merging

### Example Configurations

- **`examples/rails/`** - Example Rails application config
- **`examples/nodejs/`** - Minimal Node.js application config
- **`examples/with-vm/`** - VM-based development configuration

## Configuration System

The generic CLI uses a flexible configuration system with these priorities:

1. **Project config** (`.space.yaml` in project root) - Highest priority
2. **Global config** (`~/.config/space/config.yaml`)
3. **Defaults** - Sensible defaults for common scenarios

### Key Features

- **Auto-discovery**: Reads `docker-compose.yml` to detect services and ports
- **Provider-aware**: Adapts to OrbStack (DNS-based) vs Docker Desktop (port-based)
- **VM support**: Integrated VM management with Lima and OrbStack VM providers
- **Flexible database ops**: Configurable migrations, seeding, shell access
- **Custom commands**: Define project-specific commands
- **Port management**: Smart port allocation with persistence

## Usage Examples

### Minimal Config (Most things auto-discovered)

```yaml
# .space.yaml
project:
  name: myapp
```

### Rails App Config

See `examples/rails/.space.yaml` for:
- Standard Rails service setup
- PostgreSQL database with Rails migrations
- Custom Rails commands (console, routes, test)

### VM-Based Development

See `examples/with-vm/.space.yaml` for:
- VM configuration (Lima or OrbStack VM)
- Resource allocation (CPU, memory, disk)
- Dependency installation
- Startup commands

## Architecture

```
space-cli/                        # New generic repo
├── cmd/space/                    # CLI binary
├── pkg/
│   ├── config/                   # Configuration system
│   ├── provider/                 # Docker provider detection (OrbStack, Docker Desktop)
│   ├── compose/                  # Docker Compose operations
│   ├── ports/                    # Port allocation
│   ├── project/                  # Project naming
│   ├── sandbox/                  # Service orchestration (Docker)
│   ├── vm/                       # VM management
│   │   ├── manager.go            # VM lifecycle
│   │   └── provider/             # VM provider implementations
│   │       ├── lima.go           # Lima provider
│   │       └── orbstack.go       # OrbStack VM provider
│   └── database/                 # Database operations
└── examples/                     # Example configs

propel-cli/                       # Existing repo (becomes thin wrapper)
├── internal/
│   ├── propel/                   # Propel-specific logic ONLY
│   │   ├── config.go             # Default Propel config
│   │   ├── constants.go          # Service names (api-server, app, etc.)
│   │   └── database.go           # River integration
│   ├── commands/                 # CLI commands (wraps space-cli)
│   └── util/                     # Propel utilities (PR copy, etc.)
└── go.mod                        # imports space-cli
```

## Integration Strategy

Propel-CLI will use space-cli as a Go module:

```go
import (
    spacecli "github.com/yourorg/space-cli/pkg/sandbox"
    "propel-cli/internal/propel"
)

func NewSandboxCommand() *cobra.Command {
    return &cobra.Command{
        RunE: func(cmd *cobra.Command, args []string) error {
            // Create sandbox with Propel defaults
            config := propel.DefaultConfig()
            sb, err := spacecli.New(ctx, workDir, config)
            // ...
        },
    }
}
```

## Benefits

### For Propel
- **Lighter codebase**: ~40% reduction by offloading generic code
- **Focused development**: Concentrate on Propel-specific features
- **Better separation**: Clear boundary between generic and proprietary
- **Easier testing**: Generic code tested independently

### For Space CLI
- **Reusable**: Any team with Docker Compose or VM needs can use it
- **Open source potential**: Could be published for wider use
- **Better quality**: Designed for multiple use cases from day one
- **Complete solution**: Both Docker and VM workflows in one tool
- **Community contributions**: External developers could contribute

## Timeline

- **Week 1**: Repository setup + config system
- **Week 2**: Extract Docker Compose packages
- **Week 3**: Extract VM management + build CLI + documentation
- **Week 4**: Integrate with propel-cli + full testing

**Total: 4 weeks** for complete migration

## Command Examples

### Docker-based Development
```bash
space up                # Start services in Docker
space logs api          # View service logs
space shell app         # Open shell in container
space db shell          # Open database shell
```

### VM-based Development
```bash
space vm start          # Create and start VM
space vm status         # Check VM status
space vm shell          # Open shell in VM
space up                # Start services inside VM
```

### Database Operations
```bash
space db create mydb    # Create database
space db migrate        # Run migrations
space db seed           # Seed database
```

## Next Steps

1. ✅ Create prototypes (this directory)
2. Review and approve plan
3. Create `space-cli` repository
4. Start Phase 1: Repository setup
5. Follow extraction checklist
6. Test with sample projects
7. Integrate with propel-cli
8. Release v1.0.0

## Questions?

See:
- `../MIGRATION_PLAN.md` for detailed phases and timeline
- `EXTRACTION_CHECKLIST.md` for what moves where
- `ARCHITECTURE.md` for architecture diagrams
- `examples/` for configuration examples
