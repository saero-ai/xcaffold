package state

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/saero-ai/xcaffold/internal/output"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestMemorySeed_SortedByName(t *testing.T) {
	seeds := []MemorySeed{
		{Name: "z-entry", Target: "claude"},
		{Name: "a-entry", Target: "claude"},
		{Name: "m-entry", Target: "claude"},
	}
	sortMemorySeeds(seeds)

	require.Equal(t, "a-entry", seeds[0].Name)
	require.Equal(t, "m-entry", seeds[1].Name)
	require.Equal(t, "z-entry", seeds[2].Name)
}

func TestStateManifest_Fields(t *testing.T) {
	m := &StateManifest{
		Version:         1,
		XcaffoldVersion: "1.2.0",
		Blueprint:       "backend",
		BlueprintHash:   "sha256:abc",
		SourceFiles: []SourceFile{
			{Path: "project.xcaf", Hash: "sha256:111"},
		},
		Targets: map[string]TargetState{
			"claude": {
				LastApplied: "2026-04-20T00:00:00Z",
				Artifacts: []Artifact{
					{Path: "agents/dev.md", Hash: "sha256:222"},
				},
			},
		},
		MemorySeeds: []MemorySeed{
			{Name: "arch", Target: "claude", Path: "arch.md", Hash: "sha256:333",
				SeededAt: "2026-04-20T00:00:00Z"},
		},
	}

	data, err := yaml.Marshal(m)
	require.NoError(t, err)

	raw := string(data)
	assert.Contains(t, raw, "version: 1")
	assert.Contains(t, raw, "xcaffold-version:")
	assert.Contains(t, raw, "blueprint: backend")
	assert.Contains(t, raw, "blueprint-hash:")
	assert.Contains(t, raw, "source-files:")
	assert.Contains(t, raw, "memory-seeds:")
	assert.Contains(t, raw, "seeded-at:")
	assert.NotContains(t, raw, "seeded_at")
	assert.Contains(t, raw, "last-applied:")
}

func TestStateManifest_EmptyBlueprint(t *testing.T) {
	m := &StateManifest{
		Version:         1,
		XcaffoldVersion: "1.0.0",
		Targets:         map[string]TargetState{},
	}
	data, err := yaml.Marshal(m)
	require.NoError(t, err)
	assert.NotContains(t, string(data), "blueprint:")

	var out StateManifest
	require.NoError(t, yaml.Unmarshal(data, &out))
	assert.Equal(t, "", out.Blueprint)
}

func TestStateDir(t *testing.T) {
	assert.Equal(t, "/home/user/proj/.xcaffold", StateDir("/home/user/proj"))
}

func TestStateFilePath_Default(t *testing.T) {
	got := StateFilePath("/home/user/proj", "")
	assert.Equal(t, "/home/user/proj/.xcaffold/project.xcaf.state", got)
}

func TestStateFilePath_NamedProfile(t *testing.T) {
	got := StateFilePath("/home/user/proj", "backend")
	assert.Equal(t, "/home/user/proj/.xcaffold/backend.xcaf.state", got)
}

func TestStateFilePath_PathTraversal(t *testing.T) {
	got := StateFilePath("/home/user/proj", "../../etc/passwd")
	assert.True(t, strings.HasPrefix(got, "/home/user/proj/.xcaffold/"),
		"path must remain inside .xcaffold/: %s", got)
	assert.NotContains(t, got, "..")
}

func TestStateFilePathWithOutputDir_Default(t *testing.T) {
	got := StateFilePathWithOutputDir("/home/user/proj", "", "")
	assert.Equal(t, "/home/user/proj/.xcaffold/project.xcaf.state", got)
}

func TestStateFilePathWithOutputDir_BlueprintOnly(t *testing.T) {
	got := StateFilePathWithOutputDir("/home/user/proj", "backend", "")
	assert.Equal(t, "/home/user/proj/.xcaffold/backend.xcaf.state", got)
}

