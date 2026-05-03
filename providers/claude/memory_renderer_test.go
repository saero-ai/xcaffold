package claude

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// defaultMEMORY returns the expected path for the default agent's MEMORY.md.
func defaultMEMORY(dir string) string {
	return filepath.Join(dir, "default", "MEMORY.md")
}

// agentMEMORY returns the expected path for a named agent's MEMORY.md.
func agentMEMORY(dir, agentRef string) string {
	return filepath.Join(dir, agentRef, "MEMORY.md")
}

// TestMemoryRenderer_ConcatenatesIntoMEMORY verifies the link-list index
// format and individual .md file generation.
func TestMemoryRenderer_ConcatenatesIntoMEMORY(t *testing.T) {
	dir := t.TempDir()
	r := NewMemoryRenderer(dir)

	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Memory: map[string]ast.MemoryConfig{
				"backend-dev/user-role": {
					Name:        "user-role",
					Description: "User role info",
					Content:     "Robert is the founder.",
					AgentRef:    "backend-dev",
				},
				"backend-dev/arch-decisions": {
					Name:        "arch-decisions",
					Description: "Architecture decisions",
					Content:     "One-way compiler model.",
					AgentRef:    "backend-dev",
				},
			},
		},
	}

	output, notes, err := r.Compile(config, dir)
	require.NoError(t, err)
	require.NotNil(t, output)
	require.Empty(t, notes, "no fidelity notes for new files")

	memPath := agentMEMORY(dir, "backend-dev")
	require.FileExists(t, memPath)

	data, err := os.ReadFile(memPath)
	require.NoError(t, err)
	content := string(data)

	// Link-list format, not ## headings.
	assert.Contains(t, content, "- [arch-decisions](arch-decisions.md)")
	assert.Contains(t, content, "- [user-role](user-role.md)")
	assert.NotContains(t, content, "## ", "should use link-list format, not ## headings")

	// Individual files must exist.
	require.FileExists(t, filepath.Join(dir, "backend-dev", "user-role.md"))
	require.FileExists(t, filepath.Join(dir, "backend-dev", "arch-decisions.md"))
}

// TestMemoryRenderer_MultiAgent writes to separate per-agent directories.
func TestMemoryRenderer_MultiAgent(t *testing.T) {
	dir := t.TempDir()
	r := NewMemoryRenderer(dir)

	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Memory: map[string]ast.MemoryConfig{
				"backend-dev/backend-pref": {
					Name:        "backend-pref",
					Description: "Backend preferences",
					Content:     "Backend pref body.",
					AgentRef:    "backend-dev",
				},
				"frontend-dev/frontend-pref": {
					Name:        "frontend-pref",
					Description: "Frontend preferences",
					Content:     "Frontend pref body.",
					AgentRef:    "frontend-dev",
				},
			},
		},
	}

	_, _, err := r.Compile(config, dir)
	require.NoError(t, err)

	require.FileExists(t, agentMEMORY(dir, "backend-dev"))
	require.FileExists(t, agentMEMORY(dir, "frontend-dev"))

	backendData, _ := os.ReadFile(agentMEMORY(dir, "backend-dev"))
	assert.Contains(t, string(backendData), "[backend-pref]")
	assert.NotContains(t, string(backendData), "[frontend-pref]")

	frontendData, _ := os.ReadFile(agentMEMORY(dir, "frontend-dev"))
	assert.Contains(t, string(frontendData), "[frontend-pref]")
	assert.NotContains(t, string(frontendData), "[backend-pref]")
}

// TestMemoryRenderer_DefaultAgentRef uses "default" when AgentRef is empty.
func TestMemoryRenderer_DefaultAgentRef(t *testing.T) {
	dir := t.TempDir()
	r := NewMemoryRenderer(dir)

	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Memory: map[string]ast.MemoryConfig{
				"user-role": {
					Name:        "user-role",
					Description: "User role",
					Content:     "Robert is the founder.",
					// AgentRef intentionally left empty.
				},
			},
		},
	}

	_, _, err := r.Compile(config, dir)
	require.NoError(t, err)
	require.FileExists(t, defaultMEMORY(dir))
}

// TestCompileMemory_EmptyBody_Skipped verifies that entries with empty bodies
// do not produce files or errors.
func TestCompileMemory_EmptyBody_Skipped(t *testing.T) {
	dir := t.TempDir()
	r := NewMemoryRenderer(dir)
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Memory: map[string]ast.MemoryConfig{
				"empty": {
					Name: "empty",
					// No content.
				},
			},
		},
	}

	_, _, err := r.Compile(config, dir)
	require.NoError(t, err)

	// No agent dir should be created for all-empty entries.
	_, statErr := os.Stat(filepath.Join(dir, "default"))
	require.True(t, os.IsNotExist(statErr), "no agent dir for empty-body-only agent")
}

// TestCompileMemory_AgentRefTraversal_Rejected verifies that an AgentRef
// containing traversal sequences is rejected.
func TestCompileMemory_AgentRefTraversal_Rejected(t *testing.T) {
	dir := t.TempDir()
	r := NewMemoryRenderer(dir)
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Memory: map[string]ast.MemoryConfig{
				"evil": {
					Name:     "evil",
					Content:  "bad content",
					AgentRef: "../escaped",
				},
			},
		},
	}

	_, _, err := r.Compile(config, dir)
	require.Error(t, err)
	require.Contains(t, err.Error(), "agent-ref")
}

