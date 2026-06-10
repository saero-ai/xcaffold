package antigravity2

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/renderer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCompile_Antigravity2_ProjectInstructions verifies that a project-level context
// with a body is compiled to GEMINI.md in rootFiles.
func TestCompile_Antigravity2_ProjectInstructions(t *testing.T) {
	r := New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Contexts: map[string]ast.ContextConfig{
				"root": {
					Body:    "# Antigravity 2.0 Project Instructions\n\nFollow these guidelines for agent development.\n",
					Targets: []string{"antigravity2"},
				},
			},
		},
	}

	files, rootFiles, notes, err := r.CompileProjectInstructions(config, "")
	require.NoError(t, err)
	require.Empty(t, notes)
	require.Empty(t, files)

	content, ok := rootFiles[ProjectContextFile]
	require.True(t, ok, "expected GEMINI.md in rootFiles")
	assert.Contains(t, content, "Antigravity 2.0 Project Instructions")
	assert.Contains(t, content, "agent development")
}

// TestCompile_Antigravity2_Rules verifies that rules with activation modes
// are compiled to rules/<id>.md with correct frontmatter.
func TestCompile_Antigravity2_Rules(t *testing.T) {
	r := New()
	rules := map[string]ast.RuleConfig{
		"security": {
			Description: "Security best practices.",
			Activation:  ast.RuleActivationAlways,
			Body:        "Never expose secrets in logs.",
		},
		"format-check": {
			Description: "Code formatting rules.",
			Activation:  ast.RuleActivationPathGlob,
			Paths:       ast.ClearableList{Values: []string{"src/**/*.go"}},
			Body:        "Use gofmt for all Go files.",
		},
	}

	files, notes, err := r.CompileRules(rules, "")
	require.NoError(t, err)
	require.Empty(t, notes)

	// Verify security rule
	securityContent, ok := files["rules/security.md"]
	require.True(t, ok, "expected rules/security.md in output")
	assert.Contains(t, securityContent, "description: Security best practices.")
	assert.Contains(t, securityContent, "Never expose secrets in logs.")

	// Verify format-check rule with path-glob activation
	formatContent, ok := files["rules/format-check.md"]
	require.True(t, ok, "expected rules/format-check.md in output")
	assert.Contains(t, formatContent, "trigger: glob")
	assert.Contains(t, formatContent, "src/**/*.go")
}

// TestCompile_Antigravity2_Skills verifies that skills are compiled to
// skills/<id>/SKILL.md with agentskills.io frontmatter.
func TestCompile_Antigravity2_Skills(t *testing.T) {
	r := New()
	skills := map[string]ast.SkillConfig{
		"code-review": {
			Name:        "code-review",
			Description: "Code review automation skill.",
			WhenToUse:   "When you need to check code quality.",
			Body:        "# Code Review\n\nReview code for quality and style.\n",
		},
	}

	files, notes, err := r.CompileSkills(skills, "")
	require.NoError(t, err)
	require.Empty(t, notes)

	content, ok := files["skills/code-review/SKILL.md"]
	require.True(t, ok, "expected skills/code-review/SKILL.md in output")

	// Verify frontmatter
	assert.Contains(t, content, "name: code-review")
	assert.Contains(t, content, "description: Code review automation skill.")
	assert.Contains(t, content, "when-to-use: When you need to check code quality.")

	// Verify body
	assert.Contains(t, content, "Review code for quality and style.")
}