func TestStateFilePathWithOutputDir_WithRelativeDir(t *testing.T) {
	got := StateFilePathWithOutputDir("/home/user/proj", "backend", "custom-out/")
	assert.Equal(t, "/home/user/proj/.xcaffold/backend@custom-out.xcaf.state", got)
}

func TestStateFilePathWithOutputDir_WithNestedDir(t *testing.T) {
	got := StateFilePathWithOutputDir("/home/user/proj", "backend", "deep/nested/dir/")
	assert.Equal(t, "/home/user/proj/.xcaffold/backend@deep_nested_dir.xcaf.state", got)
}

func TestStateFilePathWithOutputDir_WithAbsoluteDir(t *testing.T) {
	got := StateFilePathWithOutputDir("/home/user/proj", "", "/tmp/xcaffold-out/")
	assert.Equal(t, "/home/user/proj/.xcaffold/project@tmp_xcaffold-out.xcaf.state", got)
}

func TestStateFilePathWithOutputDir_BackwardCompatible(t *testing.T) {
	// When outputDir is empty, must produce identical result to StateFilePath
	for _, bp := range []string{"", "alpha", "backend"} {
		old := StateFilePath("/base", bp)
		new := StateFilePathWithOutputDir("/base", bp, "")
		assert.Equal(t, old, new, "mismatch for blueprint %q", bp)
	}
}

func TestStateOpts_Fields(t *testing.T) {
	opts := StateOpts{
		Blueprint:     "backend",
		BlueprintHash: "sha256:abc",
		Target:        "claude",
		BaseDir:       "/tmp/proj",
		SourceFiles:   []string{"/tmp/proj/project.xcaf"},
		MemorySeeds:   nil,
	}
	assert.Equal(t, "claude", opts.Target)
	assert.Equal(t, "backend", opts.Blueprint)
}

func TestState_WriteRead_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := StateFilePath(dir, "")

	original := &StateManifest{
		Version:         1,
		XcaffoldVersion: "1.2.0",
		Blueprint:       "",
		SourceFiles:     []SourceFile{{Path: "project.xcaf", Hash: "sha256:aaa"}},
		Targets: map[string]TargetState{
			"claude": {
				LastApplied: "2026-04-20T01:00:00Z",
				Artifacts:   []Artifact{{Path: "agents/dev.md", Hash: "sha256:bbb"}},
			},
		},
	}

	require.NoError(t, WriteState(original, path))

	got, err := ReadState(path)
	require.NoError(t, err)

	assert.Equal(t, original.Version, got.Version)
	assert.Equal(t, original.XcaffoldVersion, got.XcaffoldVersion)
	assert.Equal(t, original.SourceFiles, got.SourceFiles)
	assert.Equal(t, original.Targets["claude"].LastApplied, got.Targets["claude"].LastApplied)
	assert.Equal(t, original.Targets["claude"].Artifacts, got.Targets["claude"].Artifacts)
}

func TestWriteState_CreatesDirectory(t *testing.T) {
	base := t.TempDir()
	path := StateFilePath(base, "")
	require.NoError(t, WriteState(&StateManifest{Version: 1, Targets: map[string]TargetState{}}, path))
	_, err := os.Stat(filepath.Join(base, ".xcaffold"))
	assert.NoError(t, err, ".xcaffold/ should be created")
}

func TestWriteState_Permissions(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root always has access")
	}
	base := t.TempDir()
	path := StateFilePath(base, "")
	require.NoError(t, WriteState(&StateManifest{Version: 1, Targets: map[string]TargetState{}}, path))
	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm())
}

func TestReadState_FileNotFound(t *testing.T) {
	_, err := ReadState("/nonexistent/.xcaffold/project.xcaf.state")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "project.xcaf.state")
}

