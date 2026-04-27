package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/saero-ai/xcaffold/internal/ast"
)

// TestImportScope_PrunesOrphanMemory verifies that memory successfully imported
// by the provider importer is preserved by pruneOrphanMemory — even when the
// agent has no local agent definition. The importer issues a warning for
// such entries but explicitly keeps them (they may belong to a global agent
// defined in ~/.claude/agents/). Memory that was never imported (i.e. written
// directly to xcf/ by a prior run but absent from the current import) is the
// only category that pruneOrphanMemory removes.
func TestImportScope_PrunesOrphanMemory(t *testing.T) {
	orig, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(orig) }()

	dir := t.TempDir()
	require.NoError(t, os.Chdir(dir))

	// Create valid agent and its memory.
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".claude", "agents"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".claude", "agents", "dev.md"), []byte("# Dev\n"), 0o644))

	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".claude", "agent-memory"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".claude", "agent-memory", "dev.md"), []byte("dev mem"), 0o644))

	// Create memory for agents with no local definition (e.g. global agents).
	// The importer imports these with a warning and adds them to config.Memory,
	// so pruneOrphanMemory must preserve them.
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".claude", "agent-memory", "global.md"), []byte("global mem"), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".claude", "agent-memory", "sub"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".claude", "agent-memory", "sub", "task.md"), []byte("task mem"), 0o644))

	err = importScope(".claude", "project.xcf", "project", "claude")
	require.NoError(t, err)

	// Memory for the declared agent must be present.
	devMd := filepath.Join(dir, "xcf", "agents", "dev", "memory", "dev.md")
	require.FileExists(t, devMd)

	// Memory imported for agents without a local definition is preserved because
	// the importer added them to config.Memory (they may be global agents).
	globalMd := filepath.Join(dir, "xcf", "agents", "global", "memory", "global.md")
	assert.FileExists(t, globalMd, "imported memory for agent without local definition should be preserved")

	subMd := filepath.Join(dir, "xcf", "agents", "sub", "memory", "task.md")
	assert.FileExists(t, subMd, "imported nested memory for agent without local definition should be preserved")
}

// TestPruneOrphanMemory_PreservesImportedMemory verifies that memory imported
// for a global agent (not in config.Agents) is NOT pruned when config.Memory
// contains an entry for that agent. This covers the case where a global agent
// like ~/.claude/agents/ceo.md writes project-scoped memory to
// .claude/agent-memory/ceo/ and the importer correctly preserves it.
func TestPruneOrphanMemory_PreservesImportedMemory(t *testing.T) {
	dir := t.TempDir()

	// Create memory file for a global agent "ceo" that has no agent definition.
	memDir := filepath.Join(dir, "xcf", "agents", "ceo", "memory")
	require.NoError(t, os.MkdirAll(memDir, 0755))
	memFile := filepath.Join(memDir, "context.md")
	require.NoError(t, os.WriteFile(memFile, []byte("ceo context"), 0o644))

	// Config has NO ceo entry in Agents, but DOES have it in Memory.
	config := &ast.XcaffoldConfig{}
	config.Agents = map[string]ast.AgentConfig{} // explicitly empty
	config.Memory = map[string]ast.MemoryConfig{
		"ceo/context": {Content: "ceo context", SourceProvider: "claude"},
	}

	err := pruneOrphanMemory(config, dir)
	require.NoError(t, err)

	assert.FileExists(t, memFile, "memory for imported global agent should be preserved")
}

// TestPruneOrphanMemory_PrunesNonImportedMemory verifies that memory on disk
// for an agent that has neither an Agents entry nor a Memory entry in config
// is still removed.
func TestPruneOrphanMemory_PrunesNonImportedMemory(t *testing.T) {
	dir := t.TempDir()

	// Create memory file for "ghost" agent — not in config at all.
	memDir := filepath.Join(dir, "xcf", "agents", "ghost", "memory")
	require.NoError(t, os.MkdirAll(memDir, 0755))
	memFile := filepath.Join(memDir, "context.md")
	require.NoError(t, os.WriteFile(memFile, []byte("ghost context"), 0o644))

	// Config has neither the ghost agent in Agents nor in Memory.
	config := &ast.XcaffoldConfig{}
	config.Agents = map[string]ast.AgentConfig{}
	config.Memory = map[string]ast.MemoryConfig{}

	err := pruneOrphanMemory(config, dir)
	require.NoError(t, err)

	assert.NoFileExists(t, memFile, "memory for non-imported orphan agent should be pruned")
}

// TestPruneOrphanMemory_CleansEmptyDirs verifies that after pruning, any
// xcf/agents/<id>/ directory that is now empty (no .xcf file and no memory/)
// is removed.
func TestPruneOrphanMemory_CleansEmptyDirs(t *testing.T) {
	dir := t.TempDir()

	// Create an empty orphan agent directory with no .xcf file and no memory/.
	orphanDir := filepath.Join(dir, "xcf", "agents", "orphan")
	require.NoError(t, os.MkdirAll(orphanDir, 0755))

	config := &ast.XcaffoldConfig{}
	config.Agents = map[string]ast.AgentConfig{}
	config.Memory = map[string]ast.MemoryConfig{}

	err := pruneOrphanMemory(config, dir)
	require.NoError(t, err)

	assert.NoDirExists(t, orphanDir, "empty agent directory should be cleaned up after pruning")
}
