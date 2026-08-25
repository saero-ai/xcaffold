package antigravity_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/renderer"
	"github.com/saero-ai/xcaffold/providers/antigravity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func findAgNote(notes []renderer.FidelityNote, code, field string) (renderer.FidelityNote, bool) {
	for _, n := range notes {
		if n.Code == code && (field == "" || n.Field == field) {
			return n, true
		}
	}
	return renderer.FidelityNote{}, false
}

// ─── Target / OutputDir / Capabilities ────────────────────────────────────────

func TestRenderer_Target(t *testing.T) {
	r := antigravity.New()
	assert.Equal(t, "antigravity", r.Target())
}

func TestRenderer_OutputDir(t *testing.T) {
	r := antigravity.New()
	assert.Equal(t, ".agents", r.OutputDir())
}

func TestRenderer_Capabilities(t *testing.T) {
	r := antigravity.New()
	caps := r.Capabilities()
	assert.True(t, caps.Agents)
	assert.True(t, caps.Skills)
	assert.True(t, caps.Rules)
	assert.True(t, caps.Workflows)
	assert.True(t, caps.Hooks)
	assert.True(t, caps.MCP)
	assert.True(t, caps.Settings)
	assert.False(t, caps.Memory)
	assert.True(t, caps.ProjectInstructions)
}

// ─── Rule tests ───────────────────────────────────────────────────────────────

func TestCompile_Rule_OutputPathIsMarkdown(t *testing.T) {
	r := antigravity.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"my-rule": {
					Description: "A rule",
					Body:        "Always format with gofmt.",
				},
			},
		},
	}

	out, _, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)

	_, ok := out.Files["rules/my-rule.md"]
	assert.True(t, ok, "expected rules/my-rule.md in output")
}

func TestCompile_Rule_NoFrontmatterDelimiters_WhenEmptyDescAndAlways(t *testing.T) {
	r := antigravity.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"plain-rule": {
					Body: "Be concise.",
				},
			},
		},
	}

	out, _, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)

	content := out.Files["rules/plain-rule.md"]
	require.NotEmpty(t, content)

	assert.False(t, strings.HasPrefix(content, "---"), "AG rules with no description and always-on must not start with --- frontmatter delimiter")
	assert.NotContains(t, content, "---", "AG rules with no description and always-on must contain no --- delimiters")
}

func TestCompile_Rule_DescriptionInFrontmatter(t *testing.T) {
	r := antigravity.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"desc-rule": {
					Description: "My Rule Description",
					Body:        "Do something important.",
				},
			},
		},
	}

	out, _, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)

	content := out.Files["rules/desc-rule.md"]
	require.NotEmpty(t, content)

	assert.True(t, strings.HasPrefix(content, "---\n"), "AG rule with description must start with frontmatter delimiter")
	assert.Contains(t, content, "description: My Rule Description", "description must appear as YAML frontmatter field")
	assert.NotContains(t, content, "# My Rule Description", "description must NOT appear as a markdown heading")
}

func TestCompile_Rule_GlobActivation(t *testing.T) {
	r := antigravity.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"path-rule": {
					Description: "A rule with paths",
					Paths:       ast.ClearableList{Values: []string{"**/*.go", "**/*.ts"}},
					Activation:  ast.RuleActivationPathGlob,
					Body:        "Check Go and TS files.",
				},
			},
		},
	}

	out, _, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)

	content := out.Files["rules/path-rule.md"]
	require.NotEmpty(t, content)

	assert.Contains(t, content, "trigger: glob", "PathGlob activation must emit trigger: glob")
	assert.Contains(t, content, "globs: **/*.go, **/*.ts", "PathGlob activation must emit comma-separated globs")
}

func TestCompile_Rule_ModelDecidedActivation(t *testing.T) {
	r := antigravity.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"smart-rule": {
					Description: "Applied when relevant.",
					Body:        "Be context-aware.",
					Activation:  ast.RuleActivationModelDecided,
				},
			},
		},
	}

	out, _, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)

	content := out.Files["rules/smart-rule.md"]
	assert.Contains(t, content, "trigger: model_decision\n", "ModelDecided activation must emit 'trigger: model_decision'")
	assert.NotContains(t, content, "trigger: glob", "ModelDecided must not emit glob trigger")
}

