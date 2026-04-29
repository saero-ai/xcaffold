package claude_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/renderer"
	"github.com/saero-ai/xcaffold/internal/renderer/claude"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompileMemory_Claude_OrchestratorPath_EmitsAgentScopedFiles(t *testing.T) {
	cfg := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Memory: map[string]ast.MemoryConfig{
				"auth-specialist/project_audit_log_owner": {
					Name:        "project_audit_log_owner",
					Description: "Who owns the audit log.",
					AgentRef:    "auth-specialist",
					Content:     "Security team owns audit log.",
				},
			},
		},
	}
	r := claude.New()
	files, notes, err := r.CompileMemory(cfg, t.TempDir(), renderer.MemoryOptions{})
	require.NoError(t, err)
	indexPath := filepath.Join("agent-memory", "auth-specialist", "MEMORY.md")
	entryPath := filepath.Join("agent-memory", "auth-specialist", "project_audit_log_owner.md")
	require.Contains(t, files, indexPath)
	require.Contains(t, files, entryPath)
	assert.Contains(t, files[indexPath], "[project_audit_log_owner](project_audit_log_owner.md)")
	assert.Contains(t, files[entryPath], "Security team owns audit log.")
	require.Empty(t, notes)
}

func TestCompileMemory_Claude_OrchestratorPath_NoMemoryReturnsEmpty(t *testing.T) {
	cfg := &ast.XcaffoldConfig{}
	r := claude.New()
	files, notes, err := r.CompileMemory(cfg, t.TempDir(), renderer.MemoryOptions{})
	require.NoError(t, err)
	require.Empty(t, files)
	require.Empty(t, notes)
}

func TestCompileMemory_Claude_OrchestratorPath_MissingAgentRefFallsBackToDefault(t *testing.T) {
	cfg := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Memory: map[string]ast.MemoryConfig{
				"orphaned": {Name: "orphaned", Content: "no owner"},
			},
		},
	}
	r := claude.New()
	files, _, err := r.CompileMemory(cfg, t.TempDir(), renderer.MemoryOptions{})
	require.NoError(t, err)
	require.Contains(t, files, filepath.Join("agent-memory", "default", "MEMORY.md"))
}

func TestCompileMemoryToMap_GroupsByAgentRef(t *testing.T) {
	cfg := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Memory: map[string]ast.MemoryConfig{
				"reviewer/user-prefs": {
					Name:        "user-prefs",
					Description: "Conciseness preference.",
					AgentRef:    "reviewer",
					Content:     "Always be concise.",
				},
				"reviewer/project-ctx": {
					Name:        "project-ctx",
					Description: "Project context.",
					AgentRef:    "reviewer",
					Content:     "This is the xcaffold project.",
				},
				"developer/feedback-style": {
					Name:        "feedback-style",
					Description: "YAML key style preference.",
					AgentRef:    "developer",
					Content:     "Use kebab-case for YAML keys.",
				},
			},
		},
	}
	r := claude.New()
	files, notes, err := r.CompileMemory(cfg, t.TempDir(), renderer.MemoryOptions{})
	require.NoError(t, err)
	require.Empty(t, notes)

	// Two MEMORY.md index files + three individual entry files = 5.
	require.Len(t, files, 5)

	reviewerIndex, ok := files["agent-memory/reviewer/MEMORY.md"]
	require.True(t, ok, "expected agent-memory/reviewer/MEMORY.md")
	assert.Contains(t, reviewerIndex, "[project-ctx](project-ctx.md)")
	assert.Contains(t, reviewerIndex, "[user-prefs](user-prefs.md)")

	require.Contains(t, files, "agent-memory/reviewer/project-ctx.md")
	assert.Contains(t, files["agent-memory/reviewer/project-ctx.md"], "This is the xcaffold project.")
	require.Contains(t, files, "agent-memory/reviewer/user-prefs.md")
	assert.Contains(t, files["agent-memory/reviewer/user-prefs.md"], "Always be concise.")

	devIndex, ok := files["agent-memory/developer/MEMORY.md"]
	require.True(t, ok, "expected agent-memory/developer/MEMORY.md")
	assert.Contains(t, devIndex, "[feedback-style](feedback-style.md)")

	require.Contains(t, files, "agent-memory/developer/feedback-style.md")
	assert.Contains(t, files["agent-memory/developer/feedback-style.md"], "Use kebab-case for YAML keys.")
}

func TestCompileMemoryToMap_EmptyBodySkipped(t *testing.T) {
	cfg := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Memory: map[string]ast.MemoryConfig{
				"dev/empty-entry": {Name: "empty-entry", AgentRef: "dev", Content: "   "},
			},
		},
	}
	r := claude.New()
	files, _, err := r.CompileMemory(cfg, t.TempDir(), renderer.MemoryOptions{})
	require.NoError(t, err)
	require.Empty(t, files)
}

func TestCompileMemoryToMap_DefaultAgentRef(t *testing.T) {
	cfg := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Memory: map[string]ast.MemoryConfig{
				"orphan": {Name: "orphan", Content: "No agent ref set."},
			},
		},
	}
	r := claude.New()
	files, _, err := r.CompileMemory(cfg, t.TempDir(), renderer.MemoryOptions{})
	require.NoError(t, err)
	_, ok := files["agent-memory/default/MEMORY.md"]
	require.True(t, ok, "expected agent-memory/default/MEMORY.md for zero AgentRef")
	// key has no "/" so fname is "orphan.md"
	_, ok = files["agent-memory/default/orphan.md"]
	require.True(t, ok, "expected agent-memory/default/orphan.md for key without /")
}

func TestCompileAgentMarkdown_MemoryUserInjected(t *testing.T) {
	cfg := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"reviewer": {
					Name:  "reviewer",
					Model: "sonnet",
				},
			},
			Memory: map[string]ast.MemoryConfig{
				"user-prefs": {AgentRef: "reviewer", Content: "Be concise."},
			},
		},
	}
	r := claude.New()
	files, _, err := renderer.Orchestrate(r, cfg, t.TempDir())
	require.NoError(t, err)
	content, ok := files.Files["agents/reviewer.md"]
	require.True(t, ok)
	assert.Contains(t, content, "memory: user")
}

func TestCompileAgentMarkdown_MemoryUserNotInjectedWhenNoMemory(t *testing.T) {
	cfg := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"reviewer": {Name: "reviewer", Model: "sonnet"},
			},
		},
	}
	r := claude.New()
	files, _, err := renderer.Orchestrate(r, cfg, t.TempDir())
	require.NoError(t, err)
	content := files.Files["agents/reviewer.md"]
	assert.NotContains(t, content, "memory:")
}

func TestCompileAgentMarkdown_ExplicitMemoryFieldNotOverwritten(t *testing.T) {
	cfg := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"dev": {Name: "dev", Memory: ast.FlexStringSlice{"project"}},
			},
			Memory: map[string]ast.MemoryConfig{
				"ctx": {AgentRef: "dev", Content: "Context."},
			},
		},
	}
	r := claude.New()
	files, _, err := renderer.Orchestrate(r, cfg, t.TempDir())
	require.NoError(t, err)
	content := files.Files["agents/dev.md"]
	assert.Equal(t, 1, strings.Count(content, "memory:"))
	assert.Contains(t, content, "memory: project")
}
