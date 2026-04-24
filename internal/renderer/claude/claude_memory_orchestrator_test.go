package claude_test

import (
	"path/filepath"
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/renderer"
	"github.com/saero-ai/xcaffold/internal/renderer/claude"
	"github.com/stretchr/testify/require"
)

func TestCompileMemory_Claude_OrchestratorPath_EmitsAgentScopedFiles(t *testing.T) {
	cfg := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Memory: map[string]ast.MemoryConfig{
				"project_audit_log_owner": {
					Name:         "project_audit_log_owner",
					AgentRef:     "auth-specialist",
					Instructions: "Security team owns audit log.",
				},
			},
		},
	}
	r := claude.New()
	files, notes, err := r.CompileMemory(cfg, t.TempDir(), renderer.MemoryOptions{})
	require.NoError(t, err)
	want := filepath.Join("agent-memory", "auth-specialist", "project_audit_log_owner.md")
	require.Contains(t, files, want)
	require.NotEmpty(t, files[want])
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
				"orphaned": {Name: "orphaned", Instructions: "no owner"},
			},
		},
	}
	r := claude.New()
	files, _, err := r.CompileMemory(cfg, t.TempDir(), renderer.MemoryOptions{})
	require.NoError(t, err)
	require.Contains(t, files, filepath.Join("agent-memory", "default", "project_orphaned.md"))
}
