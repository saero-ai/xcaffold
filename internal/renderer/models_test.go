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

func TestIsKnownClaudeAlias(t *testing.T) {
	tests := []struct {
		name     string
		alias    string
		expected bool
	}{
		{"Sonnet Exact", "sonnet", true},
		{"Opus Cap", "Opus", true},
		{"Haiku Mixed", "HaikU", true},
		{"Mapped Alias Fails", "sonnet-4", false},
		{"Unknown Model", "gpt-4", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, renderer.IsKnownClaudeAlias(tt.alias))
		})
	}
}

func TestSanitizeAgentModel(t *testing.T) {
	tests := []struct {
		name          string
		model         string
		caps          renderer.CapabilitySet
		targetName    string
		expectedModel string
		expectedNotes int
		expectedCode  string
	}{
		{
			name:  "ModelFieldFalse_Empty",
			model: "sonnet",
			caps: renderer.CapabilitySet{
				ModelField: false,
			},
			targetName:    "cursor",
			expectedModel: "",
			expectedNotes: 0,
		},
		{
			name:  "MappedAlias_TranslatesCleanly",
			model: "sonnet-4",
			caps: renderer.CapabilitySet{
				ModelField: true,
			},
			targetName:    "gemini",
			expectedModel: "gemini-2.5-flash",
			expectedNotes: 0,
		},
		{
			name:  "BareClaudeAlias_EmitsWarning",
			model: "sonnet",
			caps: renderer.CapabilitySet{
				ModelField: true,
			},
			targetName:    "gemini",
			expectedModel: "",
			expectedNotes: 1,
			expectedCode:  renderer.CodeAgentModelUnmapped,
		},
		{
			name:  "NativeLiteral_PassesThroughWithInfo",
			model: "gemini-2.5-pro",
			caps: renderer.CapabilitySet{
				ModelField: true,
			},
			targetName:    "gemini",
			expectedModel: "gemini-2.5-pro",
			expectedNotes: 1,
			expectedCode:  renderer.CodeFieldTransformed,
		},
		{
			name:  "EmptyModel_QuickReturn",
			model: "",
			caps: renderer.CapabilitySet{
				ModelField: true,
			},
			targetName:    "gemini",
			expectedModel: "",
			expectedNotes: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotModel, gotNotes := renderer.SanitizeAgentModel(tt.model, tt.caps, tt.targetName, "test_agent")

			assert.Equal(t, tt.expectedModel, gotModel)
			assert.Equal(t, tt.expectedNotes, len(gotNotes))

			if tt.expectedNotes > 0 && len(gotNotes) > 0 {
				assert.Equal(t, tt.expectedCode, gotNotes[0].Code)
			}
		})
	}
}
