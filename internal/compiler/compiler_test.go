package compiler

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/renderer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompile_SingleAgent(t *testing.T) {
	config := &ast.XcaffoldConfig{
		Project: &ast.ProjectConfig{Name: "test-project"},
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"developer": {
					Description: "An expert developer.",
					Body:        "You are a software developer.\nWrite clean code.\n",
					Model:       "claude-3-7-sonnet-20250219",
					Effort:      "high",
					Tools:       ast.ClearableList{Values: []string{"Bash", "Read", "Write"}},
				},
			},
		},
	}

	out, _, err := Compile(config, "", CompileOpts{Target: "claude"})
	require.NoError(t, err)
	require.NotNil(t, out)

	content, ok := out.Files["agents/developer.md"]
	require.True(t, ok, "expected agents/developer.md to be compiled")

	assert.Contains(t, content, "description: An expert developer.")
	assert.Contains(t, content, "model: claude-3-7-sonnet-20250219")
	assert.Contains(t, content, "effort: high")
	assert.Contains(t, content, "tools: [Bash, Read, Write]")
	assert.Contains(t, content, "You are a software developer.")
}

func TestCompile_MultipleAgents(t *testing.T) {
	config := &ast.XcaffoldConfig{
		Project: &ast.ProjectConfig{Name: "multi-agent-project"},
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"frontend": {Description: "Frontend specialist."},
				"backend":  {Description: "Backend specialist."},
			},
		},
	}

	out, _, err := Compile(config, "", CompileOpts{Target: "claude"})
	require.NoError(t, err)
	assert.Len(t, out.Files, 2)
	assert.Contains(t, out.Files, "agents/frontend.md")
	assert.Contains(t, out.Files, "agents/backend.md")
}

func TestCompile_AgentWithBlockedTools(t *testing.T) {
	config := &ast.XcaffoldConfig{
		Project: &ast.ProjectConfig{Name: "secure-project"},
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"readonly": {
					Description:     "Read-only agent.",
					Tools:           ast.ClearableList{Values: []string{"Read", "Grep"}},
					DisallowedTools: ast.ClearableList{Values: []string{"Bash", "Write"}},
				},
			},
		},
	}

	out, _, err := Compile(config, "", CompileOpts{Target: "claude"})
	require.NoError(t, err)
	assert.Contains(t, out.Files["agents/readonly.md"], "disallowed-tools: [Bash, Write]")
}

func TestCompile_EmptyAgents(t *testing.T) {
	config := &ast.XcaffoldConfig{
		Project: &ast.ProjectConfig{Name: "empty-project"},
	}
	out, _, err := Compile(config, "", CompileOpts{Target: "claude"})
	require.NoError(t, err)
	assert.Empty(t, out.Files)
}

func TestCompile_FullSchema(t *testing.T) {
	config := &ast.XcaffoldConfig{
		Project: &ast.ProjectConfig{Name: "full-project"},
		Hooks: map[string]ast.NamedHookConfig{
			"default": {
				Name: "default",
				Events: ast.HookConfig{
					"PreToolUse": []ast.HookMatcherGroup{
						{
							Matcher: "Bash",
							Hooks:   []ast.HookHandler{{Type: "command", Command: "make test"}},
						},
					},
				},
			},
		},
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"dev": {Description: "A developer."},
			},
			Skills: map[string]ast.SkillConfig{
				"git": {
					Description: "Git workflows",
					Body:        "Always use rebase.",
				},
			},
			Rules: map[string]ast.RuleConfig{
				"go": {
					Body: "Use gofmt.",
				},
			},
			MCP: map[string]ast.MCPConfig{
				"db": {
					Command: "npx",
					Args:    []string{"-y", "sqlite"},
				},
			},
		},
	}

	out, _, err := Compile(config, "", CompileOpts{Target: "claude"})
	require.NoError(t, err)

	// Agents
	assert.Contains(t, out.Files, "agents/dev.md")

	// Skills — compiled as skills/<id>/SKILL.md directories
	skillContent, ok := out.Files["skills/git/SKILL.md"]
	require.True(t, ok, "expected skills/git/SKILL.md to be compiled")
	assert.Contains(t, skillContent, "description: Git workflows")
	assert.Contains(t, skillContent, "Always use rebase.")

	// Rules
	ruleContent, ok := out.Files["rules/go.md"]
	require.True(t, ok, "expected rules/go.md to be compiled")
	assert.Contains(t, ruleContent, "Use gofmt.")

	settingsJSON, hasSettings := out.Files["settings.json"]
	require.True(t, hasSettings, "expected settings.json to be compiled")
	assert.Contains(t, settingsJSON, `"PreToolUse"`)
	assert.Contains(t, settingsJSON, `"make test"`)
	assert.Contains(t, settingsJSON, `"command"`)

	// MCP (which was previously in settings.json)
	mcpContent, ok := out.Files["mcp.json"]
	require.True(t, ok, "expected mcp.json to be compiled")
	assert.Contains(t, mcpContent, `"db"`)
	assert.Contains(t, mcpContent, `"npx"`)
	assert.Contains(t, mcpContent, `"sqlite"`)
}

