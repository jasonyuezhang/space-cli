# Space CLI - Complete Implementation Plan

## Executive Summary

**Space CLI** is a generic development environment tool that provides:
- Docker Compose orchestration with provider awareness (OrbStack, Docker Desktop)
- VM management with pluggable providers (Lima, OrbStack VM)
- Smart port allocation and service discovery
- Database operations and migrations
- Zero-config mode with auto-detection

**Timeline:** 4 weeks | **Scope:** Generic reusable library + CLI | **Philosophy:** Convention over configuration

---

## Table of Contents

1. [Quick Adoption Path](#quick-adoption-path)
2. [Architecture Overview](#architecture-overview)
3. [Configuration System](#configuration-system)
4. [Implementation Phases](#implementation-phases)
5. [Package Extraction Guide](#package-extraction-guide)
6. [Testing Strategy](#testing-strategy)
7. [Migration Helpers](#migration-helpers)
8. [API Design](#api-design)

---

## Quick Adoption Path

### Goal: Zero Friction for Existing Codebases

#### Option 1: Zero Config (Auto-detect everything)
```bash
# In any directory with docker-compose.yml
space up

# space-cli automatically:
# - Detects services from docker-compose.yml
# - Detects provider (OrbStack vs Docker Desktop)
# - Allocates ports (if needed)
# - Generates service URLs
```

#### Option 2: Minimal Config (One-liner)
```bash
# Create minimal config
space init

# Generates .space.yaml with detected services:
# project:
#   name: my-project  # from directory name or git repo
```

#### Option 3: Preset-Based (Framework-specific)
```bash
# Quick setup for known frameworks
space init --preset=rails        # Rails defaults
space init --preset=nodejs       # Node.js defaults
space init --preset=go           # Go defaults
space init --preset=propel       # Propel-specific defaults
```

#### Option 4: Migration Helper (From existing tools)
```bash
# Migrate from propel-cli
space migrate --from=propel-cli

# Generates .space.yaml with Propel-style config
# Preserves port allocations, service names, etc.
```

### Convention Over Configuration

**Default Behaviors (No Config Required):**

1. **Project Naming:** Git branch + directory hash
2. **Service Discovery:** Parse docker-compose.yml
3. **Port Allocation:** Sequential from 10000-60000
4. **Provider Detection:** Auto-detect OrbStack/Docker
5. **Database Type:** Infer from service name (postgres, mysql, mongodb)
6. **Shell:** Default to `/bin/sh` (or `/bin/bash` if available)

**When Config IS Needed:**

1. Multiple databases in same service
2. Custom migration commands
3. VM-based development
4. Non-standard ports or networking
5. Custom startup commands

---

## Architecture Overview

### Repository Structure

```
space-cli/
├── cmd/
│   └── space/
│       └── main.go                      # CLI entry point
│
├── pkg/
│   ├── config/                          # Configuration system
│   │   ├── schema.go                    # Config structs
│   │   ├── loader.go                    # Multi-level loading
│   │   ├── validator.go                 # Validation
│   │   ├── discovery.go                 # Auto-detect from compose
│   │   └── presets.go                   # Framework presets
│   │
│   ├── provider/                        # Docker provider detection
│   │   ├── provider.go                  # Interface & detection
│   │   ├── orbstack.go                  # OrbStack-specific
│   │   └── docker.go                    # Generic Docker
│   │
│   ├── compose/                         # Docker Compose operations
│   │   ├── client.go                    # Compose client
│   │   ├── parser.go                    # Parse docker-compose.yml
│   │   ├── modifier.go                  # Modify for provider
│   │   └── executor.go                  # Execute commands
│   │
│   ├── ports/                           # Port management
│   │   ├── allocator.go                 # Port allocation
│   │   ├── persistence.go               # Save/load port mappings
│   │   └── detector.go                  # Detect available ports
│   │
│   ├── project/                         # Project naming
│   │   ├── namer.go                     # Naming strategies
│   │   ├── git.go                       # Git-based naming
│   │   └── directory.go                 # Directory-based naming
│   │
│   ├── sandbox/                         # Service orchestration (Docker)
│   │   ├── sandbox.go                   # Main sandbox
│   │   ├── services.go                  # Service lifecycle
│   │   ├── urls.go                      # URL generation
│   │   ├── logs.go                      # Log streaming
│   │   └── shell.go                     # Shell access
│   │
│   ├── vm/                              # VM management
│   │   ├── manager.go                   # VM lifecycle
│   │   ├── config.go                    # VM configuration
│   │   ├── dependency.go                # Dependency installation
│   │   └── provider/
│   │       ├── interface.go             # Provider interface
│   │       ├── lima.go                  # Lima implementation
│   │       └── orbstack.go              # OrbStack VM implementation
│   │
│   ├── database/                        # Database operations
│   │   ├── interface.go                 # Database client interface
│   │   ├── postgres.go                  # PostgreSQL
│   │   ├── mysql.go                     # MySQL
│   │   ├── mongodb.go                   # MongoDB
│   │   └── operations.go                # Common operations
│   │
│   └── migration/                       # Migration helpers
│       ├── propel.go                    # Migrate from propel-cli
│       └── detector.go                  # Detect existing tools
│
├── internal/
│   ├── cli/                             # CLI commands
│   │   ├── root.go                      # Root command
│   │   ├── up.go                        # space up
│   │   ├── down.go                      # space down
│   │   ├── vm.go                        # space vm
│   │   ├── db.go                        # space db
│   │   ├── init.go                      # space init
│   │   └── migrate.go                   # space migrate
│   │
│   └── util/                            # Internal utilities
│       ├── logger.go
│       ├── terminal.go
│       └── exec.go
│
├── examples/
│   ├── rails/
│   ├── nodejs/
│   ├── go/
│   └── with-vm/
│
├── docs/
│   ├── configuration.md
│   ├── providers.md
│   ├── vm-setup.md
│   └── api-reference.md
│
└── README.md
```

### Data Flow: Zero-Config Mode

```
User: $ space up
    ↓
1. Check for .space.yaml
   NOT FOUND → Use zero-config mode
    ↓
2. Parse docker-compose.yml
   - Extract service names
   - Extract ports
   - Detect database services
    ↓
3. Detect provider
   OrbStack → Use container DNS
   Docker   → Allocate ports
    ↓
4. Start services
   docker compose up -d
    ↓
5. Display service URLs
   ✓ web: http://myapp-main-a1b2c3-web.orb.local
   ✓ api: http://myapp-main-a1b2c3-api.orb.local
```

### Data Flow: With Configuration

```
User: $ space up
    ↓
1. Load configuration
   Project → Global → Defaults
    ↓
2. Merge with auto-detected services
   Config + docker-compose.yml
    ↓
3. Apply provider-specific modifications
   OrbStack: Remove port bindings
   Docker: Inject port env vars
    ↓
4. Start services + Auto-create DBs
    ↓
5. Run startup commands (if configured)
```

---

## Configuration System

### Design Principles

1. **Optional by default** - Works without config
2. **Progressive disclosure** - Add config as needed
3. **Intuitive structure** - Mirrors docker-compose.yml
4. **Framework presets** - Quick start for common stacks
5. **Auto-detection** - Discover services, databases, ports

### Configuration Schema

```go
type Config struct {
    Project   ProjectConfig              // Project settings
    Services  map[string]ServiceConfig   // Service overrides
    Databases []DatabaseConfig           // Database configuration
    Commands  CommandsConfig             // Custom commands
    Provider  ProviderConfig             // Docker provider settings
    VM        VMConfig                   // VM configuration
    Network   NetworkConfig              // Networking
    Ports     PortsConfig                // Port allocation
}
```

### Example: Zero Config

```yaml
# No .space.yaml file needed!
# Just run: space up
```

### Example: Minimal Config

```yaml
project:
  name: myapp
```

### Example: Rails Preset

```yaml
project:
  name: myapp

services:
  web:
    port: 3000
    shell: /bin/bash

  postgres:
    port: 5432

databases:
  - name: myapp_development
    service: postgres
    type: postgres
    auto_create: true
    migrations_command: "docker compose exec web rails db:migrate"
```

### Example: VM-Based Development

```yaml
project:
  name: myapp

vm:
  enabled: true
  provider: auto
  cpus: 4
  memory: "8GB"
  dependencies:
    - docker
    - docker-compose
    - git

services:
  app:
    port: 3000
```

### Example: Propel Migration

```yaml
# Generated by: space migrate --from=propel-cli

project:
  name: propel-gtm
  prefix: propel-gtm-
  naming_strategy: git-branch

services:
  api-server:
    port: 6060
    shell: /bin/bash

  app:
    port: 3000

  postgres:
    port: 5432

databases:
  - name: propeldb
    service: postgres
    type: postgres
    user: admin

  - name: river
    service: postgres
    type: postgres
    user: admin
    auto_create: true
    migrations_path: cmd/river-migrate/main.go
    migrations_command: "go run {migrations_path} up"

commands:
  seed: "./bin/seeder"
  migrate: "go run {migrations_path} up"
```

---

## Implementation Phases

### Phase 1: Foundation (Week 1)

#### Days 1-2: Repository Setup

**Deliverables:**
- [x] GitHub repository created: `space-cli`
- [x] Go module initialized: `github.com/yourorg/space-cli`
- [x] Directory structure created
- [x] CI/CD pipeline (GitHub Actions)
- [x] Linting configured (golangci-lint)
- [x] Basic README

**Tasks:**
```bash
# Create repository
gh repo create yourorg/space-cli --public --gitignore=Go --license=MIT

# Initialize Go module
cd space-cli
go mod init github.com/yourorg/space-cli

# Create structure
mkdir -p cmd/space pkg/{config,provider,compose,ports,project,sandbox,vm,database,migration} internal/{cli,util} examples docs

# Setup CI
# Create .github/workflows/test.yml
# Create .github/workflows/lint.yml
```

#### Days 3-5: Configuration System

**Deliverables:**
- [x] Config schema (`pkg/config/schema.go`)
- [x] Config loader with multi-level merging (`pkg/config/loader.go`)
- [x] Config validator (`pkg/config/validator.go`)
- [x] Service discovery from docker-compose.yml (`pkg/config/discovery.go`)
- [x] Framework presets (`pkg/config/presets.go`)
- [x] Unit tests (>80% coverage)

**Key Features:**
1. Load config from: `.space.yaml` → `~/.config/space/config.yaml` → defaults
2. Auto-detect services from `docker-compose.yml`
3. Validate config schema
4. Framework presets (rails, nodejs, go, propel)
5. Zero-config mode support

**Files to Create:**

`pkg/config/discovery.go`:
```go
// DiscoverFromCompose parses docker-compose.yml and generates config
func DiscoverFromCompose(composePath string) (*Config, error) {
    // Parse docker-compose.yml
    // Extract services, ports, volumes
    // Infer database types from service names
    // Generate default config
}

// DetectFramework attempts to detect the project framework
func DetectFramework(workDir string) (string, error) {
    // Check for Gemfile → rails
    // Check for package.json → nodejs
    // Check for go.mod → go
    // etc.
}
```

`pkg/config/presets.go`:
```go
func RailsPreset() *Config
func NodeJSPreset() *Config
func GoPreset() *Config
func PropelPreset() *Config
```

### Phase 2: Docker Compose Integration (Week 2)

#### Days 1-2: Provider Detection

**Extract from propel-cli:**
- `internal/sandbox/provider.go` → `pkg/provider/provider.go`
- `internal/sandbox/provider_test.go` → `pkg/provider/provider_test.go`

**Changes:**
- Remove hardcoded constants
- Make DNS suffix configurable
- Support more detection methods

**Deliverables:**
- [x] Provider detection (OrbStack vs Docker Desktop)
- [x] Provider-specific behavior (DNS vs ports)
- [x] Auto-detection with fallback
- [x] Unit tests

#### Days 2-3: Port Management

**Extract from propel-cli:**
- `internal/sandbox/ports.go` → `pkg/ports/allocator.go`
- `internal/utils/port_mapping.go` → `pkg/ports/persistence.go`

**Changes:**
- Make port ranges configurable
- Support multiple allocation strategies
- Remove hardcoded service ports

**New Files:**
- `pkg/ports/detector.go` - Detect available ports

**Deliverables:**
- [x] Port allocation (sequential, random)
- [x] Port persistence (`.space-ports.json`)
- [x] Conflict detection
- [x] Port availability checking
- [x] Unit tests

#### Day 3: Project Naming

**Extract from propel-cli:**
- `internal/sandbox/project.go` → `pkg/project/namer.go`
- `internal/utils/project.go` → `pkg/project/git.go`

**New Files:**
- `pkg/project/directory.go` - Directory-based naming

**Changes:**
- Support multiple naming strategies
- Make prefix configurable

**Deliverables:**
- [x] Git-based project naming
- [x] Directory-based naming
- [x] Static naming
- [x] Configurable naming strategy
- [x] Unit tests

#### Days 4-5: Compose Client

**Extract from propel-cli:**
- `internal/utils/docker.go` → `pkg/compose/client.go`
- `internal/utils/docker_client.go` → `pkg/compose/interface.go`

**New Files:**
- `pkg/compose/parser.go` - Parse docker-compose.yml
- `pkg/compose/modifier.go` - Modify compose for providers
- `pkg/compose/executor.go` - Execute compose commands

**Changes:**
- Remove Propel-specific service knowledge
- Make operations generic
- Support multiple compose files

**Deliverables:**
- [x] Compose file parser
- [x] Provider-aware compose modification
- [x] Generic service operations
- [x] Multi-file support
- [x] Integration tests

### Phase 3: Core Services (Week 2-3)

#### Week 2, Day 5 - Week 3, Day 2: Sandbox & Database

**Extract from propel-cli:**
- `internal/sandbox/sandbox.go` → `pkg/sandbox/sandbox.go`

**New Files:**
- `pkg/sandbox/services.go` - Service lifecycle
- `pkg/sandbox/urls.go` - URL generation
- `pkg/sandbox/logs.go` - Log streaming
- `pkg/sandbox/shell.go` - Shell access
- `pkg/database/interface.go` - Database interface
- `pkg/database/postgres.go` - PostgreSQL
- `pkg/database/mysql.go` - MySQL
- `pkg/database/mongodb.go` - MongoDB

**Deliverables:**
- [x] Generic sandbox orchestration
- [x] Service lifecycle (up, down, restart)
- [x] URL generation (provider-aware)
- [x] Log streaming
- [x] Shell access
- [x] Database abstraction
- [x] Auto-create databases
- [x] Database shell access
- [x] Integration tests

#### Week 3, Days 1-2: VM Management

**Extract from propel-cli:**
- `internal/vm/manager.go` → `pkg/vm/manager.go`
- `internal/vm/config.go` → `pkg/vm/config.go`
- `internal/vm/dependency.go` → `pkg/vm/dependency.go`
- `internal/vm/provider/*.go` → `pkg/vm/provider/*.go`

**Changes:**
- Remove Propel-specific constants
- Make VM configuration generic
- Support custom dependencies

**Deliverables:**
- [x] VM lifecycle management
- [x] Lima provider
- [x] OrbStack VM provider
- [x] Dependency installation
- [x] VM configuration
- [x] Integration tests

### Phase 4: CLI & Migration (Week 3)

#### Days 3-4: CLI Implementation

**Command Structure:**
```
space
├── init [--preset=rails|nodejs|go|propel]
├── migrate --from=propel-cli
├── up [services...]
├── down
├── restart [services...]
├── build [services...]
├── rebuild [services...]
├── logs [service]
├── shell [service]
├── status
├── links
├── config
│   ├── init
│   ├── show
│   └── validate
├── db
│   ├── shell [dbname]
│   ├── create <dbname>
│   ├── drop <dbname>
│   ├── migrate [dbname]
│   └── seed [dbname]
└── vm
    ├── start
    ├── stop
    ├── restart
    ├── status
    ├── shell
    ├── logs
    └── delete
```

**Deliverables:**
- [x] All CLI commands implemented
- [x] Help text for all commands
- [x] Flags and arguments
- [x] Shell completions (bash, zsh, fish)
- [x] Progress indicators
- [x] Error messages

#### Days 4-5: Migration Helpers

**New Files:**
- `pkg/migration/propel.go` - Migrate from propel-cli
- `pkg/migration/detector.go` - Detect existing tools
- `internal/cli/migrate.go` - Migration command

**Features:**
1. Detect propel-cli usage
2. Migrate port allocations (`.sandbox-ports.json` → `.space-ports.json`)
3. Generate `.space.yaml` with Propel defaults
4. Preserve custom configurations
5. Migration report

**Deliverables:**
- [x] `space migrate --from=propel-cli` command
- [x] Port mapping migration
- [x] Config generation
- [x] Migration validation
- [x] Integration tests

### Phase 5: Testing & Documentation (Week 3-4)

#### Week 3, Days 4-5: Testing

**Test Coverage Goals:**
- Unit tests: >80% coverage
- Integration tests: Key workflows
- E2E tests: Sample projects

**Test Scenarios:**
1. **Zero-config mode:**
   - Start services without config
   - Auto-detect provider
   - Generate URLs

2. **With configuration:**
   - Load and merge configs
   - Apply custom settings
   - Validate config

3. **Provider scenarios:**
   - OrbStack: Container DNS
   - Docker Desktop: Port mapping

4. **VM scenarios:**
   - Lima: Create and start VM
   - OrbStack VM: Create and start VM

5. **Database scenarios:**
   - PostgreSQL: Create, migrate, seed
   - MySQL: Create, migrate
   - MongoDB: Create

6. **Migration scenarios:**
   - Migrate from propel-cli
   - Preserve port allocations
   - Generate config

**Test Projects:**
- Simple single-service app
- Rails app with PostgreSQL
- Node.js app with MongoDB
- Go app with PostgreSQL
- Multi-service app
- VM-based app

**Deliverables:**
- [x] Unit test suite (>80% coverage)
- [x] Integration test suite
- [x] E2E test suite
- [x] Sample test projects
- [x] CI/CD pipeline running tests

#### Week 4, Days 1-2: Documentation

**Documentation Structure:**

1. **README.md**
   - Quick start
   - Installation
   - Basic usage
   - Examples

2. **docs/configuration.md**
   - Complete config reference
   - All settings explained
   - Examples for each setting

3. **docs/providers.md**
   - OrbStack setup and usage
   - Docker Desktop setup and usage
   - Provider-specific features

4. **docs/vm-setup.md**
   - Lima installation and config
   - OrbStack VM setup
   - VM configuration options
   - Troubleshooting

5. **docs/api-reference.md**
   - Go package documentation
   - Public API
   - Usage examples

6. **docs/migration.md**
   - Migrating from propel-cli
   - Migrating from other tools
   - Breaking changes

7. **docs/presets.md**
   - Available presets
   - Creating custom presets

8. **CONTRIBUTING.md**
   - How to contribute
   - Development setup
   - Testing guidelines

**Deliverables:**
- [x] Complete documentation
- [x] Code examples
- [x] Troubleshooting guides
- [x] API documentation (godoc)

### Phase 6: Integration with propel-cli (Week 4)

#### Days 3-4: Propel-CLI Integration

**In propel-cli repository:**

1. **Add dependency:**
```go
// go.mod
require github.com/yourorg/space-cli v1.0.0
```

2. **Create Propel config:**
```go
// internal/propel/config.go
package propel

import spacecli "github.com/yourorg/space-cli/pkg/config"

func DefaultConfig() *spacecli.Config {
    return &spacecli.Config{
        Project: spacecli.ProjectConfig{
            Name:   "propel-gtm",
            Prefix: "propel-gtm-",
        },
        Services: map[string]spacecli.ServiceConfig{
            "api-server": {Port: 6060, Shell: "/bin/bash"},
            "app":        {Port: 3000, Shell: "/bin/sh"},
            "postgres":   {Port: 5432},
        },
        Databases: []spacecli.DatabaseConfig{
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
        Commands: spacecli.CommandsConfig{
            Seed: "./bin/seeder",
        },
    }
}
```

3. **Wrap sandbox commands:**
```go
// internal/commands/sandbox.go
func NewSandboxUpCommand() *cobra.Command {
    return &cobra.Command{
        Use: "up",
        RunE: func(cmd *cobra.Command, args []string) error {
            config := propel.DefaultConfig()
            sb, err := sandbox.New(ctx, workDir, config)
            if err != nil {
                return err
            }
            return sb.Up(ctx, services)
        },
    }
}
```

4. **Keep in propel-cli:**
   - `internal/propel/config.go` - Propel defaults
   - `internal/propel/constants.go` - Service names
   - `internal/propel/database.go` - River-specific logic
   - `internal/commands/util.go` - PR copy utility

5. **Remove from propel-cli:**
   - All extracted packages
   - Generic utilities
   - Provider detection
   - Port management
   - VM management (now in space-cli)

**Deliverables:**
- [x] space-cli added as dependency
- [x] All commands updated to use space-cli
- [x] Propel-specific logic isolated
- [x] Backward compatibility maintained
- [x] All tests passing

#### Day 5: Validation & Cleanup

**Validation Checklist:**
- [ ] All propel-cli commands work unchanged
- [ ] Port allocations preserved
- [ ] OrbStack integration works
- [ ] VM management works
- [ ] Performance equal or better
- [ ] All tests passing
- [ ] Documentation updated

**Cleanup:**
- [ ] Remove extracted code from propel-cli
- [ ] Update propel-cli README
- [ ] Create migration guide for Propel team
- [ ] Tag releases: space-cli v1.0.0, propel-cli v2.0.0

---

## Package Extraction Guide

### Quick Reference

| Source (propel-cli) | Destination (space-cli) | Changes Needed |
|---------------------|-------------------------|----------------|
| `internal/sandbox/provider.go` | `pkg/provider/provider.go` | Remove hardcoded DNS suffix |
| `internal/sandbox/ports.go` | `pkg/ports/allocator.go` | Make port ranges configurable |
| `internal/sandbox/project.go` | `pkg/project/namer.go` | Support multiple strategies |
| `internal/utils/docker.go` | `pkg/compose/client.go` | Remove Propel service names |
| `internal/sandbox/sandbox.go` | `pkg/sandbox/sandbox.go` | Accept config, make generic |
| `internal/utils/database.go` | `pkg/database/postgres.go` | Abstract database operations |
| `internal/vm/*` | `pkg/vm/*` | Remove Propel constants |

### Detailed Extraction Steps

#### 1. Provider Detection

**Source Files:**
- `internal/sandbox/provider.go`
- `internal/sandbox/provider_test.go`

**Destination:**
- `pkg/provider/provider.go`
- `pkg/provider/provider_test.go`

**Changes:**
```diff
- const OrbStackDNSSuffix = ".orb.local"
+ func (p ProviderType) GetDNSSuffix(config *Config) string {
+     if config.Provider.OrbStack != nil && config.Provider.OrbStack.DNSSuffix != "" {
+         return config.Provider.OrbStack.DNSSuffix
+     }
+     return ".orb.local"
+ }
```

#### 2. Port Management

**Source Files:**
- `internal/sandbox/ports.go`
- `internal/utils/port_mapping.go`

**Destination:**
- `pkg/ports/allocator.go`
- `pkg/ports/persistence.go`

**Changes:**
```diff
- const DefaultPortRangeStart = 10000
- const DefaultPortRangeEnd = 60000
+ type Allocator struct {
+     rangeStart int
+     rangeEnd   int
+ }
+ func NewAllocator(config PortsConfig) *Allocator

- func (pc *PortConfig) GetAPIServerPort() int { return 6060 }
+ func (pc *PortConfig) GetServicePort(serviceName string, config *Config) (int, error)
```

#### 3. Project Naming

**Source Files:**
- `internal/sandbox/project.go`
- `internal/utils/project.go`

**Destination:**
- `pkg/project/namer.go`
- `pkg/project/git.go`
- `pkg/project/directory.go`

**Changes:**
```diff
- const ProjectPrefix = "propel-gtm"
+ type Namer interface {
+     GenerateName(workDir string, config *Config) (string, error)
+ }

+ type GitBranchNamer struct{}
+ type DirectoryNamer struct{}
+ type StaticNamer struct{ name string }
```

#### 4. Compose Client

**Source Files:**
- `internal/utils/docker.go`
- `internal/utils/docker_client.go`

**Destination:**
- `pkg/compose/client.go`
- `pkg/compose/interface.go`
- `pkg/compose/parser.go` (new)
- `pkg/compose/modifier.go` (new)

**Changes:**
```diff
- func (d *DockerUtils) StartAPIServer(ctx context.Context) error
- func (d *DockerUtils) StartApp(ctx context.Context) error
+ func (c *Client) StartService(ctx context.Context, name string) error
+ func (c *Client) StartServices(ctx context.Context, names []string) error
```

#### 5. Sandbox

**Source Files:**
- `internal/sandbox/sandbox.go`
- `internal/sandbox/commands.go`

**Destination:**
- `pkg/sandbox/sandbox.go`
- `pkg/sandbox/services.go` (new)
- `pkg/sandbox/urls.go` (new)
- `pkg/sandbox/logs.go` (new)
- `pkg/sandbox/shell.go` (new)

**Changes:**
```diff
- const DefaultShellService = "api-server"
+ func (s *Sandbox) Shell(ctx context.Context, service string) error {
+     if service == "" {
+         service = s.config.GetDefaultShellService()
+     }
+ }

- func (s *Sandbox) CreateRiverDatabase() error
+ // Move to pkg/database, call via config
```

#### 6. Database

**Source Files:**
- `internal/utils/database.go`

**Destination:**
- `pkg/database/interface.go` (new)
- `pkg/database/postgres.go`
- `pkg/database/mysql.go` (new)
- `pkg/database/mongodb.go` (new)
- `pkg/database/operations.go` (new)

**Interface:**
```go
type Client interface {
    Connect(ctx context.Context, dsn string) error
    CreateDatabase(ctx context.Context, name string) error
    DropDatabase(ctx context.Context, name string) error
    ListDatabases(ctx context.Context) ([]string, error)
    OpenShell(ctx context.Context, dbName string) error
}
```

#### 7. VM Management

**Source Files:**
- `internal/vm/manager.go`
- `internal/vm/config.go`
- `internal/vm/dependency.go`
- `internal/vm/provider/*.go`

**Destination:**
- `pkg/vm/manager.go`
- `pkg/vm/config.go`
- `pkg/vm/dependency.go`
- `pkg/vm/provider/*.go`

**Changes:**
```diff
- // Propel-specific VM naming
+ // Generic VM naming based on project config
```

---

## Testing Strategy

### Test Pyramid

```
    E2E Tests (10%)
   ─────────────────
  Integration Tests (30%)
 ─────────────────────────
Unit Tests (60%)
```

### Unit Tests (60% of tests)

**Coverage Goal:** >80%

**Package Testing:**
- `pkg/config/` - Config loading, merging, validation, discovery
- `pkg/provider/` - Provider detection, behavior
- `pkg/ports/` - Port allocation, persistence, conflicts
- `pkg/project/` - Naming strategies
- `pkg/compose/` - Parsing, modification, execution
- `pkg/database/` - Database operations (mocked)
- `pkg/vm/` - VM operations (mocked)

**Example Test:**
```go
func TestConfigMerge(t *testing.T) {
    base := Defaults()
    override := &Config{
        Project: ProjectConfig{Name: "custom"},
    }
    merged := base.Merge(override)
    assert.Equal(t, "custom", merged.Project.Name)
    assert.Equal(t, 10000, merged.Ports.RangeStart) // From defaults
}
```

### Integration Tests (30% of tests)

**Test Scenarios:**

1. **Zero-config mode:**
   - Parse docker-compose.yml
   - Auto-detect provider
   - Start services
   - Verify URLs

2. **With configuration:**
   - Load multi-level config
   - Apply overrides
   - Start services

3. **Provider switching:**
   - Test OrbStack behavior
   - Test Docker Desktop behavior

4. **Database operations:**
   - Create database
   - Run migrations (mocked)
   - Seed database (mocked)

**Example Test:**
```go
func TestZeroConfigMode(t *testing.T) {
    // Setup test project with docker-compose.yml
    dir := createTestProject(t)

    // Run space up (no config file)
    sb, err := sandbox.New(ctx, dir, nil)
    require.NoError(t, err)

    err = sb.Up(ctx, nil)
    require.NoError(t, err)

    // Verify services started
    status, err := sb.Status(ctx)
    require.NoError(t, err)
    assert.Len(t, status, 2) // web, postgres
}
```

### E2E Tests (10% of tests)

**Test Projects:**

1. **Simple app:** Single service, no database
2. **Rails app:** web + postgres + redis
3. **Node.js app:** app + mongodb
4. **Go app:** api + postgres
5. **Multi-service:** api + web + worker + postgres + redis
6. **VM-based:** App running in VM

**Test Flow:**
```bash
# E2E test for Rails app
cd test-projects/rails-app

# Test zero-config
space up
space status  # Should show 3 services running
space links   # Should show URLs
space db shell myapp_development  # Should connect
space down

# Test with config
space init --preset=rails
space up
space db migrate
space db seed
space test  # Custom command
space down

# Cleanup
rm -rf .space-ports.json .space.yaml
```

---

## Migration Helpers

### space migrate --from=propel-cli

**Detection:**
```go
func DetectPropelCLI(workDir string) (bool, error) {
    // Check for .sandbox-ports.json
    // Check for propel-specific service names in docker-compose.yml
    // Check for River migrations
}
```

**Migration Steps:**

1. **Detect propel-cli usage**
2. **Migrate port allocations:**
   ```
   .sandbox-ports.json → .space-ports.json
   ```
3. **Generate .space.yaml:**
   - Use Propel preset
   - Add detected services
   - Add database configs
   - Add custom commands
4. **Report:**
   ```
   ✓ Migrated port allocations (3 services)
   ✓ Generated .space.yaml with Propel defaults
   ✓ Detected River database migrations

   Next steps:
   1. Review .space.yaml
   2. Run: space up
   3. Remove propel-cli: go mod edit -dropreplace propel-cli
   ```

**Generated Config:**
```yaml
# Generated by: space migrate --from=propel-cli
# Review and customize as needed

project:
  name: propel-gtm
  prefix: propel-gtm-
  naming_strategy: git-branch

services:
  api-server:
    port: 6060
    shell: /bin/bash

  app:
    port: 3000
    shell: /bin/sh

  postgres:
    port: 5432

databases:
  - name: propeldb
    service: postgres
    type: postgres
    user: admin

  - name: river
    service: postgres
    type: postgres
    user: admin
    auto_create: true
    migrations_path: cmd/river-migrate/main.go
    migrations_command: "go run {migrations_path} up"

commands:
  seed: "./bin/seeder"
  migrate: "go run cmd/river-migrate/main.go up"

provider:
  type: auto

ports:
  persistence_file: .space-ports.json
```

---

## API Design

### Public API (Exported)

**For library consumers (like propel-cli):**

```go
package sandbox

// New creates a sandbox with the given config
// If config is nil, uses zero-config mode with auto-detection
func New(ctx context.Context, workDir string, config *config.Config) (*Sandbox, error)

// Sandbox operations
func (s *Sandbox) Up(ctx context.Context, services []string) error
func (s *Sandbox) Down(ctx context.Context) error
func (s *Sandbox) Restart(ctx context.Context, services []string) error
func (s *Sandbox) Build(ctx context.Context, services []string, noCache bool) error
func (s *Sandbox) Status(ctx context.Context) ([]ServiceInfo, error)
func (s *Sandbox) Logs(ctx context.Context, service string, follow bool) error
func (s *Sandbox) Shell(ctx context.Context, service string, shell string) error
func (s *Sandbox) GetServiceURL(service string) (string, error)
func (s *Sandbox) GetServiceURLs() (map[string]string, error)

// Database operations
func (s *Sandbox) DatabaseShell(ctx context.Context, dbName string) error
func (s *Sandbox) CreateDatabase(ctx context.Context, dbName string) error
func (s *Sandbox) DropDatabase(ctx context.Context, dbName string) error
func (s *Sandbox) MigrateDatabase(ctx context.Context, dbName string) error
func (s *Sandbox) SeedDatabase(ctx context.Context, dbName string) error

// VM operations
func (s *Sandbox) VMStart(ctx context.Context) error
func (s *Sandbox) VMStop(ctx context.Context) error
func (s *Sandbox) VMStatus(ctx context.Context) (*VMStatus, error)
func (s *Sandbox) VMShell(ctx context.Context) error

// Configuration
func LoadConfig(workDir string) (*config.Config, error)
func (c *Config) Validate() error
```

### Usage Examples

**Example 1: Zero-config mode**
```go
import "github.com/yourorg/space-cli/pkg/sandbox"

// Create sandbox with auto-detection
sb, err := sandbox.New(ctx, ".", nil)
if err != nil {
    return err
}

// Start all services
if err := sb.Up(ctx, nil); err != nil {
    return err
}

// Get service URLs
urls, err := sb.GetServiceURLs()
for name, url := range urls {
    fmt.Printf("%s: %s\n", name, url)
}
```

**Example 2: With custom config (Propel)**
```go
import (
    spacecli "github.com/yourorg/space-cli/pkg/sandbox"
    "github.com/yourorg/space-cli/pkg/config"
)

// Create Propel-specific config
cfg := &config.Config{
    Project: config.ProjectConfig{
        Name:   "propel-gtm",
        Prefix: "propel-gtm-",
    },
    Services: map[string]config.ServiceConfig{
        "api-server": {Port: 6060},
        "app":        {Port: 3000},
    },
}

// Create sandbox with config
sb, err := spacecli.New(ctx, workDir, cfg)
if err != nil {
    return err
}

// Start specific services
if err := sb.Up(ctx, []string{"api-server", "postgres"}); err != nil {
    return err
}

// Auto-create River database
if err := sb.CreateDatabase(ctx, "river"); err != nil {
    return err
}
```

**Example 3: VM-based development**
```go
// Config with VM enabled
cfg := &config.Config{
    VM: config.VMConfig{
        Enabled:  true,
        Provider: "lima",
        CPUs:     4,
        Memory:   "8GB",
    },
}

sb, err := spacecli.New(ctx, workDir, cfg)

// Start VM
if err := sb.VMStart(ctx); err != nil {
    return err
}

// Start services in VM
if err := sb.Up(ctx, nil); err != nil {
    return err
}
```

---

## Success Criteria

### Must Have (v1.0.0)

- [ ] **Zero-config mode works** - `space up` works without any config file
- [ ] **Framework presets** - Rails, Node.js, Go presets available
- [ ] **Migration helper** - `space migrate --from=propel-cli` works
- [ ] **Provider detection** - Auto-detects OrbStack vs Docker Desktop
- [ ] **Port management** - Smart allocation, persistence, conflict resolution
- [ ] **Database operations** - Create, migrate, seed, shell
- [ ] **VM support** - Lima and OrbStack VM providers work
- [ ] **CLI complete** - All commands implemented with help text
- [ ] **Tests passing** - >80% unit test coverage, integration tests pass
- [ ] **Documentation** - Complete docs, API reference, examples
- [ ] **Propel integration** - propel-cli successfully uses space-cli
- [ ] **Performance** - Equal or better than current implementation

### Should Have (v1.1.0)

- [ ] **More presets** - Django, Laravel, Spring Boot
- [ ] **Health checks** - Service health monitoring
- [ ] **Better error messages** - Actionable suggestions
- [ ] **Telemetry** - Optional usage analytics
- [ ] **Plugin system** - Custom providers, commands

### Nice to Have (v1.2.0+)

- [ ] **GUI** - Web-based dashboard
- [ ] **Docker Swarm** - Swarm orchestration
- [ ] **Kubernetes** - K8s local dev
- [ ] **Cloud integration** - Deploy to cloud

---

## Timeline Summary

| Week | Focus | Deliverable | Hours |
|------|-------|-------------|-------|
| 1 | Foundation | Repo + Config System | 40h |
| 2 | Docker Integration | Compose + Ports + Provider | 40h |
| 3 | Services + VM | Sandbox + Database + VM + CLI | 40h |
| 4 | Integration | Testing + Docs + Propel integration | 40h |

**Total: 4 weeks @ 40h/week = 160 hours**

**With VM extraction:** Adds 2-3 days (~16-24 hours)

---

## Getting Started (For Implementers)

### Prerequisites

- Go 1.25+
- Docker or OrbStack
- Lima (optional, for VM support)
- git

### Setup

```bash
# Clone propel-cli (source code)
git clone https://github.com/yourorg/propel-cli
cd propel-cli

# Create space-cli repo
gh repo create yourorg/space-cli --public --gitignore=Go --license=MIT
cd ../space-cli

# Initialize
go mod init github.com/yourorg/space-cli
mkdir -p cmd/space pkg/{config,provider,compose,ports,project,sandbox,vm,database,migration} internal/{cli,util} examples docs

# Start with Phase 1
# Implement pkg/config first (foundation)
# Then move to Phase 2 (Docker integration)
# etc.
```

### Development Workflow

1. **Implement a package** (e.g., `pkg/config`)
2. **Write unit tests** (>80% coverage)
3. **Update CLI** to use the package
4. **Write integration test**
5. **Document** the package
6. **Repeat** for next package

### Testing During Development

```bash
# Run unit tests
go test ./pkg/...

# Run integration tests
go test -tags=integration ./pkg/...

# Test with sample project
cd examples/rails
space up
space status
space down
```

---

## Risk Mitigation

### Risks & Mitigation

| Risk | Impact | Mitigation |
|------|--------|------------|
| Breaking changes in propel-cli | High | Maintain backward compatibility, feature flags |
| Performance regression | Medium | Benchmark before/after, optimize hot paths |
| Complex migration | Medium | Provide migration helper, good docs |
| Missing edge cases | Medium | Comprehensive tests, beta testing |
| Adoption friction | High | Zero-config mode, presets, migration helper |

### Rollback Plan

1. Keep original code in propel-cli until stable
2. Use `go.mod replace` directive during development
3. Tag pre-migration version of propel-cli
4. Maintain propel-cli v1.x branch for critical fixes

---

## Next Steps

1. **Review this plan** with stakeholders
2. **Create space-cli repository**
3. **Start Phase 1** (Foundation)
4. **Daily standups** to track progress
5. **Weekly demos** of completed features
6. **Beta testing** with Propel team (Week 4)
7. **Release v1.0.0** after validation

---

## Questions & Feedback

For questions or feedback on this implementation plan:

1. Open an issue in the space-cli repository
2. Tag with `implementation-plan` label
3. Assign to project lead

---

## Appendix

### A. Framework Preset Specifications

#### Rails Preset
```yaml
project:
  naming_strategy: git-branch

services:
  web:
    port: 3000
    shell: /bin/bash
    health_check:
      enabled: true
      endpoint: /up
      timeout: 5s

  postgres:
    port: 5432

  redis:
    port: 6379

databases:
  - name: "{project_name}_development"
    service: postgres
    type: postgres
    user: postgres
    auto_create: true
    migrations_command: "docker compose exec web rails db:migrate"
    seed_command: "docker compose exec web rails db:seed"

commands:
  migrate: "docker compose exec web rails db:migrate"
  seed: "docker compose exec web rails db:seed"
  custom:
    console: "docker compose exec web rails console"
    routes: "docker compose exec web rails routes"
    test: "docker compose exec web rails test"
```

#### Node.js Preset
```yaml
project:
  naming_strategy: directory

services:
  app:
    port: 3000
    shell: /bin/sh
    environment:
      NODE_ENV: development

  mongodb:
    port: 27017

databases:
  - name: "{project_name}"
    service: mongodb
    type: mongodb
    auto_create: true

commands:
  custom:
    test: "npm test"
    lint: "npm run lint"
```

#### Go Preset
```yaml
project:
  naming_strategy: git-branch

services:
  api:
    port: 8080
    shell: /bin/sh
    health_check:
      enabled: true
      endpoint: /health

  postgres:
    port: 5432

databases:
  - name: "{project_name}"
    service: postgres
    type: postgres
    user: postgres
    auto_create: true

commands:
  custom:
    test: "go test ./..."
    build: "go build -o bin/api ./cmd/api"
```

### B. Environment Variable Reference

**Injected by space-cli:**

```bash
# Provider info
SPACE_PROVIDER=orbstack|docker

# Service URLs (OrbStack)
SPACE_URL_WEB=http://myapp-main-abc123-web.orb.local
SPACE_URL_API=http://myapp-main-abc123-api.orb.local

# Service ports (Docker Desktop)
SPACE_PORT_WEB=13001
SPACE_PORT_API=13002

# Database DSN
SPACE_DB_PROPELDB_DSN=postgres://admin@postgres:5432/propeldb

# Project info
SPACE_PROJECT_NAME=myapp-main-abc123
SPACE_WORK_DIR=/path/to/project
```

### C. Performance Benchmarks

**Target Performance (vs propel-cli):**

| Operation | Current | Target | Status |
|-----------|---------|--------|--------|
| `space up` (cold) | 5s | ≤5s | ✓ |
| `space up` (warm) | 2s | ≤2s | ✓ |
| `space down` | 3s | ≤3s | ✓ |
| `space status` | 0.5s | ≤0.5s | ✓ |
| Config load | 50ms | ≤50ms | ✓ |
| Port allocation | 100ms | ≤100ms | ✓ |

---

**End of Implementation Plan**

*Last updated: 2026-01-20*
*Version: 1.0.0*
*Status: Draft*
