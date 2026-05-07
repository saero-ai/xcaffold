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
			name:             "xcaf-native env var (unbraced)",
			command:          "$XCAF_PROJECT_DIR/.xcaf/hooks/script.sh",
			targetEnvVar:     "$GITHUB_WORKSPACE",
			targetPathPrefix: ".xcaf/hooks/scripts/",
			expected:         "$GITHUB_WORKSPACE/.xcaf/hooks/scripts/script.sh",
		},
		{
			name:             "xcaf-native env var (braced)",
			command:          "${XCAF_PROJECT_DIR}/.xcaf/hooks/sync.sh",
			targetEnvVar:     "${CURSOR_PROJECT_DIR}",
			targetPathPrefix: ".xcaf/hooks/",
			expected:         "${CURSOR_PROJECT_DIR}/.xcaf/hooks/sync.sh",
		},
		{
			name:             "backward compat: Claude env var imported from .claude/ project",
			command:          "\"$CLAUDE_PROJECT_DIR/.claude/hooks/script.sh\"",
			targetEnvVar:     "$GITHUB_WORKSPACE",
			targetPathPrefix: ".xcaf/hooks/scripts/",
			expected:         "\"$GITHUB_WORKSPACE/.xcaf/hooks/scripts/script.sh\"",
		},
		{
			name:             "backward compat: braced Claude env var",
			command:          "${CLAUDE_PROJECT_DIR}/.claude/hooks/sync.sh",
			targetEnvVar:     "${CURSOR_PROJECT_DIR}",
			targetPathPrefix: ".xcaf/hooks/",
			expected:         "${CURSOR_PROJECT_DIR}/.xcaf/hooks/sync.sh",
		},
		{
			name:             "xcaf abstract hook path without env var",
			command:          "npm run build && sh .xcaf/hooks/post-build.sh",
			targetEnvVar:     "$GEMINI_PROJECT_DIR",
			targetPathPrefix: ".xcaf/hooks/",
			expected:         "npm run build && sh .xcaf/hooks/post-build.sh",
		},
		{
			name:             "pass through unhandled command",
			command:          "echo 'Hello' > my-file",
			targetEnvVar:     "$CURSOR_PROJECT_DIR",
			targetPathPrefix: ".xcaf/hooks/",
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
