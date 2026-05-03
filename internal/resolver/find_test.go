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
	require.NoError(t, os.MkdirAll(filepath.Join(project, ".xcaffold"), 0755))

	xcf := filepath.Join(project, ".xcaffold", "project.xcf")
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

	// Only agents.xcf — no project.xcf
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
	require.NoError(t, os.WriteFile(filepath.Join(root, "project.xcf"), []byte("version: \"1\"\n"), 0600))

	_, err := FindConfigDir(sub, home)
	assert.Error(t, err, "should fail when no xcf found within home boundary")
}

func TestFindConfigDir_FindsAtHome(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	sub := filepath.Join(home, "sub")
	require.NoError(t, os.MkdirAll(sub, 0755))

	require.NoError(t, os.WriteFile(filepath.Join(home, "project.xcf"), []byte("version: \"1\"\n"), 0600))

	got, err := FindConfigDir(sub, home)
	require.NoError(t, err)
	assert.Equal(t, home, got)
}

func TestFindConfigDir_CwdHasXcf(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	project := filepath.Join(home, "project")
	require.NoError(t, os.MkdirAll(project, 0755))

	require.NoError(t, os.WriteFile(filepath.Join(project, "project.xcf"), []byte("version: \"1\"\n"), 0600))

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

func TestFindXCFFiles_ExcludesRegistryXcf(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "project.xcf"), []byte("s"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "registry.xcf"), []byte("r"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "agents.xcf"), []byte("a"), 0600))

	files, err := FindXCFFiles(dir)
	require.NoError(t, err)
	require.Len(t, files, 2, "registry.xcf must be excluded")
	for _, f := range files {
		assert.NotContains(t, f, "registry.xcf")
	}
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

func TestFindProjectRoot_RootFirst(t *testing.T) {
	tmp := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "project.xcf"), []byte("kind: project\nversion: \"1.0\"\nname: test\n"), 0644))
	found := FindProjectRoot(tmp)
	require.NotEmpty(t, found, "FindProjectRoot did not find project.xcf at root")
	assert.Equal(t, tmp, found)
}

func TestFindProjectRoot_FallbackToXcaffoldDir(t *testing.T) {
	tmp := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, ".xcaffold"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(tmp, ".xcaffold", "project.xcf"), []byte("kind: project\nversion: \"1.0\"\nname: test\n"), 0644))
	found := FindProjectRoot(tmp)
	require.NotEmpty(t, found, "FindProjectRoot did not find .xcaffold/project.xcf as fallback")
	assert.Equal(t, tmp, found)
}

func TestFindProjectRoot_NotFound(t *testing.T) {
	tmp := t.TempDir()
	found := FindProjectRoot(tmp)
	assert.Empty(t, found, "expected empty string when no project.xcf found")
}

func TestFindProjectRoot_WalksUptoRoot(t *testing.T) {
	tmp := t.TempDir()
	subdir := filepath.Join(tmp, "a", "b", "c")
	require.NoError(t, os.MkdirAll(subdir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "project.xcf"), []byte("kind: project\nversion: \"1.0\"\nname: test\n"), 0644))

	found := FindProjectRoot(subdir)
	require.NotEmpty(t, found, "FindProjectRoot should walk up to find project.xcf")
	assert.Equal(t, tmp, found)
}

func TestFindProjectRoot_PrefersRootOverXcaffold(t *testing.T) {
	tmp := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, ".xcaffold"), 0755))
	// Create both root and .xcaffold versions
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "project.xcf"), []byte("kind: project\nversion: \"1.0\"\nname: root\n"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmp, ".xcaffold", "project.xcf"), []byte("kind: project\nversion: \"1.0\"\nname: xcaffold\n"), 0644))

	found := FindProjectRoot(tmp)
	require.NotEmpty(t, found, "FindProjectRoot should prefer root version")
	assert.Equal(t, tmp, found)

	// Verify the root version is returned by checking file content
	projPath := filepath.Join(found, "project.xcf")
	content, err := os.ReadFile(projPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "root")
}
