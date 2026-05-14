package codex

import (
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/renderer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCompile_Agents_BasicTOML verifies that an agent with name, description, and body
// produces valid TOML output containing the expected fields.
func TestCompile_Agents_BasicTOML(t *testing.T) {
	r := New()
	agents := map[string]ast.AgentConfig{
		"reviewer": {
			Name:        "Code Reviewer",
			Description: "Code review specialist.",
			Body:        "Review code for correctness and style.\n",
		},
	}

	files, notes, err := r.CompileAgents(agents, "")
	require.NoError(t, err)
	require.NotNil(t, files)
	require.Empty(t, notes)

	content, ok := files["agents/reviewer.toml"]
	require.True(t, ok, "expected agents/reviewer.toml in output")

	// Verify TOML contains expected fields
	assert.Contains(t, content, "name = \"Code Reviewer\"")
	assert.Contains(t, content, "description = \"Code review specialist.\"")
	assert.Contains(t, content, "developer_instructions = \"Review code for correctness and style.\"")

	// Verify it parses as valid TOML
	var cfg codexAgent
	err = toml.Unmarshal([]byte(content), &cfg)
	require.NoError(t, err)
	assert.Equal(t, "Code Reviewer", cfg.Name)
	assert.Equal(t, "Code review specialist.", cfg.Description)
}

// TestCompile_Agents_ModelResolution verifies that xcaffold model aliases
// are resolved to Codex model IDs (e.g., "sonnet-4" → "gpt-5.4").
func TestCompile_Agents_ModelResolution(t *testing.T) {
	tests := []struct {
		name          string
		alias         string
		expectedModel string
	}{
		{"sonnet-4 alias", "sonnet-4", "gpt-5.4"},
		{"opus-4 alias", "opus-4", "gpt-5.5"},
		{"haiku-3.5 alias", "haiku-3.5", "gpt-5.4-mini"},
		{"native gpt-5.4", "gpt-5.4", "gpt-5.4"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := New()
			agents := map[string]ast.AgentConfig{
				"test": {
					Name:        "Test Agent",
					Description: "Test.",
					Body:        "Test body.",
					Model:       tt.alias,
				},
			}

			files, _, err := r.CompileAgents(agents, "")
			require.NoError(t, err)

			content := files["agents/test.toml"]
			require.NotEmpty(t, content)

			var cfg codexAgent
			err = toml.Unmarshal([]byte(content), &cfg)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedModel, cfg.Model, "model %q should resolve to %q", tt.alias, tt.expectedModel)
		})
	}
}

// TestCompile_Agents_EmptyMap verifies that an empty agents map produces
// an empty output with no error.
func TestCompile_Agents_EmptyMap(t *testing.T) {
	r := New()
	agents := map[string]ast.AgentConfig{}

	files, notes, err := r.CompileAgents(agents, "")
	require.NoError(t, err)
	assert.Empty(t, files)
	assert.Empty(t, notes)
}

// TestCompile_Agents_UnsupportedFields verifies that unsupported agent fields
// (tools, memory, hooks, etc.) generate fidelity notes and are dropped from output.
func TestCompile_Agents_UnsupportedFields(t *testing.T) {
	r := New()
	agents := map[string]ast.AgentConfig{
		"test": {
			Name:        "Test",
			Description: "Test.",
			Body:        "Test.",
			Tools:       ast.ClearableList{Values: []string{"Bash", "Read"}},
			Memory:      ast.FlexStringSlice{"project"},
			Hooks:       map[string][]ast.HookMatcherGroup{"PreToolUse": {}},
		},
	}

	files, notes, err := r.CompileAgents(agents, "")
	require.NoError(t, err)

	// Verify fidelity notes were generated for unsupported fields
	require.NotEmpty(t, notes)
	assert.True(t, len(notes) >= 3, "expected at least 3 fidelity notes for unsupported fields")

	// Verify notes are warnings
	for _, note := range notes {
		assert.Equal(t, renderer.LevelWarning, note.Level)
		assert.Equal(t, "codex", note.Target)
		assert.Equal(t, "agent", note.Kind)
		assert.Equal(t, "test", note.Resource)
		assert.Equal(t, renderer.CodeFieldUnsupported, note.Code)
	}

	// Verify fields are not in output
	content := files["agents/test.toml"]
	require.NotEmpty(t, content)
	assert.NotContains(t, content, "tools")
	assert.NotContains(t, content, "memory")
	assert.NotContains(t, content, "hooks")

	// Verify supported fields are still present
	assert.Contains(t, content, "name = \"Test\"")
	assert.Contains(t, content, "description = \"Test.\"")
}

