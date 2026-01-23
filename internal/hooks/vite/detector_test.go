package vite

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetector_DetectViteConfigTS(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "vite-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create vite.config.ts
	configPath := filepath.Join(tmpDir, "vite.config.ts")
	if err := os.WriteFile(configPath, []byte("export default {}"), 0644); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	detector, err := NewDetector(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create detector: %v", err)
	}

	result, err := detector.Detect()
	if err != nil {
		t.Fatalf("Detection failed: %v", err)
	}

	if !result.IsViteProject {
		t.Error("Expected IsViteProject to be true")
	}
	if result.ConfigFile != configPath {
		t.Errorf("Expected ConfigFile %s, got %s", configPath, result.ConfigFile)
	}
	if result.ConfigType != "ts" {
		t.Errorf("Expected ConfigType 'ts', got %s", result.ConfigType)
	}
}

func TestDetector_DetectViteConfigJS(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "vite-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	configPath := filepath.Join(tmpDir, "vite.config.js")
	if err := os.WriteFile(configPath, []byte("export default {}"), 0644); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	detector, err := NewDetector(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create detector: %v", err)
	}

	result, err := detector.Detect()
	if err != nil {
		t.Fatalf("Detection failed: %v", err)
	}

	if !result.IsViteProject {
		t.Error("Expected IsViteProject to be true")
	}
	if result.ConfigType != "js" {
		t.Errorf("Expected ConfigType 'js', got %s", result.ConfigType)
	}
}

func TestDetector_DetectPackageJSON(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "vite-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create package.json with vite dependency
	pkgPath := filepath.Join(tmpDir, "package.json")
	pkgContent := `{
		"name": "test",
		"devDependencies": {
			"vite": "^5.0.0"
		}
	}`
	if err := os.WriteFile(pkgPath, []byte(pkgContent), 0644); err != nil {
		t.Fatalf("Failed to create package.json: %v", err)
	}

	detector, err := NewDetector(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create detector: %v", err)
	}

	result, err := detector.Detect()
	if err != nil {
		t.Fatalf("Detection failed: %v", err)
	}

	if !result.IsViteProject {
		t.Error("Expected IsViteProject to be true")
	}
	if !result.HasViteDep {
		t.Error("Expected HasViteDep to be true")
	}
	if result.ViteVersion != "^5.0.0" {
		t.Errorf("Expected ViteVersion '^5.0.0', got %s", result.ViteVersion)
	}
}

func TestDetector_NoViteProject(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "vite-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	detector, err := NewDetector(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create detector: %v", err)
	}

	result, err := detector.Detect()
	if err != nil {
		t.Fatalf("Detection failed: %v", err)
	}

	if result.IsViteProject {
		t.Error("Expected IsViteProject to be false for empty directory")
	}
}