func TestCompile_CursorTarget_Supported(t *testing.T) {
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"test": {Description: "Test", Body: "Test rule."},
			},
		},
	}
	out, _, err := Compile(config, "", CompileOpts{Target: "cursor"})
	require.NoError(t, err)
	assert.NotEmpty(t, out.Files)
}

func TestCompile_CursorTarget_RulesUseMdc(t *testing.T) {
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"style": {Description: "Style", Body: "Format code."},
			},
		},
	}
	out, _, err := Compile(config, "", CompileOpts{Target: "cursor"})
	require.NoError(t, err)
	_, ok := out.Files["rules/style.mdc"]
	assert.True(t, ok, "Cursor rules should use .mdc extension")
}

func TestCompile_AntigravityTarget_Supported(t *testing.T) {
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"test": {Description: "Test", Body: "Test rule."},
			},
		},
	}
	out, _, err := Compile(config, "", CompileOpts{Target: "antigravity"})
	require.NoError(t, err)
	assert.NotEmpty(t, out.Files)
}

func TestCompile_AntigravityTarget_RulesEmitFrontmatter(t *testing.T) {
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"style": {Description: "Style", Body: "Format code."},
			},
		},
	}
	out, _, err := Compile(config, "", CompileOpts{Target: "antigravity"})
	require.NoError(t, err)
	content, ok := out.Files["rules/style.md"]
	assert.True(t, ok)
	assert.Contains(t, content, "---\n")
	assert.Contains(t, content, "description: Style\n")
}

func TestCompile_AntigravityTarget_AgentsIncluded(t *testing.T) {
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"reviewer": {Name: "Reviewer", Description: "test agent", Body: "Review."},
			},
			Rules: map[string]ast.RuleConfig{
				"test": {Body: "Test."},
			},
		},
	}
	out, _, err := Compile(config, "", CompileOpts{Target: "antigravity"})
	require.NoError(t, err)
	// Both rule and agent should appear (agent as note)
	assert.Len(t, out.Files, 2)
}

func TestCompileAgentMarkdown_PathTraversalPrevented(t *testing.T) {
	// An agent id containing path separators should be cleaned safely.
	config := &ast.XcaffoldConfig{
		Project: &ast.ProjectConfig{Name: "traversal-test"},
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"../evil": {Description: "Malicious agent."},
			},
		},
	}
	out, _, err := Compile(config, "", CompileOpts{Target: "claude"})
	require.NoError(t, err)
	for path := range out.Files {
		assert.NotContains(t, path, "..", "output path must not contain traversal sequences")
	}
}

func TestCompile_ResolveAttributes_SkillToolsInherited(t *testing.T) {
	config := &ast.XcaffoldConfig{
		Version: "1.0",
		ResourceScope: ast.ResourceScope{
			Skills: map[string]ast.SkillConfig{
				"tdd": {
					Description:  "TDD workflow",
					AllowedTools: ast.ClearableList{Values: []string{"Bash", "Read", "Write"}},
					Body:         "Follow TDD",
				},
			},
			Agents: map[string]ast.AgentConfig{
				"developer": {
					Description: "Dev agent",
					Model:       "sonnet",
					Tools:       ast.ClearableList{Values: []string{"${skill.tdd.allowed-tools}"}},
					Skills:      ast.ClearableList{Values: []string{"tdd"}},
					Body:        "You are a developer",
				},
			},
		},
	}

	output, _, err := Compile(config, t.TempDir(), CompileOpts{Target: "claude", Blueprint: "", VarFile: ""})
	require.NoError(t, err)

	// The compiled agent output should have the resolved tools, not the ${...} reference
	agentContent := output.Files["agents/developer.md"]
	assert.Contains(t, agentContent, "Bash")
	assert.Contains(t, agentContent, "Read")
	assert.Contains(t, agentContent, "Write")
	assert.NotContains(t, agentContent, "${skill.tdd.allowed-tools}")
}

