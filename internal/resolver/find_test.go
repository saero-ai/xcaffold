package resolver

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindConfigDir_FindsScaffoldXcaf(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	project := filepath.Join(home, "project")
	sub := filepath.Join(project, "sub")
	require.NoError(t, os.MkdirAll(sub, 0755))

	xcaf := filepath.Join(project, "project.xcaf")
	require.NoError(t, os.WriteFile(xcaf, []byte("version: \"1\"\n"), 0600))

	got, err := FindConfigDir(sub, home)
	require.NoError(t, err)
	assert.Equal(t, project, got)
}

func TestFindConfigDir_FindsAnyXcafFile(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	project := filepath.Join(home, "project")
	sub := filepath.Join(project, "sub")
	require.NoError(t, os.MkdirAll(sub, 0755))

	// Only agents.xcaf — no project.xcaf
	xcaf := filepath.Join(project, "agents.xcaf")
	require.NoError(t, os.WriteFile(xcaf, []byte("agents:\n  dev:\n    name: Dev\n"), 0600))

	got, err := FindConfigDir(sub, home)
	require.NoError(t, err)
	assert.Equal(t, project, got)
}

func TestFindConfigDir_FindsMultipleXcafFiles(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	project := filepath.Join(home, "project")
	sub := filepath.Join(project, "sub")
	require.NoError(t, os.MkdirAll(sub, 0755))

	require.NoError(t, os.WriteFile(filepath.Join(project, "agents.xcaf"), []byte("agents:\n"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(project, "rules.xcaf"), []byte("rules:\n"), 0600))

	got, err := FindConfigDir(sub, home)
	require.NoError(t, err)
	assert.Equal(t, project, got)
}

func TestFindConfigDir_StopsAtHome(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	sub := filepath.Join(home, "sub")
	require.NoError(t, os.MkdirAll(sub, 0755))

	// Place xcaf ABOVE home — must NOT be found
	require.NoError(t, os.WriteFile(filepath.Join(root, "project.xcaf"), []byte("version: \"1\"\n"), 0600))

	_, err := FindConfigDir(sub, home)
	assert.Error(t, err, "should fail when no xcaf found within home boundary")
}

func TestFindConfigDir_FindsAtHome(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	sub := filepath.Join(home, "sub")
	require.NoError(t, os.MkdirAll(sub, 0755))

	require.NoError(t, os.WriteFile(filepath.Join(home, "project.xcaf"), []byte("version: \"1\"\n"), 0600))

	got, err := FindConfigDir(sub, home)
	require.NoError(t, err)
	assert.Equal(t, home, got)
}

func TestFindConfigDir_CwdHasXcaf(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	project := filepath.Join(home, "project")
	require.NoError(t, os.MkdirAll(project, 0755))

	require.NoError(t, os.WriteFile(filepath.Join(project, "project.xcaf"), []byte("version: \"1\"\n"), 0600))

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

	// Only xcaf file is inside a hidden dir — should NOT count
	require.NoError(t, os.WriteFile(filepath.Join(hidden, "something.xcaf"), []byte("bad\n"), 0600))

	_, err := FindConfigDir(project, home)
	assert.Error(t, err, "xcaf files inside hidden dirs should not be found")
}

func TestFindXCAFFiles_ReturnsSorted(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "z.xcaf"), []byte("z"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.xcaf"), []byte("a"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "m.xcaf"), []byte("m"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "not-xcaf.yaml"), []byte("n"), 0600))

	files, err := FindXCAFFiles(dir)
	require.NoError(t, err)
	require.Len(t, files, 3)
	assert.Equal(t, filepath.Join(dir, "a.xcaf"), files[0])
	assert.Equal(t, filepath.Join(dir, "m.xcaf"), files[1])
	assert.Equal(t, filepath.Join(dir, "z.xcaf"), files[2])
}

func TestFindXCAFFiles_ExcludesRegistryXcaf(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "project.xcaf"), []byte("s"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "registry.xcaf"), []byte("r"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "agents.xcaf"), []byte("a"), 0600))

	files, err := FindXCAFFiles(dir)
	require.NoError(t, err)
	require.Len(t, files, 2, "registry.xcaf must be excluded")
	for _, f := range files {
		assert.NotContains(t, f, "registry.xcaf")
	}
}

