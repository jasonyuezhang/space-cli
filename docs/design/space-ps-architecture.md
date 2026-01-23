# Architecture Design: `space ps` Command

## Overview

The `space ps` command provides a comprehensive view of running containers managed by space-cli, with intelligent filtering, formatting options, and DNS integration.

## Command Structure

### Command Definition
```go
space ps [OPTIONS]
```

### Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--format` | `-f` | string | `table` | Output format: `table`, `json`, `yaml`, `compact` |
| `--services` | `-s` | []string | all | Filter by service names (comma-separated) |
| `--all` | `-a` | bool | `false` | Show all containers (including stopped) |
| `--project` | `-p` | string | auto | Filter by project name |
| `--no-trunc` | | bool | `false` | Don't truncate output |
| `--quiet` | `-q` | bool | `false` | Only display container IDs |

## Architecture Components

### 1. Container Discovery

```go
type ContainerFilter struct {
    ProjectName  string   // From .space.yaml or compose files
    ServiceNames []string // Service filter
    ShowAll      bool     // Include stopped containers
}

type ContainerInfo struct {
    ID          string
    Name        string
    ServiceName string
    ProjectName string
    Status      string
    State       string // running, exited, paused, etc.
    Health      string // healthy, unhealthy, starting, none
    Ports       []PortMapping
    Networks    []NetworkInfo
    Created     time.Time
    Uptime      time.Duration
}

type PortMapping struct {
    ContainerPort int
    HostPort      int
    Protocol      string // tcp, udp
    HostIP        string
}

type NetworkInfo struct {
    Name      string
    IPAddress string
    Gateway   string
}
```

### 2. Docker Integration

```go
// ContainerDiscoverer handles Docker container discovery
type ContainerDiscoverer struct {
    dockerClient *client.Client
    projectName  string
}

func (d *ContainerDiscoverer) ListContainers(filter ContainerFilter) ([]ContainerInfo, error) {
    // 1. Build Docker API filter
    dockerFilter := filters.NewArgs()

    // Filter by project label (docker-compose sets this)
    if filter.ProjectName != "" {
        dockerFilter.Add("label", fmt.Sprintf("com.docker.compose.project=%s", filter.ProjectName))
    }

    // Filter by service names
    for _, service := range filter.ServiceNames {
        dockerFilter.Add("label", fmt.Sprintf("com.docker.compose.service=%s", service))
    }

    // Include all states or just running
    if !filter.ShowAll {
        dockerFilter.Add("status", "running")
    }

    // 2. Query Docker API
    containers, err := d.dockerClient.ContainerList(context.Background(), container.ListOptions{
        All:     filter.ShowAll,
        Filters: dockerFilter,
    })

    // 3. Transform to ContainerInfo
    return d.transformContainers(containers)
}

func (d *ContainerDiscoverer) transformContainers(containers []types.Container) ([]ContainerInfo, error) {
    var result []ContainerInfo

    for _, c := range containers {
        info := ContainerInfo{
            ID:          c.ID[:12], // Short ID
            Name:        strings.TrimPrefix(c.Names[0], "/"),
            ServiceName: c.Labels["com.docker.compose.service"],
            ProjectName: c.Labels["com.docker.compose.project"],
            Status:      c.Status,
            State:       c.State,
            Created:     time.Unix(c.Created, 0),
        }

        // Parse health status
        if c.State == "running" {
            info.Health = d.getHealthStatus(c)
        }

        // Parse port mappings
        info.Ports = d.parsePorts(c.Ports)

        // Parse network info
        info.Networks = d.parseNetworks(c.NetworkSettings)

        // Calculate uptime
        if c.State == "running" {
            info.Uptime = time.Since(time.Unix(c.Created, 0))
        }

        result = append(result, info)
    }

    return result, nil
}
```

### 3. DNS Integration

```go
type DNSResolver struct {
    domain    string // e.g., "space.local"
    stateFile string
}

// EnrichWithDNS adds DNS information to container data
func (r *DNSResolver) EnrichWithDNS(containers []ContainerInfo) []ContainerInfo {
    // Check if DNS daemon is running
    state, err := loadDNSState()
    if err != nil {
        return containers // No DNS available
    }

    for i := range containers {
        if containers[i].State == "running" {
            // Add DNS hostname
            containers[i].DNSName = fmt.Sprintf("%s.%s", containers[i].ServiceName, r.domain)
            containers[i].DNSAvailable = true
        }
    }

    return containers
}

// Updated ContainerInfo with DNS fields
type ContainerInfo struct {
    // ... existing fields ...
    DNSName      string // e.g., "postgres.space.local"
    DNSAvailable bool   // Whether DNS mode is active
}
```

### 4. Output Formatting

