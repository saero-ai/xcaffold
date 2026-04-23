package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImport_HookScriptPassthrough(t *testing.T) {
	orig, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(orig) }()

	dir := t.TempDir()
	require.NoError(t, os.Chdir(dir))

	claudeHooks := filepath.Join(dir, ".claude", "hooks")
	require.NoError(t, os.MkdirAll(claudeHooks, 0755))
	testHook := filepath.Join(claudeHooks, "pre-commit.sh")
	require.NoError(t, os.WriteFile(testHook, []byte("echo hook"), 0o644))

	// Mock other provider extra
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".claude", "custom"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".claude", "custom", "test.json"), []byte("{}"), 0o644))

	err = importScope(".claude", "project.xcf", "project", "claude")
	require.NoError(t, err)

	hookOutput := filepath.Join(dir, "xcf", "hooks", "pre-commit.sh")
	require.FileExists(t, hookOutput)
	info, err := os.Stat(hookOutput)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0755), info.Mode()&os.ModePerm, "hook script must be executable")

	extraOutput := filepath.Join(dir, "xcf", "provider", "claude", "custom", "test.json")
	require.FileExists(t, extraOutput)
	extraInfo, err := os.Stat(extraOutput)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0644), extraInfo.Mode()&os.ModePerm, "provider extra must be 0644")
}
