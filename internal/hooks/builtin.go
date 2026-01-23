package hooks

import (
	"context"
	"fmt"
	"strings"
)

// RegisterBuiltinHooks registers all built-in hooks with the manager
func RegisterBuiltinHooks(m *Manager) error {
	builtins := []Hook{
		NewEnvFileHook(),
	}

	for _, h := range builtins {
		if err := m.Register(h); err != nil {
			return fmt.Errorf("failed to register builtin hook %q: %w", h.Name(), err)
		}
	}

	return nil
}

// EnvFileHook manages .env file updates for services
type EnvFileHook struct{}

// NewEnvFileHook creates a new env file hook
func NewEnvFileHook() *EnvFileHook {
	return &EnvFileHook{}
}

// Name returns the hook name
func (h *EnvFileHook) Name() string {
	return "env-file"
}

// Description returns the hook description
func (h *EnvFileHook) Description() string {
	return "Updates .env files with service DNS names and ports"
}

// Events returns the events this hook handles
func (h *EnvFileHook) Events() []EventType {
	return []EventType{PostUp, OnEnvChange}
}

// Execute updates .env files with service information
func (h *EnvFileHook) Execute(ctx context.Context, event EventType, hookCtx *HookContext) error {
	if !hookCtx.DNSEnabled {
		return nil
	}

	// Generate environment variable suggestions
	envVars := make(map[string]string)

	for name, svc := range hookCtx.Services {
		upperName := strings.ToUpper(strings.ReplaceAll(name, "-", "_"))

		if svc.URL != "" {
			envVars[upperName+"_URL"] = svc.URL
		}
		if svc.DNSName != "" {
			envVars[upperName+"_HOST"] = svc.DNSName
		}
		if svc.InternalPort > 0 {
			envVars[upperName+"_PORT"] = fmt.Sprintf("%d", svc.InternalPort)
		}
	}

	// Store in metadata for consumers
	hookCtx.SetMetadata("env_vars", envVars)

	return nil
}
