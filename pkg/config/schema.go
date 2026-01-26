package config

import "time"

// Config represents the complete configuration for space-cli
type Config struct {
	// Project configuration
	Project ProjectConfig `yaml:"project" json:"project"`

	// Services configuration
	Services map[string]ServiceConfig `yaml:"services" json:"services"`

	// Databases configuration
	Databases []DatabaseConfig `yaml:"databases,omitempty" json:"databases,omitempty"`

	// Commands configuration
	Commands CommandsConfig `yaml:"commands,omitempty" json:"commands,omitempty"`

	// Provider configuration (Docker providers)
	Provider ProviderConfig `yaml:"provider,omitempty" json:"provider,omitempty"`

	// VM configuration
	VM VMConfig `yaml:"vm,omitempty" json:"vm,omitempty"`

	// Networking configuration
	Network NetworkConfig `yaml:"network,omitempty" json:"network,omitempty"`

	// Ports configuration
	Ports PortsConfig `yaml:"ports,omitempty" json:"ports,omitempty"`

	// Hooks configuration
	Hooks HooksConfig `yaml:"hooks,omitempty" json:"hooks,omitempty"`
}

// ProjectConfig defines project-level settings
type ProjectConfig struct {
	// Name is the base project name (auto-detected from repo if not set)
	Name string `yaml:"name,omitempty" json:"name,omitempty"`

	// Prefix for Docker resources (images, containers, networks)
	// Default: project name
	Prefix string `yaml:"prefix,omitempty" json:"prefix,omitempty"`

	// NamingStrategy defines how to generate project names
	// Options: "git-branch" (default), "directory", "static"
	NamingStrategy string `yaml:"naming_strategy,omitempty" json:"naming_strategy,omitempty"`

	// ComposeFiles to use (default: ["docker-compose.yml"])
	ComposeFiles []string `yaml:"compose_files,omitempty" json:"compose_files,omitempty"`

	// WorkDir override (default: current directory)
	WorkDir string `yaml:"work_dir,omitempty" json:"work_dir,omitempty"`
}

// ServiceConfig defines configuration for a specific service
type ServiceConfig struct {
	// Port is the internal port the service listens on
	Port int `yaml:"port,omitempty" json:"port,omitempty"`

	// ExternalPort is the host port to map to (for Docker Desktop)
	// If not set, will be auto-allocated
	ExternalPort int `yaml:"external_port,omitempty" json:"external_port,omitempty"`

	// Shell to use for "shell" command (default: "sh")
	Shell string `yaml:"shell,omitempty" json:"shell,omitempty"`

	// URL template for generating service URLs
	// Variables: {host}, {port}, {service}, {project}
	// Default: "http://{host}:{port}"
	URLTemplate string `yaml:"url_template,omitempty" json:"url_template,omitempty"`

	// HealthCheck configuration
	HealthCheck *HealthCheckConfig `yaml:"health_check,omitempty" json:"health_check,omitempty"`

	// Environment variables to inject
	Environment map[string]string `yaml:"environment,omitempty" json:"environment,omitempty"`

	// Dependencies that must be running before this service
	DependsOn []string `yaml:"depends_on,omitempty" json:"depends_on,omitempty"`
}

// HealthCheckConfig defines health check settings
type HealthCheckConfig struct {
	// Enabled enables health checking
	Enabled bool `yaml:"enabled" json:"enabled"`

	// Endpoint to check (e.g., "/health", "/api/health")
	Endpoint string `yaml:"endpoint,omitempty" json:"endpoint,omitempty"`

	// Timeout for health check
	Timeout time.Duration `yaml:"timeout,omitempty" json:"timeout,omitempty"`

	// Interval between checks
	Interval time.Duration `yaml:"interval,omitempty" json:"interval,omitempty"`

	// Retries before marking unhealthy
	Retries int `yaml:"retries,omitempty" json:"retries,omitempty"`
}

