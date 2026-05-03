package claude

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

func TestClaudeRenderer_Target(t *testing.T) {
	r := New()
	assert.Equal(t, "claude", r.Target())
}

func TestClaudeRenderer_OutputDir(t *testing.T) {
	r := New()
	assert.Equal(t, ".claude", r.OutputDir())
}

func TestClaudeRenderer_Compile_EmptyConfig(t *testing.T) {
	r := New()
	config := &ast.XcaffoldConfig{}

	out, _, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Empty(t, out.Files)
}

func TestClaudeRenderer_Compile_MinimalAgent(t *testing.T) {
	r := New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"reviewer": {
					Description: "Code review specialist.",
					Body:        "Review code for correctness and style.\n",
					Model:       "claude-opus-4-5",
				},
			},
		},
	}

	out, _, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)
	require.NotNil(t, out)

	content, ok := out.Files["agents/reviewer.md"]
	require.True(t, ok, "expected agents/reviewer.md in output")

	assert.Contains(t, content, "description: Code review specialist.")
	assert.Contains(t, content, "model: claude-opus-4-5")
	assert.Contains(t, content, "Review code for correctness and style.")

	// Verify frontmatter delimiters are present.
	assert.Contains(t, content, "---\n")
}

func TestClaudeRenderer_Compile_MinimalRule(t *testing.T) {
	r := New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"security": {
					Description: "Security hardening rules.",
					Body:        "Never expose secrets in logs.\n",
				},
			},
		},
	}

	out, _, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)
	require.NotNil(t, out)

	content, ok := out.Files["rules/security.md"]
	require.True(t, ok, "expected rules/security.md in output")

	assert.Contains(t, content, "description: Security hardening rules.")
	assert.Contains(t, content, "Never expose secrets in logs.")
}

func TestClaudeRenderer_Compile_AgentFrontmatterFields(t *testing.T) {
	r := New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"developer": {
					Description: "An expert developer.",
					Model:       "claude-sonnet-4-5",
					Effort:      "high",
					Tools:       []string{"Bash", "Read", "Write"},
					Skills:      []string{"tdd", "code-review"},
				},
			},
		},
	}

	out, _, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)

	content := out.Files["agents/developer.md"]
	assert.Contains(t, content, "model: claude-sonnet-4-5")
	assert.Contains(t, content, "effort: high")
	assert.Contains(t, content, "tools: [Bash, Read, Write]")
	assert.Contains(t, content, "skills: [tdd, code-review]")
}

func TestClaudeRenderer_Compile_RuleWithPaths(t *testing.T) {
	r := New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"go-style": {
					Description: "Go coding conventions.",
					Body:        "Follow effective Go.\n",
					Paths:       []string{"**/*.go"},
				},
			},
		},
	}

	out, _, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)

	content := out.Files["rules/go-style.md"]
	assert.Contains(t, content, "paths: [**/*.go]")
}

func TestClaudeRenderer_Compile_MultipleResources(t *testing.T) {
	r := New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"frontend": {Description: "Frontend specialist."},
				"backend":  {Description: "Backend specialist."},
			},
			Rules: map[string]ast.RuleConfig{
				"security": {Description: "Security rules."},
			},
		},
	}

	out, _, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)

	assert.Contains(t, out.Files, "agents/frontend.md")
	assert.Contains(t, out.Files, "agents/backend.md")
	assert.Contains(t, out.Files, "rules/security.md")
	assert.Len(t, out.Files, 3)
}

func TestClaudeRenderer_Compile_SkillMinimal(t *testing.T) {
	r := New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Skills: map[string]ast.SkillConfig{
				"tdd": {
					Name:        "tdd-driven-development",
					Description: "Test-driven development workflow.",
					Body:        "Write tests before implementation.\n",
				},
			},
		},
	}

	out, _, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)

	content, ok := out.Files["skills/tdd/SKILL.md"]
	require.True(t, ok, "expected skills/tdd/SKILL.md in output")

	assert.Contains(t, content, "name: tdd-driven-development")
	assert.Contains(t, content, "description: Test-driven development workflow.")
	assert.Contains(t, content, "Write tests before implementation.")
}

