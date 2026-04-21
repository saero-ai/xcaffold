package parser

import (
	"path/filepath"
	"testing"
)

// TestExampleFiles verifies that every example .xcf file under
// docs/reference/examples/ parses successfully as a single-resource document.
func TestExampleFiles(t *testing.T) {
	glob := "../../docs/reference/examples/*.xcf"
	files, err := filepath.Glob(glob)
	if err != nil {
		t.Fatalf("glob failed: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("no .xcf files found under docs/reference/examples/")
	}
	for _, f := range files {
		t.Run(filepath.Base(f), func(t *testing.T) {
			_, err := ParseFile(f)
			if err != nil {
				t.Errorf("ParseFile(%q) error: %v", f, err)
			}
		})
	}
}
