package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteSplitFiles_DirectoryStructure(t *testing.T) {
	tmpDir := t.TempDir()

	config := &ast.XcaffoldConfig{
		Version: "1.0",
		Project: &ast.ProjectConfig{
			Name:        "my-project",
			Description: "Test project",
			Targets:     []string{"claude", "antigravity"},
		},
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"developer": {Name: "developer", Description: "Dev agent", Model: "sonnet"},
				"reviewer":  {Name: "reviewer", Description: "Review agent", Model: "haiku"},
			},
			Skills: map[string]ast.SkillConfig{
				"tdd": {Name: "tdd", Description: "Test-driven development"},
			},
			Rules: map[string]ast.RuleConfig{
				"security": {Name: "security", Description: "Security rules"},
			},
		},
		Hooks: map[string]ast.NamedHookConfig{
			"default": {
				Name: "default",
				Events: ast.HookConfig{
					"PreToolUse": {
						{Hooks: []ast.HookHandler{{Type: "command", Command: "echo pre"}}},
					},
				},
			},
		},
		Settings: map[string]ast.SettingsConfig{"default": {
			Model: "claude-sonnet-4-5",
		}},
	}

	err := WriteSplitFiles(config, tmpDir)
	require.NoError(t, err)

	// project.xcf must exist with kind: project
	scaffoldPath := filepath.Join(tmpDir, "project.xcf")
	assert.FileExists(t, scaffoldPath)
	scaffoldBytes, err := os.ReadFile(scaffoldPath)
	require.NoError(t, err)
	scaffoldContent := string(scaffoldBytes)
	assert.Contains(t, scaffoldContent, "kind: project")
	assert.Contains(t, scaffoldContent, "name: my-project")
	assert.Contains(t, scaffoldContent, "claude")
	assert.Contains(t, scaffoldContent, "antigravity")
	// Ref lists are no longer in project.xcf; resources are discovered from xcf/ directory
	assert.NotContains(t, scaffoldContent, "developer")
	assert.NotContains(t, scaffoldContent, "reviewer")
	assert.NotContains(t, scaffoldContent, "tdd")
	assert.NotContains(t, scaffoldContent, "security")

	// Agent files — each lives in its own subdirectory: xcf/agents/<id>/agent.xcf
	assert.FileExists(t, filepath.Join(tmpDir, "xcf", "agents", "developer", "agent.xcf"))
	developerBytes, err := os.ReadFile(filepath.Join(tmpDir, "xcf", "agents", "developer", "agent.xcf"))
	require.NoError(t, err)
	assert.Contains(t, string(developerBytes), "kind: agent")
	assert.Contains(t, string(developerBytes), "name: developer")

	assert.FileExists(t, filepath.Join(tmpDir, "xcf", "agents", "reviewer", "agent.xcf"))

	// Skill file — directory layout: xcf/skills/<name>/skill.xcf
	assert.FileExists(t, filepath.Join(tmpDir, "xcf", "skills", "tdd", "skill.xcf"))
	skillBytes, err := os.ReadFile(filepath.Join(tmpDir, "xcf", "skills", "tdd", "skill.xcf"))
	require.NoError(t, err)
	assert.Contains(t, string(skillBytes), "kind: skill")

	// Rule file
	assert.FileExists(t, filepath.Join(tmpDir, "xcf", "rules", "security", "rule.xcf"))
	ruleBytes, err := os.ReadFile(filepath.Join(tmpDir, "xcf", "rules", "security", "rule.xcf"))
	require.NoError(t, err)
	assert.Contains(t, string(ruleBytes), "kind: rule")

	// Hooks file — each named hook lives in its own subdirectory: xcf/hooks/<name>/hooks.xcf
	assert.FileExists(t, filepath.Join(tmpDir, "xcf", "hooks", "default", "hooks.xcf"))
	hooksBytes, err := os.ReadFile(filepath.Join(tmpDir, "xcf", "hooks", "default", "hooks.xcf"))
	require.NoError(t, err)
	assert.Contains(t, string(hooksBytes), "kind: hooks")
	assert.Contains(t, string(hooksBytes), "events:")

	// Settings file — each named settings lives in its own subdirectory: xcf/settings/<name>/settings.xcf
	assert.FileExists(t, filepath.Join(tmpDir, "xcf", "settings", "default", "settings.xcf"))
	settingsBytes, err := os.ReadFile(filepath.Join(tmpDir, "xcf", "settings", "default", "settings.xcf"))
	require.NoError(t, err)
	assert.Contains(t, string(settingsBytes), "kind: settings")
}

