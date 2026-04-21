package renderer_test

import (
	"testing"

	"github.com/saero-ai/xcaffold/internal/renderer"
	"github.com/stretchr/testify/assert"
)

func TestIsMappedModel_KnownAlias_Claude_ReturnsTrue(t *testing.T) {
	assert.True(t, renderer.IsMappedModel("sonnet-4", "claude"),
		"sonnet-4 is a known alias with a claude mapping")
}

func TestIsMappedModel_KnownAlias_Cursor_ReturnsTrue(t *testing.T) {
	assert.True(t, renderer.IsMappedModel("sonnet-4", "cursor"),
		"sonnet-4 has a cursor mapping — translated to a canonical model string")
}

func TestIsMappedModel_LiteralModelID_ReturnsFalse(t *testing.T) {
	assert.False(t, renderer.IsMappedModel("claude-sonnet-4-5", "claude"),
		"literal model IDs are not in modelAliases — IsMappedModel must return false")
}

func TestIsMappedModel_LiteralModelID_Cursor_ReturnsFalse(t *testing.T) {
	assert.False(t, renderer.IsMappedModel("claude-sonnet-4-5", "cursor"),
		"literal model ID on cursor target must return false")
}

func TestIsMappedModel_EmptyAlias_ReturnsFalse(t *testing.T) {
	assert.False(t, renderer.IsMappedModel("", "claude"),
		"empty alias is never mapped")
}

func TestIsMappedModel_Antigravity_ReturnsFalse(t *testing.T) {
	assert.False(t, renderer.IsMappedModel("sonnet-4", "antigravity"),
		"antigravity has no model support — always false")
}

func TestResolveModel_GeminiTarget_TranslatesSonnet(t *testing.T) {
	model, ok := renderer.ResolveModel("sonnet-4", "gemini")
	if !ok {
		t.Fatal("expected ok=true for gemini target with sonnet-4 alias")
	}
	if model == "sonnet-4" {
		t.Errorf("expected translated model for gemini, got raw alias %q", model)
	}
}

func TestResolveModel_CopilotTarget_TranslatesSonnet(t *testing.T) {
	model, ok := renderer.ResolveModel("sonnet-4", "copilot")
	if !ok {
		t.Fatal("expected ok=true for copilot target with sonnet-4 alias")
	}
	if model == "sonnet-4" {
		t.Errorf("expected translated model for copilot, got raw alias %q", model)
	}
}

func TestResolveModel_GeminiTarget_TranslatesOpus(t *testing.T) {
	model, ok := renderer.ResolveModel("opus-4", "gemini")
	if !ok {
		t.Fatal("expected ok=true for gemini target with opus-4 alias")
	}
	if model == "opus-4" {
		t.Errorf("expected translated model for gemini, got raw alias %q", model)
	}
}

func TestResolveModel_CursorTarget_ReturnsMapped(t *testing.T) {
	model, ok := renderer.ResolveModel("sonnet-4", "cursor")
	if !ok {
		t.Fatal("expected ok=true for cursor target with sonnet-4 alias")
	}
	if model == "sonnet-4" {
		t.Errorf("expected translated model for cursor, got raw alias %q", model)
	}
}

func TestResolveModel_UnknownAlias_PassesThrough(t *testing.T) {
	model, ok := renderer.ResolveModel("custom-model-v3", "gemini")
	if !ok {
		t.Fatal("expected ok=true for gemini target with unknown model literal")
	}
	if model != "custom-model-v3" {
		t.Errorf("expected literal passthrough for unknown model, got %q", model)
	}
}
