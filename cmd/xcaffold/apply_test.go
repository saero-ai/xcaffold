package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// minimalXCF is a minimal valid scaffold.xcf for apply tests.
const minimalXCF = `version: "1"
project:
  name: apply-test
`

func TestApplyScope_Project(t *testing.T) {
	dir := t.TempDir()
	xcf := filepath.Join(dir, "scaffold.xcf")
	require.NoError(t, os.WriteFile(xcf, []byte(minimalXCF), 0600))

	claudeDirPath := filepath.Join(dir, ".claude")
	lock := filepath.Join(dir, "scaffold.lock")

	err := applyScope(xcf, claudeDirPath, lock, "project")
	require.NoError(t, err)

	// scaffold.lock must be written.
	_, err = os.Stat(lock)
	assert.NoError(t, err, "scaffold.lock should exist after applyScope")

	// .claude/ subdirectories must exist.
	for _, sub := range []string{"agents", "skills", "rules"} {
		_, err = os.Stat(filepath.Join(claudeDirPath, sub))
		assert.NoError(t, err, ".claude/%s should exist", sub)
	}
}

func TestApplyScope_MissingXCF(t *testing.T) {
	dir := t.TempDir()
	claudeDirPath := filepath.Join(dir, ".claude")
	lock := filepath.Join(dir, "scaffold.lock")

	err := applyScope(filepath.Join(dir, "nonexistent.xcf"), claudeDirPath, lock, "project")
	assert.Error(t, err, "should fail when xcf file does not exist")
}

func TestRunApply_ScopeProject(t *testing.T) {
	dir := t.TempDir()
	xcf := filepath.Join(dir, "scaffold.xcf")
	require.NoError(t, os.WriteFile(xcf, []byte(minimalXCF), 0600))

	// Set package-level path vars used by runApply.
	xcfPath = xcf
	claudeDir = filepath.Join(dir, ".claude")
	lockPath = filepath.Join(dir, "scaffold.lock")
	scopeFlag = "project"

	err := runApply(nil, nil)
	require.NoError(t, err)

	_, err = os.Stat(lockPath)
	assert.NoError(t, err, "scaffold.lock should be written for project scope")
}

func TestRunApply_ScopeGlobal(t *testing.T) {
	dir := t.TempDir()
	xcf := filepath.Join(dir, "global.xcf")
	require.NoError(t, os.WriteFile(xcf, []byte(minimalXCF), 0600))

	globalXcfPath = xcf
	globalClaudeDir = filepath.Join(dir, ".claude")
	globalLockPath = filepath.Join(dir, "scaffold.lock")
	scopeFlag = "global"

	err := runApply(nil, nil)
	require.NoError(t, err)

	_, err = os.Stat(globalLockPath)
	assert.NoError(t, err, "scaffold.lock should be written for global scope")
}

func TestRunApply_ScopeAll(t *testing.T) {
	dir := t.TempDir()

	projXCF := filepath.Join(dir, "scaffold.xcf")
	require.NoError(t, os.WriteFile(projXCF, []byte(minimalXCF), 0600))

	globalXCF := filepath.Join(dir, "global.xcf")
	require.NoError(t, os.WriteFile(globalXCF, []byte(minimalXCF), 0600))

	xcfPath = projXCF
	claudeDir = filepath.Join(dir, "proj-claude")
	lockPath = filepath.Join(dir, "proj-scaffold.lock")

	globalXcfPath = globalXCF
	globalClaudeDir = filepath.Join(dir, "global-claude")
	globalLockPath = filepath.Join(dir, "global-scaffold.lock")

	scopeFlag = "all"

	err := runApply(nil, nil)
	require.NoError(t, err)

	_, err = os.Stat(lockPath)
	assert.NoError(t, err, "project scaffold.lock should be written")

	_, err = os.Stat(globalLockPath)
	assert.NoError(t, err, "global scaffold.lock should be written")
}