func TestWriteSplitFiles_RoundTrip(t *testing.T) {
	t.Setenv("XCAFFOLD_SKIP_GLOBAL", "true")
	tmpDir := t.TempDir()

	config := &ast.XcaffoldConfig{
		Version: "1.0",
		Project: &ast.ProjectConfig{
			Name: "round-trip",
		},
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"alpha": {Name: "alpha", Description: "Alpha agent", Model: "sonnet"},
			},
			Skills: map[string]ast.SkillConfig{
				"deploy": {Name: "deploy", Description: "Deploy skill"},
			},
			Rules: map[string]ast.RuleConfig{
				"lint": {Name: "lint", Description: "Lint rules"},
			},
		},
	}

	err := WriteSplitFiles(config, tmpDir)
	require.NoError(t, err)

	parsed, err := parser.ParseDirectory(tmpDir)
	require.NoError(t, err)

	assert.Equal(t, "round-trip", parsed.Project.Name)
	assert.Contains(t, parsed.Agents, "alpha")
	assert.Contains(t, parsed.Skills, "deploy")
	assert.Contains(t, parsed.Rules, "lint")
}

func TestWriteSplitFiles_Deterministic(t *testing.T) {
	t.Setenv("XCAFFOLD_SKIP_GLOBAL", "true")
	tmpDir1 := t.TempDir()
	tmpDir2 := t.TempDir()

	config := &ast.XcaffoldConfig{
		Version: "1.0",
		Project: &ast.ProjectConfig{
			Name:    "deterministic",
			Targets: []string{"claude"},
		},
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"zulu":  {Name: "zulu", Model: "sonnet"},
				"alpha": {Name: "alpha", Model: "haiku"},
			},
			Skills: map[string]ast.SkillConfig{
				"skill-b": {Name: "skill-b"},
				"skill-a": {Name: "skill-a"},
			},
		},
	}

	err := WriteSplitFiles(config, tmpDir1)
	require.NoError(t, err)
	err = WriteSplitFiles(config, tmpDir2)
	require.NoError(t, err)

	// Compare project.xcf
	b1, err := os.ReadFile(filepath.Join(tmpDir1, "project.xcf"))
	require.NoError(t, err)
	b2, err := os.ReadFile(filepath.Join(tmpDir2, "project.xcf"))
	require.NoError(t, err)
	assert.Equal(t, b1, b2, "project.xcf must be byte-identical")

	// Compare an agent file
	a1, err := os.ReadFile(filepath.Join(tmpDir1, "xcf", "agents", "alpha", "agent.xcf"))
	require.NoError(t, err)
	a2, err := os.ReadFile(filepath.Join(tmpDir2, "xcf", "agents", "alpha", "agent.xcf"))
	require.NoError(t, err)
	assert.Equal(t, a1, a2, "agent file must be byte-identical")
}

func TestWriteSplitFiles_EmptyResources(t *testing.T) {
	tmpDir := t.TempDir()

	config := &ast.XcaffoldConfig{
		Version: "1.0",
		Project: &ast.ProjectConfig{
			Name: "empty-project",
		},
	}

	err := WriteSplitFiles(config, tmpDir)
	require.NoError(t, err)

	// project.xcf must be created
	assert.FileExists(t, filepath.Join(tmpDir, "project.xcf"))

	// No xcf/agents/ directory when there are no agents
	_, statErr := os.Stat(filepath.Join(tmpDir, "xcf", "agents"))
	assert.True(t, os.IsNotExist(statErr), "xcf/agents/ should not exist when config has no agents")
}

func TestWriteSplitFiles_NoHooks_NoHooksFile(t *testing.T) {
	tmpDir := t.TempDir()

	config := &ast.XcaffoldConfig{
		Version: "1.0",
		Project: &ast.ProjectConfig{
			Name: "no-hooks",
		},
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"agent1": {Name: "agent1", Model: "sonnet"},
			},
		},
	}

	err := WriteSplitFiles(config, tmpDir)
	require.NoError(t, err)

	// No hooks directory should exist when config has no hooks
	_, statErr := os.Stat(filepath.Join(tmpDir, "xcf", "hooks"))
	assert.True(t, os.IsNotExist(statErr), "xcf/hooks/ directory should not be created when config has no hooks")
}