// ─── Readonly tests (Issue #5 — Normalization Rule 7) ────────────────────────

func TestClaudeRenderer_Compile_Agent_Readonly_EmitsToolsReadGrepGlob(t *testing.T) {
	r := New()
	ro := true
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"auditor": {
					Name:        "Auditor",
					Description: "Read-only code auditor.",
					Body:        "Audit the code.",
					Readonly:    &ro,
				},
			},
		},
	}

	out, _, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)

	content := out.Files["agents/auditor.md"]
	require.NotEmpty(t, content)

	assert.Contains(t, content, "tools: [Read, Grep, Glob]", "readonly: true must synthesize to tools: [Read, Grep, Glob] on CC")
	assert.NotContains(t, content, "readonly:", "readonly: key must not appear in CC output")
}

func TestClaudeRenderer_Compile_Agent_Readonly_ExplicitToolsTakePrecedence(t *testing.T) {
	r := New()
	ro := true
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"custom": {
					Name:     "Custom",
					Body:     "Custom tools.",
					Readonly: &ro,
					Tools:    []string{"Bash", "Read"},
				},
			},
		},
	}

	out, _, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)

	content := out.Files["agents/custom.md"]
	// Explicit tools must win over readonly synthesis
	assert.Contains(t, content, "tools: [Bash, Read]")
	assert.NotContains(t, content, "tools: [Read, Grep, Glob]", "explicit tools must take precedence over readonly synthesis")
}

func TestClaudeRenderer_Compile_Agent_ReadonlyFalse_NoToolsSynthesized(t *testing.T) {
	r := New()
	ro := false
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"writer": {
					Name:     "Writer",
					Body:     "Write code.",
					Readonly: &ro,
				},
			},
		},
	}

	out, _, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)

	content := out.Files["agents/writer.md"]
	assert.NotContains(t, content, "tools:", "readonly: false must not synthesize tools")
}

func TestClaudeRenderer_Compile_Agent_InvocationControl(t *testing.T) {
	// disable-model-invocation and user-invocable are Copilot-only agent fields.
	// The Claude renderer must silently drop them — they must not appear in the
	// emitted agent frontmatter. (These fields ARE valid for Claude skills.)
	truthy := true
	falsy := false
	r := New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"commit": {
					Description:            "Commit workflow agent.",
					DisableModelInvocation: &truthy,
					UserInvocable:          &falsy,
				},
			},
		},
	}

	out, _, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)

	content := out.Files["agents/commit.md"]
	require.NotContains(t, content, "disable-model-invocation", "disable-model-invocation is Copilot-only and must be dropped for Claude agents")
	require.NotContains(t, content, "user-invocable", "user-invocable is Copilot-only and must be dropped for Claude agents")
	// The description must still be emitted.
	require.Contains(t, content, "description: Commit workflow agent.")
}

func TestClaudeRenderer_Compile_Agent_MemoryInGroup6(t *testing.T) {
	r := New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"researcher": {
					Description: "Research agent.",
					Model:       "sonnet",
					Effort:      "high",
					MaxTurns:    10,
					Memory:      ast.FlexStringSlice{"project"},
					Isolation:   "worktree",
				},
			},
		},
	}

	out, _, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)

	content := out.Files["agents/researcher.md"]

	memoryIdx := strings.Index(content, "memory:")
	maxTurnsIdx := strings.Index(content, "max-turns:")
	isolationIdx := strings.Index(content, "isolation:")

	require.NotEqual(t, -1, memoryIdx, "memory: not found in output:\n%s", content)
	require.NotEqual(t, -1, maxTurnsIdx, "max-turns: not found")
	require.NotEqual(t, -1, isolationIdx, "isolation: not found")

	require.Greater(t, memoryIdx, maxTurnsIdx, "memory must come AFTER max-turns (Group 6 > Group 2)")
	require.Greater(t, memoryIdx, isolationIdx, "memory must come AFTER isolation (within Group 5-6 ordering)")
}

