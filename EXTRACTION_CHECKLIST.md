# Extraction Checklist

## Package: Provider Detection

### Files to Extract
- [ ] `internal/sandbox/provider.go` → `pkg/provider/provider.go`
- [ ] `internal/sandbox/provider_test.go` → `pkg/provider/provider_test.go`

### Changes Needed
```diff
// Remove hardcoded constants
- const OrbStackDNSSuffix = ".orb.local"
+ // Get DNS suffix from config
+ func (p ProviderType) GetDNSSuffix(config *Config) string
```

### Dependencies
- None (self-contained)

---

## Package: Port Management

### Files to Extract
- [ ] `internal/sandbox/ports.go` → `pkg/ports/allocator.go`
- [ ] `internal/sandbox/ports_test.go` → `pkg/ports/allocator_test.go`
- [ ] `internal/utils/port_mapping.go` → `pkg/ports/mapper.go`
- [ ] `internal/utils/port_mapping_test.go` → `pkg/ports/mapper_test.go`

### Changes Needed
```diff
// Make port ranges configurable
- const DefaultPortRangeStart = 10000
- const DefaultPortRangeEnd = 60000
+ type Config struct {
+     RangeStart int
+     RangeEnd   int
+ }

// Remove hardcoded service ports
- func (pc *PortConfig) GetAPIServerPort() int { return 6060 }
- func (pc *PortConfig) GetAppPort() int { return 3000 }
+ func (pc *PortConfig) GetServicePort(serviceName string) (int, error)

// Make persistence file configurable
- const PortsFileName = ".sandbox-ports.json"
+ persistenceFile := config.Ports.PersistenceFile
```

### Dependencies
- `pkg/config` (for configuration)

---

## Package: Project Naming

### Files to Extract
- [ ] `internal/sandbox/project.go` → `pkg/project/namer.go`
- [ ] `internal/sandbox/project_test.go` → `pkg/project/namer_test.go`
- [ ] `internal/utils/project.go` → `pkg/project/git.go`

### Changes Needed
```diff
// Make project prefix configurable
- const ProjectPrefix = "propel-gtm"
+ prefix := config.Project.Prefix

// Support multiple naming strategies
- // Always use git branch
+ type NamingStrategy interface {
+     DeriveProjectName(workDir string) (string, error)
+ }
+
+ type GitBranchStrategy struct{}
+ type DirectoryStrategy struct{}
+ type StaticStrategy struct{ Name string }
```

### Dependencies
- `pkg/config` (for configuration)

---

## Package: Docker Compose Client

### Files to Extract
- [ ] `internal/utils/docker.go` → `pkg/compose/client.go`
- [ ] `internal/utils/docker_client.go` → `pkg/compose/interface.go`

### Files to Create
- [ ] `pkg/compose/parser.go` - Parse docker-compose.yml
- [ ] `pkg/compose/modifier.go` - Modify compose for providers
- [ ] `pkg/compose/executor.go` - Execute compose commands

### Changes Needed
```diff
// Remove Propel-specific service knowledge
- func (d *DockerUtils) StartAPIServer(ctx context.Context) error
- func (d *DockerUtils) StartApp(ctx context.Context) error
+ func (d *ComposeClient) StartService(ctx context.Context, serviceName string) error

// Make compose file configurable
- composeFile := "docker-compose.yml"
+ composeFile := config.Project.ComposeFiles[0]

// Generic service operations
+ func (d *ComposeClient) ListServices(ctx context.Context) ([]string, error)
+ func (d *ComposeClient) GetServiceStatus(ctx context.Context, service string) (ServiceStatus, error)
```

### Dependencies
- `pkg/config` (for configuration)
- `pkg/provider` (for provider-aware modifications)

---

## Package: Compose Parser

### Files to Create
- [ ] `pkg/compose/parser.go`

### Features
```go
type ComposeParser struct{}

// ParseFile parses a docker-compose.yml file
func (p *ComposeParser) ParseFile(path string) (*ComposeFile, error)

// ComposeFile represents a parsed compose file
type ComposeFile struct {
    Version  string
    Services map[string]Service
    Networks map[string]Network
    Volumes  map[string]Volume
}

// Service represents a service in compose file
type Service struct {
    Name        string
    Image       string
    Build       *BuildConfig
    Ports       []PortMapping
    Environment map[string]string
    DependsOn   []string
    Volumes     []string
}

// DiscoverServices extracts service names and ports
func (p *ComposeParser) DiscoverServices(file *ComposeFile) []ServiceInfo
```

