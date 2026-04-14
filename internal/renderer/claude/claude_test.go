package claude

import (
	"strings"
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
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

	out, err := r.Compile(config, "")
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
					Description:  "Code review specialist.",
					Instructions: "Review code for correctness and style.\n",
					Model:        "claude-opus-4-5",
				},
			},
		},
	}

	out, err := r.Compile(config, "")
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
					Description:  "Security hardening rules.",
					Instructions: "Never expose secrets in logs.\n",
				},
			},
		},
	}

	out, err := r.Compile(config, "")
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

	out, err := r.Compile(config, "")
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
					Description:  "Go coding conventions.",
					Instructions: "Follow effective Go.\n",
					Paths:        []string{"**/*.go"},
				},
			},
		},
	}

	out, err := r.Compile(config, "")
	require.NoError(t, err)

	content := out.Files["rules/go-style.md"]
	assert.Contains(t, content, "paths: [**/*.go]")
}

func TestClaudeRenderer_Render(t *testing.T) {
	r := New()
	files := map[string]string{
		"agents/test.md": "---\n---\n",
	}

	result := r.Render(files)
	require.NotNil(t, result)
	assert.Equal(t, files, result.Files)
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

	out, err := r.Compile(config, "")
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
					Name:         "tdd-driven-development",
					Description:  "Test-driven development workflow.",
					Instructions: "Write tests before implementation.\n",
				},
			},
		},
	}

	out, err := r.Compile(config, "")
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
					Name:         "Auditor",
					Description:  "Read-only code auditor.",
					Instructions: "Audit the code.",
					Readonly:     &ro,
				},
			},
		},
	}

	out, err := r.Compile(config, "")
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
					Name:         "Custom",
					Instructions: "Custom tools.",
					Readonly:     &ro,
					Tools:        []string{"Bash", "Read"},
				},
			},
		},
	}

	out, err := r.Compile(config, "")
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
					Name:         "Writer",
					Instructions: "Write code.",
					Readonly:     &ro,
				},
			},
		},
	}

	out, err := r.Compile(config, "")
	require.NoError(t, err)

	content := out.Files["agents/writer.md"]
	assert.NotContains(t, content, "tools:", "readonly: false must not synthesize tools")
}

func TestClaudeRenderer_Compile_Agent_InvocationControl(t *testing.T) {
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

	out, err := r.Compile(config, "")
	require.NoError(t, err)

	content := out.Files["agents/commit.md"]
	require.Contains(t, content, "disableModelInvocation: true")
	require.Contains(t, content, "userInvocable: false")
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
					Memory:      "project",
					Isolation:   "worktree",
				},
			},
		},
	}

	out, err := r.Compile(config, "")
	require.NoError(t, err)

	content := out.Files["agents/researcher.md"]

	memoryIdx := strings.Index(content, "memory:")
	maxTurnsIdx := strings.Index(content, "maxTurns:")
	isolationIdx := strings.Index(content, "isolation:")

	require.NotEqual(t, -1, memoryIdx, "memory: not found in output:\n%s", content)
	require.NotEqual(t, -1, maxTurnsIdx, "maxTurns: not found")
	require.NotEqual(t, -1, isolationIdx, "isolation: not found")

	require.Greater(t, memoryIdx, maxTurnsIdx, "memory must come AFTER maxTurns (Group 6 > Group 2)")
	require.Greater(t, memoryIdx, isolationIdx, "memory must come AFTER isolation (within Group 5-6 ordering)")
}