// DatabaseConfig defines database-specific configuration
type DatabaseConfig struct {
	// Name of the database
	Name string `yaml:"name" json:"name"`

	// Service that provides this database
	Service string `yaml:"service" json:"service"`

	// Type of database: "postgres", "mysql", "mongodb", etc.
	Type string `yaml:"type,omitempty" json:"type,omitempty"`

	// User for database access
	User string `yaml:"user,omitempty" json:"user,omitempty"`

	// Password for database access (prefer environment variable)
	Password string `yaml:"password,omitempty" json:"password,omitempty"`

	// Host override (default: auto-detected based on provider)
	Host string `yaml:"host,omitempty" json:"host,omitempty"`

	// Port override (default: from service config)
	Port int `yaml:"port,omitempty" json:"port,omitempty"`

	// AutoCreate creates the database if it doesn't exist
	AutoCreate bool `yaml:"auto_create,omitempty" json:"auto_create,omitempty"`

	// MigrationsPath is the path to migrations
	// Can be a directory or a Go file to run
	MigrationsPath string `yaml:"migrations_path,omitempty" json:"migrations_path,omitempty"`

	// MigrationsCommand is the command to run migrations
	// Variables: {path}, {db_name}, {db_user}, {db_host}, {db_port}
	MigrationsCommand string `yaml:"migrations_command,omitempty" json:"migrations_command,omitempty"`

	// SeedCommand is the command to seed the database
	// Variables: {db_name}, {db_user}, {db_host}, {db_port}
	SeedCommand string `yaml:"seed_command,omitempty" json:"seed_command,omitempty"`
}

// CommandsConfig defines custom commands
type CommandsConfig struct {
	// Seed command template
	// Variables: {db_name}, {db_user}, {db_host}, {db_port}
	Seed string `yaml:"seed,omitempty" json:"seed,omitempty"`

	// Migrate command template
	// Variables: {migrations_path}, {db_name}, {db_user}, {db_host}, {db_port}
	Migrate string `yaml:"migrate,omitempty" json:"migrate,omitempty"`

	// Custom commands (key: command name, value: command template)
	Custom map[string]string `yaml:"custom,omitempty" json:"custom,omitempty"`
}

// ProviderConfig defines provider-specific settings
type ProviderConfig struct {
	// Type forces a specific provider: "auto", "orbstack", "docker"
	// Default: "auto" (auto-detect)
	Type string `yaml:"type,omitempty" json:"type,omitempty"`

	// OrbStack-specific configuration
	OrbStack *OrbStackConfig `yaml:"orbstack,omitempty" json:"orbstack,omitempty"`

	// Docker-specific configuration
	Docker *DockerConfig `yaml:"docker,omitempty" json:"docker,omitempty"`
}

// OrbStackConfig defines OrbStack-specific settings
type OrbStackConfig struct {
	// DNSSuffix for OrbStack containers (default: ".orb.local")
	DNSSuffix string `yaml:"dns_suffix,omitempty" json:"dns_suffix,omitempty"`

	// UseContainerDNS forces using container DNS even if not detected
	UseContainerDNS bool `yaml:"use_container_dns,omitempty" json:"use_container_dns,omitempty"`

	// RemovePortBindings removes port bindings from compose file
	// Default: true for OrbStack
	RemovePortBindings bool `yaml:"remove_port_bindings,omitempty" json:"remove_port_bindings,omitempty"`
}

// DockerConfig defines Docker Desktop specific settings
type DockerConfig struct {
	// Context to use (default: current context)
	Context string `yaml:"context,omitempty" json:"context,omitempty"`

	// ComposeCommand to use: "docker compose" or "docker-compose"
	// Default: auto-detect
	ComposeCommand string `yaml:"compose_command,omitempty" json:"compose_command,omitempty"`
}

// NetworkConfig defines networking settings
type NetworkConfig struct {
	// AllowedHosts for CORS (for Vite, webpack-dev-server, etc.)
	AllowedHosts string `yaml:"allowed_hosts,omitempty" json:"allowed_hosts,omitempty"`

	// NetworkMode: "bridge", "host", etc.
	NetworkMode string `yaml:"network_mode,omitempty" json:"network_mode,omitempty"`

	// CustomDomain for accessing services
	CustomDomain string `yaml:"custom_domain,omitempty" json:"custom_domain,omitempty"`

	// DNSHashing enables directory-based hashing for DNS names to prevent collisions
	// Default: true (enabled)
	DNSHashing bool `yaml:"dns_hashing,omitempty" json:"dns_hashing,omitempty"`
}

// PortsConfig defines port allocation settings
type PortsConfig struct {
	// RangeStart is the start of the dynamic port range
	// Default: 10000
	RangeStart int `yaml:"range_start,omitempty" json:"range_start,omitempty"`

	// RangeEnd is the end of the dynamic port range
	// Default: 60000
	RangeEnd int `yaml:"range_end,omitempty" json:"range_end,omitempty"`

	// PersistenceFile is where to save port allocations
	// Default: ".compose-cli-ports.json"
	PersistenceFile string `yaml:"persistence_file,omitempty" json:"persistence_file,omitempty"`

	// Strategy for port allocation: "sequential", "random"
	// Default: "sequential"
	Strategy string `yaml:"strategy,omitempty" json:"strategy,omitempty"`
}