func TestWriteFrontmatterFile_AgentWithBody(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "developer.xcf")

	type frontmatterDoc struct {
		Kind    string `yaml:"kind"`
		Version string `yaml:"version"`
		Name    string `yaml:"name"`
		Model   string `yaml:"model"`
	}

	doc := frontmatterDoc{Kind: "agent", Version: "1.0", Name: "developer", Model: "sonnet"}
	body := "You are a developer.\nWrite clean code."

	err := writeFrontmatterFile(path, doc, body)
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	content := string(data)
	assert.True(t, strings.HasPrefix(content, "---\n"), "must start with ---")
	assert.Contains(t, content, "kind: agent")
	assert.Contains(t, content, "name: developer")
	assert.Contains(t, content, "---\nYou are a developer.")
	assert.NotContains(t, content, "instructions:")
}

func TestWriteSplitFiles_ProviderPassthrough(t *testing.T) {
	config := &ast.XcaffoldConfig{}
	config.ProviderExtras = map[string]map[string][]byte{
		"claude": {
			"skills/adr-management/TEMPLATE.md": []byte("# Template"),
		},
	}

	tmpDir := t.TempDir()

	err := WriteSplitFiles(config, tmpDir)
	require.NoError(t, err)

	expected := filepath.Join(tmpDir, "xcf", "provider", "claude", "skills", "adr-management", "TEMPLATE.md")
	assert.FileExists(t, expected, "expected file at xcf/provider/claude/...")

	unexpected := filepath.Join(tmpDir, "xcf", "extras")
	_, statErr := os.Stat(unexpected)
	assert.True(t, os.IsNotExist(statErr), "xcf/extras/ should not exist — should be xcf/provider/")
}

func TestWriteSplitFiles_AgentFrontmatter(t *testing.T) {
	dir := t.TempDir()
	config := &ast.XcaffoldConfig{
		Version: "1.0",
		Project: &ast.ProjectConfig{Name: "test"},
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"developer": {
					Name:        "developer",
					Description: "General developer.",
					Model:       "sonnet",
					Body:        "You are a developer.\nWrite clean code.",
				},
			},
		},
	}
	err := WriteSplitFiles(config, dir)
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(dir, "xcf", "agents", "developer", "agent.xcf"))
	require.NoError(t, err)
	content := string(data)

	assert.True(t, strings.HasPrefix(content, "---\n"), "agent with body must use frontmatter")
	assert.Contains(t, content, "kind: agent")
	assert.Contains(t, content, "name: developer")
	assert.Contains(t, content, "---\nYou are a developer.")
	assert.NotContains(t, content, "instructions:")
}

func TestWriteSplitFiles_SkillFrontmatter(t *testing.T) {
	dir := t.TempDir()
	config := &ast.XcaffoldConfig{
		Version: "1.0",
		Project: &ast.ProjectConfig{Name: "test"},
		ResourceScope: ast.ResourceScope{
			Skills: map[string]ast.SkillConfig{
				"tdd": {
					Name:         "tdd",
					Description:  "Test-driven development.",
					AllowedTools: ast.ClearableList{Values: []string{"Read", "Edit", "Bash"}},
					Body:         "Follow Red-Green-Refactor.",
				},
			},
		},
	}
	err := WriteSplitFiles(config, dir)
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(dir, "xcf", "skills", "tdd", "skill.xcf"))
	require.NoError(t, err)
	content := string(data)

	assert.True(t, strings.HasPrefix(content, "---\n"), "skill with body must use frontmatter")
	assert.Contains(t, content, "kind: skill")
	assert.Contains(t, content, "---\nFollow Red-Green-Refactor.")
	assert.NotContains(t, content, "instructions:")
}

func TestWriteSplitFiles_RuleFrontmatter(t *testing.T) {
	dir := t.TempDir()
	config := &ast.XcaffoldConfig{
		Version: "1.0",
		Project: &ast.ProjectConfig{Name: "test"},
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"conventions": {
					Name:        "conventions",
					Description: "Coding conventions.",
					Body:        "Write clean code.\nUse 2-space indentation.",
				},
			},
		},
	}
	err := WriteSplitFiles(config, dir)
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(dir, "xcf", "rules", "conventions", "rule.xcf"))
	require.NoError(t, err)
	content := string(data)

	assert.True(t, strings.HasPrefix(content, "---\n"), "rule with body must use frontmatter")
	assert.Contains(t, content, "kind: rule")
	assert.Contains(t, content, "---\nWrite clean code.")
	assert.NotContains(t, content, "instructions:")
}

