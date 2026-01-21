# Architecture: Generic vs Propel-Specific

## Component Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                         User                                     │
└────────────────────────┬────────────────────────────────────────┘
                         │
                         │ $ propel sandbox up
                         │
┌────────────────────────▼────────────────────────────────────────┐
│                    propel-cli                                    │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │  internal/commands/sandbox.go                            │  │
│  │  • Wraps compose-dev-cli with Propel defaults           │  │
│  │  • Injects Propel-specific config                       │  │
│  └──────────────┬───────────────────────────────────────────┘  │
│                 │                                                │
│  ┌──────────────▼───────────────────────────────────────────┐  │
│  │  internal/propel/config.go                               │  │
│  │  • Defines Propel service names (api-server, app, etc.) │  │
│  │  • Sets database configs (propeldb, river)              │  │
│  │  • Configures River migrations & seeder                 │  │
│  └──────────────┬───────────────────────────────────────────┘  │
│                 │                                                │
└─────────────────┼────────────────────────────────────────────────┘
                  │
                  │ config := propel.DefaultConfig()
                  │ sandbox.New(ctx, workDir, config)
                  │
┌─────────────────▼────────────────────────────────────────────────┐
│              compose-dev-cli (Go module)                          │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │  pkg/sandbox/sandbox.go                                  │   │
│  │  • Generic service orchestration                         │   │
│  │  • Provider-aware networking                             │   │
│  │  • Service lifecycle (up, down, restart)                 │   │
│  └──┬───────────────────────────────────────────────────────┘   │
│     │                                                             │
│     │ Uses:                                                       │
│     │                                                             │
│  ┌──▼───────────────────┐  ┌──────────────────────────────┐    │
│  │ pkg/config/          │  │ pkg/provider/                │    │
│  │ • Load configs       │  │ • Detect OrbStack/Docker     │    │
│  │ • Merge layers       │  │ • Provider-specific logic    │    │
│  │ • Validate           │  │ • DNS vs port mapping        │    │
│  └──────────────────────┘  └──────────────────────────────┘    │
│                                                                   │
│  ┌──────────────────────┐  ┌──────────────────────────────┐    │
│  │ pkg/ports/           │  │ pkg/compose/                 │    │
│  │ • Port allocation    │  │ • Parse docker-compose.yml   │    │
│  │ • Persistence        │  │ • Modify for provider        │    │
│  │ • Range management   │  │ • Execute commands           │    │
│  └──────────────────────┘  └──────────────────────────────┘    │
│                                                                   │
│  ┌──────────────────────┐  ┌──────────────────────────────┐    │
│  │ pkg/project/         │  │ pkg/database/                │    │
│  │ • Git-based naming   │  │ • Generic DB operations      │    │
│  │ • Directory naming   │  │ • Postgres, MySQL, MongoDB   │    │
│  │ • Static naming      │  │ • Migrations, seeding        │    │
│  └──────────────────────┘  └──────────────────────────────┘    │
│                                                                   │
└───────────────────────────────────────────────────────────────────┘
                  │
                  │ docker compose up
                  │
┌─────────────────▼────────────────────────────────────────────────┐
│                      Docker Engine                                │
│                   (OrbStack or Docker Desktop)                    │
└───────────────────────────────────────────────────────────────────┘
```

## Code Flow Example: `propel sandbox up`

### 1. User Command
```bash
$ propel sandbox up
```

### 2. Propel CLI (Wrapper)
```go
// internal/commands/sandbox.go
func NewSandboxUpCommand() *cobra.Command {
    return &cobra.Command{
        Use: "up",
        RunE: func(cmd *cobra.Command, args []string) error {
            // 1. Load Propel-specific config
            config := propel.DefaultConfig()

            // 2. Create generic sandbox with Propel config
            sb, err := sandbox.New(ctx, workDir, config)
            if err != nil {
                return err
            }

            // 3. Start services (generic)
            return sb.Up(ctx, services)
        },
    }
}
```

### 3. Propel Config
```go
// internal/propel/config.go
func DefaultConfig() *config.Config {
    return &config.Config{
        Project: config.ProjectConfig{
            Name:   "propel-gtm",
            Prefix: "propel-gtm-",
        },
        Services: map[string]config.ServiceConfig{
            "api-server": {Port: 6060, Shell: "/bin/bash"},
            "app":        {Port: 3000, Shell: "/bin/sh"},
            "postgres":   {Port: 5432},
        },
        Databases: []config.DatabaseConfig{
            {Name: "propeldb", Service: "postgres", User: "admin"},
            {
                Name:              "river",
                Service:           "postgres",
                User:              "admin",
                AutoCreate:        true,
                MigrationsPath:    "cmd/river-migrate/main.go",
                MigrationsCommand: "go run {migrations_path} up",
            },
        },
        Commands: config.CommandsConfig{
            Seed:    "./bin/seeder",
            Migrate: "go run {migrations_path} up",
        },
    }
}
```

### 4. Generic Sandbox (Compose-Dev-CLI)
```go
// pkg/sandbox/sandbox.go
func (s *Sandbox) Up(ctx context.Context, services []string) error {
    // 1. Detect provider (OrbStack or Docker)
    provider := provider.Detect(ctx)

    // 2. Allocate ports (if Docker Desktop)
    if !provider.SupportsContainerDNS() {
        if err := s.ports.AllocatePorts(s.config.Services); err != nil {
            return err
        }
    }

    // 3. Modify compose file for provider
    composeFile, err := s.compose.LoadAndModify(provider)
    if err != nil {
        return err
    }

    // 4. Start services
    if err := s.compose.Up(ctx, services); err != nil {
        return err
    }

    // 5. Auto-create databases if configured
    for _, db := range s.config.Databases {
        if db.AutoCreate {
            if err := s.database.Create(ctx, db); err != nil {
                return err
            }
        }
    }

    // 6. Display service URLs
    s.displayServiceURLs()

    return nil
}
```

## Data Flow

### Configuration Loading
```
1. Defaults (hardcoded)
   ↓