// HooksConfig defines hooks configuration
type HooksConfig struct {
	// Vite-specific hooks for frontend development
	Vite *ViteHooksConfig `yaml:"vite,omitempty" json:"vite,omitempty"`

	// Database-specific hooks for database setup
	Database *DatabaseHooksConfig `yaml:"database,omitempty" json:"database,omitempty"`

	// Custom hooks for arbitrary commands
	Custom []CustomHookConfig `yaml:"custom,omitempty" json:"custom,omitempty"`
}

// DatabaseHooksConfig defines database-specific hook settings
type DatabaseHooksConfig struct {
	// River queue database configuration
	River *RiverHooksConfig `yaml:"river,omitempty" json:"river,omitempty"`
}

// RiverHooksConfig defines River queue database hook settings
type RiverHooksConfig struct {
	// Enabled enables River database setup
	Enabled bool `yaml:"enabled" json:"enabled"`

	// PostgresService is the name of the postgres service (default: "postgres")
	PostgresService string `yaml:"postgres_service,omitempty" json:"postgres_service,omitempty"`

	// DatabaseName is the River database name (default: "river")
	DatabaseName string `yaml:"database_name,omitempty" json:"database_name,omitempty"`

	// Username for postgres connection (default: "admin")
	Username string `yaml:"username,omitempty" json:"username,omitempty"`

	// Password for postgres connection (default: "test")
	Password string `yaml:"password,omitempty" json:"password,omitempty"`

	// Port for postgres (default: 5432)
	Port int `yaml:"port,omitempty" json:"port,omitempty"`
}

// ViteHooksConfig defines Vite-specific hook settings
type ViteHooksConfig struct {
	// Enabled enables Vite hooks
	Enabled bool `yaml:"enabled" json:"enabled"`

	// AutoDetect automatically detects Vite projects and configures hooks
	AutoDetect bool `yaml:"auto_detect,omitempty" json:"auto_detect,omitempty"`

	// EnvVars maps environment variable names to service references
	// Example: VITE_API_BASE_URL -> "api-server:6060"
	EnvVars map[string]string `yaml:"env_vars,omitempty" json:"env_vars,omitempty"`

	// AllowedHostsPattern for Vite's server.host configuration
	// Default: derived from network.allowed_hosts
	AllowedHostsPattern string `yaml:"allowed_hosts_pattern,omitempty" json:"allowed_hosts_pattern,omitempty"`
}

// CustomHookConfig defines a custom hook configuration
type CustomHookConfig struct {
	// Name is the unique identifier for the hook
	Name string `yaml:"name" json:"name"`

	// Events that trigger this hook
	// Supported: "pre-up", "post-up", "pre-down", "post-down", "on-dns-ready"
	Events []string `yaml:"events" json:"events"`

	// Command to execute when the hook is triggered
	Command string `yaml:"command" json:"command"`

	// Environment variables to set when running the command
	Environment map[string]string `yaml:"environment,omitempty" json:"environment,omitempty"`

	// WorkDir overrides the working directory for the command
	WorkDir string `yaml:"work_dir,omitempty" json:"work_dir,omitempty"`

	// Timeout for command execution (default: 30s)
	Timeout time.Duration `yaml:"timeout,omitempty" json:"timeout,omitempty"`

	// ContinueOnError if true, doesn't fail the operation if hook fails
	ContinueOnError bool `yaml:"continue_on_error,omitempty" json:"continue_on_error,omitempty"`
}

