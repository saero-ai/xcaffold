package state

import (
	"testing"

	"github.com/saero-ai/xcaffold/internal/compiler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