// TestCompile_Skills_BasicSKILLMD verifies that a skill with name, description, and body
// produces a SKILL.md file with frontmatter and body.
func TestCompile_Skills_BasicSKILLMD(t *testing.T) {
	r := New()
	skills := map[string]ast.SkillConfig{
		"tdd": {
			Name:        "tdd-driven-development",
			Description: "Test-driven development workflow.",
			Body:        "Write tests before implementation.\n",
		},
	}

	files, notes, err := r.CompileSkills(skills, "")
	require.NoError(t, err)
	require.Empty(t, notes)

	content, ok := files["skills/tdd/SKILL.md"]
	require.True(t, ok, "expected skills/tdd/SKILL.md in output")

	// Verify frontmatter
	assert.Contains(t, content, "---\n")
	assert.Contains(t, content, "name: tdd-driven-development")
	assert.Contains(t, content, "description: Test-driven development workflow.")

	// Verify body
	assert.Contains(t, content, "Write tests before implementation.")

	// Verify structure: frontmatter must come before body
	frontmatterEnd := strings.LastIndex(content, "---\n")
	bodyStart := strings.Index(content, "Write tests")
	assert.True(t, frontmatterEnd < bodyStart, "frontmatter must end before body starts")
}

// TestCompile_Skills_OutputPath verifies that skill files are placed in the
// "skills/" directory (before Finalize moves them to rootFiles).
func TestCompile_Skills_OutputPath(t *testing.T) {
	r := New()
	skills := map[string]ast.SkillConfig{
		"test-skill": {
			Name:        "test-skill",
			Description: "Test skill.",
			Body:        "Test body.",
		},
	}

	files, _, err := r.CompileSkills(skills, "")
	require.NoError(t, err)

	// Verify path starts with "skills/"
	found := false
	for path := range files {
		if strings.HasPrefix(path, "skills/test-skill/") && strings.HasSuffix(path, "SKILL.md") {
			found = true
			break
		}
	}
	require.True(t, found, "expected skill output path to start with 'skills/', got keys: %v", mapKeys(files))
}

// TestCompile_Rules_Unsupported verifies that CompileRules returns an empty map
// without error (unsupported in Codex).
func TestCompile_Rules_Unsupported(t *testing.T) {
	r := New()
	rules := map[string]ast.RuleConfig{
		"security": {
			Description: "Security rules.",
			Body:        "Never expose secrets.",
		},
	}

	files, notes, err := r.CompileRules(rules, "")
	require.NoError(t, err)
	assert.Empty(t, files)
	assert.Empty(t, notes) // Orchestrator emits RENDERER_KIND_UNSUPPORTED
}

// TestCompile_Hooks_JSON verifies that hook config is encoded as valid JSON
// in a hooks.json file.
func TestCompile_Hooks_JSON(t *testing.T) {
	r := New()
	timeout := 60
	hooks := ast.HookConfig{
		"PreToolUse": []ast.HookMatcherGroup{
			{
				Matcher: "",
				Hooks: []ast.HookHandler{
					{
						Type:    "command",
						Command: "echo test",
						Timeout: &timeout,
					},
				},
			},
		},
	}

	files, notes, err := r.CompileHooks(hooks, "")
	require.NoError(t, err)
	require.Empty(t, notes)

	content, ok := files["hooks.json"]
	require.True(t, ok, "expected hooks.json in output")

	// Verify the top-level "hooks" wrapper is present (Codex native round-trip format).
	assert.Contains(t, content, "\"hooks\"")
	// Verify the event type and command are present inside the wrapper.
	assert.Contains(t, content, "\"PreToolUse\"")
	assert.Contains(t, content, "echo test")
}

// TestCompile_Hooks_Empty verifies that an empty hooks config produces
// an empty output.
func TestCompile_Hooks_Empty(t *testing.T) {
	r := New()
	hooks := ast.HookConfig{}

	files, notes, err := r.CompileHooks(hooks, "")
	require.NoError(t, err)
	assert.Empty(t, files)
	assert.Empty(t, notes)
}

