package shared_test

import (
	"testing"

	"github.com/saero-ai/xcaffold/internal/renderer/shared"
	"github.com/stretchr/testify/assert"
)

func TestTranslateHookCommand(t *testing.T) {
	tests := []struct {
		name             string
		cmd              string
		providerDirDepth int
		expected         string
	}{
		{
			name:             "Claude depth 1",
			cmd:              "xcf/hooks/pre-build.sh",
			providerDirDepth: 1,
			expected:         "../xcf/hooks/pre-build.sh",
		},
		{
			name:             "Cursor depth 2",
			cmd:              "xcf/hooks/post-sync.sh",
			providerDirDepth: 2,
			expected:         "../../xcf/hooks/post-sync.sh",
		},
		{
			name:             "Zero depth",
			cmd:              "xcf/hooks/test.sh",
			providerDirDepth: 0,
			expected:         "xcf/hooks/test.sh",
		},
		{
			name:             "Not an xcf hook",
			cmd:              "npm run test",
			providerDirDepth: 1,
			expected:         "npm run test",
		},
		{
			name:             "Absolute path",
			cmd:              "/usr/local/bin/xcf/hooks/script.sh",
			providerDirDepth: 1,
			expected:         "/usr/local/bin/xcf/hooks/script.sh",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := shared.TranslateHookCommand(tt.cmd, tt.providerDirDepth)
			assert.Equal(t, tt.expected, actual)
		})
	}
}