```go
type OutputFormatter interface {
    Format(containers []ContainerInfo) (string, error)
}

// TableFormatter - default human-readable table
type TableFormatter struct {
    noTrunc bool
}

func (f *TableFormatter) Format(containers []ContainerInfo) (string, error) {
    if len(containers) == 0 {
        return "No containers found.\n", nil
    }

    // Build table with tabwriter
    var buf bytes.Buffer
    w := tabwriter.NewWriter(&buf, 0, 0, 2, ' ', 0)

    // Header
    fmt.Fprintln(w, "SERVICE\tSTATUS\tHEALTH\tPORTS\tACCESS\tUPTIME")

    // Rows
    for _, c := range containers {
        serviceName := f.truncate(c.ServiceName, 20)
        status := f.formatStatus(c.State, c.Status)
        health := f.formatHealth(c.Health)
        ports := f.formatPorts(c.Ports)
        access := f.formatAccess(c)
        uptime := f.formatUptime(c.Uptime)

        fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
            serviceName, status, health, ports, access, uptime)
    }

    w.Flush()
    return buf.String(), nil
}

func (f *TableFormatter) formatAccess(c ContainerInfo) string {
    if c.DNSAvailable {
        // DNS mode - show .space.local domain
        return fmt.Sprintf("http://%s", c.DNSName)
    }

    // Port binding mode - show localhost with port
    if len(c.Ports) > 0 {
        port := c.Ports[0]
        if port.HostPort > 0 {
            return fmt.Sprintf("localhost:%d", port.HostPort)
        }
    }

    return "-"
}

func (f *TableFormatter) formatStatus(state, status string) string {
    switch state {
    case "running":
        return "🟢 Running"
    case "exited":
        return "🔴 Exited"
    case "paused":
        return "🟡 Paused"
    case "restarting":
        return "🔄 Restarting"
    default:
        return state
    }
}

func (f *TableFormatter) formatHealth(health string) string {
    switch health {
    case "healthy":
        return "✅"
    case "unhealthy":
        return "❌"
    case "starting":
        return "⏳"
    default:
        return "-"
    }
}

// CompactFormatter - one-line per container
type CompactFormatter struct{}

func (f *CompactFormatter) Format(containers []ContainerInfo) (string, error) {
    var buf bytes.Buffer
    for _, c := range containers {
        fmt.Fprintf(&buf, "%s: %s (%s)\n", c.ServiceName, c.State, c.ID)
    }
    return buf.String(), nil
}

// JSONFormatter - JSON output
type JSONFormatter struct {
    pretty bool
}

func (f *JSONFormatter) Format(containers []ContainerInfo) (string, error) {
    var data []byte
    var err error

    if f.pretty {
        data, err = json.MarshalIndent(containers, "", "  ")
    } else {
        data, err = json.Marshal(containers)
    }

    if err != nil {
        return "", err
    }

    return string(data) + "\n", nil
}

// YAMLFormatter - YAML output
type YAMLFormatter struct{}

func (f *YAMLFormatter) Format(containers []ContainerInfo) (string, error) {
    data, err := yaml.Marshal(containers)
    if err != nil {
        return "", err
    }
    return string(data), nil
}
```

### 5. Project Name Resolution

```go
// ProjectResolver determines the project name from context
type ProjectResolver struct {
    workDir string
    config  *config.Config
}

func (r *ProjectResolver) ResolveProjectName() (string, error) {
    // 1. Try explicit --project flag (handled by cobra)

    // 2. Load .space.yaml config
    loader, err := config.NewLoader(r.workDir)
    if err == nil {
        cfg, err := loader.Load()
        if err == nil && cfg.Project.Name != "" {
            return r.generateProjectName(cfg), nil
        }
    }

    // 3. Try to detect from docker-compose labels
    // Look for containers with com.docker.compose.project label in current directory

    // 4. Fallback to directory name
    return filepath.Base(r.workDir), nil
}

func (r *ProjectResolver) generateProjectName(cfg *config.Config) string {
    // Use same logic as `space up`
    baseName := cfg.Project.Name
    if baseName == "" {
        baseName = filepath.Base(r.workDir)
    }

    prefix := cfg.Project.Prefix
    if prefix == "" {
        prefix = baseName + "-"
    }

    switch cfg.Project.NamingStrategy {
    case "git-branch":
        branch := getGitBranch(r.workDir)
        if branch != "" {
            branch = strings.ReplaceAll(branch, "/", "-")
            return prefix + branch
        }
        fallthrough
    case "directory":
        return prefix + filepath.Base(r.workDir)
    default:
        return baseName
    }
}
```

## Implementation Plan

### Phase 1: Core Functionality
1. Implement `ContainerDiscoverer` with Docker API integration
2. Add basic table formatter
3. Support `--services` and `--all` flags
4. Integrate with project name resolution from `space up`