// TestCompile_Antigravity2_Agents verifies that agents with all 12 fields populated
// are compiled to agents/<id>/agent.json with correct JSON structure.
func TestCompile_Antigravity2_Agents(t *testing.T) {
	r := New()
	maxTurns := 10
	readonly := false
	userInvocable := true

	agents := map[string]ast.AgentConfig{
		"reviewer": {
			Name:            "Code Reviewer",
			Description:     "Automated code review specialist.",
			Model:           "gpt-5.5",
			MaxTurns:        &maxTurns,
			Tools:           ast.ClearableList{Values: []string{"Read", "Write"}},
			DisallowedTools: ast.ClearableList{Values: []string{"Delete"}},
			Readonly:        &readonly,
			UserInvocable:   &userInvocable,
			InitialPrompt:   "Review the provided code.",
			Skills:          ast.ClearableList{Values: []string{"code-review", "security-audit"}},
			Rules:           ast.ClearableList{Values: []string{"security", "formatting"}},
			Body:            "Perform thorough code reviews.",
		},
	}

	files, notes, err := r.CompileAgents(agents, "")
	require.NoError(t, err)
	require.Empty(t, notes)

	content, ok := files["agents/reviewer/agent.json"]
	require.True(t, ok, "expected agents/reviewer/agent.json in output")

	// Verify JSON structure and all fields
	var agentObj agentJSON
	err = json.Unmarshal([]byte(content), &agentObj)
	require.NoError(t, err)

	assert.Equal(t, "Code Reviewer", agentObj.Name)
	assert.Equal(t, "Automated code review specialist.", agentObj.Description)
	assert.Equal(t, "gpt-5.5", agentObj.Model)
	assert.Equal(t, &maxTurns, agentObj.MaxTurns)
	assert.Equal(t, []string{"Read", "Write"}, agentObj.Tools)
	assert.Equal(t, []string{"Delete"}, agentObj.DisabledTools)
	assert.Equal(t, &readonly, agentObj.Readonly)
	assert.Equal(t, &userInvocable, agentObj.UserInvocable)
	assert.Equal(t, "Review the provided code.", agentObj.InitialPrompt)
	assert.Equal(t, []string{"code-review", "security-audit"}, agentObj.Skills)
	assert.Equal(t, []string{"security", "formatting"}, agentObj.Rules)
	assert.Contains(t, agentObj.Instructions, "Perform thorough code reviews.")
}

// TestCompile_Antigravity2_Hooks verifies that hooks configuration is
// compiled to hooks.json with correct JSON structure.
func TestCompile_Antigravity2_Hooks(t *testing.T) {
	r := New()
	timeout := 30
	hooks := ast.HookConfig{
		"PreToolUse": []ast.HookMatcherGroup{
			{
				Matcher: "",
				Hooks: []ast.HookHandler{
					{
						Type:    "command",
						Command: "echo pre-tool",
						Timeout: &timeout,
					},
				},
			},
		},
		"PostToolUse": []ast.HookMatcherGroup{
			{
				Matcher: "",
				Hooks: []ast.HookHandler{
					{
						Type:    "command",
						Command: "echo post-tool",
					},
				},
			},
		},
	}

	files, notes, err := r.CompileHooks(hooks, "")
	require.NoError(t, err)
	require.Empty(t, notes)

	content, ok := files[HooksFile]
	require.True(t, ok, "expected hooks.json in output")

	// Verify valid JSON
	var hooksObj ast.HookConfig
	err = json.Unmarshal([]byte(content), &hooksObj)
	require.NoError(t, err)

	// Verify structure
	assert.Contains(t, content, "PreToolUse")
	assert.Contains(t, content, "PostToolUse")
	assert.Contains(t, content, "echo pre-tool")
	assert.Contains(t, content, "echo post-tool")
}

// TestCompile_Antigravity2_MCP verifies that MCP configuration is compiled
// to mcp_config.json with serverUrl field.
func TestCompile_Antigravity2_MCP(t *testing.T) {
	r := New()
	servers := map[string]ast.MCPConfig{
		"postgres": {
			URL:           "stdio:///usr/bin/mcp-postgres",
			Command:       "/usr/bin/mcp-postgres",
			Args:          []string{"--host", "localhost"},
			Env:           map[string]string{"DB_HOST": "localhost"},
			DisabledTools: []string{"delete"},
		},
	}

	files, notes, err := r.CompileMCP(servers)
	require.NoError(t, err)
	require.Empty(t, notes)

	content, ok := files[MCPConfigFile]
	require.True(t, ok, "expected mcp_config.json in output")

	// Verify JSON structure
	var wrapper map[string]interface{}
	err = json.Unmarshal([]byte(content), &wrapper)
	require.NoError(t, err)

	assert.Contains(t, content, "mcpServers")
	assert.Contains(t, content, "postgres")
	assert.Contains(t, content, "serverUrl")
	assert.Contains(t, content, "stdio:///usr/bin/mcp-postgres")
	assert.Contains(t, content, "disabledTools")
}