func TestReadState_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.xcaf.state")
	require.NoError(t, os.WriteFile(path, []byte("version: [invalid\n  broken:"), 0600))
	_, err := ReadState(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse")
}

func TestGenerateState_DefaultBlueprint(t *testing.T) {
	out := &output.Output{Files: map[string]string{
		"agents/dev.md": "# dev",
	}}
	opts := StateOpts{Target: "claude", BaseDir: t.TempDir()}
	m, err := GenerateState(out, opts, nil)
	require.NoError(t, err)

	assert.Equal(t, 1, m.Version)
	ts, ok := m.Targets["claude"]
	require.True(t, ok)
	assert.Len(t, ts.Artifacts, 1)
	assert.Equal(t, "agents/dev.md", ts.Artifacts[0].Path)
	assert.True(t, strings.HasPrefix(ts.Artifacts[0].Hash, "sha256:"))
}

func TestGenerateState_MergesTargets(t *testing.T) {
	out1 := &output.Output{Files: map[string]string{"agents/dev.md": "# dev"}}
	out2 := &output.Output{Files: map[string]string{"agents/dev.md": "# dev cursor"}}

	opts1 := StateOpts{Target: "claude", BaseDir: t.TempDir()}
	opts2 := StateOpts{Target: "cursor", BaseDir: t.TempDir()}

	m1, err := GenerateState(out1, opts1, nil)
	require.NoError(t, err)
	m2, err := GenerateState(out2, opts2, m1)
	require.NoError(t, err)

	assert.Contains(t, m2.Targets, "claude")
	assert.Contains(t, m2.Targets, "cursor")
}

func TestFindOrphansFromState_NilOldState(t *testing.T) {
	orphans := FindOrphansFromState(nil, "claude", map[string]string{"agents/dev.md": "# dev"}, nil)
	assert.Nil(t, orphans)
}

func TestFindOrphansFromState_TargetNotInOldState(t *testing.T) {
	old := &StateManifest{
		Targets: map[string]TargetState{
			"cursor": {Artifacts: []Artifact{{Path: "agents/dev.md", Hash: "sha256:aaa"}}},
		},
	}
	orphans := FindOrphansFromState(old, "claude", map[string]string{}, nil)
	assert.Nil(t, orphans)
}

func TestFindOrphansFromState_NoOrphans(t *testing.T) {
	old := &StateManifest{
		Targets: map[string]TargetState{
			"claude": {Artifacts: []Artifact{
				{Path: "agents/dev.md", Hash: "sha256:aaa"},
				{Path: "skills/tdd/SKILL.md", Hash: "sha256:bbb"},
			}},
		},
	}
	newFiles := map[string]string{
		"agents/dev.md":       "# dev",
		"skills/tdd/SKILL.md": "# tdd",
	}
	orphans := FindOrphansFromState(old, "claude", newFiles, nil)
	assert.Empty(t, orphans)
}

func TestFindOrphansFromState_OrphansFound(t *testing.T) {
	old := &StateManifest{
		Targets: map[string]TargetState{
			"claude": {Artifacts: []Artifact{
				{Path: "agents/dev.md", Hash: "sha256:aaa"},
				{Path: "agents/old.md", Hash: "sha256:bbb"},
				{Path: "skills/tdd/SKILL.md", Hash: "sha256:ccc"},
			}},
		},
	}
	newFiles := map[string]string{
		"agents/dev.md": "# dev",
	}
	orphans := FindOrphansFromState(old, "claude", newFiles, nil)
	assert.Equal(t, []string{"agents/old.md", "skills/tdd/SKILL.md"}, orphans)
}

