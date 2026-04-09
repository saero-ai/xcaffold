package resolver

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindConfigDir_FindsScaffoldXcf(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	project := filepath.Join(home, "project")
	sub := filepath.Join(project, "sub")
	require.NoError(t, os.MkdirAll(sub, 0755))

	xcf := filepath.Join(project, "scaffold.xcf")
	require.NoError(t, os.WriteFile(xcf, []byte("version: \"1\"\n"), 0600))

	got, err := FindConfigDir(sub, home)
	require.NoError(t, err)
	assert.Equal(t, project, got)
}

func TestFindConfigDir_FindsAnyXcfFile(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	project := filepath.Join(home, "project")
	sub := filepath.Join(project, "sub")
	require.NoError(t, os.MkdirAll(sub, 0755))

	// Only agents.xcf — no scaffold.xcf
	xcf := filepath.Join(project, "agents.xcf")
	require.NoError(t, os.WriteFile(xcf, []byte("agents:\n  dev:\n    name: Dev\n"), 0600))

	got, err := FindConfigDir(sub, home)
	require.NoError(t, err)
	assert.Equal(t, project, got)
}

func TestFindConfigDir_FindsMultipleXcfFiles(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	project := filepath.Join(home, "project")
	sub := filepath.Join(project, "sub")
	require.NoError(t, os.MkdirAll(sub, 0755))

	require.NoError(t, os.WriteFile(filepath.Join(project, "agents.xcf"), []byte("agents:\n"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(project, "rules.xcf"), []byte("rules:\n"), 0600))

	got, err := FindConfigDir(sub, home)
	require.NoError(t, err)
	assert.Equal(t, project, got)
}

func TestFindConfigDir_StopsAtHome(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	sub := filepath.Join(home, "sub")
	require.NoError(t, os.MkdirAll(sub, 0755))

	// Place xcf ABOVE home — must NOT be found
	require.NoError(t, os.WriteFile(filepath.Join(root, "scaffold.xcf"), []byte("version: \"1\"\n"), 0600))

	_, err := FindConfigDir(sub, home)
	assert.Error(t, err, "should fail when no xcf found within home boundary")
}

func TestFindConfigDir_FindsAtHome(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	sub := filepath.Join(home, "sub")
	require.NoError(t, os.MkdirAll(sub, 0755))

	require.NoError(t, os.WriteFile(filepath.Join(home, "scaffold.xcf"), []byte("version: \"1\"\n"), 0600))

	got, err := FindConfigDir(sub, home)
	require.NoError(t, err)
	assert.Equal(t, home, got)
}

func TestFindConfigDir_CwdHasXcf(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	project := filepath.Join(home, "project")
	require.NoError(t, os.MkdirAll(project, 0755))

	require.NoError(t, os.WriteFile(filepath.Join(project, "scaffold.xcf"), []byte("version: \"1\"\n"), 0600))

	got, err := FindConfigDir(project, home)
	require.NoError(t, err)
	assert.Equal(t, project, got)
}

func TestFindConfigDir_IgnoresHiddenDirs(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	project := filepath.Join(home, "project")
	hidden := filepath.Join(project, ".claude")
	require.NoError(t, os.MkdirAll(hidden, 0755))

	// Only xcf file is inside a hidden dir — should NOT count
	require.NoError(t, os.WriteFile(filepath.Join(hidden, "something.xcf"), []byte("bad\n"), 0600))

	_, err := FindConfigDir(project, home)
	assert.Error(t, err, "xcf files inside hidden dirs should not be found")
}

func TestFindXCFFiles_ReturnsSorted(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "z.xcf"), []byte("z"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.xcf"), []byte("a"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "m.xcf"), []byte("m"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "not-xcf.yaml"), []byte("n"), 0600))

	files, err := FindXCFFiles(dir)
	require.NoError(t, err)
	require.Len(t, files, 3)
	assert.Equal(t, filepath.Join(dir, "a.xcf"), files[0])
	assert.Equal(t, filepath.Join(dir, "m.xcf"), files[1])
	assert.Equal(t, filepath.Join(dir, "z.xcf"), files[2])
}

func TestFindXCFFiles_Recursive(t *testing.T) {
	dir := t.TempDir()
	subdir := filepath.Join(dir, "agents")
	require.NoError(t, os.MkdirAll(subdir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "project.xcf"), []byte("p"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(subdir, "dev.xcf"), []byte("d"), 0600))

	// Hidden dir should be skipped
	hidden := filepath.Join(dir, ".git")
	require.NoError(t, os.MkdirAll(hidden, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(hidden, "bad.xcf"), []byte("b"), 0600))

	files, err := FindXCFFiles(dir)
	require.NoError(t, err)
	require.Len(t, files, 2)
}