func TestCompile_ResolveAttributes_NoRefsPassthrough(t *testing.T) {
	config := &ast.XcaffoldConfig{
		Version: "1.0",
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"developer": {
					Description: "Dev agent",
					Model:       "sonnet",
					Tools:       ast.ClearableList{Values: []string{"Bash", "Read"}},
					Body:        "You are a developer",
				},
			},
		},
	}

	output, _, err := Compile(config, t.TempDir(), CompileOpts{Target: "claude", Blueprint: "", VarFile: ""})
	require.NoError(t, err)
	assert.Contains(t, output.Files["agents/developer.md"], "Dev agent")
}

// Plan A4: ensure notes emitted by a target renderer are threaded through
// compiler.Compile's return values.
func TestOutputDir_AllTargets(t *testing.T) {
	tests := []struct {
		target string
		want   string
	}{
		{"", ""},
		{"claude", ".claude"},
		{"cursor", ".cursor"},
		{"antigravity", ".agents"},
		{"copilot", ".github"},
		{"gemini", ".gemini"},
		{"unknown", ""},
	}
	for _, tt := range tests {
		t.Run(tt.target, func(t *testing.T) {
			got := OutputDir(tt.target)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCompile_Gemini_DispatchesGeminiRenderer(t *testing.T) {
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"style": {Description: "Style guide", Body: "Format code."},
			},
		},
	}
	out, _, err := Compile(config, t.TempDir(), CompileOpts{Target: "gemini", Blueprint: "", VarFile: ""})
	require.NoError(t, err)
	assert.NotNil(t, out)
	assert.NotEmpty(t, out.Files)
}

func TestOutputDir_Gemini_ReturnsDotGemini(t *testing.T) {
	got := OutputDir("gemini")
	assert.Equal(t, ".gemini", got)
}

func TestCompile_FidelityNotes_Propagated_FromCursor(t *testing.T) {
	// Two-layer fidelity check: permission-mode has Role:["rendering"] and is
	// unsupported by cursor. The orchestrator silently skips it — no note is
	// emitted. Compile must succeed and must NOT emit FIELD_UNSUPPORTED for
	// permission-mode.
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"reviewer": {
					Description:    "Code review agent.",
					PermissionMode: "acceptEdits",
				},
			},
		},
	}

	_, notes, err := Compile(config, t.TempDir(), CompileOpts{Target: "cursor", Blueprint: "", VarFile: ""})
	require.NoError(t, err)

	for _, n := range notes {
		if n.Code == renderer.CodeFieldUnsupported && n.Field == "permission-mode" {
			t.Errorf("permission-mode has an xcaf role; FIELD_UNSUPPORTED must not be emitted for cursor, got: %s", n.Reason)
		}
	}
}

func TestCompile_Blueprint_FiltersResources(t *testing.T) {
	cfg := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"developer": {Name: "Developer", Description: "A developer", Body: "dev instructions"},
				"designer":  {Name: "Designer", Description: "A designer", Body: "design instructions"},
			},
		},
		Blueprints: map[string]ast.BlueprintConfig{
			"backend": {Name: "backend", Agents: ast.ClearableList{Values: []string{"developer"}}},
		},
	}
	out, _, err := Compile(cfg, t.TempDir(), CompileOpts{Target: "claude", Blueprint: "backend"})
	require.NoError(t, err)

	hasAgent := func(name string) bool {
		for path := range out.Files {
			if strings.Contains(path, name) {
				return true
			}
		}
		return false
	}
	require.True(t, hasAgent("developer"), "developer agent must be present in blueprint output")
	require.False(t, hasAgent("designer"), "designer agent must be excluded by blueprint filter")
}

func TestCompile_NoBlueprint_CompilesAll(t *testing.T) {
	cfg := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"developer": {Name: "Developer", Description: "d", Body: "x"},
				"designer":  {Name: "Designer", Description: "d", Body: "x"},
			},
		},
	}
	out, _, err := Compile(cfg, t.TempDir(), CompileOpts{Target: "claude", Blueprint: ""})
	require.NoError(t, err)

	count := 0
	for path := range out.Files {
		if strings.Contains(path, "agents/") {
			count++
		}
	}
	require.Equal(t, 2, count, "all agents must be compiled when blueprintName is empty")
}