// TestCompile_MCP_TOMLConfig verifies that MCP servers are written to
// config.toml with [mcp_servers.*] sections.
func TestCompile_MCP_TOMLConfig(t *testing.T) {
	r := New()
	servers := map[string]ast.MCPConfig{
		"postgres": {
			Type:    "stdio",
			Command: "/usr/bin/mcp-postgres",
			Args:    []string{"--host", "localhost"},
			Env: map[string]string{
				"DB_HOST": "localhost",
				"DB_PORT": "5432",
			},
		},
	}

	files, notes, err := r.CompileMCP(servers)
	require.NoError(t, err)
	require.Empty(t, notes)

	content, ok := files["config.toml"]
	require.True(t, ok, "expected config.toml in output")

	// Verify TOML structure contains [mcp_servers.postgres]
	assert.Contains(t, content, "[mcp_servers.postgres]")
	assert.Contains(t, content, "type = \"stdio\"")
	assert.Contains(t, content, "command = \"/usr/bin/mcp-postgres\"")

	// Verify it parses as valid TOML
	var doc map[string]interface{}
	err = toml.Unmarshal([]byte(content), &doc)
	require.NoError(t, err, "config.toml must be valid TOML")
}

// TestCompile_MCP_Empty verifies that an empty MCP config produces
// an empty output.
func TestCompile_MCP_Empty(t *testing.T) {
	r := New()
	servers := map[string]ast.MCPConfig{}

	files, notes, err := r.CompileMCP(servers)
	require.NoError(t, err)
	assert.Empty(t, files)
	assert.Empty(t, notes)
}

// TestCompile_ProjectInstructions_AGENTSMD verifies that a project context
// body is written to AGENTS.md in rootFiles (not files).
func TestCompile_ProjectInstructions_AGENTSMD(t *testing.T) {
	r := New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Contexts: map[string]ast.ContextConfig{
				"root": {
					Body:    "# Project Instructions for Codex\n\nUse the following guidelines:\n",
					Targets: []string{"codex"},
				},
			},
		},
	}

	files, rootFiles, notes, err := r.CompileProjectInstructions(config, "")
	require.NoError(t, err)
	require.Empty(t, notes)
	require.Empty(t, files)

	content, ok := rootFiles["AGENTS.md"]
	require.True(t, ok, "expected AGENTS.md in rootFiles")
	assert.Contains(t, content, "Project Instructions for Codex")
}

// TestCompile_ProjectInstructions_NoContext verifies that missing context
// produces empty output without error.
func TestCompile_ProjectInstructions_NoContext(t *testing.T) {
	r := New()
	config := &ast.XcaffoldConfig{}

	files, rootFiles, notes, err := r.CompileProjectInstructions(config, "")
	require.NoError(t, err)
	assert.Empty(t, files)
	assert.Empty(t, rootFiles)
	assert.Empty(t, notes)
}

// TestCompile_Memory_Unsupported verifies that CompileMemory returns
// an empty map without error (unsupported in Codex).
func TestCompile_Memory_Unsupported(t *testing.T) {
	r := New()
	config := &ast.XcaffoldConfig{}

	files, notes, err := r.CompileMemory(config, "", renderer.MemoryOptions{})
	require.NoError(t, err)
	assert.Empty(t, files)
	assert.Empty(t, notes) // Orchestrator emits RENDERER_KIND_UNSUPPORTED
}

// TestFinalize_MovesSkillsToRootFiles verifies that skills entries in files
// are moved from "skills/" to rootFiles with ".agents/skills/" prefix.
func TestFinalize_MovesSkillsToRootFiles(t *testing.T) {
	r := New()
	files := map[string]string{
		"agents/test.toml":               "name = \"test\"",
		"skills/my-skill/SKILL.md":       "# My Skill",
		"skills/other-skill/SKILL.md":    "# Other",
		"skills/my-skill/examples/ex.md": "example",
		"hooks.json":                     "{}",
	}
	rootFiles := map[string]string{
		"AGENTS.md": "# Root",
	}

	filesOut, rootFilesOut, notes, err := r.Finalize(files, rootFiles)
	require.NoError(t, err)
	require.Empty(t, notes)

	// Verify skills are moved to rootFiles with .agents/ prefix
	assert.Contains(t, rootFilesOut, ".agents/skills/my-skill/SKILL.md")
	assert.Equal(t, "# My Skill", rootFilesOut[".agents/skills/my-skill/SKILL.md"])

	assert.Contains(t, rootFilesOut, ".agents/skills/other-skill/SKILL.md")
	assert.Equal(t, "# Other", rootFilesOut[".agents/skills/other-skill/SKILL.md"])

	assert.Contains(t, rootFilesOut, ".agents/skills/my-skill/examples/ex.md")
	assert.Equal(t, "example", rootFilesOut[".agents/skills/my-skill/examples/ex.md"])

	// Verify skills removed from files
	assert.NotContains(t, filesOut, "skills/my-skill/SKILL.md")
	assert.NotContains(t, filesOut, "skills/other-skill/SKILL.md")

	// Verify other entries remain in files
	assert.Contains(t, filesOut, "agents/test.toml")
	assert.Contains(t, filesOut, "hooks.json")

	// Verify rootFiles preserved
	assert.Contains(t, rootFilesOut, "AGENTS.md")
}