// VMConfig defines VM configuration
type VMConfig struct {
	// Enabled enables VM-based development (default: false, uses Docker)
	Enabled bool `yaml:"enabled,omitempty" json:"enabled,omitempty"`

	// Provider: "auto" (default), "lima", "orbstack"
	Provider string `yaml:"provider,omitempty" json:"provider,omitempty"`

	// CPUs allocated to the VM (default: 4)
	CPUs int `yaml:"cpus,omitempty" json:"cpus,omitempty"`

	// Memory allocated to the VM (e.g., "8GB", "4096MB")
	Memory string `yaml:"memory,omitempty" json:"memory,omitempty"`

	// Disk size for the VM (e.g., "50GB", "100GB")
	Disk string `yaml:"disk,omitempty" json:"disk,omitempty"`

	// MountType: "reverse-sshfs", "9p", "virtiofs"
	// Default: auto-detect based on provider
	MountType string `yaml:"mount_type,omitempty" json:"mount_type,omitempty"`

	// Dependencies to install in the VM
	Dependencies []string `yaml:"dependencies,omitempty" json:"dependencies,omitempty"`

	// StartupCommands to run when VM starts
	StartupCommands []string `yaml:"startup_commands,omitempty" json:"startup_commands,omitempty"`

	// Lima-specific configuration
	Lima *LimaConfig `yaml:"lima,omitempty" json:"lima,omitempty"`

	// OrbStack VM-specific configuration
	OrbStackVM *OrbStackVMConfig `yaml:"orbstack_vm,omitempty" json:"orbstack_vm,omitempty"`
}

// LimaConfig defines Lima-specific VM settings
type LimaConfig struct {
	// Template to use (e.g., "default", "docker", "k8s")
	Template string `yaml:"template,omitempty" json:"template,omitempty"`

	// Arch: "host", "x86_64", "aarch64"
	Arch string `yaml:"arch,omitempty" json:"arch,omitempty"`

	// Images to use for the VM
	Images []LimaImage `yaml:"images,omitempty" json:"images,omitempty"`
}

// LimaImage defines a Lima VM image
type LimaImage struct {
	// Location of the image (URL or local path)
	Location string `yaml:"location" json:"location"`

	// Arch: "x86_64", "aarch64"
	Arch string `yaml:"arch,omitempty" json:"arch,omitempty"`

	// Digest for verification
	Digest string `yaml:"digest,omitempty" json:"digest,omitempty"`
}

// OrbStackVMConfig defines OrbStack VM-specific settings
type OrbStackVMConfig struct {
	// Distribution: "ubuntu", "debian", etc.
	Distribution string `yaml:"distribution,omitempty" json:"distribution,omitempty"`

	// Version of the distribution
	Version string `yaml:"version,omitempty" json:"version,omitempty"`
}

// Defaults returns a config with default values
func Defaults() *Config {
	return &Config{
		Project: ProjectConfig{
			NamingStrategy: "git-branch",
			ComposeFiles:   []string{"docker-compose.yml"},
		},
		Provider: ProviderConfig{
			Type: "auto",
			OrbStack: &OrbStackConfig{
				DNSSuffix:          ".orb.local",
				UseContainerDNS:    false,
				RemovePortBindings: true,
			},
		},
		Network: NetworkConfig{
			AllowedHosts: ".orb.local,localhost,127.0.0.1",
			NetworkMode:  "bridge",
			DNSHashing:   true, // Enable hashing by default
		},
		Ports: PortsConfig{
			RangeStart:      10000,
			RangeEnd:        60000,
			PersistenceFile: ".space-ports.json",
			Strategy:        "sequential",
		},
		VM: VMConfig{
			Enabled:  false,
			Provider: "auto",
			CPUs:     4,
			Memory:   "8GB",
			Disk:     "50GB",
			Dependencies: []string{
				"docker",
				"docker-compose",
				"git",
			},
		},
	}
}

