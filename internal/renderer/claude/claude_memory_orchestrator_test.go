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
				"project_audit_log_owner": {
					Name:     "project_audit_log_owner",
					AgentRef: "auth-specialist",
					Content:  "Security team owns audit log.",
				},
			},
		},
	}
	r := claude.New()
	files, notes, err := r.CompileMemory(cfg, t.TempDir(), renderer.MemoryOptions{})
	require.NoError(t, err)
	want := filepath.Join("agent-memory", "auth-specialist", "MEMORY.md")
	require.Contains(t, files, want)
	assert.Contains(t, files[want], "## project_audit_log_owner")
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
				"user-prefs": {
					AgentRef: "reviewer",
					Content:  "Always be concise.",
				},
				"project-ctx": {
					AgentRef: "reviewer",
					Content:  "This is the xcaffold project.",
				},
				"feedback-style": {
					AgentRef: "developer",
					Content:  "Use kebab-case for YAML keys.",
				},
			},
		},
	}
	r := claude.New()
	files, notes, err := r.CompileMemory(cfg, t.TempDir(), renderer.MemoryOptions{})
	require.NoError(t, err)
	require.Empty(t, notes)

	// Two agent dirs, not three individual files.
	require.Len(t, files, 2)

	reviewerMem, ok := files["agent-memory/reviewer/MEMORY.md"]
	require.True(t, ok, "expected agent-memory/reviewer/MEMORY.md")
	assert.Contains(t, reviewerMem, "## project-ctx")
	assert.Contains(t, reviewerMem, "## user-prefs")
	assert.Contains(t, reviewerMem, "This is the xcaffold project.")
	assert.Contains(t, reviewerMem, "Always be concise.")

	devMem, ok := files["agent-memory/developer/MEMORY.md"]
	require.True(t, ok, "expected agent-memory/developer/MEMORY.md")
	assert.Contains(t, devMem, "## feedback-style")
	assert.Contains(t, devMem, "Use kebab-case for YAML keys.")
}

func TestCompileMemoryToMap_EmptyBodySkipped(t *testing.T) {
	cfg := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Memory: map[string]ast.MemoryConfig{
				"empty-entry": {AgentRef: "dev", Content: "   "},
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
				"orphan": {Content: "No agent ref set."},
			},
		},
	}
	r := claude.New()
	files, _, err := r.CompileMemory(cfg, t.TempDir(), renderer.MemoryOptions{})
	require.NoError(t, err)
	_, ok := files["agent-memory/default/MEMORY.md"]
	require.True(t, ok, "expected agent-memory/default/MEMORY.md for zero AgentRef")
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
				"dev": {Name: "dev", Memory: "project"},
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
