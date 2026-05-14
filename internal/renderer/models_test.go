package renderer_test

import (
	"testing"

	"github.com/saero-ai/xcaffold/internal/renderer"
	"github.com/stretchr/testify/assert"
)

func TestIsMappedModel_KnownAlias_Claude_ReturnsTrue(t *testing.T) {
	assert.True(t, renderer.IsMappedModel("balanced", "claude"),
		"balanced is a known alias with a claude mapping")
}

func TestIsMappedModel_KnownAlias_Cursor_ReturnsTrue(t *testing.T) {
	assert.True(t, renderer.IsMappedModel("balanced", "cursor"),
		"balanced has a cursor mapping — translated to a canonical model string")
}

func TestIsMappedModel_LiteralModelID_Claude_ReturnsFalse(t *testing.T) {
	assert.False(t, renderer.IsMappedModel("claude-sonnet-4-5", "claude"),
		"literal model IDs are not xcaffold-mapped aliases — only flagship, balanced, fast are mapped")
}

func TestIsMappedModel_LiteralModelID_Cursor_ReturnsFalse(t *testing.T) {
	assert.False(t, renderer.IsMappedModel("claude-sonnet-4-5", "cursor"),
		"literal model ID on cursor target must return false because it's not a xcaffold alias")
}

func TestIsMappedModel_EmptyAlias_ReturnsFalse(t *testing.T) {
	assert.False(t, renderer.IsMappedModel("", "claude"),
		"empty alias is never mapped")
}

func TestIsMappedModel_Antigravity_ReturnsFalse(t *testing.T) {
	assert.False(t, renderer.IsMappedModel("balanced", "antigravity"),
		"antigravity has no model support — always false")
}

func TestResolveModel_GeminiTarget_TranslatesBalanced(t *testing.T) {
	model, ok := renderer.ResolveModel("balanced", "gemini")
	if !ok {
		t.Fatal("expected ok=true for gemini target with balanced alias")
	}
	if model == "balanced" {
		t.Errorf("expected translated model for gemini, got raw alias %q", model)
	}
}

func TestResolveModel_CopilotTarget_TranslatesBalanced(t *testing.T) {
	model, ok := renderer.ResolveModel("balanced", "copilot")
	if !ok {
		t.Fatal("expected ok=true for copilot target with balanced alias")
	}
	if model == "balanced" {
		t.Errorf("expected translated model for copilot, got raw alias %q", model)
	}
}

func TestResolveModel_GeminiTarget_TranslatesFlagship(t *testing.T) {
	model, ok := renderer.ResolveModel("flagship", "gemini")
	if !ok {
		t.Fatal("expected ok=true for gemini target with flagship alias")
	}
	if model == "flagship" {
		t.Errorf("expected translated model for gemini, got raw alias %q", model)
	}
}

func TestResolveModel_CursorTarget_ReturnsMapped(t *testing.T) {
	model, ok := renderer.ResolveModel("balanced", "cursor")
	if !ok {
		t.Fatal("expected ok=true for cursor target with balanced alias")
	}
	if model == "balanced" {
		t.Errorf("expected translated model for cursor, got raw alias %q", model)
	}
}

func TestResolveModel_UnknownAlias_GeminiRejectsNonGemini(t *testing.T) {
	// Gemini resolver only accepts "gemini-*" prefixed models or mapped aliases
	model, ok := renderer.ResolveModel("custom-model-v3", "gemini")
	if ok {
		t.Fatalf("expected ok=false for gemini target with custom non-gemini model, got %q", model)
	}
}