func TestWriteSplitFiles_ScopeFilter_EmptyRefs_WritesAll(t *testing.T) {
	dir := t.TempDir()
	config := &ast.XcaffoldConfig{
		Version: "1.0",
		Project: &ast.ProjectConfig{Name: "test"},
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"developer": {Name: "developer"},
				"reviewer":  {Name: "reviewer"},
			},
		},
	}
	err := WriteSplitFiles(config, dir)
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(dir, "xcf", "agents", "developer", "agent.xcf"))
	require.NoError(t, err)
	_, err = os.Stat(filepath.Join(dir, "xcf", "agents", "reviewer", "agent.xcf"))
	require.NoError(t, err)
}

func TestWriteSplitFiles_SkillSubdirFields(t *testing.T) {
	outDir := t.TempDir()
	config := &ast.XcaffoldConfig{
		Version: "1.0",
		Project: &ast.ProjectConfig{Name: "test"},
		ResourceScope: ast.ResourceScope{
			Skills: map[string]ast.SkillConfig{
				"my-skill": {
					Name:        "my-skill",
					Description: "Test skill",
					References:  ast.ClearableList{Values: []string{"xcf/skills/my-skill/references/ref.md"}},
					Scripts:     ast.ClearableList{Values: []string{"xcf/skills/my-skill/scripts/run.sh"}},
					Assets:      ast.ClearableList{Values: []string{"xcf/skills/my-skill/assets/icon.svg"}},
					Examples:    ast.ClearableList{Values: []string{"xcf/skills/my-skill/examples/sample.md"}},
					Body:        "Do the thing.",
				},
			},
		},
	}
	err := WriteSplitFiles(config, outDir)
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(outDir, "xcf", "skills", "my-skill", "skill.xcf"))
	require.NoError(t, err)
	content := string(data)

	// Verify frontmatter mode is used for skills with a body
	assert.True(t, strings.HasPrefix(content, "---\n"), "skill with body must use frontmatter")
	// Verify all 4 subdir fields appear in frontmatter
	assert.Contains(t, content, "references:")
	assert.Contains(t, content, "scripts:")
	assert.Contains(t, content, "assets:")
	assert.Contains(t, content, "examples:")
	// Verify body appears after the closing --- delimiter
	assert.Contains(t, content, "---\nDo the thing.")
	// Verify instructions field is NOT duplicated in YAML
	assert.NotContains(t, content, "instructions:")
}

func TestWriteSplitFiles_Rules_DirectoryLayout(t *testing.T) {
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"secure-coding": {Name: "secure-coding", Body: "No secrets."},
			},
		},
	}
	dir := t.TempDir()
	if err := WriteSplitFiles(config, dir); err != nil {
		t.Fatal(err)
	}

	// Should be in directory layout: xcf/rules/secure-coding/rule.xcf
	expected := filepath.Join(dir, "xcf", "rules", "secure-coding", "rule.xcf")
	if _, err := os.Stat(expected); os.IsNotExist(err) {
		t.Fatalf("expected rule at %s", expected)
	}

	// Should NOT be flat: xcf/rules/secure-coding.xcf
	flat := filepath.Join(dir, "xcf", "rules", "secure-coding.xcf")
	if _, err := os.Stat(flat); !os.IsNotExist(err) {
		t.Fatal("rule should NOT be in flat layout")
	}
}

func TestWriteSplitFiles_Skill_DirectoryLayout(t *testing.T) {
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Skills: map[string]ast.SkillConfig{
				"tdd": {Name: "tdd", Body: "Do TDD."},
			},
		},
	}
	dir := t.TempDir()
	if err := WriteSplitFiles(config, dir); err != nil {
		t.Fatal(err)
	}

	expected := filepath.Join(dir, "xcf", "skills", "tdd", "skill.xcf")
	if _, err := os.Stat(expected); os.IsNotExist(err) {
		t.Fatal("expected skill at directory layout path")
	}

	flat := filepath.Join(dir, "xcf", "skills", "tdd.xcf")
	if _, err := os.Stat(flat); !os.IsNotExist(err) {
		t.Fatal("skill should NOT be in flat layout")
	}
}