// TestCompileMemory_DeterministicOrder verifies that entries in a single
// MEMORY.md are sorted by name regardless of map iteration order.
func TestCompileMemory_DeterministicOrder(t *testing.T) {
	dir := t.TempDir()
	r := NewMemoryRenderer(dir)
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Memory: map[string]ast.MemoryConfig{
				"zzz-last": {
					Name:        "zzz-last",
					Description: "Z desc",
					Content:     "Z body.",
				},
				"aaa-first": {
					Name:        "aaa-first",
					Description: "A desc",
					Content:     "A body.",
				},
			},
		},
	}

	_, _, err := r.Compile(config, dir)
	require.NoError(t, err)

	data, _ := os.ReadFile(defaultMEMORY(dir))
	content := string(data)

	posFirst := strings.Index(content, "aaa-first")
	posLast := strings.Index(content, "zzz-last")
	assert.Greater(t, posLast, posFirst, "aaa-first must appear before zzz-last")
}

// TestCompileMemory_Seeds_Recorded verifies that Seeds() returns a MemorySeed
// for each written agent MEMORY.md after Compile.
func TestCompileMemory_Seeds_Recorded(t *testing.T) {
	dir := t.TempDir()

	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Memory: map[string]ast.MemoryConfig{
				"arch": {Name: "arch", Description: "Architecture", Content: "Original xcf content."},
			},
		},
	}

	r := NewMemoryRenderer(dir)
	_, _, err := r.Compile(config, dir)
	require.NoError(t, err)

	seeds := r.Seeds()
	require.Len(t, seeds, 1, "one seed per agent must be recorded")
	assert.Equal(t, "default", seeds[0].Name)
	assert.Equal(t, "claude", seeds[0].Target)
	assert.Contains(t, seeds[0].Hash, "sha256:")
}

// TestCompileMemory_GeneratesLinkListIndex verifies the link-list index format.
func TestCompileMemory_GeneratesLinkListIndex(t *testing.T) {
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Memory: map[string]ast.MemoryConfig{
				"dev/orm-decision": {
					Name: "ORM Decision", Description: "Always use Drizzle",
					Content: "We chose Drizzle ORM.", AgentRef: "dev",
				},
				"dev/api-patterns": {
					Name: "API Patterns", Description: "REST conventions",
					Content: "All endpoints use JSON:API.", AgentRef: "dev",
				},
			},
		},
	}
	dir := t.TempDir()
	mr := NewMemoryRenderer(dir)
	_, _, err := mr.Compile(config, "")
	require.NoError(t, err)

	memPath := filepath.Join(dir, "dev", "MEMORY.md")
	data, err := os.ReadFile(memPath)
	require.NoError(t, err)
	content := string(data)
	assert.Contains(t, content, "- [API Patterns](api-patterns.md) — REST conventions")
	assert.Contains(t, content, "- [ORM Decision](orm-decision.md) — Always use Drizzle")
	assert.NotContains(t, content, "## ", "should use link-list format, not ## headings")
}

// TestCompileMemory_CopiesIndividualFiles verifies individual .md files are written.
func TestCompileMemory_CopiesIndividualFiles(t *testing.T) {
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Memory: map[string]ast.MemoryConfig{
				"dev/orm-decision": {
					Name: "ORM Decision", Description: "Always use Drizzle",
					Content: "We chose Drizzle ORM.", AgentRef: "dev",
				},
			},
		},
	}
	dir := t.TempDir()
	mr := NewMemoryRenderer(dir)
	_, _, err := mr.Compile(config, "")
	require.NoError(t, err)

	indivPath := filepath.Join(dir, "dev", "orm-decision.md")
	data, err := os.ReadFile(indivPath)
	require.NoError(t, err)
	assert.Contains(t, string(data), "We chose Drizzle ORM.")
}

// TestCompileMemory_AlwaysOverwrites verifies that apply always overwrites
// existing MEMORY.md (no seed-once behavior).
func TestCompileMemory_AlwaysOverwrites(t *testing.T) {
	dir := t.TempDir()
	agentDir := filepath.Join(dir, "dev")
	require.NoError(t, os.MkdirAll(agentDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(agentDir, "MEMORY.md"), []byte("old content"), 0o644))

	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Memory: map[string]ast.MemoryConfig{
				"dev/new-entry": {
					Name: "New", Description: "New entry",
					Content: "New content.", AgentRef: "dev",
				},
			},
		},
	}
	mr := NewMemoryRenderer(dir)
	_, _, err := mr.Compile(config, "")
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(dir, "dev", "MEMORY.md"))
	require.NoError(t, err)
	assert.Contains(t, string(data), "New")
	assert.NotContains(t, string(data), "old content")
}

// TestCompileMemory_SortedDeterministic verifies alphabetical sorting of entries.
func TestCompileMemory_SortedDeterministic(t *testing.T) {
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Memory: map[string]ast.MemoryConfig{
				"dev/zebra": {Name: "Zebra", Description: "Z desc", Content: "Z content", AgentRef: "dev"},
				"dev/alpha": {Name: "Alpha", Description: "A desc", Content: "A content", AgentRef: "dev"},
			},
		},
	}
	dir := t.TempDir()
	mr := NewMemoryRenderer(dir)
	_, _, err := mr.Compile(config, "")
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(dir, "dev", "MEMORY.md"))
	require.NoError(t, err)
	content := string(data)
	alphaIdx := strings.Index(content, "Alpha")
	zebraIdx := strings.Index(content, "Zebra")
	assert.Less(t, alphaIdx, zebraIdx, "entries should be sorted alphabetically")
}