func TestResolveModel_GeminiLiteralModel_Accepted(t *testing.T) {
	// But Gemini accepts full gemini- prefixed IDs
	model, ok := renderer.ResolveModel("gemini-2.5-pro", "gemini")
	if !ok {
		t.Fatal("expected ok=true for gemini target with gemini- prefixed model")
	}
	if model != "gemini-2.5-pro" {
		t.Errorf("expected literal passthrough for gemini model, got %q", model)
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
		// The schema registry gates model support per target.
		// "cursor" has model: optional in the registry, so it passes through.
		// A bare Claude alias on a non-Claude target emits a warning and is dropped.
		{
			name:          "RegistryOptional_CursorTarget_BareAliasDrop",
			model:         "sonnet",
			caps:          renderer.CapabilitySet{},
			targetName:    "cursor",
			expectedModel: "",
			expectedNotes: 1,
			expectedCode:  renderer.CodeAgentModelUnmapped,
		},
		{
			name:          "MappedAlias_TranslatesCleanly",
			model:         "balanced",
			caps:          renderer.CapabilitySet{},
			targetName:    "gemini",
			expectedModel: "gemini-2.5-flash",
			expectedNotes: 0,
		},
		{
			name:          "BareClaudeAlias_EmitsWarning",
			model:         "sonnet",
			caps:          renderer.CapabilitySet{},
			targetName:    "gemini",
			expectedModel: "",
			expectedNotes: 1,
			expectedCode:  renderer.CodeAgentModelUnmapped,
		},
		// Bare Claude aliases are valid Claude Code native aliases and must pass
		// through as-is when the target is "claude". Ground truth (models.json,
		// verified 2026-04-30): "sonnet", "opus", "haiku" are documented aliases.
		{
			name:          "BareClaudeAlias_ClaudeTarget_PassesThrough",
			model:         "sonnet",
			caps:          renderer.CapabilitySet{},
			targetName:    "claude",
			expectedModel: "sonnet",
			expectedNotes: 1,
			expectedCode:  renderer.CodeFieldTransformed,
		},
		{
			name:          "BareOpusAlias_ClaudeTarget_PassesThrough",
			model:         "opus",
			caps:          renderer.CapabilitySet{},
			targetName:    "claude",
			expectedModel: "opus",
			expectedNotes: 1,
			expectedCode:  renderer.CodeFieldTransformed,
		},
		{
			name:          "BareHaikuAlias_ClaudeTarget_PassesThrough",
			model:         "haiku",
			caps:          renderer.CapabilitySet{},
			targetName:    "claude",
			expectedModel: "haiku",
			expectedNotes: 1,
			expectedCode:  renderer.CodeFieldTransformed,
		},
		// A mapped xcaffold alias (e.g. "balanced") must still resolve to the
		// target-specific literal and emit no note.
		{
			name:          "MappedAlias_ClaudeTarget_TranslatesToLiteral",
			model:         "balanced",
			caps:          renderer.CapabilitySet{},
			targetName:    "claude",
			expectedModel: "claude-sonnet-4-5",
			expectedNotes: 0,
		},
		// A native literal passed directly must pass through with an info note
		// regardless of target.
		{
			name:          "NativeLiteral_ClaudeTarget_PassesThrough",
			model:         "claude-sonnet-4-5",
			caps:          renderer.CapabilitySet{},
			targetName:    "claude",
			expectedModel: "claude-sonnet-4-5",
			expectedNotes: 1,
			expectedCode:  renderer.CodeFieldTransformed,
		},
		// Bare Claude alias on a non-Claude target must still be dropped with a
		// warning (existing behavior must not regress).
		{
			name:          "BareClaudeAlias_CursorTarget_Dropped",
			model:         "sonnet",
			caps:          renderer.CapabilitySet{},
			targetName:    "cursor",
			expectedModel: "",
			expectedNotes: 1,
			expectedCode:  renderer.CodeAgentModelUnmapped,
		},
		{
			name:          "NativeLiteral_PassesThroughWithInfo",
			model:         "gemini-2.5-pro",
			caps:          renderer.CapabilitySet{},
			targetName:    "gemini",
			expectedModel: "gemini-2.5-pro",
			expectedNotes: 1,
			expectedCode:  renderer.CodeFieldTransformed,
		},
		{
			name:          "EmptyModel_QuickReturn",
			model:         "",
			caps:          renderer.CapabilitySet{},
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
	caps := renderer.CapabilitySet{}

	// claude has model: optional in the schema registry — mapped alias resolves cleanly.
	gotModel, gotNotes := renderer.SanitizeAgentModel("balanced", caps, "claude", "my-agent")
	assert.Equal(t, "claude-sonnet-4-5", gotModel,
		"registry lookup: claude target with mapped alias must resolve to provider literal")
	assert.Empty(t, gotNotes,
		"registry lookup: clean alias resolution must emit no fidelity notes")

	// gemini has model: optional in the schema registry — mapped alias resolves cleanly.
	gotModel, gotNotes = renderer.SanitizeAgentModel("balanced", caps, "gemini", "my-agent")
	assert.Equal(t, "gemini-2.5-flash", gotModel,
		"registry lookup: gemini target with mapped alias must resolve to provider literal")
	assert.Empty(t, gotNotes,
		"registry lookup: clean alias resolution must emit no fidelity notes")
}

func TestIsMappedModel_NewTierAlias_Flagship(t *testing.T) {
	assert.True(t, renderer.IsMappedModel("flagship", "claude"),
		"flagship must be a mapped xcaffold alias")
}

func TestIsMappedModel_NewTierAlias_Balanced(t *testing.T) {
	assert.True(t, renderer.IsMappedModel("balanced", "claude"),
		"balanced must be a mapped xcaffold alias")
}

func TestIsMappedModel_NewTierAlias_Fast(t *testing.T) {
	assert.True(t, renderer.IsMappedModel("fast", "claude"),
		"fast must be a mapped xcaffold alias")
}

func TestIsMappedModel_OldAlias_SonnetRejected(t *testing.T) {
	assert.False(t, renderer.IsMappedModel("sonnet-4", "claude"),
		"old alias sonnet-4 must NOT be a mapped alias after rename")
}

func TestIsMappedModel_OldAlias_OpusRejected(t *testing.T) {
	assert.False(t, renderer.IsMappedModel("opus-4", "claude"),
		"old alias opus-4 must NOT be a mapped alias after rename")
}

func TestIsMappedModel_OldAlias_HaikuRejected(t *testing.T) {
	assert.False(t, renderer.IsMappedModel("haiku-3.5", "claude"),
		"old alias haiku-3.5 must NOT be a mapped alias after rename")
}