func TestCompile_UnknownBlueprint_Error(t *testing.T) {
	cfg := &ast.XcaffoldConfig{
		Blueprints: map[string]ast.BlueprintConfig{
			"backend": {Name: "backend"},
		},
	}
	_, _, err := Compile(cfg, t.TempDir(), CompileOpts{Target: "claude", Blueprint: "ghost"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "ghost")
}

// TestCompile_BlueprintExtends_InheritedResources verifies that extends
// resolution runs before ApplyBlueprint so a child blueprint inherits
// the parent's ref-lists and the inherited agents are compiled.
func TestCompile_BlueprintExtends_InheritedResources(t *testing.T) {
	cfg := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"base-agent":  {Name: "BaseAgent", Description: "base", Body: "base instructions"},
				"child-agent": {Name: "ChildAgent", Description: "child", Body: "child instructions"},
			},
		},
		Blueprints: map[string]ast.BlueprintConfig{
			"base":  {Name: "base", Agents: ast.ClearableList{Values: []string{"base-agent"}}},
			"child": {Name: "child", Extends: "base", Agents: ast.ClearableList{Values: []string{"child-agent"}}},
		},
	}

	out, _, err := Compile(cfg, t.TempDir(), CompileOpts{Target: "claude", Blueprint: "child"})
	require.NoError(t, err)

	hasAgent := func(name string) bool {
		for path := range out.Files {
			if strings.Contains(path, name) {
				return true
			}
		}
		return false
	}
	require.True(t, hasAgent("child-agent"), "child-agent must be compiled for 'child' blueprint")
	require.True(t, hasAgent("base-agent"), "base-agent must be compiled — inherited via extends from 'base'")
}

// TestCompile_BlueprintTransitiveDeps_AutoExpandsSkills verifies that
// ResolveTransitiveDeps is called so an agent's skills are auto-included
// when the blueprint lists no explicit skills.
func TestCompile_BlueprintTransitiveDeps_AutoExpandsSkills(t *testing.T) {
	cfg := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"dev": {
					Name:        "Dev",
					Description: "developer",
					Body:        "dev instructions",
					Skills:      ast.ClearableList{Values: []string{"tdd"}},
				},
			},
			Skills: map[string]ast.SkillConfig{
				"tdd": {Name: "tdd", Description: "TDD skill", Body: "follow tdd"},
			},
		},
		Blueprints: map[string]ast.BlueprintConfig{
			// Only agents listed; skills intentionally empty so transitive dep
			// resolution should populate them automatically.
			"backend": {Name: "backend", Agents: ast.ClearableList{Values: []string{"dev"}}},
		},
	}

	out, _, err := Compile(cfg, t.TempDir(), CompileOpts{Target: "claude", Blueprint: "backend"})
	require.NoError(t, err)

	hasSkill := func(name string) bool {
		for path := range out.Files {
			if strings.Contains(path, name) {
				return true
			}
		}
		return false
	}
	require.True(t, hasSkill("tdd"), "tdd skill must be compiled via transitive dep expansion")
}

// TestCompile_BlueprintValidation_RunsAfterExtends ensures that
// ValidateBlueprintRefs is evaluated after extends resolution. A child
// blueprint that references resources only available through the parent
// must not produce a validation error.
func TestCompile_BlueprintValidation_RunsAfterExtends(t *testing.T) {
	// "child" extends "base" and picks up "base-agent" through inheritance.
	// Without post-extends validation this would erroneously report
	// "base-agent" as unknown for the "child" blueprint.
	cfg := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"base-agent": {Name: "BaseAgent", Description: "base", Body: "base instructions"},
			},
		},
		Blueprints: map[string]ast.BlueprintConfig{
			"base":  {Name: "base", Agents: ast.ClearableList{Values: []string{"base-agent"}}},
			"child": {Name: "child", Extends: "base"},
		},
	}

	_, _, err := Compile(cfg, t.TempDir(), CompileOpts{Target: "claude", Blueprint: "child"})
	require.NoError(t, err, "child blueprint inheriting base-agent via extends must compile without error")
}

