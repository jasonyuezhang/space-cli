package hooks

import (
	"context"

	"github.com/happy-sdk/space-cli/internal/provider"
)

// EventType represents the type of hook event
type EventType string

const (
	// Lifecycle events
	PreUp    EventType = "pre-up"
	PostUp   EventType = "post-up"
	PreDown  EventType = "pre-down"
	PostDown EventType = "post-down"

	// DNS events
	OnDNSReady EventType = "on-dns-ready"

	// Environment events
	OnEnvChange EventType = "on-env-change"

	// Service events
	OnServiceStart EventType = "on-service-start"
	OnServiceStop  EventType = "on-service-stop"
)

// String returns the string representation of the event type
func (e EventType) String() string {
	return string(e)
}

// AllEventTypes returns all available event types
func AllEventTypes() []EventType {
	return []EventType{
		PreUp,
		PostUp,
		PreDown,
		PostDown,
		OnDNSReady,
		OnEnvChange,
		OnServiceStart,
		OnServiceStop,
	}
}

// IsValid checks if the event type is valid
func (e EventType) IsValid() bool {
	for _, valid := range AllEventTypes() {
		if e == valid {
			return true
		}
	}
	return false
}

// ServiceInfo contains information about a running service
type ServiceInfo struct {
	// Name is the service name
	Name string

	// DNSName is the full DNS name (e.g., "web-a1b2c3.space.local")
	DNSName string

	// ContainerName is the Docker container name
	ContainerName string

	// InternalPort is the port inside the container
	InternalPort int

	// ExternalPort is the mapped host port (0 if DNS mode)
	ExternalPort int

	// Host is the hostname/IP to reach this service
	Host string

	// URL is the full URL to access the service
	URL string

	// Status is the service status (running, stopped, etc.)
	Status string

	// Environment variables for this service
	Environment map[string]string
}

// HookContext contains context information passed to hooks
type HookContext struct {
	// WorkDir is the working directory for the project
	WorkDir string

	// ProjectName is the Docker Compose project name
	ProjectName string

	// Hash is the unique hash for this project (for DNS)
	Hash string

	// Services maps service name to service info
	Services map[string]*ServiceInfo

	// Provider is the detected Docker provider
	Provider provider.Provider

	// DNSEnabled indicates if DNS mode is active
	DNSEnabled bool

	// BaseDomain is the DNS base domain (e.g., "space.local")
	BaseDomain string

	// DNSAddress is the DNS server address (if running)
	DNSAddress string

	// ComposeFiles is the list of docker-compose files in use
	ComposeFiles []string

	// Metadata allows hooks to store arbitrary data
	Metadata map[string]interface{}

	// ServiceName is set for service-specific events
	ServiceName string

	// Error is set if an error occurred (for post-* events)
	Error error

	// EnvironmentChanges tracks env var changes (for OnEnvChange)
	EnvironmentChanges map[string]EnvChange
}

// EnvChange represents a change to an environment variable
type EnvChange struct {
	Key      string
	OldValue string
	NewValue string
	Action   string // "set", "unset", "changed"
}

// NewHookContext creates a new HookContext with initialized maps
func NewHookContext() *HookContext {
	return &HookContext{
		Services:           make(map[string]*ServiceInfo),
		Metadata:           make(map[string]interface{}),
		EnvironmentChanges: make(map[string]EnvChange),
		BaseDomain:         "space.local",
	}
}

// GetService returns service info by name, or nil if not found
func (c *HookContext) GetService(name string) *ServiceInfo {
	if c.Services == nil {
		return nil
	}
	return c.Services[name]
}

// SetMetadata sets a metadata value
func (c *HookContext) SetMetadata(key string, value interface{}) {
	if c.Metadata == nil {
		c.Metadata = make(map[string]interface{})
	}
	c.Metadata[key] = value
}

// GetMetadata gets a metadata value
func (c *HookContext) GetMetadata(key string) (interface{}, bool) {
	if c.Metadata == nil {
		return nil, false
	}
	v, ok := c.Metadata[key]
	return v, ok
}

// Hook defines the interface that all hooks must implement
type Hook interface {
	// Name returns the unique name of this hook
	Name() string

	// Description returns a human-readable description
	Description() string

	// Events returns the event types this hook handles
	Events() []EventType

	// Execute runs the hook for the given event
	// Should return nil if successful, or an error if something went wrong
	// Note: Hook errors are typically logged but don't fail the parent operation
	Execute(ctx context.Context, event EventType, hookCtx *HookContext) error
}

// Priority levels for hook execution order
type Priority int

const (
	PriorityLowest  Priority = -100
	PriorityLow     Priority = -50
	PriorityNormal  Priority = 0
	PriorityHigh    Priority = 50
	PriorityHighest Priority = 100
)

// PriorityHook extends Hook with priority support
type PriorityHook interface {
	Hook

	// Priority returns the execution priority (higher = earlier)
	Priority() Priority
}

// ConditionalHook extends Hook with conditional execution
type ConditionalHook interface {
	Hook

	// ShouldExecute returns true if the hook should run for this context
	ShouldExecute(ctx context.Context, event EventType, hookCtx *HookContext) bool
}

// AsyncHook extends Hook for asynchronous execution
type AsyncHook interface {
	Hook

	// IsAsync returns true if the hook can be run asynchronously
	IsAsync() bool
}