### Phase 2: DNS Integration
1. Add DNS state detection
2. Enhance `ContainerInfo` with DNS fields
3. Update table formatter to show DNS addresses
4. Add visual indicators for DNS mode

### Phase 3: Advanced Formatting
1. Implement JSON formatter
2. Implement YAML formatter
3. Implement compact formatter
4. Add `--quiet` mode

### Phase 4: Enhanced Features
1. Add health check status display
2. Show network information
3. Add uptime calculation
4. Support `--no-trunc` for full IDs/names

## Example Output

### Table Format (Default)
```
SERVICE    STATUS       HEALTH  PORTS       ACCESS                          UPTIME
postgres   🟢 Running   ✅      5432/tcp    http://postgres.space.local     2h 15m
redis      🟢 Running   -       6379/tcp    http://redis.space.local        2h 15m
api        🟢 Running   ✅      3000/tcp    http://api.space.local:3000     2h 14m
worker     🟢 Running   ⏳      -           -                               1h 5m
```

### Table Format (No DNS)
```
SERVICE    STATUS       HEALTH  PORTS       ACCESS           UPTIME
postgres   🟢 Running   ✅      5432/tcp    localhost:5432   2h 15m
redis      🟢 Running   -       6379/tcp    localhost:6379   2h 15m
api        🟢 Running   ✅      3000/tcp    localhost:3000   2h 14m
```

### Compact Format
```
postgres: running (95871d3cf1d8)
redis: running (7a3c368cda05)
api: running (0750943c7911)
```

### JSON Format
```json
[
  {
    "id": "95871d3cf1d8",
    "name": "space-main-postgres-1",
    "service_name": "postgres",
    "project_name": "space-main",
    "status": "Up 2 hours",
    "state": "running",
    "health": "healthy",
    "dns_name": "postgres.space.local",
    "dns_available": true,
    "ports": [
      {
        "container_port": 5432,
        "host_port": 0,
        "protocol": "tcp"
      }
    ],
    "uptime": "2h15m30s"
  }
]
```

## Error Handling

### No Containers Found
```
No containers found for project 'space-main'.

💡 Tip: Run 'space up' to start services
💡 Tip: Use '--all' to show stopped containers
```

### Docker Not Available
```
❌ Failed to connect to Docker

Please ensure Docker is running and try again.
```

### Invalid Format
```
❌ Invalid format 'xml'

Supported formats: table, json, yaml, compact
```

## Dependencies

### External Libraries
- `github.com/docker/docker/client` - Docker API client
- `github.com/docker/docker/api/types` - Docker types
- `github.com/docker/docker/api/types/filters` - Docker filters
- `text/tabwriter` - Table formatting
- `gopkg.in/yaml.v3` - YAML formatting

### Internal Packages
- `pkg/config` - Configuration loading
- `internal/dns` - DNS state management
- `internal/provider` - Provider detection

## Testing Strategy

### Unit Tests
1. `ContainerDiscoverer.ListContainers` - Mock Docker client
2. `TableFormatter.Format` - Table output generation
3. `ProjectResolver.ResolveProjectName` - Project name logic
4. Port parsing and formatting
5. DNS enrichment logic

### Integration Tests
1. Full command execution with test containers
2. DNS mode vs port binding mode
3. Service filtering
4. Multiple output formats

### Test Fixtures
```go
var testContainers = []types.Container{
    {
        ID:    "95871d3cf1d8abc123",
        Names: []string{"/space-main-postgres-1"},
        State: "running",
        Status: "Up 2 hours (healthy)",
        Labels: map[string]string{
            "com.docker.compose.project": "space-main",
            "com.docker.compose.service": "postgres",
        },
        Ports: []types.Port{
            {PrivatePort: 5432, Type: "tcp"},
        },
    },
}
```

## Performance Considerations

1. **Docker API Calls**: Single API call to list all containers with filters
2. **Caching**: No caching needed - real-time status is required
3. **Sorting**: Sort by service name for consistent output
4. **Pagination**: Not needed - typical projects have <50 services

## Future Enhancements

1. **Watch Mode**: `space ps --watch` for real-time updates
2. **Resource Usage**: Add CPU/Memory usage columns
3. **Logs Preview**: Show last log line per service
4. **Color Themes**: Customizable color schemes
5. **Export**: Save output to file
6. **Filter by State**: `--state running,exited`
7. **Tree View**: Show container dependencies

## Security Considerations

1. **Docker Socket Access**: Requires read access to Docker socket
2. **Label Filtering**: Only show containers with compose labels to avoid leaking info about other containers
3. **Project Isolation**: Automatic filtering by project prevents cross-project visibility
4. **Credential Hiding**: Never show environment variables or secrets in output