func TestDiscoverAgentMemory_FindsMdFiles(t *testing.T) {
	dir := t.TempDir()
	agentMemDir := filepath.Join(dir, "xcaf", "agents", "backend-dev", "memory")
	require.NoError(t, os.MkdirAll(agentMemDir, 0o755))

	content := "---\nname: ORM Decision\ndescription: \"Always use Drizzle\"\n---\n\nWe chose Drizzle ORM.\n"
	require.NoError(t, os.WriteFile(filepath.Join(agentMemDir, "orm-decision.md"), []byte(content), 0o644))

	result := DiscoverAgentMemory(dir, nil, nil)
	require.Contains(t, result, "backend-dev/orm-decision")
	entry := result["backend-dev/orm-decision"]
	assert.Equal(t, "ORM Decision", entry.Name)
	assert.Equal(t, "Always use Drizzle", entry.Description)
	assert.Contains(t, entry.Content, "We chose Drizzle ORM.")
	assert.Equal(t, "backend-dev", entry.AgentRef)
}

func TestDiscoverAgentMemory_SkipsMemoryMd(t *testing.T) {
	dir := t.TempDir()
	agentMemDir := filepath.Join(dir, "xcaf", "agents", "dev", "memory")
	require.NoError(t, os.MkdirAll(agentMemDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(agentMemDir, "MEMORY.md"), []byte("index"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(agentMemDir, "real-entry.md"), []byte("content"), 0o644))

	result := DiscoverAgentMemory(dir, nil, nil)
	assert.NotContains(t, result, "dev/MEMORY")
	assert.Contains(t, result, "dev/real-entry")
}

func TestDiscoverAgentMemory_FallbackNameDescription(t *testing.T) {
	dir := t.TempDir()
	agentMemDir := filepath.Join(dir, "xcaf", "agents", "dev", "memory")
	require.NoError(t, os.MkdirAll(agentMemDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(agentMemDir, "simple.md"), []byte("First line of content.\nSecond line."), 0o644))

	result := DiscoverAgentMemory(dir, nil, nil)
	entry := result["dev/simple"]
	assert.Equal(t, "simple", entry.Name)
	assert.Equal(t, "First line of content.", entry.Description)
	assert.Equal(t, "First line of content.\nSecond line.", entry.Content)
}

func TestDiscoverAgentMemory_FallbackDescriptionTruncated(t *testing.T) {
	dir := t.TempDir()
	agentMemDir := filepath.Join(dir, "xcaf", "agents", "dev", "memory")
	require.NoError(t, os.MkdirAll(agentMemDir, 0o755))
	longLine := strings.Repeat("a", 200)
	require.NoError(t, os.WriteFile(filepath.Join(agentMemDir, "long.md"), []byte(longLine), 0o644))

	result := DiscoverAgentMemory(dir, nil, nil)
	entry := result["dev/long"]
	assert.Len(t, []rune(entry.Description), 120)
}

func TestDiscoverAgentMemory_IgnoresXcafFiles(t *testing.T) {
	dir := t.TempDir()
	agentMemDir := filepath.Join(dir, "xcaf", "agents", "dev", "memory")
	require.NoError(t, os.MkdirAll(agentMemDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(agentMemDir, "old.xcaf"), []byte("kind: memory"), 0o644))

	result := DiscoverAgentMemory(dir, nil, nil)
	assert.Empty(t, result)
}

func TestCompile_OverrideMerge_AppliesForTarget(t *testing.T) {
	// Base agent uses the balanced alias. The claude-target override swaps it to
	// flagship, which maps to "claude-opus-4-7" in the Claude renderer's model table.
	config := &ast.XcaffoldConfig{
		Version: "1.0",
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"developer": {
					Description: "A developer.",
					Model:       "balanced",
					Body:        "You write code.",
				},
			},
		},
		Overrides: &ast.ResourceOverrides{},
	}
	config.Overrides.AddAgent("developer", "claude", ast.AgentConfig{Model: "flagship"})

	out, notes, err := Compile(config, t.TempDir(), CompileOpts{Target: "claude", Blueprint: "", VarFile: ""})
	require.NoError(t, err)
	require.NotNil(t, out)

	// The override model (flagship → opus bare alias) must appear in the compiled output.
	agentContent, ok := out.Files["agents/developer.md"]
	require.True(t, ok, "expected agents/developer.md to be compiled")
	assert.Contains(t, agentContent, "model: opus", "override model must appear in compiled agent")
	assert.NotContains(t, agentContent, "model: sonnet", "base model must be replaced by override")

	// No RESOURCE_TARGET_SKIPPED notes: the agent has no Targets restriction.
	for _, n := range notes {
		assert.NotEqual(t, CodeResourceTargetSkipped, n.Code,
			"universal (no Targets) agent must never emit RESOURCE_TARGET_SKIPPED")
	}
}

