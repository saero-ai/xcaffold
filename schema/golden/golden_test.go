package golden_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/saero-ai/xcaffold/internal/parser"
	"github.com/saero-ai/xcaffold/pkg/schema"
)

func TestGoldenManifests_AllParse(t *testing.T) {
	goldenDir := "."
	if _, err := os.Stat(filepath.Join(goldenDir, "agent.xcf")); err != nil {
		// Running from repo root — adjust path.
		goldenDir = "schema/golden"
	}

	entries, err := os.ReadDir(goldenDir)
	if err != nil {
		t.Fatalf("failed to read golden manifest directory: %v", err)
	}

	// Kinds whose parser support is not yet implemented or has been removed.
	unparseable := map[string]bool{
		"template.xcf": true,
		"system.xcf":   true,
		// memory.xcf: kind:memory is no longer a parsed resource kind.
		// Memory is convention-based (.md files in xcf/agents/<id>/memory/).
		"memory.xcf": true,
	}

	xcfCount := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".xcf") {
			continue
		}
		if unparseable[entry.Name()] {
			t.Logf("SKIP (parser support pending): %s", entry.Name())
			continue
		}
		xcfCount++
		t.Run(entry.Name(), func(t *testing.T) {
			path := filepath.Join(goldenDir, entry.Name())
			f, err := os.Open(path)
			if err != nil {
				t.Fatalf("failed to open %s: %v", path, err)
			}
			defer f.Close()

			_, parseErr := parser.Parse(f)
			if parseErr != nil {
				t.Errorf("golden manifest %s failed to parse: %v", entry.Name(), parseErr)
			}
		})
	}

	if xcfCount == 0 {
		t.Fatal("no .xcf golden manifests found — test is misconfigured")
	}
	t.Logf("validated %d golden manifests", xcfCount)
}

func TestGoldenManifests_Completeness(t *testing.T) {
	goldenDir := "."
	if _, err := os.Stat(filepath.Join(goldenDir, "agent.xcf")); err != nil {
		goldenDir = "schema/golden"
	}

	// Kinds in the registry that have no standalone golden file.
	skip := map[string]bool{
		"memory": true,
	}

	for _, kindName := range schema.KindNames() {
		if skip[kindName] {
			continue
		}
		t.Run(kindName, func(t *testing.T) {
			path := filepath.Join(goldenDir, kindName+".xcf")
			content, err := os.ReadFile(path)
			if err != nil {
				t.Skipf("no golden manifest for kind %q: %v", kindName, err)
				return
			}

			ks, ok := schema.LookupKind(kindName)
			if !ok {
				t.Fatalf("kind %q in KindNames() but not in Registry", kindName)
			}

			text := string(content)
			for _, f := range ks.Fields {
				if !strings.Contains(text, f.YAMLKey+":") {
					t.Errorf("golden manifest for %q missing field %q", kindName, f.YAMLKey)
				}
			}
		})
	}
}
