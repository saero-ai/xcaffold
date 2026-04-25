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
	scaffoldPath := filepath.Join(tmpDir, ".xcaffold", "project.xcf")
	assert.FileExists(t, scaffoldPath)
	scaffoldBytes, err := os.ReadFile(scaffoldPath)
	require.NoError(t, err)
	scaffoldContent := string(scaffoldBytes)
	assert.Contains(t, scaffoldContent, "kind: project")
	assert.Contains(t, scaffoldContent, "name: my-project")
	assert.Contains(t, scaffoldContent, "claude")
	assert.Contains(t, scaffoldContent, "antigravity")
	assert.Contains(t, scaffoldContent, "developer")
	assert.Contains(t, scaffoldContent, "reviewer")
	assert.Contains(t, scaffoldContent, "tdd")
	assert.Contains(t, scaffoldContent, "security")

	// Agent files — each lives in its own subdirectory: xcf/agents/<id>/<id>.xcf
	assert.FileExists(t, filepath.Join(tmpDir, "xcf", "agents", "developer", "developer.xcf"))
	developerBytes, err := os.ReadFile(filepath.Join(tmpDir, "xcf", "agents", "developer", "developer.xcf"))
	require.NoError(t, err)
	assert.Contains(t, string(developerBytes), "kind: agent")
	assert.Contains(t, string(developerBytes), "name: developer")

	assert.FileExists(t, filepath.Join(tmpDir, "xcf", "agents", "reviewer", "reviewer.xcf"))

	// Skill file
	assert.FileExists(t, filepath.Join(tmpDir, "xcf", "skills", "tdd.xcf"))
	skillBytes, err := os.ReadFile(filepath.Join(tmpDir, "xcf", "skills", "tdd.xcf"))
	require.NoError(t, err)
	assert.Contains(t, string(skillBytes), "kind: skill")

	// Rule file
	assert.FileExists(t, filepath.Join(tmpDir, "xcf", "rules", "security.xcf"))
	ruleBytes, err := os.ReadFile(filepath.Join(tmpDir, "xcf", "rules", "security.xcf"))
	require.NoError(t, err)
	assert.Contains(t, string(ruleBytes), "kind: rule")

	// Hooks file
	assert.FileExists(t, filepath.Join(tmpDir, "xcf", "hooks.xcf"))
	hooksBytes, err := os.ReadFile(filepath.Join(tmpDir, "xcf", "hooks.xcf"))
	require.NoError(t, err)
	assert.Contains(t, string(hooksBytes), "kind: hooks")
	assert.Contains(t, string(hooksBytes), "events:")

	// Settings file
	assert.FileExists(t, filepath.Join(tmpDir, "xcf", "settings.xcf"))
	settingsBytes, err := os.ReadFile(filepath.Join(tmpDir, "xcf", "settings.xcf"))
	require.NoError(t, err)
	assert.Contains(t, string(settingsBytes), "kind: settings")
}

func TestWriteSplitFiles_RoundTrip(t *testing.T) {
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
	b1, err := os.ReadFile(filepath.Join(tmpDir1, ".xcaffold", "project.xcf"))
	require.NoError(t, err)
	b2, err := os.ReadFile(filepath.Join(tmpDir2, ".xcaffold", "project.xcf"))
	require.NoError(t, err)
	assert.Equal(t, b1, b2, ".xcaffold/project.xcf must be byte-identical")

	// Compare an agent file
	a1, err := os.ReadFile(filepath.Join(tmpDir1, "xcf", "agents", "alpha", "alpha.xcf"))
	require.NoError(t, err)
	a2, err := os.ReadFile(filepath.Join(tmpDir2, "xcf", "agents", "alpha", "alpha.xcf"))
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
	assert.FileExists(t, filepath.Join(tmpDir, ".xcaffold", "project.xcf"))

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

	_, statErr := os.Stat(filepath.Join(tmpDir, "xcf", "hooks.xcf"))
	assert.True(t, os.IsNotExist(statErr), "xcf/hooks.xcf should not be created when config has no hooks")
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
					Name:         "developer",
					Description:  "General developer.",
					Model:        "sonnet",
					Instructions: "You are a developer.\nWrite clean code.",
				},
			},
		},
	}
	err := WriteSplitFiles(config, dir)
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(dir, "xcf", "agents", "developer", "developer.xcf"))
	require.NoError(t, err)
	content := string(data)

	assert.True(t, strings.HasPrefix(content, "---\n"), "agent with body must use frontmatter")
	assert.Contains(t, content, "kind: agent")
	assert.Contains(t, content, "name: developer")
	assert.Contains(t, content, "---\nYou are a developer.")
	assert.NotContains(t, content, "instructions:")
}