func TestDiscoverAgentMemory_NoMemoryDir(t *testing.T) {
	dir := t.TempDir()
	agentDir := filepath.Join(dir, "xcaf", "agents", "dev")
	require.NoError(t, os.MkdirAll(agentDir, 0o755))

	result := DiscoverAgentMemory(dir, nil, nil)
	assert.Empty(t, result)
}

// ---- mergeResourceScope tests ----

func TestMergeResourceScope_Policies(t *testing.T) {
	root := &ast.ResourceScope{
		Policies: map[string]ast.PolicyConfig{
			"global-policy": {Description: "global"},
		},
	}
	project := &ast.ResourceScope{
		Policies: map[string]ast.PolicyConfig{
			"project-policy": {Description: "project"},
		},
	}
	mergeResourceScope(root, project)
	assert.Len(t, root.Policies, 2)
	assert.Contains(t, root.Policies, "global-policy")
	assert.Contains(t, root.Policies, "project-policy")
}

func TestMergeResourceScope_Memory(t *testing.T) {
	root := &ast.ResourceScope{
		Memory: map[string]ast.MemoryConfig{
			"global-mem": {Name: "global"},
		},
	}
	project := &ast.ResourceScope{
		Memory: map[string]ast.MemoryConfig{
			"project-mem": {Name: "project"},
		},
	}
	mergeResourceScope(root, project)
	assert.Len(t, root.Memory, 2)
	assert.Contains(t, root.Memory, "global-mem")
	assert.Contains(t, root.Memory, "project-mem")
}

func TestMergeResourceScope_Contexts(t *testing.T) {
	root := &ast.ResourceScope{
		Contexts: map[string]ast.ContextConfig{
			"global-ctx": {Description: "global"},
		},
	}
	project := &ast.ResourceScope{
		Contexts: map[string]ast.ContextConfig{
			"project-ctx": {Description: "project"},
		},
	}
	mergeResourceScope(root, project)
	assert.Len(t, root.Contexts, 2)
	assert.Contains(t, root.Contexts, "global-ctx")
	assert.Contains(t, root.Contexts, "project-ctx")
}

func TestMergeResourceScope_Templates(t *testing.T) {
	root := &ast.ResourceScope{
		Templates: map[string]ast.TemplateConfig{
			"global-tmpl": {Description: "global"},
		},
	}
	project := &ast.ResourceScope{
		Templates: map[string]ast.TemplateConfig{
			"project-tmpl": {Description: "project"},
		},
	}
	mergeResourceScope(root, project)
	assert.Len(t, root.Templates, 2)
	assert.Contains(t, root.Templates, "global-tmpl")
	assert.Contains(t, root.Templates, "project-tmpl")
}

func TestMergeResourceScope_All9Kinds(t *testing.T) {
	root := &ast.ResourceScope{
		Agents:    map[string]ast.AgentConfig{"g-agent": {}},
		Skills:    map[string]ast.SkillConfig{"g-skill": {}},
		Rules:     map[string]ast.RuleConfig{"g-rule": {}},
		MCP:       map[string]ast.MCPConfig{"g-mcp": {}},
		Workflows: map[string]ast.WorkflowConfig{"g-wf": {}},
		Policies:  map[string]ast.PolicyConfig{"g-policy": {}},
		Memory:    map[string]ast.MemoryConfig{"g-mem": {}},
		Contexts:  map[string]ast.ContextConfig{"g-ctx": {}},
		Templates: map[string]ast.TemplateConfig{"g-tmpl": {}},
	}
	project := &ast.ResourceScope{
		Agents:    map[string]ast.AgentConfig{"p-agent": {}},
		Skills:    map[string]ast.SkillConfig{"p-skill": {}},
		Rules:     map[string]ast.RuleConfig{"p-rule": {}},
		MCP:       map[string]ast.MCPConfig{"p-mcp": {}},
		Workflows: map[string]ast.WorkflowConfig{"p-wf": {}},
		Policies:  map[string]ast.PolicyConfig{"p-policy": {}},
		Memory:    map[string]ast.MemoryConfig{"p-mem": {}},
		Contexts:  map[string]ast.ContextConfig{"p-ctx": {}},
		Templates: map[string]ast.TemplateConfig{"p-tmpl": {}},
	}
	mergeResourceScope(root, project)

	assert.Len(t, root.Agents, 2)
	assert.Len(t, root.Skills, 2)
	assert.Len(t, root.Rules, 2)
	assert.Len(t, root.MCP, 2)
	assert.Len(t, root.Workflows, 2)
	assert.Len(t, root.Policies, 2)
	assert.Len(t, root.Memory, 2)
	assert.Len(t, root.Contexts, 2)
	assert.Len(t, root.Templates, 2)
}