func TestGenerateState_SourceFilesHashed(t *testing.T) {
	base := t.TempDir()
	src := filepath.Join(base, "project.xcaf")
	require.NoError(t, os.WriteFile(src, []byte("kind: project\n"), 0644))

	out := &output.Output{Files: map[string]string{}}
	opts := StateOpts{
		Target:      "claude",
		BaseDir:     base,
		SourceFiles: []string{src},
	}
	m, err := GenerateState(out, opts, nil)
	require.NoError(t, err)

	claudeTarget := m.Targets["claude"]
	require.Len(t, claudeTarget.SourceFiles, 1)
	assert.Equal(t, "project.xcaf", claudeTarget.SourceFiles[0].Path)
	assert.True(t, strings.HasPrefix(claudeTarget.SourceFiles[0].Hash, "sha256:"))
}

func TestGenerateState_TargetKeyedSources(t *testing.T) {
	base := t.TempDir()
	src := filepath.Join(base, "project.xcaf")
	require.NoError(t, os.WriteFile(src, []byte("kind: project\n"), 0644))

	out := &output.Output{Files: map[string]string{"agents/dev.md": "# dev"}}
	opts := StateOpts{
		Target:      "claude",
		BaseDir:     base,
		SourceFiles: []string{src},
	}
	m, err := GenerateState(out, opts, nil)
	require.NoError(t, err)

	// Source files should be stored in the target-specific state
	claudeTarget := m.Targets["claude"]
	require.Len(t, claudeTarget.SourceFiles, 1)
	assert.Equal(t, "project.xcaf", claudeTarget.SourceFiles[0].Path)
	assert.True(t, strings.HasPrefix(claudeTarget.SourceFiles[0].Hash, "sha256:"))
}

func TestGenerateState_IndependentSourcesPerTarget(t *testing.T) {
	base := t.TempDir()
	src1 := filepath.Join(base, "claude.xcaf")
	src2 := filepath.Join(base, "cursor.xcaf")
	require.NoError(t, os.WriteFile(src1, []byte("kind: project\nname: claude\n"), 0644))
	require.NoError(t, os.WriteFile(src2, []byte("kind: project\nname: cursor\n"), 0644))

	out1 := &output.Output{Files: map[string]string{"agents/dev.md": "# claude"}}
	opts1 := StateOpts{
		Target:      "claude",
		BaseDir:     base,
		SourceFiles: []string{src1},
	}
	m1, err := GenerateState(out1, opts1, nil)
	require.NoError(t, err)

	// Apply cursor target with different sources
	out2 := &output.Output{Files: map[string]string{"agents/dev.md": "# cursor"}}
	opts2 := StateOpts{
		Target:      "cursor",
		BaseDir:     base,
		SourceFiles: []string{src2},
	}
	m2, err := GenerateState(out2, opts2, m1)
	require.NoError(t, err)

	// Each target should have independent source tracking
	claudeSources := m2.Targets["claude"].SourceFiles
	cursorSources := m2.Targets["cursor"].SourceFiles

	require.Len(t, claudeSources, 1)
	require.Len(t, cursorSources, 1)
	assert.Equal(t, "claude.xcaf", claudeSources[0].Path)
	assert.Equal(t, "cursor.xcaf", cursorSources[0].Path)
	assert.NotEqual(t, claudeSources[0].Hash, cursorSources[0].Hash)
}

func TestSourcesChanged_UsesTargetSpecificSources(t *testing.T) {
	base := t.TempDir()
	src1 := filepath.Join(base, "claude.xcaf")
	src2 := filepath.Join(base, "cursor.xcaf")
	require.NoError(t, os.WriteFile(src1, []byte("kind: project\nname: claude\n"), 0644))
	require.NoError(t, os.WriteFile(src2, []byte("kind: project\nname: cursor\n"), 0644))

	// Initial state for claude
	out1 := &output.Output{Files: map[string]string{}}
	opts1 := StateOpts{
		Target:      "claude",
		BaseDir:     base,
		SourceFiles: []string{src1},
	}
	m1, _ := GenerateState(out1, opts1, nil)

	// Now add cursor target
	out2 := &output.Output{Files: map[string]string{}}
	opts2 := StateOpts{
		Target:      "cursor",
		BaseDir:     base,
		SourceFiles: []string{src2},
	}
	m2, _ := GenerateState(out2, opts2, m1)

	// Modifying src1 should be detected by claude's check
	require.NoError(t, os.WriteFile(src1, []byte("kind: project\nname: claude\nchanged: yes\n"), 0644))
	changed, _ := SourcesChanged(m2.Targets["claude"].SourceFiles, []string{src1}, base)
	assert.True(t, changed, "claude should detect src1 change")

	// src2 should be unchanged for cursor
	changed2, _ := SourcesChanged(m2.Targets["cursor"].SourceFiles, []string{src2}, base)
	assert.False(t, changed2, "cursor should not detect change in src2")
}

