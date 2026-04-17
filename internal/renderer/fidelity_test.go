package renderer_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/renderer"
	"github.com/saero-ai/xcaffold/internal/renderer/antigravity"
	"github.com/saero-ai/xcaffold/internal/renderer/claude"
	"github.com/saero-ai/xcaffold/internal/renderer/cursor"
	"github.com/saero-ai/xcaffold/internal/renderer/gemini"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFidelityNote_JSON_RoundTrip(t *testing.T) {
	note := renderer.NewNote(
		renderer.LevelWarning,
		"cursor",
		"agent",
		"code-review",
		"permissionMode",
		"AGENT_SECURITY_FIELDS_DROPPED",
		"permissionMode has no Cursor equivalent and was dropped",
		"Remove permissionMode from the cursor target override",
	)

	data, err := json.Marshal(note)
	require.NoError(t, err)

	var got renderer.FidelityNote
	require.NoError(t, json.Unmarshal(data, &got))

	assert.Equal(t, renderer.LevelWarning, got.Level)
	assert.Equal(t, "cursor", got.Target)
	assert.Equal(t, "agent", got.Kind)
	assert.Equal(t, "code-review", got.Resource)
	assert.Equal(t, "permissionMode", got.Field)
	assert.Equal(t, "AGENT_SECURITY_FIELDS_DROPPED", got.Code)
	assert.Equal(t, "permissionMode has no Cursor equivalent and was dropped", got.Reason)
	assert.Equal(t, "Remove permissionMode from the cursor target override", got.Mitigation)
}

func TestFidelityNote_JSON_OmitsEmptyField(t *testing.T) {
	note := renderer.NewNote(
		renderer.LevelInfo,
		"claude",
		"settings",
		"global",
		"",
		"FIELD_TRANSFORMED",
		"mcpServers merged from settings block",
		"",
	)

	data, err := json.Marshal(note)
	require.NoError(t, err)

	assert.NotContains(t, string(data), `"field"`)
	assert.NotContains(t, string(data), `"mitigation"`)
}

func TestFidelityNote_AllCodes_NoBlanks(t *testing.T) {
	for _, code := range renderer.AllCodes() {
		assert.NotEmpty(t, code, "catalog entry must not be blank")
	}
}

func TestFidelityNote_AllCodes_Unique(t *testing.T) {
	seen := make(map[string]int)
	for _, code := range renderer.AllCodes() {
		seen[code]++
	}
	for code, count := range seen {
		assert.Equal(t, 1, count, "catalog code %q appears %d times; codes must be unique", code, count)
	}
}

// TestFidelityNote_AllCodes_ReferencedByConstant asserts every entry in
// AllCodes() matches an exported Code* constant. This catches the class
// of drift where a new constant is added but not added to the slice
// (or vice-versa), which a simple length assertion would miss.
func TestFidelityNote_AllCodes_ReferencedByConstant(t *testing.T) {
	expected := map[string]bool{
		renderer.CodeRendererKindUnsupported:             true,
		renderer.CodeFieldUnsupported:                    true,
		renderer.CodeFieldTransformed:                    true,
		renderer.CodeActivationDegraded:                  true,
		renderer.CodeInstructionsFlattened:               true,
		renderer.CodeInstructionsClosestWinsForcedConcat: true,
		renderer.CodeMemoryNoNativeTarget:                true,
		renderer.CodeMemoryPartialFidelity:               true,
		renderer.CodeMemoryBodyEmpty:                     true,
		renderer.CodeMemorySeedSkipped:                   true,
		renderer.CodeMemoryIndexUpdateFailed:             true,
		renderer.CodeWorkflowLoweredToRulePlusSkill:      true,
		renderer.CodeWorkflowLoweredToPromptFile:         true,
		renderer.CodeWorkflowLoweredToCustomCommand:      true,
		renderer.CodeWorkflowLoweredToNative:             true,
		renderer.CodeWorkflowNoNativeTarget:              true,
		renderer.CodeReservedOutputPathRejected:          true,
		renderer.CodeSettingsFieldUnsupported:            true,
		renderer.CodeHookInterpolationRequiresEnvSyntax:  true,
		renderer.CodeAgentModelUnmapped:                  true,
		renderer.CodeAgentSecurityFieldsDropped:          true,
		renderer.CodeSkillScriptsDropped:                 true,
		renderer.CodeSkillAssetsDropped:                  true,
		renderer.CodeRuleActivationUnsupported:           true,
		renderer.CodeRuleExcludeAgentsDropped:            true,
		renderer.CodeInstructionsImportInlined:           true,
		renderer.CodeReconciliationUnionLossy:            true,
		renderer.CodeReconciliationDriftDetected:         true,
		renderer.CodeMemoryDriftDetected:                 true,
		renderer.CodeOptimizerPassReordered:              true,
	}

	got := make(map[string]bool)
	for _, code := range renderer.AllCodes() {
		got[code] = true
	}

	for code := range expected {
		assert.True(t, got[code], "catalog code %q is declared as a constant but not in AllCodes()", code)
	}
	for code := range got {
		assert.True(t, expected[code], "AllCodes() returns %q which is not declared as an exported constant", code)
	}
	assert.Equal(t, len(expected), len(got), "catalog size mismatch")
}