func TestCompile_Rule_ManualMention_FidelityNote(t *testing.T) {
	r := antigravity.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"manual-rule": {
					Description: "Only when mentioned.",
					Body:        "Only on explicit request.",
					Activation:  ast.RuleActivationManualMention,
				},
			},
		},
	}

	_, notes, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)

	note, found := findAgNote(notes, renderer.CodeRuleActivationUnsupported, "activation")
	assert.True(t, found, "ManualMention must emit CodeRuleActivationUnsupported fidelity note")
	assert.Equal(t, renderer.LevelInfo, note.Level)
}

func TestCompile_Rule_12KCharacterLimitWarning(t *testing.T) {
	r := antigravity.New()
	longBody := strings.Repeat("a", 12001)
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"long-rule": {
					Body: longBody,
				},
			},
		},
	}

	out, _, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)

	content := out.Files["rules/long-rule.md"]
	assert.True(t, strings.HasPrefix(content, "<!--"), "long rule must begin with warning HTML comment")
	assert.Contains(t, content, "12000", "warning must mention the 12000-char limit")
}

func TestCompile_Rule_NestedDirectoryPathPreserved(t *testing.T) {
	r := antigravity.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"backend/api-conventions": {
					Description: "Backend API Conventions",
					Paths:       ast.ClearableList{Values: []string{"**/backend/**/*.ts"}},
					Activation:  ast.RuleActivationPathGlob,
					Body:        "# API Conventions\n\nUse camelCase for responses.",
				},
				"backend/architecture": {
					Description: "Backend Architecture",
					Paths:       ast.ClearableList{Values: []string{"**/backend/**/*.ts"}},
					Activation:  ast.RuleActivationPathGlob,
					Body:        "# Architecture\n\nModular design.",
				},
			},
		},
	}

	out, _, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)

	apiContent, ok := out.Files["rules/backend/api-conventions.md"]
	require.True(t, ok, "expected rules/backend/api-conventions.md in output")
	assert.Contains(t, apiContent, "description: Backend API Conventions")
	assert.Contains(t, apiContent, "trigger: glob")
	assert.Contains(t, apiContent, "Use camelCase for responses.")

	archContent, ok := out.Files["rules/backend/architecture.md"]
	require.True(t, ok, "expected rules/backend/architecture.md in output")
	assert.Contains(t, archContent, "description: Backend Architecture")
	assert.Contains(t, archContent, "trigger: glob")
	assert.Contains(t, archContent, "Modular design.")
}

// ─── Agent tests ─────────────────────────────────────────────────────────────

func TestCompile_Agent_NativeMarkdownFile(t *testing.T) {
	r := antigravity.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"coder": {
					Name:        "Code Assistant",
					Description: "Writes clean code.",
					Model:       "gemini-3.1-pro",
					Tools:       ast.ClearableList{Values: []string{"run_command", "view_file"}},
					Body:        "You are an expert coder.",
				},
			},
		},
	}

	out, _, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)

	content, ok := out.Files["agents/coder.md"]
	require.True(t, ok, "expected agents/coder.md in output files")
	assert.Contains(t, content, "name: Code Assistant")
	assert.Contains(t, content, "description: Writes clean code.")
	assert.Contains(t, content, "model: gemini-3.1-pro-high")
	assert.Contains(t, content, "- run_command")
	assert.Contains(t, content, "- view_file")
	assert.Contains(t, content, "mainAgent: true")
	assert.Contains(t, content, "subagent: true")
	assert.Contains(t, content, "commandExecutionPolicy: sandbox")
	assert.Contains(t, content, "You are an expert coder.")
}

func TestCompile_Agent_ClaudeNativeTools_SanitizedAndDropped(t *testing.T) {
	r := antigravity.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"researcher": {
					Name:        "Researcher",
					Description: "Investigates codebase.",
					Tools:       ast.ClearableList{Values: []string{"Read", "Write", "Edit", "Glob", "Grep", "Bash"}},
					Body:        "You are a researcher.",
				},
			},
		},
	}

	out, notes, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)

	content, ok := out.Files["agents/researcher.md"]
	require.True(t, ok, "expected agents/researcher.md in output files")

	assert.NotContains(t, content, "tools:\n", "Claude-native tools must be sanitized away, omitting tools: key")
	assert.NotContains(t, content, "- Read")
	assert.NotContains(t, content, "- Write")
	assert.NotContains(t, content, "- Edit")
	assert.NotContains(t, content, "- Glob")
	assert.NotContains(t, content, "- Grep")
	assert.NotContains(t, content, "- Bash")

	note, found := findAgNote(notes, renderer.CodeAgentToolsDropped, "tools")
	assert.True(t, found, "expected CodeAgentToolsDropped note for sanitized Claude tools")
	assert.Equal(t, "researcher", note.Resource)
}

