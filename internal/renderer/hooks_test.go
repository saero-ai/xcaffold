package renderer_test

import (
	"testing"

	"github.com/saero-ai/xcaffold/internal/renderer"
	"github.com/stretchr/testify/assert"
)

func TestTranslateHookCommand(t *testing.T) {
	tests := []struct {
		name             string
		command          string
		targetEnvVar     string
		targetPathPrefix string
		expected         string
	}{
		{
			name:             "Copilot translation",
			command:          "\"$CLAUDE_PROJECT_DIR/.claude/hooks/script.sh\"",
			targetEnvVar:     "$GITHUB_WORKSPACE",
			targetPathPrefix: ".github/hooks/scripts/",
			expected:         "\"$GITHUB_WORKSPACE/.github/hooks/scripts/script.sh\"",
		},
		{
			name:             "Gemini translation with xcf/hooks abstract path",
			command:          "npm run build && sh .xcf/hooks/post-build.sh",
			targetEnvVar:     "$GEMINI_PROJECT_DIR",
			targetPathPrefix: ".gemini/hooks/",
			expected:         "npm run build && sh .gemini/hooks/post-build.sh",
		},
		{
			name:             "Cursor translation with braced env var",
			command:          "${CLAUDE_PROJECT_DIR}/.claude/hooks/sync.sh",
			targetEnvVar:     "${CURSOR_PROJECT_DIR}",
			targetPathPrefix: ".cursor/hooks/",
			expected:         "${CURSOR_PROJECT_DIR}/.cursor/hooks/sync.sh",
		},
		{
			name:             "Pass through unhandled path",
			command:          "echo 'Hello' > my-file",
			targetEnvVar:     "$CURSOR_PROJECT_DIR",
			targetPathPrefix: ".cursor/hooks/",
			expected:         "echo 'Hello' > my-file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := renderer.TranslateHookCommand(tt.command, tt.targetEnvVar, tt.targetPathPrefix)
			assert.Equal(t, tt.expected, actual)
		})
	}
}
