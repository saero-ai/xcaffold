package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImportScope_PrunesOrphanMemory(t *testing.T) {
	orig, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(orig) }()

	dir := t.TempDir()
	require.NoError(t, os.Chdir(dir))

	// Create valid agent and memory
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".claude", "agents"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".claude", "agents", "dev.md"), []byte("# Dev\n"), 0o644))

	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".claude", "agent-memory"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".claude", "agent-memory", "dev.md"), []byte("dev mem"), 0o644))

	// Create ORPHAN memory
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".claude", "agent-memory", "global.md"), []byte("global mem"), 0o644))
	// Create nested ORPHAN memory
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".claude", "agent-memory", "sub"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".claude", "agent-memory", "sub", "task.md"), []byte("task mem"), 0o644))

	err = importScope(".claude", "project.xcf", "project", "claude")
	require.NoError(t, err)

	devMd := filepath.Join(dir, "xcf", "agents", "dev", "memory", "dev.md")
	require.FileExists(t, devMd)

	orphanMd := filepath.Join(dir, "xcf", "agents", "global", "memory", "global.md")
	assert.NoFileExists(t, orphanMd, "orphan memory should be pruned")

	orphanSubMd := filepath.Join(dir, "xcf", "agents", "sub", "memory", "task.md")
	assert.NoFileExists(t, orphanSubMd, "nested orphan memory should be pruned")
}
