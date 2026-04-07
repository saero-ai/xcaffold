package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// walkUpFindXCF mirrors the walk-up logic in resolveProjectConfig so it can be
// tested with an injected home boundary.  The production code calls
// os.UserHomeDir() directly; this helper accepts home as a parameter so tests
// can control the boundary without touching the real $HOME.
func walkUpFindXCF(start, home string) string {
	curr := start
	for {
		candidate := filepath.Join(curr, "scaffold.xcf")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		if curr == home {
			return filepath.Join(start, "scaffold.xcf") // fallback
		}
		parent := filepath.Dir(curr)
		if parent == curr {
			return filepath.Join(start, "scaffold.xcf")
		}
		curr = parent
	}
}

// TestWalkUp_FindsFileWithinHome verifies the walk-up locates scaffold.xcf
// when it sits inside the home boundary.
func TestWalkUp_FindsFileWithinHome(t *testing.T) {
	// Layout:
	//   tmp/home/project/sub/   ← cwd
	//   tmp/home/project/scaffold.xcf  ← should be found
	root := t.TempDir()
	home := filepath.Join(root, "home")
	project := filepath.Join(home, "project")
	sub := filepath.Join(project, "sub")
	require.NoError(t, os.MkdirAll(sub, 0755))

	xcf := filepath.Join(project, "scaffold.xcf")
	require.NoError(t, os.WriteFile(xcf, []byte(`version: "1"`+"\n"), 0600))

	got := walkUpFindXCF(sub, home)
	assert.Equal(t, xcf, got, "should find scaffold.xcf at project level")
}

// TestWalkUp_StopsAtHome verifies the walk-up does NOT traverse above $HOME.
// scaffold.xcf is placed above home; the function must fall back to cwd.
func TestWalkUp_StopsAtHome(t *testing.T) {
	// Layout:
	//   tmp/above/scaffold.xcf  ← must NOT be found
	//   tmp/above/home/sub/      ← cwd  (home == tmp/above/home)
	root := t.TempDir()
	above := root // treat root itself as the directory above home
	home := filepath.Join(above, "home")
	sub := filepath.Join(home, "sub")
	require.NoError(t, os.MkdirAll(sub, 0755))

	xcfAboveHome := filepath.Join(above, "scaffold.xcf")
	require.NoError(t, os.WriteFile(xcfAboveHome, []byte(`version: "1"`+"\n"), 0600))

	got := walkUpFindXCF(sub, home)

	// The result should be the cwd fallback, not the file above home.
	assert.Equal(t, filepath.Join(sub, "scaffold.xcf"), got,
		"walk-up must not cross the home boundary")
	assert.NotEqual(t, xcfAboveHome, got,
		"must not return scaffold.xcf located above $HOME")
}

// TestWalkUp_FinisAtHome verifies the walk-up finds scaffold.xcf when it sits
// exactly at $HOME (the boundary is inclusive).
func TestWalkUp_FinisAtHome(t *testing.T) {
	// Layout:
	//   tmp/home/scaffold.xcf  ← exactly at home boundary
	//   tmp/home/sub/           ← cwd
	root := t.TempDir()
	home := filepath.Join(root, "home")
	sub := filepath.Join(home, "sub")
	require.NoError(t, os.MkdirAll(sub, 0755))

	xcf := filepath.Join(home, "scaffold.xcf")
	require.NoError(t, os.WriteFile(xcf, []byte(`version: "1"`+"\n"), 0600))

	got := walkUpFindXCF(sub, home)
	assert.Equal(t, xcf, got, "scaffold.xcf at $HOME itself must be found")
}

// TestWalkUp_CwdIsHome verifies behaviour when cwd equals home and no xcf
// is present — fallback to cwd.
func TestWalkUp_CwdIsHome(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	require.NoError(t, os.MkdirAll(home, 0755))

	// No scaffold.xcf anywhere.
	got := walkUpFindXCF(home, home)
	assert.Equal(t, filepath.Join(home, "scaffold.xcf"), got,
		"fallback should be cwd/scaffold.xcf when nothing is found")
}