// TestCompile_Antigravity2_Workflows verifies that workflows are compiled
// to workflows/<id>.md files.
func TestCompile_Antigravity2_Workflows(t *testing.T) {
	r := New()
	workflows := map[string]ast.WorkflowConfig{
		"code-review-flow": {
			Name:        "Code Review Flow",
			Description: "Multi-step code review workflow.",
			Steps: []ast.WorkflowStep{
				{
					Name:         "Analyze",
					Skill:        "code-review",
					Instructions: "Analyze code structure.",
				},
				{
					Name:         "Report",
					Instructions: "Generate report.",
				},
			},
		},
	}

	files, notes, err := r.CompileWorkflows(workflows, "")
	require.NoError(t, err)
	require.Empty(t, notes)

	content, ok := files["workflows/code-review-flow.md"]
	require.True(t, ok, "expected workflows/code-review-flow.md in output")

	assert.Contains(t, content, "## Analyze")
	assert.Contains(t, content, "Invoke `/code-review`")
	assert.Contains(t, content, "Analyze code structure.")
	assert.Contains(t, content, "## Report")
	assert.Contains(t, content, "Generate report.")
}

// TestCompile_Antigravity2_Memory verifies that memory entries are compiled
// to knowledge/<name>.md files.
func TestCompile_Antigravity2_Memory(t *testing.T) {
	r := New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Memory: map[string]ast.MemoryConfig{
				"context": {
					Name:        "Project Context",
					Description: "High-level project facts.",
					Content:     "This is a code review automation system.",
				},
			},
		},
	}

	files, notes, err := r.CompileMemory(config, "", renderer.MemoryOptions{})
	require.NoError(t, err)
	require.Empty(t, notes)
	require.NotEmpty(t, files)

	var content string
	for _, v := range files {
		content = v
		break
	}

	// Verify frontmatter
	assert.Contains(t, content, "title: Project Context")
	assert.Contains(t, content, "description: High-level project facts.")
	assert.Contains(t, content, "tags:")
	assert.Contains(t, content, "- memory")

	// Verify body
	assert.Contains(t, content, "This is a code review automation system.")
}

// TestCompile_Antigravity2_EmptyConfig verifies that compiling an empty
// configuration does not crash and produces no output.
func TestCompile_Antigravity2_EmptyConfig(t *testing.T) {
	r := New()
	config := &ast.XcaffoldConfig{}

	// Empty agents
	agents, notes, err := r.CompileAgents(map[string]ast.AgentConfig{}, "")
	require.NoError(t, err)
	assert.Empty(t, agents)
	assert.Empty(t, notes)

	// Empty skills
	skills, notes, err := r.CompileSkills(map[string]ast.SkillConfig{}, "")
	require.NoError(t, err)
	assert.Empty(t, skills)
	assert.Empty(t, notes)

	// Empty rules
	rules, notes, err := r.CompileRules(map[string]ast.RuleConfig{}, "")
	require.NoError(t, err)
	assert.Empty(t, rules)
	assert.Empty(t, notes)

	// Empty workflows
	workflows, notes, err := r.CompileWorkflows(map[string]ast.WorkflowConfig{}, "")
	require.NoError(t, err)
	assert.Empty(t, workflows)
	assert.Empty(t, notes)

	// Empty hooks
	hooks, notes, err := r.CompileHooks(ast.HookConfig{}, "")
	require.NoError(t, err)
	assert.Empty(t, hooks)
	assert.Empty(t, notes)

	// Empty memory
	memory, notes, err := r.CompileMemory(config, "", renderer.MemoryOptions{})
	require.NoError(t, err)
	assert.Empty(t, memory)
	assert.Empty(t, notes)
}

// TestCompile_Antigravity2_UnsupportedFields verifies that compiling settings
// with unsupported fields emits warnings.
func TestCompile_Antigravity2_UnsupportedFields(t *testing.T) {
	r := New()
	settings := ast.SettingsConfig{
		Name:        "global",
		Description: "Global settings.",
		Permissions: &ast.PermissionsConfig{},
		Sandbox:     &ast.SandboxConfig{},
	}

	files, notes, err := r.CompileSettings(settings)
	require.NoError(t, err)
	require.Empty(t, files)

	// Verify fidelity notes for unsupported fields
	require.NotEmpty(t, notes)
	assert.True(t, len(notes) >= 2, "expected at least 2 fidelity notes for unsupported fields")

	// Verify warning levels and codes
	for _, note := range notes {
		assert.Equal(t, renderer.LevelWarning, note.Level)
		assert.Equal(t, targetName, note.Target)
		assert.Equal(t, "settings", note.Kind)
	}
}