func TestWriteSplitFiles_Context_DirectoryLayout(t *testing.T) {
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Contexts: map[string]ast.ContextConfig{
				"project-readme": {Name: "project-readme", Body: "README content."},
			},
		},
	}
	dir := t.TempDir()
	if err := WriteSplitFiles(config, dir); err != nil {
		t.Fatal(err)
	}

	expected := filepath.Join(dir, "xcf", "context", "project-readme", "context.xcf")
	if _, err := os.Stat(expected); os.IsNotExist(err) {
		t.Fatalf("expected context at %s", expected)
	}

	flat := filepath.Join(dir, "xcf", "context", "project-readme.xcf")
	if _, err := os.Stat(flat); !os.IsNotExist(err) {
		t.Fatal("context should NOT be in flat layout")
	}
}

func TestWriteSplitFiles_Agent_CanonicalFilename(t *testing.T) {
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"developer": {Name: "developer", Model: "sonnet", Body: "Dev."},
			},
		},
	}
	dir := t.TempDir()
	if err := WriteSplitFiles(config, dir); err != nil {
		t.Fatal(err)
	}

	canonical := filepath.Join(dir, "xcf", "agents", "developer", "agent.xcf")
	if _, err := os.Stat(canonical); os.IsNotExist(err) {
		t.Fatal("expected canonical filename agent.xcf")
	}

	old := filepath.Join(dir, "xcf", "agents", "developer", "developer.xcf")
	if _, err := os.Stat(old); !os.IsNotExist(err) {
		t.Fatal("should not use resource name as filename")
	}
}

func TestWriteSplitFiles_AllKinds_DirectoryLayout(t *testing.T) {
	tmpDir := t.TempDir()

	config := &ast.XcaffoldConfig{
		Version: "1.0",
		Project: &ast.ProjectConfig{
			Name: "all-kinds-test",
		},
		ResourceScope: ast.ResourceScope{
			Workflows: map[string]ast.WorkflowConfig{
				"deploy": {Name: "deploy", Description: "Deploy workflow"},
			},
			MCP: map[string]ast.MCPConfig{
				"server-a": {Name: "server-a", Type: "stdio"},
			},
			Policies: map[string]ast.PolicyConfig{
				"security": {
					Name:        "security",
					Description: "Security policy",
					Severity:    "high",
					Target:      "agents",
				},
			},
		},
		Hooks: map[string]ast.NamedHookConfig{
			"pre-compile": {
				Name: "pre-compile",
				Events: ast.HookConfig{
					"PreCompile": {
						{Hooks: []ast.HookHandler{{Type: "command", Command: "echo pre"}}},
					},
				},
			},
			"post-compile": {
				Name: "post-compile",
				Events: ast.HookConfig{
					"PostCompile": {
						{Hooks: []ast.HookHandler{{Type: "command", Command: "echo post"}}},
					},
				},
			},
		},
		Settings: map[string]ast.SettingsConfig{
			"default": {Model: "claude-sonnet-4-5"},
			"compact": {Model: "claude-haiku"},
		},
	}

	err := WriteSplitFiles(config, tmpDir)
	require.NoError(t, err)

	// Workflows: xcf/workflows/<name>/workflow.xcf
	workflowPath := filepath.Join(tmpDir, "xcf", "workflows", "deploy", "workflow.xcf")
	assert.FileExists(t, workflowPath)
	workflowBytes, err := os.ReadFile(workflowPath)
	require.NoError(t, err)
	assert.Contains(t, string(workflowBytes), "kind: workflow")
	assert.Contains(t, string(workflowBytes), "name: deploy")

	// MCP: xcf/mcp/<name>/mcp.xcf
	mcpPath := filepath.Join(tmpDir, "xcf", "mcp", "server-a", "mcp.xcf")
	assert.FileExists(t, mcpPath)
	mcpBytes, err := os.ReadFile(mcpPath)
	require.NoError(t, err)
	assert.Contains(t, string(mcpBytes), "kind: mcp")
	assert.Contains(t, string(mcpBytes), "name: server-a")

	// Policy: xcf/policy/<name>/policy.xcf
	policyPath := filepath.Join(tmpDir, "xcf", "policy", "security", "policy.xcf")
	assert.FileExists(t, policyPath)
	policyBytes, err := os.ReadFile(policyPath)
	require.NoError(t, err)
	assert.Contains(t, string(policyBytes), "kind: policy")
	assert.Contains(t, string(policyBytes), "name: security")

	// Hooks: xcf/hooks/<name>/hooks.xcf (each named hook gets a subdirectory)
	preCompilePath := filepath.Join(tmpDir, "xcf", "hooks", "pre-compile", "hooks.xcf")
	assert.FileExists(t, preCompilePath)
	preCompileBytes, err := os.ReadFile(preCompilePath)
	require.NoError(t, err)
	assert.Contains(t, string(preCompileBytes), "kind: hooks")
	assert.Contains(t, string(preCompileBytes), "events:")

	postCompilePath := filepath.Join(tmpDir, "xcf", "hooks", "post-compile", "hooks.xcf")
	assert.FileExists(t, postCompilePath)
	postCompileBytes, err := os.ReadFile(postCompilePath)
	require.NoError(t, err)
	assert.Contains(t, string(postCompileBytes), "kind: hooks")

	// Settings: xcf/settings/<name>/settings.xcf (each named settings gets a subdirectory)
	defaultSettingsPath := filepath.Join(tmpDir, "xcf", "settings", "default", "settings.xcf")
	assert.FileExists(t, defaultSettingsPath)
	defaultSettingsBytes, err := os.ReadFile(defaultSettingsPath)
	require.NoError(t, err)
	assert.Contains(t, string(defaultSettingsBytes), "kind: settings")

	compactSettingsPath := filepath.Join(tmpDir, "xcf", "settings", "compact", "settings.xcf")
	assert.FileExists(t, compactSettingsPath)
}

