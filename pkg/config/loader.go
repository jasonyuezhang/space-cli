package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	// ConfigFileName is the default config file name
	ConfigFileName = ".space.yaml"

	// AlternateConfigFileName is an alternate config file name
	AlternateConfigFileName = "space.yaml"

	// GlobalConfigDir is the global config directory
	GlobalConfigDir = ".config/space"
)

// Loader loads and merges configurations from multiple sources
type Loader struct {
	workDir string
	homeDir string
}

// NewLoader creates a new config loader
func NewLoader(workDir string) (*Loader, error) {
	// Resolve absolute work directory
	absWorkDir, err := filepath.Abs(workDir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve work directory: %w", err)
	}

	// Get home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	return &Loader{
		workDir: absWorkDir,
		homeDir: homeDir,
	}, nil
}

// Load loads and merges configurations from all sources
// Priority (highest to lowest):
// 1. Project-level config (.space.yaml in workDir)
// 2. Global config (~/.config/space/config.yaml)
// 3. Defaults
func (l *Loader) Load() (*Config, error) {
	// Start with defaults
	config := Defaults()

	// Load global config
	globalConfig, err := l.loadGlobalConfig()
	if err != nil {
		// Global config is optional, just log error
		// In real implementation, use logger here
		_ = err
	} else if globalConfig != nil {
		config = config.Merge(globalConfig)
	}

	// Load project config
	projectConfig, err := l.loadProjectConfig()
	if err != nil {
		// Project config is optional for generic use
		_ = err
	} else if projectConfig != nil {
		config = config.Merge(projectConfig)
	}

	// Validate merged config
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return config, nil
}

// LoadFromFile loads config from a specific file
func (l *Loader) LoadFromFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}

// loadGlobalConfig loads the global configuration
func (l *Loader) loadGlobalConfig() (*Config, error) {
	configPath := filepath.Join(l.homeDir, GlobalConfigDir, "config.yaml")

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, nil // No global config is okay
	}

	return l.LoadFromFile(configPath)
}

// loadProjectConfig loads the project-level configuration
func (l *Loader) loadProjectConfig() (*Config, error) {
	// Try .space.yaml first
	configPath := filepath.Join(l.workDir, ConfigFileName)
	if _, err := os.Stat(configPath); err == nil {
		return l.LoadFromFile(configPath)
	}

	// Try space.yaml
	configPath = filepath.Join(l.workDir, AlternateConfigFileName)
	if _, err := os.Stat(configPath); err == nil {
		return l.LoadFromFile(configPath)
	}

	return nil, nil // No project config is okay
}

// FindConfigFile finds the config file in the work directory
func (l *Loader) FindConfigFile() (string, error) {
	// Try .space.yaml first
	configPath := filepath.Join(l.workDir, ConfigFileName)
	if _, err := os.Stat(configPath); err == nil {
		return configPath, nil
	}

	// Try space.yaml
	configPath = filepath.Join(l.workDir, AlternateConfigFileName)
	if _, err := os.Stat(configPath); err == nil {
		return configPath, nil
	}

	return "", fmt.Errorf("config file not found in %s", l.workDir)
}

// SaveProjectConfig saves the config to the project directory
func (l *Loader) SaveProjectConfig(config *Config) error {
	configPath := filepath.Join(l.workDir, ConfigFileName)

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// InitProjectConfig creates a default project config file
func (l *Loader) InitProjectConfig() error {
	// Check if config already exists
	if _, err := l.FindConfigFile(); err == nil {
		return fmt.Errorf("config file already exists")
	}

	// Create default config with discovered services
	config := Defaults()

	// TODO: Auto-discover services from docker-compose.yml
	// This would be implemented in discovery.go

	return l.SaveProjectConfig(config)
}
