package vite

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	// SpaceLocalDomain is the domain to add to allowed hosts
	SpaceLocalDomain = ".space.local"
)

// ConfigUpdater updates vite.config.js/ts files
type ConfigUpdater struct {
	workDir string
}

// ConfigUpdateResult contains the result of config file update
type ConfigUpdateResult struct {
	FilePath       string
	Updated        bool
	BackedUp       bool
	BackupPath     string
	AddedHosts     []string
	AlreadyPresent bool
}

// NewConfigUpdater creates a new Vite config updater
func NewConfigUpdater(workDir string) (*ConfigUpdater, error) {
	absWorkDir, err := filepath.Abs(workDir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve work directory: %w", err)
	}
	return &ConfigUpdater{workDir: absWorkDir}, nil
}

// UpdateAllowedHosts adds space.local to server.allowedHosts in vite.config
func (u *ConfigUpdater) UpdateAllowedHosts(configPath string) (*ConfigUpdateResult, error) {
	result := &ConfigUpdateResult{
		FilePath:   configPath,
		AddedHosts: []string{SpaceLocalDomain},
	}

	// Read existing config
	content, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	contentStr := string(content)

	// Check if space.local is already configured
	if u.hasSpaceLocalHost(contentStr) {
		result.AlreadyPresent = true
		return result, nil
	}

	// Backup the original file
	backupPath := configPath + ".backup"
	if err := os.WriteFile(backupPath, content, 0644); err == nil {
		result.BackedUp = true
		result.BackupPath = backupPath
	}

	// Update the config
	updatedContent, updated := u.addAllowedHosts(contentStr)
	if !updated {
		// Could not find a place to add, try adding server block
		updatedContent, updated = u.addServerBlock(contentStr)
	}

	if !updated {
		return result, fmt.Errorf("could not update config: unable to locate defineConfig or server block")
	}

	// Write updated config
	if err := os.WriteFile(configPath, []byte(updatedContent), 0644); err != nil {
		return nil, fmt.Errorf("failed to write updated config: %w", err)
	}

	result.Updated = true
	return result, nil
}

// hasSpaceLocalHost checks if space.local is already in allowedHosts
func (u *ConfigUpdater) hasSpaceLocalHost(content string) bool {
	// Check for various patterns
	patterns := []string{
		`\.space\.local`,
		`'\.space\.local'`,
		`"\.space\.local"`,
		`space\.local`,
	}

	for _, pattern := range patterns {
		matched, _ := regexp.MatchString(pattern, content)
		if matched {
			return true
		}
	}

	return false
}

// addAllowedHosts adds allowedHosts to existing server block
func (u *ConfigUpdater) addAllowedHosts(content string) (string, bool) {
	// Pattern 1: server block with existing properties
	// server: { ... }
	serverBlockRegex := regexp.MustCompile(`(server\s*:\s*\{)([^}]*?)(\})`)
	if matches := serverBlockRegex.FindStringSubmatch(content); len(matches) > 0 {
		serverContent := matches[2]

		// Check if allowedHosts already exists in server block
		if strings.Contains(serverContent, "allowedHosts") {
			// Add to existing allowedHosts array
			return u.appendToAllowedHosts(content)
		}

		// Add allowedHosts to server block
		newServerContent := serverContent
		if strings.TrimSpace(serverContent) != "" && !strings.HasSuffix(strings.TrimSpace(serverContent), ",") {
			newServerContent = serverContent + ","
		}
		newServerContent += fmt.Sprintf("\n    allowedHosts: ['%s']", SpaceLocalDomain)

		updated := serverBlockRegex.ReplaceAllString(content, "${1}"+newServerContent+"\n  ${3}")
		return updated, true
	}

	return content, false
}

// appendToAllowedHosts appends space.local to existing allowedHosts array
func (u *ConfigUpdater) appendToAllowedHosts(content string) (string, bool) {
	// Pattern: allowedHosts: [...] or allowedHosts: [...]
	allowedHostsRegex := regexp.MustCompile(`(allowedHosts\s*:\s*\[)([^\]]*?)(\])`)

	if matches := allowedHostsRegex.FindStringSubmatch(content); len(matches) > 0 {
		existingHosts := strings.TrimSpace(matches[2])

		var newHosts string
		if existingHosts == "" {
			newHosts = fmt.Sprintf("'%s'", SpaceLocalDomain)
		} else {
			newHosts = existingHosts + fmt.Sprintf(", '%s'", SpaceLocalDomain)
		}

		updated := allowedHostsRegex.ReplaceAllString(content, "${1}"+newHosts+"${3}")
		return updated, true
	}

	return content, false
}