func TestCompile_Agent_MixedTools_ClaudeDroppedMCPAndNativePreserved(t *testing.T) {
	r := antigravity.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"hybrid": {
					Name:        "Hybrid Agent",
					Description: "Uses mixed tools.",
					Tools:       ast.ClearableList{Values: []string{"Read", "view_file", "mcp_custom_tool"}},
					Body:        "You are a hybrid agent.",
				},
			},
		},
	}

	out, notes, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)

	content, ok := out.Files["agents/hybrid.md"]
	require.True(t, ok, "expected agents/hybrid.md in output files")

	assert.Contains(t, content, "tools:\n")
	assert.Contains(t, content, "  - view_file")
	assert.Contains(t, content, "  - mcp_custom_tool")
	assert.NotContains(t, content, "Read")

	note, found := findAgNote(notes, renderer.CodeAgentToolsDropped, "tools")
	assert.True(t, found, "expected CodeAgentToolsDropped note for dropped Read tool")
	assert.Equal(t, "hybrid", note.Resource)
}

// ─── Skill tests ──────────────────────────────────────────────────────────────

func TestCompile_Skill_OutputAtCorrectPath(t *testing.T) {
	r := antigravity.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Skills: map[string]ast.SkillConfig{
				"my-skill": {
					Name:        "My Skill",
					Description: "A test skill",
					Body:        "Do the skill thing.",
				},
			},
		},
	}

	out, _, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)

	_, ok := out.Files["skills/my-skill/SKILL.md"]
	assert.True(t, ok, "expected skills/my-skill/SKILL.md in output")
}

func TestCompile_Skill_FrontmatterHasProgressiveDisclosureFields(t *testing.T) {
	r := antigravity.New()
	userInvocable := true
	disableModelInvocation := false
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Skills: map[string]ast.SkillConfig{
				"fmt-skill": {
					Name:                   "Format Skill",
					Description:            "Formats code",
					WhenToUse:              "When code needs formatting",
					License:                "Apache-2.0",
					AllowedTools:           ast.ClearableList{Values: []string{"run_command"}},
					UserInvocable:          &userInvocable,
					DisableModelInvocation: &disableModelInvocation,
					ArgumentHint:           "[path]",
					Body:                   "Run gofmt first.",
				},
			},
		},
	}

	out, _, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)

	content := out.Files["skills/fmt-skill/SKILL.md"]
	require.NotEmpty(t, content)

	assert.Contains(t, content, "name: Format Skill")
	assert.Contains(t, content, "description: Formats code")
	assert.Contains(t, content, "when-to-use: When code needs formatting")
	assert.Contains(t, content, "license: Apache-2.0")
	assert.Contains(t, content, "- run_command")
	assert.Contains(t, content, "user-invocable: true")
	assert.Contains(t, content, "disable-model-invocation: false")
	assert.Contains(t, content, "argument-hint: \"[path]\"")
	assert.Contains(t, content, "Run gofmt first.")
}

func TestCompile_Skill_References_CompiledToExamples(t *testing.T) {
	tmpDir := t.TempDir()
	skillBase := filepath.Join(tmpDir, "xcaf", "skills", "test-skill")
	require.NoError(t, os.MkdirAll(filepath.Join(skillBase, "references"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(skillBase, "references", "doc.md"), []byte("# Doc"), 0o644))

	r := antigravity.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Skills: map[string]ast.SkillConfig{
				"test-skill": {
					Name:        "test-skill",
					Description: "A skill with references",
					Body:        "Do things.",
					Artifacts:   []string{"references"},
				},
			},
		},
	}
	files, notes, err := renderer.Orchestrate(r, config, tmpDir)
	require.NoError(t, err)

	_, ok := files.Files["skills/test-skill/examples/doc.md"]
	assert.True(t, ok, "expected references compiled to examples/ subdirectory")

	for _, n := range notes {
		assert.NotEqual(t, renderer.CodeSkillReferencesDropped, n.Code, "references must not produce a drop note")
	}
}