func TestWriteSplitFiles_OverrideFiles_AgentWritten(t *testing.T) {
	config := &ast.XcaffoldConfig{
		Version: "1.0",
		Project: &ast.ProjectConfig{Name: "test"},
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"developer": {Name: "developer", Model: "sonnet", Body: "Universal agent."},
			},
		},
		Overrides: &ast.ResourceOverrides{},
	}
	config.Overrides.AddAgent("developer", "claude", ast.AgentConfig{
		Model: "opus", Body: "Claude-specific.",
	})

	dir := t.TempDir()
	if err := WriteSplitFiles(config, dir); err != nil {
		t.Fatal(err)
	}

	// Base file should exist
	basePath := filepath.Join(dir, "xcf", "agents", "developer", "agent.xcf")
	if _, err := os.Stat(basePath); os.IsNotExist(err) {
		t.Fatal("expected base agent file")
	}

	// Override file: agent.claude.xcf
	overridePath := filepath.Join(dir, "xcf", "agents", "developer", "agent.claude.xcf")
	if _, err := os.Stat(overridePath); os.IsNotExist(err) {
		t.Fatal("expected claude override file at " + overridePath)
	}

	// Override should contain overridden fields
	data, _ := os.ReadFile(overridePath)
	content := string(data)
	if !strings.Contains(content, "model: opus") {
		t.Error("override should contain overridden model")
	}
	if !strings.Contains(content, "Claude-specific.") {
		t.Error("override should contain override body")
	}
	// Override should NOT contain kind or version (partial config)
	if strings.Contains(content, "kind:") {
		t.Error("override should not contain kind field")
	}
	if strings.Contains(content, "version:") {
		t.Error("override should not contain version field")
	}
}

func TestWriteSplitFiles_OverrideFiles_SkillWritten(t *testing.T) {
	config := &ast.XcaffoldConfig{
		Version: "1.0",
		Project: &ast.ProjectConfig{Name: "test"},
		ResourceScope: ast.ResourceScope{
			Skills: map[string]ast.SkillConfig{
				"tdd": {Name: "tdd", Body: "Base TDD."},
			},
		},
		Overrides: &ast.ResourceOverrides{},
	}
	config.Overrides.AddSkill("tdd", "cursor", ast.SkillConfig{
		Body: "Cursor-specific TDD.",
	})

	dir := t.TempDir()
	if err := WriteSplitFiles(config, dir); err != nil {
		t.Fatal(err)
	}

	// Override file for skills: xcf/skills/<name>/skill.<provider>.xcf
	overridePath := filepath.Join(dir, "xcf", "skills", "tdd", "skill.cursor.xcf")
	if _, err := os.Stat(overridePath); os.IsNotExist(err) {
		t.Fatal("expected cursor skill override file at " + overridePath)
	}

	data, _ := os.ReadFile(overridePath)
	content := string(data)
	if !strings.Contains(content, "Cursor-specific TDD.") {
		t.Error("override should contain override body")
	}
	// Should NOT have kind/version
	if strings.Contains(content, "kind:") {
		t.Error("override should not contain kind field")
	}
}