// TestCapabilities_SupportedKinds verifies that Capabilities returns true
// for all supported kinds in Antigravity 2.0.
func TestCapabilities_SupportedKinds(t *testing.T) {
	r := New()
	caps := r.Capabilities()

	assert.True(t, caps.Agents)
	assert.True(t, caps.Skills)
	assert.True(t, caps.Rules)
	assert.True(t, caps.Workflows)
	assert.True(t, caps.Hooks)
	assert.True(t, caps.Settings)
	assert.True(t, caps.MCP)
	assert.True(t, caps.Memory)
	assert.True(t, caps.ProjectInstructions)
}

// TestCapabilities_RuleActivations verifies that rule activation modes are declared.
func TestCapabilities_RuleActivations(t *testing.T) {
	r := New()
	caps := r.Capabilities()

	expected := []string{"always", "path-glob", "model-decided", "manual-mention"}
	assert.ElementsMatch(t, expected, caps.RuleActivations)
}

// TestTarget returns the correct target identifier.
func TestTarget(t *testing.T) {
	r := New()
	assert.Equal(t, "antigravity2", r.Target())
}

// TestOutputDir returns the correct output directory.
func TestOutputDir(t *testing.T) {
	r := New()
	assert.Equal(t, ".agents", r.OutputDir())
}

// TestFinalize_NoOp verifies that Finalize is a no-op that preserves input unchanged.
func TestFinalize_NoOp(t *testing.T) {
	r := New()
	files := map[string]string{
		"agents/test.json":     "{}",
		"skills/test/SKILL.md": "# Test",
	}
	rootFiles := map[string]string{
		"GEMINI.md": "# Root",
	}

	filesOut, rootFilesOut, notes, err := r.Finalize(files, rootFiles)
	require.NoError(t, err)
	require.Empty(t, notes)

	// Verify no-op: files unchanged
	assert.Equal(t, files, filesOut)
	assert.Equal(t, rootFiles, rootFilesOut)
}

// TestCompile_Antigravity2_RuleBodyCharLimit verifies that rules exceeding
// the character limit include a warning comment.
func TestCompile_Antigravity2_RuleBodyCharLimit(t *testing.T) {
	r := New()
	// Generate a rule body that exceeds ruleCharLimit (12000)
	longBody := strings.Repeat("a", 12500)
	rules := map[string]ast.RuleConfig{
		"long-rule": {
			Description: "A rule with a very long body.",
			Activation:  ast.RuleActivationAlways,
			Body:        longBody,
		},
	}

	files, notes, err := r.CompileRules(rules, "")
	require.NoError(t, err)
	require.Empty(t, notes)

	content, ok := files["rules/long-rule.md"]
	require.True(t, ok)

	// Verify warning comment is present
	assert.Contains(t, content, "WARNING: rule body exceeds 12000 characters")
}

// TestCompile_Antigravity2_SkillWithAllowedTools verifies that allowed-tools
// frontmatter is correctly rendered.
func TestCompile_Antigravity2_SkillWithAllowedTools(t *testing.T) {
	r := New()
	skills := map[string]ast.SkillConfig{
		"restricted-skill": {
			Name:         "restricted-skill",
			Description:  "Skill with tool restrictions.",
			AllowedTools: ast.ClearableList{Values: []string{"Read", "Write"}},
			Body:         "Do something with restricted tools.",
		},
	}

	files, notes, err := r.CompileSkills(skills, "")
	require.NoError(t, err)
	require.Empty(t, notes)

	content, ok := files["skills/restricted-skill/SKILL.md"]
	require.True(t, ok)

	assert.Contains(t, content, "allowed-tools: [Read, Write]")
	assert.Contains(t, content, "Do something with restricted tools.")
}

// TestCompile_Antigravity2_AgentWithoutOptionalFields verifies that agents
// with minimal fields (name, description only) compile correctly.
func TestCompile_Antigravity2_AgentWithoutOptionalFields(t *testing.T) {
	r := New()
	agents := map[string]ast.AgentConfig{
		"simple": {
			Name:        "Simple Agent",
			Description: "A minimal agent.",
		},
	}

	files, notes, err := r.CompileAgents(agents, "")
	require.NoError(t, err)
	require.Empty(t, notes)

	content, ok := files["agents/simple/agent.json"]
	require.True(t, ok)

	var agentObj agentJSON
	err = json.Unmarshal([]byte(content), &agentObj)
	require.NoError(t, err)

	assert.Equal(t, "Simple Agent", agentObj.Name)
	assert.Equal(t, "A minimal agent.", agentObj.Description)
	assert.Nil(t, agentObj.MaxTurns)
	assert.Empty(t, agentObj.Tools)
	assert.Empty(t, agentObj.Skills)
}