### Dependencies
- `gopkg.in/yaml.v3`

---

## Package: Compose Modifier

### Files to Create
- [ ] `pkg/compose/modifier.go`

### Features
```go
type ComposeModifier struct {
    provider provider.ProviderType
}

// ModifyForProvider modifies compose file for specific provider
func (m *ComposeModifier) ModifyForProvider(file *ComposeFile) (*ComposeFile, error)

// For OrbStack: remove port bindings
// For Docker: keep port bindings, maybe add port env vars
```

### Dependencies
- `pkg/provider`

---

## Package: Sandbox

### Files to Extract
- [ ] `internal/sandbox/sandbox.go` → `pkg/sandbox/sandbox.go`
- [ ] `internal/sandbox/commands.go` → `pkg/sandbox/commands.go`

### Files to Create
- [ ] `pkg/sandbox/services.go` - Service lifecycle
- [ ] `pkg/sandbox/urls.go` - URL generation
- [ ] `pkg/sandbox/logs.go` - Log streaming
- [ ] `pkg/sandbox/shell.go` - Shell access

### Changes Needed
```diff
// Accept config instead of hardcoded values
- const DefaultShellService = "api-server"
+ defaultShellService := config.Services.GetDefaultShell()

// Remove database operations (move to separate package)
- func (s *Sandbox) CreateRiverDatabase() error
- func (s *Sandbox) RunMigrations() error
+ // Use pkg/database instead

// Generic service operations
+ func (s *Sandbox) StartService(ctx context.Context, name string) error
+ func (s *Sandbox) GetServiceURL(name string) (string, error)
+ func (s *Sandbox) GetServiceStatus(ctx context.Context, name string) (ServiceStatus, error)
```

### Dependencies
- `pkg/config`
- `pkg/provider`
- `pkg/ports`
- `pkg/compose`
- `pkg/project`

---

## Package: Database

### Files to Create
- [ ] `pkg/database/interface.go`
- [ ] `pkg/database/postgres.go`
- [ ] `pkg/database/mysql.go`
- [ ] `pkg/database/mongodb.go`
- [ ] `pkg/database/operations.go`

### Features
```go
// DatabaseClient is the interface for database operations
type DatabaseClient interface {
    // Connect to the database
    Connect(ctx context.Context, dsn string) error

    // CreateDatabase creates a database if it doesn't exist
    CreateDatabase(ctx context.Context, name string) error

    // DropDatabase drops a database
    DropDatabase(ctx context.Context, name string) error

    // ListDatabases lists all databases
    ListDatabases(ctx context.Context) ([]string, error)

    // OpenShell opens an interactive shell
    OpenShell(ctx context.Context, dbName string) error

    // ExecuteFile executes SQL from a file
    ExecuteFile(ctx context.Context, dbName, filePath string) error
}

// PostgresClient implements DatabaseClient for PostgreSQL
type PostgresClient struct{}

// MySQLClient implements DatabaseClient for MySQL
type MySQLClient struct{}

// MongoDBClient implements DatabaseClient for MongoDB
type MongoDBClient struct{}
```

### Changes from Propel
```diff
// Extract from internal/utils/database.go
- Propel-specific River database logic
+ Generic database operations

// Make configurable
- hardcoded "propeldb", "river"
+ config.Databases[i].Name
```

### Dependencies
- `pkg/config`
- Database drivers (lib/pq, go-sql-driver/mysql, mongo-driver)

---

## Files to Keep in propel-cli

### Propel-Specific
- [ ] `internal/propel/constants.go` - Propel service names
- [ ] `internal/propel/config.go` - Default Propel config
- [ ] `internal/propel/database.go` - River-specific logic
- [ ] `internal/propel/seeder.go` - Propel seeder integration

### VM Management (stays in propel-cli)
- [ ] `internal/vm/*` - All VM-related code
- [ ] `internal/commands/vm.go`

### Utilities (stays in propel-cli)
- [ ] `internal/commands/util.go` - PR copy utility
- [ ] `internal/prcopy/*`

