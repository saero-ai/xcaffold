package resolver

import (
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