// TestFidelityNote_EmittedCodes_AreInCatalog dispatches each concrete renderer
// with a fixture that exercises as many fidelity emit sites as possible, then
// asserts that every emitted code is present in AllCodes(). This catches the
// class of bug where a renderer emits a code that was never registered.
func TestFidelityNote_EmittedCodes_AreInCatalog(t *testing.T) {
	baseDir := t.TempDir()
	config := buildFidelityFixture(t, baseDir)

	catalog := make(map[string]bool, len(renderer.AllCodes()))
	for _, c := range renderer.AllCodes() {
		catalog[c] = true
	}

	renderers := []renderer.TargetRenderer{
		claude.New(),
		cursor.New(),
		antigravity.New(),
		gemini.New(),
	}

	for _, r := range renderers {
		r := r
		t.Run(r.Target(), func(t *testing.T) {
			_, notes, err := r.Compile(config, baseDir)
			require.NoError(t, err)

			// The claude renderer is the native target; it intentionally emits no
			// fidelity notes by design. All other renderers must emit at least one.
			if r.Target() != "claude" {
				require.NotEmpty(t, notes,
					"renderer %q returned zero FidelityNotes; the fixture must exercise at least one emit site",
					r.Target())
			}

			for _, note := range notes {
				assert.True(t, catalog[note.Code],
					"renderer %q emitted unknown code %q — add it to AllCodes()", r.Target(), note.Code)
				assert.NotEmpty(t, note.Level, "note with code %q has empty Level", note.Code)
				assert.NotEmpty(t, note.Target, "note with code %q has empty Target", note.Code)
				assert.NotEmpty(t, note.Kind, "note with code %q has empty Kind", note.Code)
				assert.NotEmpty(t, note.Resource, "note with code %q has empty Resource", note.Code)
				assert.NotEmpty(t, note.Reason, "note with code %q has empty Reason", note.Code)
			}
		})
	}
}

