package importer_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/importer"
	"github.com/stretchr/testify/assert"
)

// mockImporter implements ProviderImporter for testing DetectProviders.
type mockImporter struct {
	provider string
	inputDir string
}

func (m *mockImporter) Provider() string { return m.provider }
func (m *mockImporter) InputDir() string { return m.inputDir }
func (m *mockImporter) Classify(rel string, isDir bool) (importer.Kind, importer.Layout) {
	return importer.KindUnknown, importer.LayoutUnknown
}
func (m *mockImporter) Extract(rel string, data []byte, config *ast.XcaffoldConfig) error {
	return nil
}
func (m *mockImporter) Import(dir string, config *ast.XcaffoldConfig) error { return nil }

func TestDetectProviders_FindsExistingDirs(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, ".claude"), 0755)
	os.MkdirAll(filepath.Join(root, ".cursor"), 0755)

	all := []importer.ProviderImporter{
		&mockImporter{provider: "claude", inputDir: ".claude"},
		&mockImporter{provider: "cursor", inputDir: ".cursor"},
		&mockImporter{provider: "gemini", inputDir: ".gemini"},
	}

	found := importer.DetectProviders(root, all)
	assert.Len(t, found, 2)
	assert.Equal(t, "claude", found[0].Provider())
	assert.Equal(t, "cursor", found[1].Provider())
}

func TestDetectProviders_EmptyWhenNoDirs(t *testing.T) {
	root := t.TempDir()
	all := []importer.ProviderImporter{
		&mockImporter{provider: "claude", inputDir: ".claude"},
	}
	found := importer.DetectProviders(root, all)
	assert.Len(t, found, 0)
}

func TestDefaultImporters_ReturnsSlice(t *testing.T) {
	importers := importer.DefaultImporters()
	assert.NotNil(t, importers)
}