func TestMergeResourceScope_ProjectOverridesGlobal(t *testing.T) {
	root := &ast.ResourceScope{
		Agents:    map[string]ast.AgentConfig{"shared": {Description: "global"}},
		Policies:  map[string]ast.PolicyConfig{"shared": {Description: "global"}},
		Memory:    map[string]ast.MemoryConfig{"shared": {Name: "global"}},
		Contexts:  map[string]ast.ContextConfig{"shared": {Description: "global"}},
		Templates: map[string]ast.TemplateConfig{"shared": {Description: "global"}},
	}
	project := &ast.ResourceScope{
		Agents:    map[string]ast.AgentConfig{"shared": {Description: "project"}},
		Policies:  map[string]ast.PolicyConfig{"shared": {Description: "project"}},
		Memory:    map[string]ast.MemoryConfig{"shared": {Name: "project"}},
		Contexts:  map[string]ast.ContextConfig{"shared": {Description: "project"}},
		Templates: map[string]ast.TemplateConfig{"shared": {Description: "project"}},
	}
	mergeResourceScope(root, project)

	assert.Equal(t, "project", root.Agents["shared"].Description)
	assert.Equal(t, "project", root.Policies["shared"].Description)
	assert.Equal(t, "project", root.Memory["shared"].Name)
	assert.Equal(t, "project", root.Contexts["shared"].Description)
	assert.Equal(t, "project", root.Templates["shared"].Description)
}

// ---- Policy additive-only merge enforcement tests ----

// TestCompile_PolicyConflict_SameName_Errors verifies that a project policy with
// the same name as an inherited global policy returns an error at compile time.
func TestCompile_PolicyConflict_SameName_Errors(t *testing.T) {
	inheritedPolicy := ast.PolicyConfig{
		Name:        "no-bash",
		Description: "inherited global",
		Severity:    "error",
		Target:      "agent",
		Inherited:   true,
	}
	conflictingPolicy := ast.PolicyConfig{
		Name:        "no-bash",
		Description: "project override attempt",
		Severity:    "warning",
		Target:      "agent",
	}

	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Policies: map[string]ast.PolicyConfig{
				"no-bash": inheritedPolicy,
			},
		},
		Project: &ast.ProjectConfig{
			Name: "test-project",
			ResourceScope: ast.ResourceScope{
				Policies: map[string]ast.PolicyConfig{
					"no-bash": conflictingPolicy,
				},
			},
		},
	}

	_, _, err := Compile(config, t.TempDir(), CompileOpts{Target: "claude"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), `policy "no-bash"`)
	assert.Contains(t, err.Error(), "projects can add new policies but cannot override inherited ones")
}

// TestCompile_PolicyAdditive_NewPolicy_OK verifies that a project can add a new
// policy that does not exist in global scope without error.
func TestCompile_PolicyAdditive_NewPolicy_OK(t *testing.T) {
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Policies: map[string]ast.PolicyConfig{
				"global-policy": {
					Name:        "global-policy",
					Description: "inherited global policy",
					Severity:    "error",
					Target:      "agent",
					Inherited:   true,
				},
			},
		},
		Project: &ast.ProjectConfig{
			Name: "test-project",
			ResourceScope: ast.ResourceScope{
				Policies: map[string]ast.PolicyConfig{
					"project-policy": {
						Name:        "project-policy",
						Description: "new project policy",
						Severity:    "warning",
						Target:      "agent",
					},
				},
			},
		},
	}

	_, _, err := Compile(config, t.TempDir(), CompileOpts{Target: "claude"})
	require.NoError(t, err)
}

