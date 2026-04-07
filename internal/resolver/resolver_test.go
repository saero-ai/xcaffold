package resolver

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStripFrontmatter(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "No frontmatter",
			input:    "Just normal text\nLine 2",
			expected: "Just normal text\nLine 2",
		},
		{
			name:     "Well-formed frontmatter",
			input:    "---\ntitle: test\n---\nActual content",
			expected: "Actual content",
		},
		{
			name:     "Windows line endings",
			input:    "---\r\ntitle: test\r\n---\r\nActual content",
			expected: "Actual content",
		},
		{
			name:     "Malformed missing closing",
			input:    "---\ntitle: test\nActual content without closing separator",
			expected: "---\ntitle: test\nActual content without closing separator",
		},
		{
			name:     "Frontmatter with empty content",
			input:    "---\ntitle: test\n---",
			expected: "",
		},
		{
			name:     "Multiple frontend markers payload",
			input:    "---\ntitle: test\n---\nActual ---\ncontent",
			expected: "Actual ---\ncontent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := StripFrontmatter(tt.input)
			if actual != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, actual)
			}
		})
	}
}

func TestResolveInstructions(t *testing.T) {
	// Setup test files
	dir := t.TempDir()
	file1Path := filepath.Join(dir, "file1.md")
	_ = os.WriteFile(file1Path, []byte("---\ntitle: file1\n---\ncontent1"), 0600)

	conventionPath := filepath.Join(dir, "convention.md")
	_ = os.WriteFile(conventionPath, []byte("convention_content"), 0600)

	tests := []struct {
		name           string
		inline         string
		filePath       string
		conventionPath string
		baseDir        string
		expected       string
		expectError    bool
	}{
		{
			name:           "Inline takes precedence",
			inline:         "inline_content",
			filePath:       "file1.md",
			conventionPath: "convention.md",
			baseDir:        dir,
			expected:       "inline_content",
		},
		{
			name:           "FilePath resolves and strips",
			inline:         "",
			filePath:       "file1.md",
			conventionPath: "convention.md",
			baseDir:        dir,
			expected:       "content1",
		},
		{
			name:           "Convention path resolves",
			inline:         "",
			filePath:       "",
			conventionPath: "convention.md",
			baseDir:        dir,
			expected:       "convention_content",
		},
		{
			name:           "Convention path missing is OK",
			inline:         "",
			filePath:       "",
			conventionPath: "nonexistent_convention.md",
			baseDir:        dir,
			expected:       "",
		},
		{
			name:           "FilePath missing fails",
			inline:         "",
			filePath:       "nonexistent.md",
			conventionPath: "convention.md",
			baseDir:        dir,
			expectError:    true,
		},
		{
			name:           "All empty",
			inline:         "",
			filePath:       "",
			conventionPath: "",
			baseDir:        dir,
			expected:       "",
		},
		{
			name:           "Absolute filePath",
			inline:         "",
			filePath:       file1Path,
			conventionPath: "convention.md",
			baseDir:        "/some/other/dir",
			expected:       "content1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, err := ResolveInstructions(tt.inline, tt.filePath, tt.conventionPath, tt.baseDir)
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if actual != tt.expected {
					t.Errorf("expected %q, got %q", tt.expected, actual)
				}
			}
		})
	}
}