func TestWriteSplitFiles_OverrideFiles_RuleWritten(t *testing.T) {
	config := &ast.XcaffoldConfig{
		Version: "1.0",
		Project: &ast.ProjectConfig{Name: "test"},
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"secure": {Name: "secure", Body: "No secrets."},
			},
		},
		Overrides: &ast.ResourceOverrides{},
	}
	config.Overrides.AddRule("secure", "gemini", ast.RuleConfig{
		Body: "Gemini-specific rules.",
	})

	dir := t.TempDir()
	if err := WriteSplitFiles(config, dir); err != nil {
		t.Fatal(err)
	}

	// Override file: rule.gemini.xcf (in the rule's directory)
	overridePath := filepath.Join(dir, "xcf", "rules", "secure", "rule.gemini.xcf")
	if _, err := os.Stat(overridePath); os.IsNotExist(err) {
		t.Fatal("expected gemini rule override file at " + overridePath)
	}

	data, _ := os.ReadFile(overridePath)
	content := string(data)
	if !strings.Contains(content, "Gemini-specific rules.") {
		t.Error("override should contain override body")
	}
	if strings.Contains(content, "kind:") {
		t.Error("override should not contain kind field")
	}
}

func TestWriteSplitFiles_OverrideFiles_WorkflowWritten(t *testing.T) {
	config := &ast.XcaffoldConfig{
		Version: "1.0",
		Project: &ast.ProjectConfig{Name: "test"},
		ResourceScope: ast.ResourceScope{
			Workflows: map[string]ast.WorkflowConfig{
				"deploy": {Name: "deploy", Description: "Base deploy."},
			},
		},
		Overrides: &ast.ResourceOverrides{},
	}
	config.Overrides.AddWorkflow("deploy", "antigravity", ast.WorkflowConfig{
		Description: "Antigravity-specific deploy.",
	})

	dir := t.TempDir()
	if err := WriteSplitFiles(config, dir); err != nil {
		t.Fatal(err)
	}

	// Override file: workflow.antigravity.xcf
	overridePath := filepath.Join(dir, "xcf", "workflows", "deploy", "workflow.antigravity.xcf")
	if _, err := os.Stat(overridePath); os.IsNotExist(err) {
		t.Fatal("expected antigravity workflow override file at " + overridePath)
	}

	data, _ := os.ReadFile(overridePath)
	content := string(data)
	if !strings.Contains(content, "Antigravity-specific deploy.") {
		t.Error("override should contain override description")
	}
	if strings.Contains(content, "kind:") {
		t.Error("override should not contain kind field")
	}
}

func TestWriteSplitFiles_OverrideFiles_MCPWritten(t *testing.T) {
	config := &ast.XcaffoldConfig{
		Version: "1.0",
		Project: &ast.ProjectConfig{Name: "test"},
		ResourceScope: ast.ResourceScope{
			MCP: map[string]ast.MCPConfig{
				"server": {Name: "server", Type: "stdio"},
			},
		},
		Overrides: &ast.ResourceOverrides{},
	}
	config.Overrides.AddMCP("server", "claude", ast.MCPConfig{
		Type: "sse",
	})

	dir := t.TempDir()
	if err := WriteSplitFiles(config, dir); err != nil {
		t.Fatal(err)
	}

	// Override file: mcp.claude.xcf
	overridePath := filepath.Join(dir, "xcf", "mcp", "server", "mcp.claude.xcf")
	if _, err := os.Stat(overridePath); os.IsNotExist(err) {
		t.Fatal("expected claude mcp override file at " + overridePath)
	}

	data, _ := os.ReadFile(overridePath)
	content := string(data)
	if !strings.Contains(content, "type: sse") {
		t.Error("override should contain overridden type")
	}
	if strings.Contains(content, "kind:") {
		t.Error("override should not contain kind field")
	}
}