func TestClaudeRenderer_Compile_Skill_NewFrontmatterFields(t *testing.T) {
	truthy := true
	falsy := false
	r := New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Skills: map[string]ast.SkillConfig{
				"deploy": {
					Name:                   "deploy",
					Description:            "Deploy the app",
					WhenToUse:              "When asked to ship",
					License:                "MIT",
					AllowedTools:           []string{"Bash(git *)", "Read"},
					DisableModelInvocation: &truthy,
					UserInvocable:          &falsy,
					ArgumentHint:           "[env]",
					Body:                   "Run the deploy script.",
				},
			},
		},
	}

	out, _, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)

	md, ok := out.Files["skills/deploy/SKILL.md"]
	require.True(t, ok, "expected skills/deploy/SKILL.md in output")

	require.Contains(t, md, "name: deploy")
	require.Contains(t, md, "description: Deploy the app")
	require.Contains(t, md, "when_to_use: When asked to ship")
	require.Contains(t, md, "license: MIT")
	require.Contains(t, md, "allowed-tools: Bash(git *) Read")
	require.Contains(t, md, "disable-model-invocation: true")
	require.Contains(t, md, "user-invocable: false")
	require.Contains(t, md, "argument-hint: '[env]'")
}

func TestClaudeRenderer_Compile_Skill_ClaudeProviderPassthrough(t *testing.T) {
	r := New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Skills: map[string]ast.SkillConfig{
				"deep-research": {
					Name:        "deep-research",
					Description: "Research a topic deeply",
					Targets: map[string]ast.TargetOverride{
						"claude": {
							Provider: map[string]any{
								"context": "fork",
								"agent":   "Explore",
								"model":   "sonnet",
								"effort":  "high",
								"shell":   "bash",
								"paths":   []any{"docs/**"},
							},
						},
					},
					Body: "Research deeply.",
				},
			},
		},
	}

	out, _, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)

	md, ok := out.Files["skills/deep-research/SKILL.md"]
	require.True(t, ok)

	require.Contains(t, md, "context: fork")
	require.Contains(t, md, "agent: Explore")
	require.Contains(t, md, "model: sonnet")
	require.Contains(t, md, "effort: high")
	require.Contains(t, md, "shell: bash")
	require.Contains(t, md, "paths:")
}

func TestClaudeRenderer_Compile_Skill_ProviderIsolation(t *testing.T) {
	// Cursor provider keys must NOT leak into Claude SKILL.md.
	r := New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Skills: map[string]ast.SkillConfig{
				"x": {
					Name:        "x",
					Description: "x",
					Targets: map[string]ast.TargetOverride{
						"cursor": {
							Provider: map[string]any{
								"compatibility": "cursor >= 2.4",
							},
						},
					},
					Body: "body",
				},
			},
		},
	}

	out, _, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)
	md := out.Files["skills/x/SKILL.md"]
	require.NotContains(t, md, "compatibility")
	require.NotContains(t, md, "cursor >= 2.4")
}

func TestClaudeRenderer_Compile_Skill_ProviderInjectionSafety(t *testing.T) {
	r := New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Skills: map[string]ast.SkillConfig{
				"inject": {
					Name:        "inject",
					Description: "injection test",
					Targets: map[string]ast.TargetOverride{
						"claude": {
							Provider: map[string]any{
								"context": "fork\n---\nmalicious: true",
							},
						},
					},
					Body: "body",
				},
			},
		},
	}
	out, _, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)
	md := out.Files["skills/inject/SKILL.md"]
	require.NotEmpty(t, md)
	// The value must be escaped such that the literal "\n---\n" inside the
	// value cannot terminate the frontmatter block. The frontmatter must end
	// only with its legitimate closing "---" line.
	require.NotContains(t, md, "\nmalicious: true\n")
}