func TestFindXCAFFiles_Recursive(t *testing.T) {
	dir := t.TempDir()
	subdir := filepath.Join(dir, "agents")
	require.NoError(t, os.MkdirAll(subdir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "project.xcaf"), []byte("p"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(subdir, "dev.xcaf"), []byte("d"), 0600))

	// Hidden dir should be skipped
	hidden := filepath.Join(dir, ".git")
	require.NoError(t, os.MkdirAll(hidden, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(hidden, "bad.xcaf"), []byte("b"), 0600))

	files, err := FindXCAFFiles(dir)
	require.NoError(t, err)
	require.Len(t, files, 2)
}

func TestFindProjectRoot_RootFirst(t *testing.T) {
	tmp := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "project.xcaf"), []byte("kind: project\nversion: \"1.0\"\nname: test\n"), 0644))
	found := FindProjectRoot(tmp)
	require.NotEmpty(t, found, "FindProjectRoot did not find project.xcaf at root")
	assert.Equal(t, tmp, found)
}

func TestFindProjectRoot_NotFound(t *testing.T) {
	tmp := t.TempDir()
	found := FindProjectRoot(tmp)
	assert.Empty(t, found, "expected empty string when no project.xcaf found")
}

func TestFindProjectRoot_WalksUptoRoot(t *testing.T) {
	tmp := t.TempDir()
	subdir := filepath.Join(tmp, "a", "b", "c")
	require.NoError(t, os.MkdirAll(subdir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "project.xcaf"), []byte("kind: project\nversion: \"1.0\"\nname: test\n"), 0644))

	found := FindProjectRoot(subdir)
	require.NotEmpty(t, found, "FindProjectRoot should walk up to find project.xcaf")
	assert.Equal(t, tmp, found)
}

func TestFindProjectRoot_PrefersRootOverXcaffold(t *testing.T) {
	tmp := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, ".xcaffold"), 0755))
	// Create both root and .xcaffold versions
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "project.xcaf"), []byte("kind: project\nversion: \"1.0\"\nname: root\n"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmp, ".xcaffold", "project.xcaf"), []byte("kind: project\nversion: \"1.0\"\nname: xcaffold\n"), 0644))

	found := FindProjectRoot(tmp)
	require.NotEmpty(t, found, "FindProjectRoot should prefer root version")
	assert.Equal(t, tmp, found)

	// Verify the root version is returned by checking file content
	projPath := filepath.Join(found, "project.xcaf")
	content, err := os.ReadFile(projPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "root")
}

func TestFindProjectRoot_IgnoresProjectXCF(t *testing.T) {
	dir := t.TempDir()

	// Create project.xcf (old extension) — should NOT be found
	require.NoError(t, os.WriteFile(filepath.Join(dir, "project.xcf"), []byte("kind: project\nversion: \"1.0\"\nname: test\n"), 0644))

	result := FindProjectRoot(dir)
	assert.Empty(t, result, "project.xcf (old extension) must not be discovered")
}

func TestFindProjectRoot_FindsProjectXCAF(t *testing.T) {
	dir := t.TempDir()

	// Create project.xcaf (correct extension) — should be found
	require.NoError(t, os.WriteFile(filepath.Join(dir, "project.xcaf"), []byte("kind: project\nversion: \"1.0\"\nname: test\n"), 0644))

	result := FindProjectRoot(dir)
	assert.Equal(t, dir, result, "project.xcaf must be discovered")
}

func TestDirContainsXCAF_IgnoresXCFFiles(t *testing.T) {
	dir := t.TempDir()

	// Only .xcf files — should NOT count as containing xcaf
	require.NoError(t, os.WriteFile(filepath.Join(dir, "agent.xcf"), []byte("kind: agent\n"), 0644))

	// dirContainsXCAF is unexported, but we can test it indirectly via FindConfigDir
	// Create a home boundary to test the behavior
	home := t.TempDir()
	_, err := FindConfigDir(dir, home)
	assert.Error(t, err, "directory with only .xcf files must not be discovered as a config dir")
}