2. Global config (~/.config/compose-cli/config.yaml)
   ↓
3. Project config (.compose-cli.yaml)
   ↓
4. Programmatic config (propel.DefaultConfig())
   ↓
5. Merged final config
```

### Service Startup Flow
```
User Command
   ↓
Propel CLI Command
   ↓
Load Propel Config
   ↓
Create Generic Sandbox
   ↓
Detect Provider (OrbStack/Docker)
   ↓
┌─────────────────────┬─────────────────────┐
│ OrbStack            │ Docker Desktop      │
├─────────────────────┼─────────────────────┤
│ Use Container DNS   │ Allocate Ports      │
│ Remove port bindings│ Set port env vars   │
│ *.orb.local         │ localhost:PORT      │
└─────────────────────┴─────────────────────┘
   ↓
Parse & Modify docker-compose.yml
   ↓
Execute: docker compose up
   ↓
Auto-create databases (if configured)
   ↓
Display service URLs
   ↓
Done
```

## Package Dependencies

```
compose-dev-cli packages:

pkg/config
  ↓
pkg/provider ──→ pkg/compose ──→ pkg/sandbox
  ↓                ↓               ↑
pkg/project ─────┘                │
  ↓                                │
pkg/ports ─────────────────────────┘
  ↓
pkg/database ──────────────────────┘
```

**No circular dependencies!**

## Interface Boundaries

### Public API (exported from compose-dev-cli)
```go
// Main entry point
func New(ctx context.Context, workDir string, config *Config) (*Sandbox, error)

// Sandbox operations
func (s *Sandbox) Up(ctx context.Context, services []string) error
func (s *Sandbox) Down(ctx context.Context) error
func (s *Sandbox) Restart(ctx context.Context, services []string) error
func (s *Sandbox) Status(ctx context.Context) ([]ServiceInfo, error)
func (s *Sandbox) Logs(ctx context.Context, service string) error
func (s *Sandbox) Shell(ctx context.Context, service string) error

// Database operations
func (s *Sandbox) DatabaseShell(ctx context.Context, dbName string) error
func (s *Sandbox) CreateDatabase(ctx context.Context, dbName string) error
func (s *Sandbox) Migrate(ctx context.Context, dbName string) error
func (s *Sandbox) Seed(ctx context.Context, dbName string) error

// Configuration
func LoadConfig(workDir string) (*Config, error)
func (c *Config) Validate() error
```

### Propel-Specific API (internal to propel-cli)
```go
// Generate Propel config
func DefaultConfig() *config.Config

// Propel constants
const (
    APIServiceName      = "api-server"
    AppServiceName      = "app"
    WorkerServiceName   = "worker"
    PostgresServiceName = "postgres"
)

// Propel-specific database ops
func SetupRiverDatabase(ctx context.Context, sb *sandbox.Sandbox) error
func RunSeeder(ctx context.Context, sb *sandbox.Sandbox) error
```

## Testing Strategy

### Unit Tests (compose-dev-cli)
- Config loading/merging
- Service discovery
- Port allocation
- Provider detection
- Project naming

### Integration Tests (compose-dev-cli)
- End-to-end with sample compose files
- Different providers (mocked)
- Database operations

### End-to-End Tests (propel-cli)
- Full `propel sandbox` workflow
- Verify all commands work
- Check backward compatibility
- Performance benchmarks

## Migration Path

### Phase 1: Side-by-Side
```
propel-cli (old code) ←─── Still works
compose-dev-cli (new)  ←─── Being developed
```

### Phase 2: Integration
```
propel-cli (wrapper) ──→ compose-dev-cli (library)
```

### Phase 3: Cleanup
```
propel-cli (thin wrapper)
   ├─ Propel-specific code only
   └─ VM management
compose-dev-cli (fully independent)
```

## Versioning Strategy

### compose-dev-cli
- Start at v1.0.0 after stabilization
- Follow semantic versioning
- Breaking changes = major version bump

### propel-cli
- Pin to stable compose-dev-cli version
- Test before upgrading compose-dev-cli
- Can use `replace` directive during development:
  ```go
  replace github.com/yourorg/compose-dev-cli => ../compose-dev-cli
  ```

## Success Criteria

✅ compose-dev-cli works with non-Propel projects
✅ All propel-cli commands maintain functionality
✅ Performance is equal or better
✅ Test coverage >80%
✅ Clear separation of concerns
✅ Documentation complete
✅ Zero breaking changes for users
