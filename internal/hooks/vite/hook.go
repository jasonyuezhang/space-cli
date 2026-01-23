package vite

import (
	"context"
	"fmt"

	"github.com/happy-sdk/space-cli/internal/hooks"
	"github.com/happy-sdk/space-cli/pkg/config"
)

// Hook coordinates Vite project setup for space-cli
type Hook struct {
	workDir  string
	detector *Detector
	envGen   *EnvGenerator
	cfgUpd   *ConfigUpdater
}

// HookResult contains the complete result of Vite hook execution
type HookResult struct {
	Detection    *DetectionResult
	EnvResult    *EnvGeneratorResult
	ConfigResult *ConfigUpdateResult
	Errors       []error
}

// NewHook creates a new Vite hook coordinator
func NewHook(workDir string) (*Hook, error) {
	detector, err := NewDetector(workDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create detector: %w", err)
	}

	envGen, err := NewEnvGenerator(workDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create env generator: %w", err)
	}

	cfgUpd, err := NewConfigUpdater(workDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create config updater: %w", err)
	}

	return &Hook{
		workDir:  workDir,
		detector: detector,
		envGen:   envGen,
		cfgUpd:   cfgUpd,
	}, nil
}

// Name returns the hook name
func (h *Hook) Name() string {
	return "vite"
}

// Description returns the hook description
func (h *Hook) Description() string {
	return "Configures Vite projects for DNS mode (allowed hosts, environment variables)"
}

// Events returns the events this hook handles
func (h *Hook) Events() []hooks.EventType {
	return []hooks.EventType{hooks.PostUp, hooks.OnDNSReady}
}

// Priority returns the hook priority (run early to set up environment)
func (h *Hook) Priority() hooks.Priority {
	return hooks.PriorityHigh
}

// ShouldExecute checks if this hook should run
func (h *Hook) ShouldExecute(ctx context.Context, event hooks.EventType, hookCtx *hooks.HookContext) bool {
	// Only run if DNS is enabled
	if !hookCtx.DNSEnabled {
		return false
	}

	// Check if this is a Vite project
	detection, err := h.detector.Detect()
	if err != nil {
		return false
	}

	return detection.IsViteProject
}

// Execute runs the Vite hook
func (h *Hook) Execute(ctx context.Context, event hooks.EventType, hookCtx *hooks.HookContext) error {
	// Detect Vite project
	detection, err := h.detector.Detect()
	if err != nil {
		return fmt.Errorf("detection failed: %w", err)
	}

	if !detection.IsViteProject {
		return nil // Not a Vite project, nothing to do
	}

	// Build service configs from hook context
	services := make([]ServiceEnvConfig, 0)
	for name, svc := range hookCtx.Services {
		if svc.InternalPort > 0 {
			services = append(services, ServiceEnvConfig{
				ServiceName: name,
				Port:        svc.InternalPort,
			})
		}
	}

	// Generate .env.development.local
	if len(services) > 0 {
		envResult, err := h.envGen.GenerateWithServices(services)
		if err != nil {
			return fmt.Errorf("env generation failed: %w", err)
		}

		if envResult.Generated {
			hookCtx.SetMetadata("vite.env_file", envResult.FilePath)
			hookCtx.SetMetadata("vite.env_vars", envResult.Variables)
		}
	}

	// Update vite.config.js/ts
	if detection.ConfigFile != "" {
		configResult, err := h.cfgUpd.UpdateAllowedHosts(detection.ConfigFile)
		if err != nil {
			return fmt.Errorf("config update failed: %w", err)
		}

		if configResult.Updated {
			// Validate the updated config
			if err := h.cfgUpd.ValidateConfig(detection.ConfigFile); err != nil {
				// Restore backup on validation failure
				if configResult.BackedUp {
					h.cfgUpd.RestoreBackup(detection.ConfigFile)
				}
				return fmt.Errorf("config validation failed: %w", err)
			}

			hookCtx.SetMetadata("vite.config_file", configResult.FilePath)
			hookCtx.SetMetadata("vite.config_updated", true)
		} else if configResult.AlreadyPresent {
			hookCtx.SetMetadata("vite.config_already_configured", true)
		}
	}

	return nil
}

// ExecuteStandalone runs the Vite hook with explicit configuration
func (h *Hook) ExecuteStandalone(cfg *config.Config) (*HookResult, error) {
	result := &HookResult{
		Errors: make([]error, 0),
	}

	// Step 1: Detect Vite project
	detection, err := h.detector.Detect()
	if err != nil {
		return nil, fmt.Errorf("detection failed: %w", err)
	}
	result.Detection = detection

	if !detection.IsViteProject {
		return result, nil // Not a Vite project, nothing to do
	}

	// Step 2: Generate .env.development.local
	envResult, err := h.envGen.Generate(cfg)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("env generation failed: %w", err))
	} else {
		result.EnvResult = envResult
	}

	// Step 3: Update vite.config.js/ts
	if detection.ConfigFile != "" {
		configResult, err := h.cfgUpd.UpdateAllowedHosts(detection.ConfigFile)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("config update failed: %w", err))
		} else {
			result.ConfigResult = configResult

			// Validate the updated config
			if configResult.Updated {
				if err := h.cfgUpd.ValidateConfig(detection.ConfigFile); err != nil {
					result.Errors = append(result.Errors, fmt.Errorf("config validation failed: %w", err))
					// Restore backup on validation failure
					if configResult.BackedUp {
						if restoreErr := h.cfgUpd.RestoreBackup(detection.ConfigFile); restoreErr != nil {
							result.Errors = append(result.Errors, fmt.Errorf("backup restore failed: %w", restoreErr))
						}
					}
				}
			}
		}
	}

	return result, nil
}

// ExecuteWithServices runs the hook with explicit service configurations
func (h *Hook) ExecuteWithServices(services []ServiceEnvConfig) (*HookResult, error) {
	result := &HookResult{
		Errors: make([]error, 0),
	}

	// Step 1: Detect Vite project
	detection, err := h.detector.Detect()
	if err != nil {
		return nil, fmt.Errorf("detection failed: %w", err)
	}
	result.Detection = detection

	if !detection.IsViteProject {
		return result, nil
	}

	// Step 2: Generate .env.development.local with explicit services
	envResult, err := h.envGen.GenerateWithServices(services)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("env generation failed: %w", err))
	} else {
		result.EnvResult = envResult
	}

	// Step 3: Update vite.config
	if detection.ConfigFile != "" {
		configResult, err := h.cfgUpd.UpdateAllowedHosts(detection.ConfigFile)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("config update failed: %w", err))
		} else {
			result.ConfigResult = configResult
		}
	}

	return result, nil
}

// Detector returns the underlying detector
func (h *Hook) Detector() *Detector {
	return h.detector
}

// EnvGenerator returns the underlying env generator
func (h *Hook) EnvGenerator() *EnvGenerator {
	return h.envGen
}

// ConfigUpdater returns the underlying config updater
func (h *Hook) ConfigUpdater() *ConfigUpdater {
	return h.cfgUpd
}