// buildFidelityFixture constructs an XcaffoldConfig that exercises the maximum
// number of fidelity emit sites across all three concrete renderers.
//
// Stub files (scripts/helper.sh, assets/logo.png, docs/ref.md) are created in
// baseDir so that the claude renderer — which physically copies skill subfiles —
// does not fail with "file not found" errors. The files contain placeholder text.
//
// Emit sites covered per renderer:
//
//	cursor:
//	  SKILL_SCRIPTS_DROPPED        — skill.Scripts non-empty
//	  SKILL_ASSETS_DROPPED         — skill.Assets non-empty
//	  AGENT_MODEL_UNMAPPED         — agent.Model is unknown (not in modelAliases)
//	  AGENT_SECURITY_FIELDS_DROPPED — agent.PermissionMode, DisallowedTools, Isolation
//	  HOOK_INTERPOLATION_REQUIRES_ENV_SYNTAX — hook command with ${VAR} + MCP command with ${VAR}
//	  SETTINGS_FIELD_UNSUPPORTED   — settings.Permissions + settings.Sandbox
//
//	antigravity:
//	  SKILL_SCRIPTS_DROPPED        — same skill
//	  SKILL_ASSETS_DROPPED         — same skill
//	  AGENT_SECURITY_FIELDS_DROPPED — same agent
//	  HOOK_INTERPOLATION_REQUIRES_ENV_SYNTAX — MCP env with ${VAR}
//	  SETTINGS_FIELD_UNSUPPORTED   — settings.Permissions + settings.Sandbox
//
//	gemini:
//	  SKILL_SCRIPTS_DROPPED        — same skill
//	  SKILL_ASSETS_DROPPED         — same skill
//	  AGENT_SECURITY_FIELDS_DROPPED — agent.Effort, PermissionMode, DisallowedTools, Isolation
//	  FIELD_UNSUPPORTED            — skill.AllowedTools, agent unsupported fields (hooks, memory, etc.)
//
//	claude: (native target — no fidelity notes by design)
func buildFidelityFixture(t *testing.T, baseDir string) *ast.XcaffoldConfig {
	t.Helper()

	// Create stub files that the claude renderer reads when copying skill subfiles.
	stubFiles := map[string]string{
		"scripts/helper.sh": "#!/bin/sh\necho stub\n",
		"assets/logo.png":   "stub-png-data",
		"docs/ref.md":       "# stub reference\n",
	}
	for rel, content := range stubFiles {
		full := filepath.Join(baseDir, rel)
		require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
		require.NoError(t, os.WriteFile(full, []byte(content), 0o644))
	}

	trueVal := true
	permMode := "bypassPermissions"
	isolation := "container"

	return &ast.XcaffoldConfig{
		Settings: ast.SettingsConfig{
			Permissions: &ast.PermissionsConfig{
				Allow: []string{"Bash(*)"},
			},
			Sandbox: &ast.SandboxConfig{
				Enabled: &trueVal,
			},
		},
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"fidelity-agent": {
					Name:            "Fidelity Agent",
					Description:     "triggers security and model fidelity notes",
					Model:           "unknown-model-xyz-not-in-catalog",
					Effort:          "high",
					MaxTurns:        5,
					Mode:            "auto",
					Tools:           []string{"Read", "Write"},
					DisallowedTools: []string{"Bash"},
					Readonly:        &trueVal,
					PermissionMode:  permMode,
					Background:      &trueVal,
					Isolation:       isolation,
					Memory:          "compact",
					Color:           "blue",
					InitialPrompt:   "hello",
					Skills:          []string{"fidelity-skill"},
					Rules:           []string{"fidelity-rule"},
					MCP:             []string{"fidelity-mcp"},
					Assertions:      []string{"assert-something"},
					MCPServers: map[string]ast.MCPConfig{
						"inline-mcp": {Command: "node server.js"},
					},
					Hooks: ast.HookConfig{
						"PreToolUse": {
							{
								Matcher: "Bash",
								Hooks: []ast.HookHandler{
									{Type: "command", Command: "echo ${MY_VAR}"},
								},
							},
						},
					},
					Targets: map[string]ast.TargetOverride{
						"cursor": {},
					},
					Instructions: "Do the thing.",
				},
			},
			Skills: map[string]ast.SkillConfig{
				"fidelity-skill": {
					Name:         "Fidelity Skill",
					Description:  "triggers scripts and assets dropped notes",
					AllowedTools: []string{"Read"},
					References:   []string{"docs/ref.md"},
					Scripts:      []string{"scripts/helper.sh"},
					Assets:       []string{"assets/logo.png"},
					Instructions: "Use this skill.",
				},
			},
			Rules: map[string]ast.RuleConfig{
				"fidelity-rule": {
					Description:  "triggers alwaysApply fidelity note",
					AlwaysApply:  &trueVal,
					Instructions: "Follow this rule.",
				},
			},
			Hooks: ast.HookConfig{
				"PreToolUse": {
					{
						Matcher: "Bash",
						Hooks: []ast.HookHandler{
							{Type: "command", Command: "echo ${MY_ENV_VAR}"},
						},
					},
				},
			},
			MCP: map[string]ast.MCPConfig{
				"fidelity-mcp": {
					Command: "node ${MCP_SERVER_PATH}/index.js",
					Args:    []string{"--port", "${MCP_PORT}"},
					Env: map[string]string{
						"TOKEN": "${API_TOKEN}",
					},
				},
			},
		},
	}
}
