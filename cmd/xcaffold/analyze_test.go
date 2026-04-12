package main

import (
	"os"
	"testing"

	"github.com/spf13/cobra"
)

// TestAnalyzeCmd_DoesNotRequireXcfFile verifies that the analyze command is
// excluded from resolveProjectConfig so it can run on a fresh project that has
// no scaffold.xcf yet — its primary purpose is to CREATE that file.
func TestAnalyzeCmd_DoesNotRequireXcfFile(t *testing.T) {
	// Change to a temp dir that has no scaffold.xcf.
	tmp := t.TempDir()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	defer func() { _ = os.Chdir(orig) }()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("failed to chdir to tmp: %v", err)
	}

	// Reset the global configFlag so resolveProjectConfig uses CWD walk-up.
	configFlag = ""

	cmd := &cobra.Command{Use: "analyze"}
	if err := resolveProjectConfig(cmd); err != nil {
		t.Errorf("resolveProjectConfig returned error for 'analyze' in a dir with no scaffold.xcf: %v", err)
	}
}