### Shared Utilities (might extract some)
- [ ] `internal/utils/logger.go` → Consider extracting
- [ ] `internal/utils/terminal.go` → Consider extracting
- [ ] `internal/utils/exec.go` → Consider extracting
- [ ] `internal/utils/file_path.go` → Keep in propel-cli

---

## Dependencies to Add to compose-dev-cli

```go
// go.mod for compose-dev-cli
module github.com/yourorg/compose-dev-cli

go 1.25

require (
    github.com/spf13/cobra v1.8.0
    go.uber.org/zap v1.27.0
    gopkg.in/yaml.v3 v3.0.1
    github.com/lib/pq v1.10.9              // PostgreSQL driver
    github.com/go-sql-driver/mysql v1.8.1  // MySQL driver (optional)
)
```

---

## Testing Strategy

### Unit Tests
- [ ] Test config loading and merging
- [ ] Test service discovery from compose file
- [ ] Test port allocation with different strategies
- [ ] Test provider detection
- [ ] Test project naming with different strategies
- [ ] Test database operations (mocked)

### Integration Tests
- [ ] Test with sample compose files
  - [ ] Simple app (single service)
  - [ ] App with database
  - [ ] Multi-service app
  - [ ] App with multiple databases
- [ ] Test OrbStack integration (if available)
- [ ] Test Docker Desktop integration
- [ ] Test port persistence across restarts

### Test Fixtures
Create test compose files:
- [ ] `testdata/simple/docker-compose.yml`
- [ ] `testdata/with-db/docker-compose.yml`
- [ ] `testdata/multi-service/docker-compose.yml`

---

## Documentation Checklist

- [ ] README.md with quick start
- [ ] CONFIGURATION.md - Full config reference
- [ ] PROVIDERS.md - Provider-specific guides
- [ ] MIGRATION.md - Migration guide from propel-cli
- [ ] EXAMPLES.md - Example configurations
- [ ] CONTRIBUTING.md - Contribution guidelines
- [ ] API.md - Go package documentation

---

## Pre-Launch Checklist

### Code Quality
- [ ] All packages have >80% test coverage
- [ ] All exported functions have godoc comments
- [ ] No hardcoded values (everything configurable)
- [ ] Error messages are clear and actionable
- [ ] Logging is consistent and useful

### Functionality
- [ ] Works with non-Propel compose files
- [ ] OrbStack detection and DNS work correctly
- [ ] Docker Desktop port mapping works correctly
- [ ] Port allocation and persistence works
- [ ] Service URLs are generated correctly
- [ ] Database operations work for all supported DBs

### User Experience
- [ ] Commands have helpful descriptions
- [ ] Error messages suggest solutions
- [ ] Progress indicators for long operations
- [ ] Config validation gives clear feedback
- [ ] Examples work out of the box

### Integration
- [ ] propel-cli successfully uses compose-dev-cli
- [ ] All propel-cli sandbox commands still work
- [ ] Backward compatibility maintained
- [ ] Performance is equal or better

---

## Migration Execution Order

1. **Week 1**
   - [ ] Create compose-dev-cli repo
   - [ ] Implement config system (schema, loader, validator)
   - [ ] Extract provider detection
   - [ ] Extract project naming
   - [ ] Write unit tests

2. **Week 2**
   - [ ] Extract port management
   - [ ] Extract compose client
   - [ ] Create compose parser
   - [ ] Create compose modifier
   - [ ] Write integration tests

3. **Week 3**
   - [ ] Extract sandbox core
   - [ ] Create database abstraction
   - [ ] Build CLI commands
   - [ ] Write documentation
   - [ ] Test with sample projects

4. **Week 4**
   - [ ] Update propel-cli to use compose-dev-cli
   - [ ] Refactor propel commands
   - [ ] Run full test suite
   - [ ] Fix any issues
   - [ ] Release v1.0.0

---

## Success Metrics

- [ ] compose-dev-cli works with at least 3 different project types (Rails, Node, Go)
- [ ] All existing propel-cli tests pass
- [ ] Test coverage >80% in compose-dev-cli
- [ ] Performance within 10% of current implementation
- [ ] Zero breaking changes for propel-cli users
- [ ] Documentation complete and reviewed
- [ ] At least 2 team members have successfully used it