func TestListStateFiles(t *testing.T) {
	base := t.TempDir()
	stateDir := filepath.Join(base, ".xcaffold")
	require.NoError(t, os.MkdirAll(stateDir, 0755))

	// Create multiple state files
	require.NoError(t, os.WriteFile(filepath.Join(stateDir, "project.xcaf.state"), []byte("version: 1\n"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(stateDir, "backend.xcaf.state"), []byte("version: 1\n"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(stateDir, "frontend.xcaf.state"), []byte("version: 1\n"), 0600))

	files, err := ListStateFiles(stateDir)
	require.NoError(t, err)
	require.Len(t, files, 3)

	// Verify they're in the state dir
	for _, f := range files {
		assert.True(t, strings.HasPrefix(f, stateDir))
	}
}

func TestListStateFiles_Empty(t *testing.T) {
	base := t.TempDir()
	stateDir := filepath.Join(base, ".xcaffold")
	require.NoError(t, os.MkdirAll(stateDir, 0755))

	files, err := ListStateFiles(stateDir)
	require.NoError(t, err)
	assert.Empty(t, files)
}

func TestListStateFiles_IgnoresNonStateFiles(t *testing.T) {
	base := t.TempDir()
	stateDir := filepath.Join(base, ".xcaffold")
	require.NoError(t, os.MkdirAll(stateDir, 0755))

	require.NoError(t, os.WriteFile(filepath.Join(stateDir, "project.xcaf.state"), []byte("version: 1\n"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(stateDir, "readme.txt"), []byte("not a state file\n"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(stateDir, "project.xcaf"), []byte("kind: project\n"), 0644))

	files, err := ListStateFiles(stateDir)
	require.NoError(t, err)
	require.Len(t, files, 1)
	assert.True(t, strings.HasSuffix(files[0], "project.xcaf.state"))
}

func TestFindMostRecentState_NoStateDir(t *testing.T) {
	nonExistent := filepath.Join(t.TempDir(), "nonexistent")
	manifest, blueprint, err := FindMostRecentState(nonExistent)
	assert.Nil(t, manifest)
	assert.Equal(t, "", blueprint)
	assert.NoError(t, err)
}

func TestFindMostRecentState_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	stateDir := filepath.Join(dir, ".xcaffold")
	require.NoError(t, os.MkdirAll(stateDir, 0755))

	manifest, blueprint, err := FindMostRecentState(stateDir)
	assert.Nil(t, manifest)
	assert.Equal(t, "", blueprint)
	assert.NoError(t, err)
}

func TestFindMostRecentState_SingleProjectState(t *testing.T) {
	dir := t.TempDir()
	manifest := &StateManifest{
		Version:         1,
		XcaffoldVersion: "1.0.0",
		Blueprint:       "",
		Targets: map[string]TargetState{
			"claude": {
				LastApplied: "2026-05-19T10:00:00Z",
				Artifacts:   []Artifact{{Path: "test.md", Hash: "sha256:abc"}},
			},
		},
	}
	require.NoError(t, WriteState(manifest, filepath.Join(dir, ".xcaffold", "project.xcaf.state")))

	result, blueprint, err := FindMostRecentState(filepath.Join(dir, ".xcaffold"))
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "", blueprint)
	assert.Equal(t, 1, result.Version)
}

func TestFindMostRecentState_SingleBlueprintState(t *testing.T) {
	dir := t.TempDir()
	manifest := &StateManifest{
		Version:         1,
		XcaffoldVersion: "1.0.0",
		Blueprint:       "backend",
		Targets: map[string]TargetState{
			"claude": {
				LastApplied: "2026-05-19T10:00:00Z",
				Artifacts:   []Artifact{{Path: "test.md", Hash: "sha256:abc"}},
			},
		},
	}
	require.NoError(t, WriteState(manifest, filepath.Join(dir, ".xcaffold", "backend.xcaf.state")))

	result, blueprint, err := FindMostRecentState(filepath.Join(dir, ".xcaffold"))
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "backend", blueprint)
	assert.Equal(t, 1, result.Version)
}

func TestFindMostRecentState_MultipleStates_MostRecentWins(t *testing.T) {
	dir := t.TempDir()

	// project at 10:00
	project := &StateManifest{
		Version: 1, XcaffoldVersion: "1.0.0",
		Targets: map[string]TargetState{
			"claude": {LastApplied: "2026-05-19T10:00:00Z", Artifacts: []Artifact{{Path: "p.md", Hash: "sha256:p"}}},
		},
	}
	require.NoError(t, WriteState(project, filepath.Join(dir, ".xcaffold", "project.xcaf.state")))

	// backend at 12:00 (most recent)
	backend := &StateManifest{
		Version: 1, XcaffoldVersion: "1.0.0", Blueprint: "backend",
		Targets: map[string]TargetState{
			"claude": {LastApplied: "2026-05-19T12:00:00Z", Artifacts: []Artifact{{Path: "b.md", Hash: "sha256:b"}}},
		},
	}
	require.NoError(t, WriteState(backend, filepath.Join(dir, ".xcaffold", "backend.xcaf.state")))

	// frontend at 11:00
	frontend := &StateManifest{
		Version: 1, XcaffoldVersion: "1.0.0", Blueprint: "frontend",
		Targets: map[string]TargetState{
			"claude": {LastApplied: "2026-05-19T11:00:00Z", Artifacts: []Artifact{{Path: "f.md", Hash: "sha256:f"}}},
		},
	}
	require.NoError(t, WriteState(frontend, filepath.Join(dir, ".xcaffold", "frontend.xcaf.state")))

	result, blueprint, err := FindMostRecentState(filepath.Join(dir, ".xcaffold"))
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "backend", blueprint)
	assert.Equal(t, "backend", result.Blueprint)
}