func TestWriteSplitFiles_Rules_NamespacedPath(t *testing.T) {
	config := &ast.XcaffoldConfig{
		Version: "1.0",
		Project: &ast.ProjectConfig{Name: "test"},
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"cli/build-go-cli": {Name: "build-go-cli", Description: "Build Go CLI.", Body: "Build rules."},
			},
		},
	}
	dir := t.TempDir()
	if err := WriteSplitFiles(config, dir); err != nil {
		t.Fatal(err)
	}

	// Namespaced rule produces directory: xcf/rules/cli/build-go-cli/rule.xcf
	expected := filepath.Join(dir, "xcf", "rules", "cli", "build-go-cli", "rule.xcf")
	if _, err := os.Stat(expected); os.IsNotExist(err) {
		t.Fatalf("expected namespaced rule at %s", expected)
	}

	data, err := os.ReadFile(expected)
	require.NoError(t, err)
	content := string(data)
	assert.Contains(t, content, "kind: rule")
	assert.Contains(t, content, "---\nBuild rules.")
}

func TestWriteSplitFiles_HooksOverrideFiles(t *testing.T) {
	config := &ast.XcaffoldConfig{
		Version: "1.0",
		Project: &ast.ProjectConfig{Name: "test"},
		Hooks: map[string]ast.NamedHookConfig{
			"pre-compile": {
				Name: "pre-compile",
				Events: ast.HookConfig{
					"PreCompile": {
						{Hooks: []ast.HookHandler{{Type: "command", Command: "echo base"}}},
					},
				},
			},
		},
		Overrides: &ast.ResourceOverrides{},
	}

	// Add a provider-specific override for hooks
	config.Overrides.AddHooks("pre-compile", "claude", ast.NamedHookConfig{
		Name: "pre-compile",
		Events: ast.HookConfig{
			"PreCompile": {
				{Hooks: []ast.HookHandler{{Type: "command", Command: "echo claude"}}},
			},
		},
	})

	dir := t.TempDir()
	if err := WriteSplitFiles(config, dir); err != nil {
		t.Fatal(err)
	}

	// Base file should exist
	basePath := filepath.Join(dir, "xcf", "hooks", "pre-compile", "hooks.xcf")
	if _, err := os.Stat(basePath); os.IsNotExist(err) {
		t.Fatal("expected base hooks file")
	}

	// Override file: hooks.claude.xcf
	overridePath := filepath.Join(dir, "xcf", "hooks", "pre-compile", "hooks.claude.xcf")
	if _, err := os.Stat(overridePath); os.IsNotExist(err) {
		t.Fatal("expected claude override file at " + overridePath)
	}

	data, _ := os.ReadFile(overridePath)
	content := string(data)
	if !strings.Contains(content, "events:") {
		t.Error("override should contain events field")
	}
	if strings.Contains(content, "kind:") {
		t.Error("override should not contain kind field")
	}
	if strings.Contains(content, "version:") {
		t.Error("override should not contain version field")
	}
}

func TestWriteSplitFiles_SettingsOverrideFiles(t *testing.T) {
	config := &ast.XcaffoldConfig{
		Version: "1.0",
		Project: &ast.ProjectConfig{Name: "test"},
		Settings: map[string]ast.SettingsConfig{
			"default": {
				Model: "claude-sonnet-4-5",
			},
		},
		Overrides: &ast.ResourceOverrides{},
	}

	// Add a provider-specific override for settings
	config.Overrides.AddSettings("default", "gemini", ast.SettingsConfig{
		Model: "gemini-2.5-flash",
	})

	dir := t.TempDir()
	if err := WriteSplitFiles(config, dir); err != nil {
		t.Fatal(err)
	}

	// Base file should exist
	basePath := filepath.Join(dir, "xcf", "settings", "default", "settings.xcf")
	if _, err := os.Stat(basePath); os.IsNotExist(err) {
		t.Fatal("expected base settings file")
	}

	// Override file: settings.gemini.xcf
	overridePath := filepath.Join(dir, "xcf", "settings", "default", "settings.gemini.xcf")
	if _, err := os.Stat(overridePath); os.IsNotExist(err) {
		t.Fatal("expected gemini override file at " + overridePath)
	}

	data, _ := os.ReadFile(overridePath)
	content := string(data)
	if !strings.Contains(content, "model: gemini-2.5-flash") {
		t.Error("override should contain overridden model")
	}
	if strings.Contains(content, "kind:") {
		t.Error("override should not contain kind field")
	}
	if strings.Contains(content, "version:") {
		t.Error("override should not contain version field")
	}
}