// Merge merges another config into this one (other takes precedence)
func (c *Config) Merge(other *Config) *Config {
	if other == nil {
		return c
	}

	merged := *c

	// Merge project config
	if other.Project.Name != "" {
		merged.Project.Name = other.Project.Name
	}
	if other.Project.Prefix != "" {
		merged.Project.Prefix = other.Project.Prefix
	}
	if other.Project.NamingStrategy != "" {
		merged.Project.NamingStrategy = other.Project.NamingStrategy
	}
	if len(other.Project.ComposeFiles) > 0 {
		merged.Project.ComposeFiles = other.Project.ComposeFiles
	}
	if other.Project.WorkDir != "" {
		merged.Project.WorkDir = other.Project.WorkDir
	}

	// Merge services (deep merge)
	if len(other.Services) > 0 {
		if merged.Services == nil {
			merged.Services = make(map[string]ServiceConfig)
		}
		for k, v := range other.Services {
			merged.Services[k] = v
		}
	}

	// Merge databases
	if len(other.Databases) > 0 {
		merged.Databases = other.Databases
	}

	// Merge commands
	if other.Commands.Seed != "" {
		merged.Commands.Seed = other.Commands.Seed
	}
	if other.Commands.Migrate != "" {
		merged.Commands.Migrate = other.Commands.Migrate
	}
	if len(other.Commands.Custom) > 0 {
		if merged.Commands.Custom == nil {
			merged.Commands.Custom = make(map[string]string)
		}
		for k, v := range other.Commands.Custom {
			merged.Commands.Custom[k] = v
		}
	}

	// Merge provider config
	if other.Provider.Type != "" {
		merged.Provider.Type = other.Provider.Type
	}

	// Merge network config
	if other.Network.AllowedHosts != "" {
		merged.Network.AllowedHosts = other.Network.AllowedHosts
	}
	if other.Network.NetworkMode != "" {
		merged.Network.NetworkMode = other.Network.NetworkMode
	}

	// Merge ports config
	if other.Ports.RangeStart > 0 {
		merged.Ports.RangeStart = other.Ports.RangeStart
	}
	if other.Ports.RangeEnd > 0 {
		merged.Ports.RangeEnd = other.Ports.RangeEnd
	}
	if other.Ports.PersistenceFile != "" {
		merged.Ports.PersistenceFile = other.Ports.PersistenceFile
	}
	if other.Ports.Strategy != "" {
		merged.Ports.Strategy = other.Ports.Strategy
	}

	// Merge VM config
	if other.VM.Enabled {
		merged.VM.Enabled = other.VM.Enabled
	}
	if other.VM.Provider != "" {
		merged.VM.Provider = other.VM.Provider
	}
	if other.VM.CPUs > 0 {
		merged.VM.CPUs = other.VM.CPUs
	}
	if other.VM.Memory != "" {
		merged.VM.Memory = other.VM.Memory
	}
	if other.VM.Disk != "" {
		merged.VM.Disk = other.VM.Disk
	}
	if other.VM.MountType != "" {
		merged.VM.MountType = other.VM.MountType
	}
	if len(other.VM.Dependencies) > 0 {
		merged.VM.Dependencies = other.VM.Dependencies
	}
	if len(other.VM.StartupCommands) > 0 {
		merged.VM.StartupCommands = other.VM.StartupCommands
	}

	// Merge hooks config
	if other.Hooks.Vite != nil {
		if merged.Hooks.Vite == nil {
			merged.Hooks.Vite = &ViteHooksConfig{}
		}
		if other.Hooks.Vite.Enabled {
			merged.Hooks.Vite.Enabled = other.Hooks.Vite.Enabled
		}
		if other.Hooks.Vite.AutoDetect {
			merged.Hooks.Vite.AutoDetect = other.Hooks.Vite.AutoDetect
		}
		if other.Hooks.Vite.AllowedHostsPattern != "" {
			merged.Hooks.Vite.AllowedHostsPattern = other.Hooks.Vite.AllowedHostsPattern
		}
		if len(other.Hooks.Vite.EnvVars) > 0 {
			if merged.Hooks.Vite.EnvVars == nil {
				merged.Hooks.Vite.EnvVars = make(map[string]string)
			}
			for k, v := range other.Hooks.Vite.EnvVars {
				merged.Hooks.Vite.EnvVars[k] = v
			}
		}
	}
	if len(other.Hooks.Custom) > 0 {
		merged.Hooks.Custom = other.Hooks.Custom
	}

	// Merge database hooks config
	if other.Hooks.Database != nil {
		if merged.Hooks.Database == nil {
			merged.Hooks.Database = &DatabaseHooksConfig{}
		}
		if other.Hooks.Database.River != nil {
			if merged.Hooks.Database.River == nil {
				merged.Hooks.Database.River = &RiverHooksConfig{}
			}
			if other.Hooks.Database.River.Enabled {
				merged.Hooks.Database.River.Enabled = other.Hooks.Database.River.Enabled
			}
			if other.Hooks.Database.River.PostgresService != "" {
				merged.Hooks.Database.River.PostgresService = other.Hooks.Database.River.PostgresService
			}
			if other.Hooks.Database.River.DatabaseName != "" {
				merged.Hooks.Database.River.DatabaseName = other.Hooks.Database.River.DatabaseName
			}
			if other.Hooks.Database.River.Username != "" {
				merged.Hooks.Database.River.Username = other.Hooks.Database.River.Username
			}
			if other.Hooks.Database.River.Password != "" {
				merged.Hooks.Database.River.Password = other.Hooks.Database.River.Password
			}
			if other.Hooks.Database.River.Port > 0 {
				merged.Hooks.Database.River.Port = other.Hooks.Database.River.Port
			}
		}
	}

	return &merged
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Validation logic will be in validator.go
	return nil
}