func TestFindMostRecentState_IdenticalTimestamps_LexTiebreak(t *testing.T) {
	dir := t.TempDir()
	ts := "2026-05-19T10:00:00Z"

	// backend (should win on lexicographic order)
	backend := &StateManifest{
		Version: 1, XcaffoldVersion: "1.0.0", Blueprint: "backend",
		Targets: map[string]TargetState{
			"claude": {LastApplied: ts, Artifacts: []Artifact{{Path: "b.md", Hash: "sha256:b"}}},
		},
	}
	require.NoError(t, WriteState(backend, filepath.Join(dir, ".xcaffold", "backend.xcaf.state")))

	// frontend (same timestamp)
	frontend := &StateManifest{
		Version: 1, XcaffoldVersion: "1.0.0", Blueprint: "frontend",
		Targets: map[string]TargetState{
			"claude": {LastApplied: ts, Artifacts: []Artifact{{Path: "f.md", Hash: "sha256:f"}}},
		},
	}
	require.NoError(t, WriteState(frontend, filepath.Join(dir, ".xcaffold", "frontend.xcaf.state")))

	result, blueprint, err := FindMostRecentState(filepath.Join(dir, ".xcaffold"))
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "backend", blueprint)
}

func TestFindMostRecentState_UnreadableFile_Skipped(t *testing.T) {
	dir := t.TempDir()
	stateDir := filepath.Join(dir, ".xcaffold")
	require.NoError(t, os.MkdirAll(stateDir, 0755))

	// Write one valid state file
	valid := &StateManifest{
		Version: 1, XcaffoldVersion: "1.0.0", Blueprint: "backend",
		Targets: map[string]TargetState{
			"claude": {LastApplied: "2026-05-19T10:00:00Z", Artifacts: []Artifact{{Path: "b.md", Hash: "sha256:b"}}},
		},
	}
	require.NoError(t, WriteState(valid, filepath.Join(stateDir, "backend.xcaf.state")))

	// Write one corrupt file
	require.NoError(t, os.WriteFile(filepath.Join(stateDir, "corrupt.xcaf.state"), []byte("{{invalid yaml"), 0600))

	result, blueprint, err := FindMostRecentState(stateDir)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "backend", blueprint)
}