// ─── Hooks tests ─────────────────────────────────────────────────────────────

func TestCompile_Hooks_EmittedToHooksJSON(t *testing.T) {
	r := antigravity.New()
	config := &ast.XcaffoldConfig{
		Hooks: map[string]ast.NamedHookConfig{
			"default": {
				Name: "default",
				Events: ast.HookConfig{
					"PreToolUse": []ast.HookMatcherGroup{
						{
							Hooks: []ast.HookHandler{
								{Type: "command", Command: "echo pre-tool"},
							},
						},
					},
					"PostInvocation": []ast.HookMatcherGroup{
						{
							Hooks: []ast.HookHandler{
								{Type: "command", Command: "echo post-invocation"},
							},
						},
					},
				},
			},
		},
	}

	out, notes, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)
	require.Empty(t, notes)

	content, ok := out.Files["hooks.json"]
	require.True(t, ok, "expected hooks.json in output files")

	var parsed ast.HookConfig
	require.NoError(t, json.Unmarshal([]byte(content), &parsed))
	assert.Len(t, parsed["PreToolUse"], 1)
	assert.Len(t, parsed["PostInvocation"], 1)
}

// ─── MCP tests ───────────────────────────────────────────────────────────────

func TestCompile_MCP_EmittedToMCPConfigFile(t *testing.T) {
	r := antigravity.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			MCP: map[string]ast.MCPConfig{
				"db-server": {
					Command:       "npx",
					Args:          []string{"-y", "@modelcontextprotocol/server-postgres"},
					URL:           "http://localhost:8000/sse",
					DisabledTools: []string{"drop_database"},
				},
			},
		},
	}

	out, notes, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)
	require.Empty(t, notes)

	content, ok := out.Files["mcp_config.json"]
	require.True(t, ok, "expected mcp_config.json in output files")

	assert.Contains(t, content, "db-server")
	assert.Contains(t, content, "http://localhost:8000/sse")
	assert.Contains(t, content, "drop_database")
}

// ─── Memory tests ────────────────────────────────────────────────────────────

func TestCompile_Memory_Unsupported_EmitsFidelityNote(t *testing.T) {
	r := antigravity.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Memory: map[string]ast.MemoryConfig{
				"user-profile": {
					Name:    "user-profile",
					Content: "User preferences.",
				},
			},
		},
	}

	out, notes, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)

	for k := range out.Files {
		assert.False(t, strings.HasPrefix(k, "knowledge/"), "memory files must not be emitted")
		assert.False(t, strings.HasPrefix(k, "memory/"), "memory files must not be emitted")
	}

	note, found := findAgNote(notes, renderer.CodeRendererKindUnsupported, "")
	assert.True(t, found, "expected CodeRendererKindUnsupported note for memory")
	assert.Equal(t, "memory", note.Kind)
	assert.Equal(t, "user-profile", note.Resource)
}

// ─── Workflows tests ─────────────────────────────────────────────────────────

func TestCompile_Workflow_EmittedToWorkflows(t *testing.T) {
	r := antigravity.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Workflows: map[string]ast.WorkflowConfig{
				"release": {
					Name:        "release",
					Description: "Release workflow",
					Steps: []ast.WorkflowStep{
						{Name: "build", Instructions: "Run make build"},
						{Name: "test", Instructions: "Run make test"},
					},
				},
			},
		},
	}

	out, _, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)

	content, ok := out.Files["workflows/release.md"]
	require.True(t, ok, "expected workflows/release.md in output files")
	assert.Contains(t, content, "description: Release workflow")
	assert.Contains(t, content, "## build")
	assert.Contains(t, content, "Run make build")
	assert.Contains(t, content, "## test")
	assert.Contains(t, content, "Run make test")
}

// ─── Project Context tests ───────────────────────────────────────────────────

func TestCompile_ProjectInstructions_EmittedToRootGEMINIMD(t *testing.T) {
	r := antigravity.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Contexts: map[string]ast.ContextConfig{
				"default": {
					Body: "Project overview and guidelines.",
				},
			},
		},
	}

	out, _, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)

	content, ok := out.RootFiles["GEMINI.md"]
	require.True(t, ok, "expected GEMINI.md in root files")
	assert.Equal(t, "Project overview and guidelines.", content)
}
