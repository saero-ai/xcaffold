package importer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
)

func TestBaseImporter_Provider(t *testing.T) {
	bi := &BaseImporter{ProviderName: "test-provider"}
	if bi.Provider() != "test-provider" {
		t.Errorf("Provider() = %q, want test-provider", bi.Provider())
	}
}

func TestBaseImporter_InputDir(t *testing.T) {
	bi := &BaseImporter{Dir: ".test/"}
	if bi.InputDir() != ".test/" {
		t.Errorf("InputDir() = %q, want .test/", bi.InputDir())
	}
}

func TestBaseImporter_GetWarnings(t *testing.T) {
	bi := &BaseImporter{Warnings: []string{"w1", "w2"}}
	w := bi.GetWarnings()
	if len(w) != 2 || w[0] != "w1" || w[1] != "w2" {
		t.Errorf("GetWarnings() = %v, want [w1 w2]", w)
	}
}

func TestBaseImporter_AppendWarning(t *testing.T) {
	bi := &BaseImporter{}
	bi.AppendWarning("first")
	bi.AppendWarning("second")
	if len(bi.Warnings) != 2 || bi.Warnings[0] != "first" || bi.Warnings[1] != "second" {
		t.Errorf("AppendWarning() produced %v, want [first second]", bi.Warnings)
	}
}

// mockImporter is a minimal implementation of ProviderImporter for testing.
type mockImporter struct {
	BaseImporter
	classifyCalls []string
	extractCalls  []string
	extractErr    error
}

func (m *mockImporter) Classify(rel string, isDir bool) (Kind, Layout) {
	m.classifyCalls = append(m.classifyCalls, rel)
	if rel == "agents/test.md" {
		return KindAgent, FlatFile
	}
	return KindUnknown, LayoutUnknown
}

func (m *mockImporter) Extract(rel string, data []byte, config *ast.XcaffoldConfig) error {
	m.extractCalls = append(m.extractCalls, rel)
	return m.extractErr
}

func (m *mockImporter) Import(dir string, config *ast.XcaffoldConfig) error {
	return RunImport(m, dir, config)
}

func TestRunImport_CallsClassifyAndExtract(t *testing.T) {
	// Create a temporary directory structure
	tmpDir := t.TempDir()
	inputDir := filepath.Join(tmpDir, ".test")
	if err := os.Mkdir(inputDir, 0755); err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	// Create a test agent file
	agentFile := filepath.Join(inputDir, "agents", "test.md")
	if err := os.MkdirAll(filepath.Dir(agentFile), 0755); err != nil {
		t.Fatalf("failed to create agents dir: %v", err)
	}
	if err := os.WriteFile(agentFile, []byte("---\nname: test\n---\ntest body"), 0644); err != nil {
		t.Fatalf("failed to write agent file: %v", err)
	}

	// Set up mock importer
	mock := &mockImporter{
		BaseImporter: BaseImporter{
			ProviderName: "test",
			Dir:          ".test",
		},
	}

	config := &ast.XcaffoldConfig{}
	if err := RunImport(mock, tmpDir, config); err != nil {
		t.Fatalf("RunImport failed: %v", err)
	}

	// Verify that Classify was called
	if len(mock.classifyCalls) == 0 {
		t.Errorf("Classify was not called")
	}

	// Verify that Extract was called for the recognized kind
	if len(mock.extractCalls) == 0 {
		t.Errorf("Extract was not called for recognized kind")
	}
	if mock.extractCalls[0] != "agents/test.md" {
		t.Errorf("Extract called with %q, want agents/test.md", mock.extractCalls[0])
	}
}

func TestRunImport_HandlesExtractionErrors(t *testing.T) {
	tmpDir := t.TempDir()
	inputDir := filepath.Join(tmpDir, ".test")
	if err := os.Mkdir(inputDir, 0755); err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	agentFile := filepath.Join(inputDir, "agents", "test.md")
	if err := os.MkdirAll(filepath.Dir(agentFile), 0755); err != nil {
		t.Fatalf("failed to create agents dir: %v", err)
	}
	if err := os.WriteFile(agentFile, []byte("invalid data"), 0644); err != nil {
		t.Fatalf("failed to write agent file: %v", err)
	}

	mock := &mockImporter{
		BaseImporter: BaseImporter{
			ProviderName: "test",
			Dir:          ".test",
		},
		extractErr: NewExtractionError("test error"),
	}

	config := &ast.XcaffoldConfig{}
	err := RunImport(mock, tmpDir, config)
	if err != nil {
		t.Fatalf("RunImport failed: %v", err)
	}

	// Verify that error was recorded as a warning
	if len(mock.Warnings) == 0 {
		t.Errorf("extraction error was not recorded as a warning")
	}

	// Verify that file data was stored in ProviderExtras
	if config.ProviderExtras == nil {
		t.Errorf("ProviderExtras was not initialized")
	}
	if config.ProviderExtras["test"] == nil {
		t.Errorf("provider extras for 'test' was not created")
	}
	if config.ProviderExtras["test"]["agents/test.md"] == nil {
		t.Errorf("file data was not stored in ProviderExtras")
	}
}

// NewExtractionError is a helper for testing.
func NewExtractionError(msg string) error {
	return &ExtractionError{Message: msg}
}

type ExtractionError struct {
	Message string
}

func (e *ExtractionError) Error() string {
	return e.Message
}
