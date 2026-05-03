package renderer_test

import (
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/renderer"
	antigravity "github.com/saero-ai/xcaffold/providers/antigravity"
	"github.com/saero-ai/xcaffold/providers/claude"
	copilot "github.com/saero-ai/xcaffold/providers/copilot"
	"github.com/saero-ai/xcaffold/providers/cursor"
	gemini "github.com/saero-ai/xcaffold/providers/gemini"
	"github.com/stretchr/testify/assert"
)

func TestCompileMemory_AllRenderers_SatisfyInterface(t *testing.T) {
	renderers := []renderer.TargetRenderer{
		claude.New(),
		cursor.New(),
		gemini.New(),
		copilot.New(),
		antigravity.New(),
	}
	for _, r := range renderers {
		t.Run(r.Target(), func(t *testing.T) {
			config := &ast.XcaffoldConfig{}
			opts := renderer.MemoryOptions{}
			files, notes, err := r.CompileMemory(config, "/tmp/test", opts)
			assert.NoError(t, err)
			assert.NotNil(t, files)
			_ = notes
		})
	}
}