func TestOutputDir_Serialization(t *testing.T) {
	ts := TargetState{
		LastApplied: "2026-05-20T00:00:00Z",
		OutputDir:   ".worktrees/backend/",
		Artifacts:   []Artifact{{Path: "agents/foo.md", Hash: "sha256:abc"}},
	}
	data, err := yaml.Marshal(ts)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(data), "output-dir: .worktrees/backend/") {
		t.Errorf("expected output-dir in YAML, got:\n%s", data)
	}

	var roundTrip TargetState
	if err := yaml.Unmarshal(data, &roundTrip); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if roundTrip.OutputDir != ".worktrees/backend/" {
		t.Errorf("round-trip OutputDir = %q, want %q", roundTrip.OutputDir, ".worktrees/backend/")
	}
}

func TestOutputDir_BackwardCompat(t *testing.T) {
	oldYAML := `last-applied: "2026-01-01T00:00:00Z"
artifacts:
  - path: agents/foo.md
    hash: "sha256:abc"
`
	var ts TargetState
	if err := yaml.Unmarshal([]byte(oldYAML), &ts); err != nil {
		t.Fatalf("unmarshal old state: %v", err)
	}
	if ts.OutputDir != "" {
		t.Errorf("OutputDir should be empty for old state files, got %q", ts.OutputDir)
	}
}

func TestGlobalStateDir_ReturnsCorrectPath(t *testing.T) {
	t.Setenv("XCAFFOLD_HOME", "")
	// This will use the real home directory, so just verify the suffix
	globalStateDir, err := GlobalStateDir()
	require.NoError(t, err)
	assert.True(t, strings.HasSuffix(globalStateDir, "/.xcaffold/state"),
		"GlobalStateDir should end with /.xcaffold/state, got %s", globalStateDir)
}

func TestGlobalStateDir_RespectsXCAFFOLD_HOME(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("XCAFFOLD_HOME", tmpHome)

	globalStateDir, err := GlobalStateDir()
	require.NoError(t, err)

	expectedPath := filepath.Join(tmpHome, "state")
	assert.Equal(t, expectedPath, globalStateDir)
}

func TestGlobalStateDir_CreatesDirectoryIfNeeded(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("XCAFFOLD_HOME", tmpHome)

	globalStateDir, err := GlobalStateDir()
	require.NoError(t, err)

	// Directory shouldn't exist yet
	_, err = os.Stat(globalStateDir)
	assert.True(t, os.IsNotExist(err), "GlobalStateDir should not create the directory itself")

	// But WriteState should be able to create it
	manifest := &StateManifest{
		Version: 1,
		Targets: map[string]TargetState{},
	}
	statePath := filepath.Join(globalStateDir, "test.xcaf.state")
	err = WriteState(manifest, statePath)
	require.NoError(t, err)

	// Now it should exist
	_, err = os.Stat(globalStateDir)
	assert.NoError(t, err, "globalStateDir should exist after WriteState")
}
