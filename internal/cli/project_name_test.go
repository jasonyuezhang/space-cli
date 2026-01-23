package cli

import (
	"testing"
)

func TestNormalizeProjectName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "leading underscore",
			input:    "_wt1-feature",
			expected: "wt1-feature",
		},
		{
			name:     "leading hyphen",
			input:    "-project-name",
			expected: "project-name",
		},
		{
			name:     "multiple leading underscores",
			input:    "___test-project",
			expected: "test-project",
		},
		{
			name:     "mixed leading invalid chars",
			input:    "_-_project",
			expected: "project",
		},
		{
			name:     "uppercase to lowercase",
			input:    "MyProject-Name",
			expected: "myproject-name",
		},
		{
			name:     "valid name unchanged",
			input:    "valid-project-name",
			expected: "valid-project-name",
		},
		{
			name:     "starts with number",
			input:    "123-project",
			expected: "123-project",
		},
		{
			name:     "only underscores",
			input:    "___",
			expected: "p",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "p",
		},
		{
			name:     "special chars replaced",
			input:    "project@#$name",
			expected: "project---name",
		},
		{
			name:     "underscores preserved in middle",
			input:    "my_project_name",
			expected: "my_project_name",
		},
		{
			name:     "git branch with slashes",
			input:    "feature/my-feature",
			expected: "feature-my-feature",
		},
		{
			name:     "complex case",
			input:    "_wt1-jason-feat-split-febe",
			expected: "wt1-jason-feat-split-febe",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeProjectName(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeProjectName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIsAlphanumeric(t *testing.T) {
	tests := []struct {
		name     string
		input    byte
		expected bool
	}{
		{"lowercase a", 'a', true},
		{"lowercase z", 'z', true},
		{"digit 0", '0', true},
		{"digit 9", '9', true},
		{"uppercase A", 'A', false},
		{"hyphen", '-', false},
		{"underscore", '_', false},
		{"special char", '@', false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isAlphanumeric(tt.input)
			if result != tt.expected {
				t.Errorf("isAlphanumeric(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestNormalizeProjectName_DockerCompliance(t *testing.T) {
	// Test cases that previously caused Docker errors
	problematicNames := []string{
		"_wt1-jason-feat-split-febe-cross-repo-search-feature-flag",
		"_test",
		"-project",
		"__my_project",
	}

	for _, name := range problematicNames {
		t.Run(name, func(t *testing.T) {
			result := normalizeProjectName(name)

			// Must not be empty
			if len(result) == 0 {
				t.Errorf("normalizeProjectName(%q) returned empty string", name)
			}

			// Must start with letter or number
			if !isAlphanumeric(result[0]) {
				t.Errorf("normalizeProjectName(%q) = %q, does not start with letter or number", name, result)
			}

			// All characters must be valid
			for i, c := range result {
				if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' || c == '_') {
					t.Errorf("normalizeProjectName(%q) = %q, contains invalid character at position %d: %q",
						name, result, i, c)
				}
			}
		})
	}
}