func TestWriteSplitFiles_AgentInstructionsFile_PureYAML(t *testing.T) {
	dir := t.TempDir()
	config := &ast.XcaffoldConfig{
		Version: "1.0",
		Project: &ast.ProjectConfig{Name: "test"},
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"ceo": {
					Name:             "ceo",
					InstructionsFile: "agents/ceo.md",
				},
			},
		},
	}
	err := WriteSplitFiles(config, dir)
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(dir, "xcf", "agents", "ceo", "ceo.xcf"))
	require.NoError(t, err)
	content := string(data)

	assert.False(t, strings.HasPrefix(content, "---\n"), "instructions-file agent must be pure YAML")
	assert.Contains(t, content, "instructions-file: agents/ceo.md")
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
					AllowedTools: []string{"Read", "Edit", "Bash"},
					Instructions: "Follow Red-Green-Refactor.",
				},
			},
		},
	}
	err := WriteSplitFiles(config, dir)
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(dir, "xcf", "skills", "tdd.xcf"))
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
					Name:         "conventions",
					Description:  "Coding conventions.",
					Instructions: "Write clean code.\nUse 2-space indentation.",
				},
			},
		},
	}
	err := WriteSplitFiles(config, dir)
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(dir, "xcf", "rules", "conventions.xcf"))
	require.NoError(t, err)
	content := string(data)

	assert.True(t, strings.HasPrefix(content, "---\n"), "rule with body must use frontmatter")
	assert.Contains(t, content, "kind: rule")
	assert.Contains(t, content, "---\nWrite clean code.")
	assert.NotContains(t, content, "instructions:")
}

func TestWriteSplitFiles_ScopeFilter_OnlyDeclaredAgents(t *testing.T) {
	dir := t.TempDir()
	config := &ast.XcaffoldConfig{
		Version: "1.0",
		Project: &ast.ProjectConfig{
			Name:      "test",
			AgentRefs: []ast.AgentManifestEntry{{ID: "developer"}},
		},
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"developer": {Name: "developer", Instructions: "Local agent."},
				"ceo":       {Name: "ceo", InstructionsFile: "/home/.claude/agents/ceo.md"},
				"cfo":       {Name: "cfo", InstructionsFile: "/home/.claude/agents/cfo.md"},
			},
		},
	}
	err := WriteSplitFiles(config, dir)
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(dir, "xcf", "agents", "developer", "developer.xcf"))
	require.NoError(t, err, "declared agent must be written")

	_, err = os.Stat(filepath.Join(dir, "xcf", "agents", "ceo", "ceo.xcf"))
	assert.True(t, os.IsNotExist(err), "undeclared agent must NOT be written")

	_, err = os.Stat(filepath.Join(dir, "xcf", "agents", "cfo", "cfo.xcf"))
	assert.True(t, os.IsNotExist(err), "undeclared agent must NOT be written")
}

func TestWriteSplitFiles_ScopeFilter_EmptyRefs_WritesAll(t *testing.T) {
	dir := t.TempDir()
	config := &ast.XcaffoldConfig{
		Version: "1.0",
		Project: &ast.ProjectConfig{Name: "test"},
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"developer": {Name: "developer", Instructions: "Dev."},
				"reviewer":  {Name: "reviewer", Instructions: "Rev."},
			},
		},
	}
	err := WriteSplitFiles(config, dir)
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(dir, "xcf", "agents", "developer", "developer.xcf"))
	require.NoError(t, err)
	_, err = os.Stat(filepath.Join(dir, "xcf", "agents", "reviewer", "reviewer.xcf"))
	require.NoError(t, err)
}

func TestWriteSplitFiles_ScopeFilter_SkillsAndRules(t *testing.T) {
	dir := t.TempDir()
	config := &ast.XcaffoldConfig{
		Version: "1.0",
		Project: &ast.ProjectConfig{
			Name:      "test",
			SkillRefs: []string{"tdd"},
			RuleRefs:  []string{"conventions"},
		},
		ResourceScope: ast.ResourceScope{
			Skills: map[string]ast.SkillConfig{
				"tdd":       {Name: "tdd", Instructions: "Red-Green-Refactor."},
				"global-sk": {Name: "global-sk", InstructionsFile: "/home/.claude/skills/global/SKILL.md"},
			},
			Rules: map[string]ast.RuleConfig{
				"conventions": {Name: "conventions", Instructions: "Write clean code."},
				"global-rule": {Name: "global-rule", InstructionsFile: "/home/.claude/rules/global.md"},
			},
		},
	}
	err := WriteSplitFiles(config, dir)
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(dir, "xcf", "skills", "tdd.xcf"))
	require.NoError(t, err, "declared skill must be written")
	_, err = os.Stat(filepath.Join(dir, "xcf", "skills", "global-sk.xcf"))
	assert.True(t, os.IsNotExist(err), "undeclared skill must NOT be written")

	_, err = os.Stat(filepath.Join(dir, "xcf", "rules", "conventions.xcf"))
	require.NoError(t, err, "declared rule must be written")
	_, err = os.Stat(filepath.Join(dir, "xcf", "rules", "global-rule.xcf"))
	assert.True(t, os.IsNotExist(err), "undeclared rule must NOT be written")
}