// TestCompile_PolicyInherited_NoConflict verifies that a project policy with the
// same name as a non-inherited global policy does NOT produce a conflict error.
// Only policies marked Inherited=true are protected from override.
func TestCompile_PolicyInherited_NoConflict(t *testing.T) {
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Policies: map[string]ast.PolicyConfig{
				"local-policy": {
					Name:        "local-policy",
					Description: "non-inherited global policy",
					Severity:    "error",
					Target:      "agent",
					Inherited:   false, // explicitly NOT inherited
				},
			},
		},
		Project: &ast.ProjectConfig{
			Name: "test-project",
			ResourceScope: ast.ResourceScope{
				Policies: map[string]ast.PolicyConfig{
					"local-policy": {
						Name:        "local-policy",
						Description: "project override of non-inherited policy",
						Severity:    "warning",
						Target:      "agent",
					},
				},
			},
		},
	}

	_, _, err := Compile(config, t.TempDir(), CompileOpts{Target: "claude"})
	require.NoError(t, err)
}

// ── Context path-scoped uniqueness through Compile ────────────────────────────

// TestCompile_TwoContexts_SameTarget_DifferentPaths_OK verifies that two
// contexts targeting the same provider but at different paths (root vs "backend")
// are not treated as ambiguous by ValidateContextUniqueness when called through
// Compile in bare-apply mode (no blueprint).
func TestCompile_TwoContexts_SameTarget_DifferentPaths_OK(t *testing.T) {
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Contexts: map[string]ast.ContextConfig{
				"root-ctx": {
					Name:    "root-ctx",
					Body:    "Root workspace instructions.",
					Targets: []string{"claude"},
					Path:    "",
				},
				"backend-ctx": {
					Name:    "backend-ctx",
					Body:    "Backend-scoped instructions.",
					Targets: []string{"claude"},
					Path:    "backend",
				},
			},
		},
	}
	_, _, err := Compile(config, t.TempDir(), CompileOpts{Target: "claude"})
	require.NoError(t, err, "two contexts at different paths must not trigger a uniqueness error")
}

// TestCompile_TwoContexts_SameTarget_SamePath_NoDefault_Error verifies that two
// contexts both targeting "claude" at the same path ("backend"), neither marked
// default, cause Compile to return a context validation error in bare-apply mode.
func TestCompile_TwoContexts_SameTarget_SamePath_NoDefault_Error(t *testing.T) {
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Contexts: map[string]ast.ContextConfig{
				"backend-a": {
					Name:    "backend-a",
					Body:    "First backend context.",
					Targets: []string{"claude"},
					Path:    "backend",
				},
				"backend-b": {
					Name:    "backend-b",
					Body:    "Second backend context.",
					Targets: []string{"claude"},
					Path:    "backend",
				},
			},
		},
	}
	_, _, err := Compile(config, t.TempDir(), CompileOpts{Target: "claude"})
	require.Error(t, err, "two contexts at the same path with no default must return an error")
	assert.Contains(t, err.Error(), "context validation failed")
	assert.Contains(t, err.Error(), `"claude"`)
	assert.Contains(t, err.Error(), `"backend"`)
}

func TestMergeResourceScope_NilMaps(t *testing.T) {
	// nil global + non-nil project: no panic, project entries present
	t.Run("nil_root_non_nil_project", func(t *testing.T) {
		root := &ast.ResourceScope{}
		project := &ast.ResourceScope{
			Policies:  map[string]ast.PolicyConfig{"p": {}},
			Memory:    map[string]ast.MemoryConfig{"p": {}},
			Contexts:  map[string]ast.ContextConfig{"p": {}},
			Templates: map[string]ast.TemplateConfig{"p": {}},
		}
		require.NotPanics(t, func() { mergeResourceScope(root, project) })
		assert.Contains(t, root.Policies, "p")
		assert.Contains(t, root.Memory, "p")
		assert.Contains(t, root.Contexts, "p")
		assert.Contains(t, root.Templates, "p")
	})

	// non-nil global + nil project: no panic, global entries preserved
	t.Run("non_nil_root_nil_project", func(t *testing.T) {
		root := &ast.ResourceScope{
			Policies:  map[string]ast.PolicyConfig{"g": {}},
			Memory:    map[string]ast.MemoryConfig{"g": {}},
			Contexts:  map[string]ast.ContextConfig{"g": {}},
			Templates: map[string]ast.TemplateConfig{"g": {}},
		}
		project := &ast.ResourceScope{}
		require.NotPanics(t, func() { mergeResourceScope(root, project) })
		assert.Contains(t, root.Policies, "g")
		assert.Contains(t, root.Memory, "g")
		assert.Contains(t, root.Contexts, "g")
		assert.Contains(t, root.Templates, "g")
	})
}