// ─── Rule activation + fidelity note tests ───────────────────────────────────

func TestCompileRuleMarkdown_Activation_PathGlob(t *testing.T) {
	r := New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"api-style": {
					Description: "API style guide.",
					Activation:  ast.RuleActivationPathGlob,
					Paths:       []string{"src/**"},
					Body:        "Use REST conventions.",
				},
			},
		},
	}
	out, _, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)

	content := out.Files["rules/api-style.md"]
	require.Contains(t, content, "paths:")
	require.Contains(t, content, "src/**")
}

func TestCompileRuleMarkdown_Activation_Always(t *testing.T) {
	r := New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"security": {
					Description: "Security checklist.",
					Activation:  ast.RuleActivationAlways,
					Body:        "Follow OWASP.",
				},
			},
		},
	}
	out, _, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)

	content := out.Files["rules/security.md"]
	require.NotContains(t, content, "paths:")
}

func TestCompileRuleMarkdown_Activation_ManualMention_FidelityNote(t *testing.T) {
	r := New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"commit-style": {
					Description: "Commit formatting.",
					Activation:  ast.RuleActivationManualMention,
					Body:        "Use Conventional Commits.",
				},
			},
		},
	}
	out, notes, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)
	require.NotEmpty(t, out.Files["rules/commit-style.md"])

	// Must produce a FidelityNote warning for unsupported activation.
	require.NotEmpty(t, notes)
	require.Equal(t, renderer.LevelWarning, notes[0].Level)
	require.Contains(t, notes[0].Reason, "manual-mention")
}

func TestCompile_SkillWithExamples_Claude(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmpDir, "examples"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "examples", "sample.md"), []byte("# Sample Output"), 0o644); err != nil {
		t.Fatal(err)
	}

	skills := map[string]ast.SkillConfig{
		"my-skill": {
			Description: "test skill",
			Body:        "Do the thing.",
			Examples:    []string{"examples/sample.md"},
		},
	}

	r := New()
	files, _, err := r.CompileSkills(skills, tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	// Claude: examples flattened alongside SKILL.md (no subdirectory)
	if _, ok := files["skills/my-skill/sample.md"]; !ok {
		keys := make([]string, 0, len(files))
		for k := range files {
			keys = append(keys, k)
		}
		t.Errorf("expected examples flattened to skills/my-skill/sample.md, got: %v", keys)
	}
}

func TestCompileRuleMarkdown_ExcludeAgents_FidelityNote(t *testing.T) {
	r := New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"pr-review": {
					Description:   "PR review standards.",
					Activation:    ast.RuleActivationAlways,
					ExcludeAgents: []string{"code-review"},
					Body:          "Review carefully.",
				},
			},
		},
	}
	out, notes, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)

	content := out.Files["rules/pr-review.md"]
	require.NotContains(t, content, "excludeAgent")
	require.NotContains(t, content, "exclude-agents")

	// Must produce an info-level FidelityNote for the dropped field.
	require.NotEmpty(t, notes)
	require.Equal(t, renderer.LevelInfo, notes[0].Level)
	require.Contains(t, notes[0].Reason, "exclude-agents")
}

func TestCompileAgents_RulesNotInFrontmatter(t *testing.T) {
	agents := map[string]ast.AgentConfig{
		"test": {
			Name:        "test",
			Description: "Test agent",
			Rules:       []string{"security", "coding-standards"},
			Body:        "You are a test agent.",
		},
	}
	r := New()
	files, _, err := r.CompileAgents(agents, ".")
	require.NoError(t, err)

	content, ok := files["agents/test.md"]
	require.True(t, ok, "expected agents/test.md in output")

	assert.NotContains(t, content, "rules:")
	assert.NotContains(t, content, "security")
	assert.NotContains(t, content, "coding-standards")

	assert.Contains(t, content, "name: test")
	assert.Contains(t, content, "You are a test agent.")
}