// addServerBlock adds a new server block with allowedHosts
func (u *ConfigUpdater) addServerBlock(content string) (string, bool) {
	// Find defineConfig({ ... }) and add server block inside
	defineConfigRegex := regexp.MustCompile(`(defineConfig\s*\(\s*\{)([^}]*?)(\}\s*\))`)

	if matches := defineConfigRegex.FindStringSubmatch(content); len(matches) > 0 {
		configContent := matches[2]

		// Add server block
		serverBlock := fmt.Sprintf("\n  server: {\n    allowedHosts: ['%s'],\n  },", SpaceLocalDomain)

		var newContent string
		trimmed := strings.TrimSpace(configContent)
		if trimmed == "" {
			newContent = serverBlock + "\n"
		} else if strings.HasSuffix(trimmed, ",") {
			newContent = configContent + serverBlock
		} else {
			newContent = configContent + "," + serverBlock
		}

		updated := defineConfigRegex.ReplaceAllString(content, "${1}"+newContent+"\n${3}")
		return updated, true
	}

	// Try export default { ... } pattern (without defineConfig)
	exportDefaultRegex := regexp.MustCompile(`(export\s+default\s*\{)([^}]*?)(\})`)

	if matches := exportDefaultRegex.FindStringSubmatch(content); len(matches) > 0 {
		configContent := matches[2]

		serverBlock := fmt.Sprintf("\n  server: {\n    allowedHosts: ['%s'],\n  },", SpaceLocalDomain)

		var newContent string
		trimmed := strings.TrimSpace(configContent)
		if trimmed == "" {
			newContent = serverBlock + "\n"
		} else if strings.HasSuffix(trimmed, ",") {
			newContent = configContent + serverBlock
		} else {
			newContent = configContent + "," + serverBlock
		}

		updated := exportDefaultRegex.ReplaceAllString(content, "${1}"+newContent+"\n${3}")
		return updated, true
	}

	return content, false
}

// GenerateMinimalConfig generates a minimal vite.config.js with allowedHosts
func (u *ConfigUpdater) GenerateMinimalConfig() string {
	return fmt.Sprintf(`import { defineConfig } from 'vite'

export default defineConfig({
  server: {
    allowedHosts: ['%s'],
  },
})
`, SpaceLocalDomain)
}

// GenerateMinimalTSConfig generates a minimal vite.config.ts with allowedHosts
func (u *ConfigUpdater) GenerateMinimalTSConfig() string {
	return fmt.Sprintf(`import { defineConfig } from 'vite'

export default defineConfig({
  server: {
    allowedHosts: ['%s'],
  },
})
`, SpaceLocalDomain)
}

// CreateConfigIfNotExists creates a vite.config file if none exists
func (u *ConfigUpdater) CreateConfigIfNotExists(preferTS bool) (*ConfigUpdateResult, error) {
	// Check if any config exists
	configFiles := []string{
		"vite.config.ts",
		"vite.config.js",
		"vite.config.mts",
		"vite.config.mjs",
	}

	for _, cf := range configFiles {
		path := filepath.Join(u.workDir, cf)
		if _, err := os.Stat(path); err == nil {
			// Config exists, update it
			return u.UpdateAllowedHosts(path)
		}
	}

	// No config exists, create one
	var configPath string
	var content string

	if preferTS {
		configPath = filepath.Join(u.workDir, "vite.config.ts")
		content = u.GenerateMinimalTSConfig()
	} else {
		configPath = filepath.Join(u.workDir, "vite.config.js")
		content = u.GenerateMinimalConfig()
	}

	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		return nil, fmt.Errorf("failed to create config file: %w", err)
	}

	return &ConfigUpdateResult{
		FilePath:   configPath,
		Updated:    true,
		AddedHosts: []string{SpaceLocalDomain},
	}, nil
}

// ValidateConfig checks if the vite config is valid after update
func (u *ConfigUpdater) ValidateConfig(configPath string) error {
	content, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	contentStr := string(content)

	// Basic validation: check for balanced braces
	openBraces := strings.Count(contentStr, "{")
	closeBraces := strings.Count(contentStr, "}")
	if openBraces != closeBraces {
		return fmt.Errorf("unbalanced braces in config file: %d open, %d close", openBraces, closeBraces)
	}

	// Check for syntax errors in common patterns
	if strings.Contains(contentStr, ",,") {
		return fmt.Errorf("double comma detected in config file")
	}

	// Verify allowedHosts is present
	if !u.hasSpaceLocalHost(contentStr) {
		return fmt.Errorf("space.local not found in allowedHosts after update")
	}

	return nil
}

// RestoreBackup restores the config from backup
func (u *ConfigUpdater) RestoreBackup(configPath string) error {
	backupPath := configPath + ".backup"
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("backup file does not exist: %s", backupPath)
	}

	data, err := os.ReadFile(backupPath)
	if err != nil {
		return fmt.Errorf("failed to read backup file: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to restore config from backup: %w", err)
	}

	return nil
}

// WorkDir returns the working directory
func (u *ConfigUpdater) WorkDir() string {
	return u.workDir
}
