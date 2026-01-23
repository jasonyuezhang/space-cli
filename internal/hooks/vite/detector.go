package vite

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Detector detects Vite projects in a directory
type Detector struct {
	workDir string
}

// DetectionResult contains the result of Vite project detection
type DetectionResult struct {
	IsViteProject   bool
	ConfigFile      string // Path to vite.config.js or vite.config.ts
	ConfigType      string // "js" or "ts"
	PackageJSONPath string
	HasViteDep      bool
	ViteVersion     string
}

// NewDetector creates a new Vite project detector
func NewDetector(workDir string) (*Detector, error) {
	absWorkDir, err := filepath.Abs(workDir)
	if err != nil {
		return nil, err
	}
	return &Detector{workDir: absWorkDir}, nil
}

// Detect checks if the directory contains a Vite project
func (d *Detector) Detect() (*DetectionResult, error) {
	result := &DetectionResult{}

	// Check for vite.config.ts first (preferred)
	tsConfig := filepath.Join(d.workDir, "vite.config.ts")
	if _, err := os.Stat(tsConfig); err == nil {
		result.IsViteProject = true
		result.ConfigFile = tsConfig
		result.ConfigType = "ts"
	}

	// Check for vite.config.js
	if result.ConfigFile == "" {
		jsConfig := filepath.Join(d.workDir, "vite.config.js")
		if _, err := os.Stat(jsConfig); err == nil {
			result.IsViteProject = true
			result.ConfigFile = jsConfig
			result.ConfigType = "js"
		}
	}

	// Check for vite.config.mjs
	if result.ConfigFile == "" {
		mjsConfig := filepath.Join(d.workDir, "vite.config.mjs")
		if _, err := os.Stat(mjsConfig); err == nil {
			result.IsViteProject = true
			result.ConfigFile = mjsConfig
			result.ConfigType = "js"
		}
	}

	// Check for vite.config.mts
	if result.ConfigFile == "" {
		mtsConfig := filepath.Join(d.workDir, "vite.config.mts")
		if _, err := os.Stat(mtsConfig); err == nil {
			result.IsViteProject = true
			result.ConfigFile = mtsConfig
			result.ConfigType = "ts"
		}
	}

	// Check package.json for vite dependency
	pkgPath := filepath.Join(d.workDir, "package.json")
	if _, err := os.Stat(pkgPath); err == nil {
		result.PackageJSONPath = pkgPath
		viteVersion, hasVite := d.checkPackageJSON(pkgPath)
		result.HasViteDep = hasVite
		result.ViteVersion = viteVersion
		if hasVite {
			result.IsViteProject = true
		}
	}

	return result, nil
}

// checkPackageJSON checks if package.json contains vite as a dependency
func (d *Detector) checkPackageJSON(path string) (version string, hasVite bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}

	var pkg packageJSON
	if err := json.Unmarshal(data, &pkg); err != nil {
		return "", false
	}

	// Check devDependencies first (most common)
	if v, ok := pkg.DevDependencies["vite"]; ok {
		return v, true
	}

	// Check dependencies
	if v, ok := pkg.Dependencies["vite"]; ok {
		return v, true
	}

	return "", false
}

// packageJSON represents a minimal package.json structure
type packageJSON struct {
	Name            string            `json:"name"`
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
}

// WorkDir returns the working directory
func (d *Detector) WorkDir() string {
	return d.workDir
}