// TestFinalize_NonSkillsUnchanged verifies that non-skill entries in files
// remain in files unchanged.
func TestFinalize_NonSkillsUnchanged(t *testing.T) {
	r := New()
	files := map[string]string{
		"agents/reviewer.toml": "name = \"reviewer\"",
		"hooks.json":           "{}",
		"config.toml":          "[mcp_servers]",
	}
	rootFiles := map[string]string{}

	filesOut, rootFilesOut, notes, err := r.Finalize(files, rootFiles)
	require.NoError(t, err)
	require.Empty(t, notes)

	// Verify non-skill entries remain in files
	assert.Equal(t, "name = \"reviewer\"", filesOut["agents/reviewer.toml"])
	assert.Equal(t, "{}", filesOut["hooks.json"])
	assert.Equal(t, "[mcp_servers]", filesOut["config.toml"])

	// Verify rootFiles empty (no skills were present)
	assert.Empty(t, rootFilesOut)
}

// TestCompile_Agents_EffortField verifies that the model_reasoning_effort field
// is correctly populated when Effort is set.
func TestCompile_Agents_EffortField(t *testing.T) {
	r := New()
	agents := map[string]ast.AgentConfig{
		"thinker": {
			Name:        "Thinker",
			Description: "Thoughtful analysis.",
			Body:        "Think carefully.",
			Effort:      "high",
		},
	}

	files, notes, err := r.CompileAgents(agents, "")
	require.NoError(t, err)
	require.Empty(t, notes)

	content := files["agents/thinker.toml"]
	assert.Contains(t, content, "model_reasoning_effort = \"high\"")

	var cfg codexAgent
	err = toml.Unmarshal([]byte(content), &cfg)
	require.NoError(t, err)
	assert.Equal(t, "high", cfg.ModelReasoningEffort)
}

// TestCompile_Agents_InheritedSkipped verifies that inherited agents are
// not compiled.
func TestCompile_Agents_InheritedSkipped(t *testing.T) {
	r := New()
	agents := map[string]ast.AgentConfig{
		"parent": {
			Name:        "Parent",
			Description: "Parent agent.",
			Body:        "Parent body.",
		},
		"child": {
			Name:        "Child",
			Description: "Child agent.",
			Body:        "Child body.",
			Inherited:   true,
		},
	}

	files, notes, err := r.CompileAgents(agents, "")
	require.NoError(t, err)
	require.Empty(t, notes)

	// Verify only non-inherited agent is compiled
	assert.Contains(t, files, "agents/parent.toml")
	assert.NotContains(t, files, "agents/child.toml")
}

// TestCompile_Skills_WithWhenToUse verifies that the when-to-use frontmatter
// field is included when set.
func TestCompile_Skills_WithWhenToUse(t *testing.T) {
	r := New()
	skills := map[string]ast.SkillConfig{
		"deploy": {
			Name:        "deploy",
			Description: "Deployment skill.",
			WhenToUse:   "When you need to ship code.",
			Body:        "Run deployment script.",
		},
	}

	files, _, err := r.CompileSkills(skills, "")
	require.NoError(t, err)

	content := files["skills/deploy/SKILL.md"]
	require.NotEmpty(t, content)
	assert.Contains(t, content, "when-to-use: When you need to ship code.")
}

// TestCompile_Agents_EmptyBody verifies that an agent with Body="" produces
// valid TOML with an empty developer_instructions field.
func TestCompile_Agents_EmptyBody(t *testing.T) {
	r := New()
	agents := map[string]ast.AgentConfig{
		"silent": {
			Name:        "Silent Agent",
			Description: "Agent with no instructions.",
			Body:        "",
		},
	}

	files, notes, err := r.CompileAgents(agents, "")
	require.NoError(t, err)
	require.Empty(t, notes)

	content := files["agents/silent.toml"]
	require.NotEmpty(t, content)

	// Verify TOML parses and developer_instructions is empty string
	var cfg codexAgent
	err = toml.Unmarshal([]byte(content), &cfg)
	require.NoError(t, err)
	assert.Equal(t, "", cfg.DeveloperInstructions)
}