func TestWriteSplitFiles_SkillSubdirFields(t *testing.T) {
	outDir := t.TempDir()
	config := &ast.XcaffoldConfig{
		Version: "1.0",
		Project: &ast.ProjectConfig{Name: "test"},
		ResourceScope: ast.ResourceScope{
			Skills: map[string]ast.SkillConfig{
				"my-skill": {
					Name:         "my-skill",
					Description:  "Test skill",
					References:   []string{"xcf/skills/my-skill/references/ref.md"},
					Scripts:      []string{"xcf/skills/my-skill/scripts/run.sh"},
					Assets:       []string{"xcf/skills/my-skill/assets/icon.svg"},
					Examples:     []string{"xcf/skills/my-skill/examples/sample.md"},
					Instructions: "Do the thing.",
				},
			},
		},
	}
	err := WriteSplitFiles(config, outDir)
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(outDir, "xcf", "skills", "my-skill.xcf"))
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

func TestWriteSplitFiles_Memory_SlashKey(t *testing.T) {
	dir := t.TempDir()
	config := &ast.XcaffoldConfig{
		Version: "1.0",
		Project: &ast.ProjectConfig{Name: "test"},
		ResourceScope: ast.ResourceScope{
			Memory: map[string]ast.MemoryConfig{
				"developer/architecture-decisions": {
					Name:    "architecture-decisions",
					Content: "Keep decisions short.",
				},
			},
		},
	}
	err := WriteSplitFiles(config, dir)
	require.NoError(t, err)

	expected := filepath.Join(dir, "xcf", "agents", "developer", "memory", "architecture-decisions.xcf")
	assert.FileExists(t, expected, "slash-key memory must go to xcf/agents/<agentID>/memory/<name>.xcf")

	data, err := os.ReadFile(expected)
	require.NoError(t, err)
	assert.Contains(t, string(data), "kind: memory")
}

// TestWriteSplitFiles_Memory_AgentRef verifies that when a memory key has no slash,
// AgentRef on MemoryConfig is used to determine the agent directory.
func TestWriteSplitFiles_Memory_AgentRef(t *testing.T) {
	dir := t.TempDir()
	config := &ast.XcaffoldConfig{
		Version: "1.0",
		Project: &ast.ProjectConfig{Name: "test"},
		ResourceScope: ast.ResourceScope{
			Memory: map[string]ast.MemoryConfig{
				"architecture-decisions": {
					Name:     "architecture-decisions",
					AgentRef: "developer",
					Content:  "Keep decisions short.",
				},
			},
		},
	}
	err := WriteSplitFiles(config, dir)
	require.NoError(t, err)

	expected := filepath.Join(dir, "xcf", "agents", "developer", "memory", "architecture-decisions.xcf")
	assert.FileExists(t, expected, "AgentRef must determine xcf/agents/<agentID>/memory/<name>.xcf when key has no slash")

	data, err := os.ReadFile(expected)
	require.NoError(t, err)
	assert.Contains(t, string(data), "kind: memory")
}

func TestWriteFrontmatterFile_EmptyBody_FallsBackToYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ceo.xcf")

	type frontmatterDoc struct {
		Kind             string `yaml:"kind"`
		Version          string `yaml:"version"`
		Name             string `yaml:"name"`
		InstructionsFile string `yaml:"instructions-file"`
	}

	doc := frontmatterDoc{Kind: "agent", Version: "1.0", Name: "ceo", InstructionsFile: "agents/ceo.md"}

	err := writeFrontmatterFile(path, doc, "")
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	content := string(data)
	assert.False(t, strings.HasPrefix(content, "---\n"), "empty body must NOT use frontmatter")
	assert.Contains(t, content, "kind: agent")
	assert.Contains(t, content, "instructions-file: agents/ceo.md")
}
