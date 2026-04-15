package state

import (
	"testing"

	"github.com/saero-ai/xcaffold/internal/compiler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestGenerate_ProducesCorrectArtifacts(t *testing.T) {
	out := &compiler.Output{
		Files: map[string]string{
			"agents/developer.md": "---\ndescription: dev\n---\n",
		},
	}

	manifest := Generate(out)
	require.NotNil(t, manifest)

	assert.Equal(t, lockFileVersion, manifest.Version)
	assert.Equal(t, XcaffoldVersion, manifest.XcaffoldVersion)
	assert.Len(t, manifest.Artifacts, 1)
	assert.Equal(t, "agents/developer.md", manifest.Artifacts[0].Path)
	assert.True(t, len(manifest.Artifacts[0].Hash) > 0)
	assert.Contains(t, manifest.Artifacts[0].Hash, "sha256:")
}

func TestGenerate_EmptyOutput_ProducesEmptyArtifacts(t *testing.T) {
	out := &compiler.Output{Files: map[string]string{}}
	manifest := Generate(out)
	assert.Empty(t, manifest.Artifacts)
}

func TestWriteAndRead_RoundTrip(t *testing.T) {
	out := &compiler.Output{
		Files: map[string]string{
			"agents/backend.md": "---\ndescription: backend\n---\n",
		},
	}
	manifest := Generate(out)

	tmpFile := t.TempDir() + "/scaffold.lock"

	err := Write(manifest, tmpFile)
	require.NoError(t, err)

	recovered, err := Read(tmpFile)
	require.NoError(t, err)

	assert.Equal(t, manifest.Version, recovered.Version)
	assert.Equal(t, manifest.XcaffoldVersion, recovered.XcaffoldVersion)
	assert.Len(t, recovered.Artifacts, 1)
	assert.Equal(t, manifest.Artifacts[0].Hash, recovered.Artifacts[0].Hash)
}

func TestRead_NonExistentFile_ReturnsError(t *testing.T) {
	_, err := Read("/tmp/this-file-does-not-exist-xcaffold.lock")
	require.Error(t, err)
}

func TestLockManifest_MemorySeeds_Serialization(t *testing.T) {
	manifest := LockManifest{
		LastApplied:     "2026-04-15T16:00:00Z",
		XcaffoldVersion: "1.3.0",
		Target:          "claude",
		Version:         2,
		MemorySeeds: []MemorySeed{
			{
				Name:      "user-role",
				Target:    "claude",
				Path:      "~/.claude/projects/test/memory/user-role.md",
				Hash:      "sha256:abc123",
				SeededAt:  "2026-04-15T16:00:00Z",
				Lifecycle: "seed-once",
			},
		},
	}

	data, err := yaml.Marshal(manifest)
	require.NoError(t, err)
	content := string(data)

	require.Contains(t, content, "memory_seeds:")
	require.Contains(t, content, "name: user-role")
	require.Contains(t, content, "lifecycle: seed-once")
	require.Contains(t, content, "sha256:abc123")
}

func TestLockManifest_MemorySeeds_OmitEmptyWhenAbsent(t *testing.T) {
	manifest := LockManifest{
		LastApplied: "2026-04-15T16:00:00Z",
		Version:     2,
	}
	data, err := yaml.Marshal(manifest)
	require.NoError(t, err)
	require.NotContains(t, string(data), "memory_seeds")
}

func TestMemorySeed_SortedByName(t *testing.T) {
	seeds := []MemorySeed{
		{Name: "z-entry", Target: "claude", Lifecycle: "seed-once"},
		{Name: "a-entry", Target: "claude", Lifecycle: "tracked"},
		{Name: "m-entry", Target: "claude", Lifecycle: "seed-once"},
	}
	sortMemorySeeds(seeds)

	require.Equal(t, "a-entry", seeds[0].Name)
	require.Equal(t, "m-entry", seeds[1].Name)
	require.Equal(t, "z-entry", seeds[2].Name)
}

func TestGenerateWithOpts_MemorySeeds_CopiedAndSorted(t *testing.T) {
	seeds := []MemorySeed{
		{Name: "z-seed", Target: "claude", Lifecycle: "tracked", Hash: "sha256:z", SeededAt: "2026-04-15T00:00:00Z"},
		{Name: "a-seed", Target: "claude", Lifecycle: "seed-once", Hash: "sha256:a", SeededAt: "2026-04-15T00:00:00Z"},
	}

	out := &compiler.Output{Files: map[string]string{"x.md": "x"}}
	manifest := GenerateWithOpts(out, GenerateOpts{MemorySeeds: seeds})

	require.Len(t, manifest.MemorySeeds, 2)
	require.Equal(t, "a-seed", manifest.MemorySeeds[0].Name)
	require.Equal(t, "z-seed", manifest.MemorySeeds[1].Name)

	// Defensive copy: mutating source must not affect manifest
	seeds[0].Name = "mutated"
	require.Equal(t, "z-seed", manifest.MemorySeeds[1].Name)
}

func TestGenerateWithOpts_MemorySeeds_AutoPopulatesSeededAt(t *testing.T) {
	seeds := []MemorySeed{
		{Name: "a-seed", Target: "claude", Lifecycle: "seed-once", Hash: "sha256:a"}, // no SeededAt
	}

	out := &compiler.Output{Files: map[string]string{"x.md": "x"}}
	manifest := GenerateWithOpts(out, GenerateOpts{MemorySeeds: seeds})

	require.Len(t, manifest.MemorySeeds, 1)
	require.NotEmpty(t, manifest.MemorySeeds[0].SeededAt, "GenerateWithOpts must fill empty SeededAt")

	// Caller's slice must still have the empty SeededAt (copy is defensive)
	require.Empty(t, seeds[0].SeededAt)
}

func TestLockManifest_BackwardCompat_NoMemorySeeds(t *testing.T) {
	raw := `
last_applied: "2026-04-01T10:00:00Z"
xcaffold_version: 1.2.0
target: claude
version: 2
`
	var manifest LockManifest
	require.NoError(t, yaml.Unmarshal([]byte(raw), &manifest))
	require.Nil(t, manifest.MemorySeeds)
}