// TestCompile_Agents_MultilineBody verifies that an agent with multiline Body
// produces valid TOML with multi-line string handling.
func TestCompile_Agents_MultilineBody(t *testing.T) {
	r := New()
	agents := map[string]ast.AgentConfig{
		"detailed": {
			Name:        "Detailed Agent",
			Description: "Agent with multiline instructions.",
			Body:        "Step 1: Analyze the request.\nStep 2: Check for edge cases.\nStep 3: Provide answer.\n",
		},
	}

	files, notes, err := r.CompileAgents(agents, "")
	require.NoError(t, err)
	require.Empty(t, notes)

	content := files["agents/detailed.toml"]
	require.NotEmpty(t, content)

	// Verify it parses as valid TOML
	var cfg codexAgent
	err = toml.Unmarshal([]byte(content), &cfg)
	require.NoError(t, err)

	// Verify multiline content is preserved (trailing newline stripped by StripFrontmatter/TrimRight)
	assert.Contains(t, cfg.DeveloperInstructions, "Step 1: Analyze the request.")
	assert.Contains(t, cfg.DeveloperInstructions, "Step 2: Check for edge cases.")
	assert.Contains(t, cfg.DeveloperInstructions, "Step 3: Provide answer.")
}

// TestCompile_Skills_EmptyMap verifies that an empty skills map produces
// empty output with no error.
func TestCompile_Skills_EmptyMap(t *testing.T) {
	r := New()
	skills := map[string]ast.SkillConfig{}

	files, notes, err := r.CompileSkills(skills, "")
	require.NoError(t, err)
	assert.Empty(t, files)
	assert.Empty(t, notes)
}

// TestFinalize_EmptyMaps verifies that Finalize with empty files and rootFiles
// produces empty output with no error.
func TestFinalize_EmptyMaps(t *testing.T) {
	r := New()
	files := map[string]string{}
	rootFiles := map[string]string{}

	filesOut, rootFilesOut, notes, err := r.Finalize(files, rootFiles)
	require.NoError(t, err)
	assert.Empty(t, filesOut)
	assert.Empty(t, rootFilesOut)
	assert.Empty(t, notes)
}

// TestCompile_Workflows_Unsupported verifies that CompileWorkflows returns
// empty output without error (unsupported in Codex).
func TestCompile_Workflows_Unsupported(t *testing.T) {
	r := New()
	workflows := map[string]ast.WorkflowConfig{
		"deploy": {
			Description: "Deploy workflow.",
		},
	}

	files, notes, err := r.CompileWorkflows(workflows, "")
	require.NoError(t, err)
	assert.Empty(t, files)
	assert.Empty(t, notes)
}

// TestCompile_Settings_Unsupported verifies that CompileSettings returns
// empty output without error (unsupported in Codex).
func TestCompile_Settings_Unsupported(t *testing.T) {
	r := New()
	settings := ast.SettingsConfig{
		Name:        "debug-settings",
		Description: "Debug configuration",
	}

	files, notes, err := r.CompileSettings(settings)
	require.NoError(t, err)
	assert.Empty(t, files)
	assert.Empty(t, notes)
}

// TestCapabilities_SupportedKinds verifies that Capabilities returns true
// for supported kinds: Agents, Skills, Hooks, MCP, ProjectInstructions.
func TestCapabilities_SupportedKinds(t *testing.T) {
	r := New()
	caps := r.Capabilities()

	assert.True(t, caps.Agents, "Agents should be supported")
	assert.True(t, caps.Skills, "Skills should be supported")
	assert.True(t, caps.Hooks, "Hooks should be supported")
	assert.True(t, caps.MCP, "MCP should be supported")
	assert.True(t, caps.ProjectInstructions, "ProjectInstructions should be supported")
}

// TestCapabilities_UnsupportedKinds verifies that Capabilities returns false
// for unsupported kinds: Rules, Workflows, Settings, Memory.
func TestCapabilities_UnsupportedKinds(t *testing.T) {
	r := New()
	caps := r.Capabilities()

	assert.False(t, caps.Rules, "Rules should not be supported")
	assert.False(t, caps.Workflows, "Workflows should not be supported")
	assert.False(t, caps.Settings, "Settings should not be supported")
	assert.False(t, caps.Memory, "Memory should not be supported")
}

// ─── Helper functions ────────────────────────────────────────────────────────

// mapKeys returns the keys of a map as a slice for debugging.
func mapKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
