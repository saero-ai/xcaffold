package renderer

import (
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/stretchr/testify/assert"
)

func TestResolveContextBody_ReturnsEmpty_NoContexts(t *testing.T) {
	config := &ast.XcaffoldConfig{}
	got := ResolveContextBody(config, "claude")
	assert.Equal(t, "", got)
}

func TestResolveContextBody_MatchesTarget(t *testing.T) {
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Contexts: map[string]ast.ContextConfig{
				"root": {
					Targets: []string{"claude"},
					Body:    "claude only",
				},
				"all": {
					Targets: nil,
					Body:    "all targets",
				},
			},
		},
	}

	got := ResolveContextBody(config, "claude")
	assert.Contains(t, got, "claude only")
	assert.Contains(t, got, "all targets")

	gotGemini := ResolveContextBody(config, "gemini")
	assert.NotContains(t, gotGemini, "claude only")
	assert.Contains(t, gotGemini, "all targets")
}
