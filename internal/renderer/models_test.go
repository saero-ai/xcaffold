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
		// The registry gates model support; caps.ModelField is no longer the gate.
		// "cursor" has model: optional in the registry, so it passes through.
		// A bare Claude alias on a non-Claude target emits a warning and is dropped.
		{
			name:  "RegistryOptional_CursorTarget_BareAliasDrop",
			model: "sonnet",
			caps: renderer.CapabilitySet{
				ModelField: true,
			},
			targetName:    "cursor",
			expectedModel: "",
			expectedNotes: 1,
			expectedCode:  renderer.CodeAgentModelUnmapped,
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
		// Bare Claude aliases are valid Claude Code native aliases and must pass
		// through as-is when the target is "claude". Ground truth (models.json,
		// verified 2026-04-30): "sonnet", "opus", "haiku" are documented aliases.
		{
			name:  "BareClaudeAlias_ClaudeTarget_PassesThrough",
			model: "sonnet",
			caps: renderer.CapabilitySet{
				ModelField: true,
			},
			targetName:    "claude",
			expectedModel: "sonnet",
			expectedNotes: 1,
			expectedCode:  renderer.CodeFieldTransformed,
		},
		{
			name:  "BareOpusAlias_ClaudeTarget_PassesThrough",
			model: "opus",
			caps: renderer.CapabilitySet{
				ModelField: true,
			},
			targetName:    "claude",
			expectedModel: "opus",
			expectedNotes: 1,
			expectedCode:  renderer.CodeFieldTransformed,
		},
		{
			name:  "BareHaikuAlias_ClaudeTarget_PassesThrough",
			model: "haiku",
			caps: renderer.CapabilitySet{
				ModelField: true,
			},
			targetName:    "claude",
			expectedModel: "haiku",
			expectedNotes: 1,
			expectedCode:  renderer.CodeFieldTransformed,
		},
		// A mapped xcaffold alias (e.g. "sonnet-4") must still resolve to the
		// target-specific literal and emit no note.
		{
			name:  "MappedAlias_ClaudeTarget_TranslatesToLiteral",
			model: "sonnet-4",
			caps: renderer.CapabilitySet{
				ModelField: true,
			},
			targetName:    "claude",
			expectedModel: "claude-sonnet-4-5",
			expectedNotes: 0,
		},
		// A native literal passed directly must pass through with an info note
		// regardless of target.
		{
			name:  "NativeLiteral_ClaudeTarget_PassesThrough",
			model: "claude-sonnet-4-5",
			caps: renderer.CapabilitySet{
				ModelField: true,
			},
			targetName:    "claude",
			expectedModel: "claude-sonnet-4-5",
			expectedNotes: 1,
			expectedCode:  renderer.CodeFieldTransformed,
		},
		// Bare Claude alias on a non-Claude target must still be dropped with a
		// warning (existing behavior must not regress).
		{
			name:  "BareClaudeAlias_CursorTarget_Dropped",
			model: "sonnet",
			caps: renderer.CapabilitySet{
				ModelField: true,
			},
			targetName:    "cursor",
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

// TestSanitizeAgentModel_RegistryLookup verifies that SanitizeAgentModel processes
// the model normally for targets where the registry marks model as "optional".
func TestSanitizeAgentModel_RegistryLookup(t *testing.T) {
	caps := renderer.CapabilitySet{ModelField: true}

	// claude has model: optional in the schema registry — mapped alias resolves cleanly.
	gotModel, gotNotes := renderer.SanitizeAgentModel("sonnet-4", caps, "claude", "my-agent")
	assert.Equal(t, "claude-sonnet-4-5", gotModel,
		"registry lookup: claude target with mapped alias must resolve to provider literal")
	assert.Empty(t, gotNotes,
		"registry lookup: clean alias resolution must emit no fidelity notes")

	// gemini has model: optional in the schema registry — mapped alias resolves cleanly.
	gotModel, gotNotes = renderer.SanitizeAgentModel("sonnet-4", caps, "gemini", "my-agent")
	assert.Equal(t, "gemini-2.5-flash", gotModel,
		"registry lookup: gemini target with mapped alias must resolve to provider literal")
	assert.Empty(t, gotNotes,
		"registry lookup: clean alias resolution must emit no fidelity notes")
}
