package cursor

import (
	"fmt"
	"sort"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/output"
	"github.com/saero-ai/xcaffold/internal/renderer"
)

// MemoryRenderer is a no-op renderer for Cursor. Cursor has no native per-file
// memory primitive; this renderer emits one FidelityNote per declared memory
// entry telling the operator to use .cursorrules or an always-on rule.
type MemoryRenderer struct{}

// NewMemoryRenderer returns a new MemoryRenderer for the cursor target.
func NewMemoryRenderer() *MemoryRenderer {
	return &MemoryRenderer{}
}

// Compile emits no files and produces one FidelityNote per memory entry
// advising the operator to use .cursorrules.
func (r *MemoryRenderer) Compile(config *ast.XcaffoldConfig, _ string) (*output.Output, []renderer.FidelityNote, error) {
	out := &output.Output{Files: make(map[string]string)}
	if len(config.Memory) == 0 {
		return out, nil, nil
	}

	names := make([]string, 0, len(config.Memory))
	for k := range config.Memory {
		names = append(names, k)
	}
	sort.Strings(names)

	notes := make([]renderer.FidelityNote, 0, len(names))
	for _, name := range names {
		notes = append(notes, renderer.NewNote(
			renderer.LevelInfo,
			targetName,
			"memory",
			name,
			"",
			renderer.CodeMemoryNoNativeTarget,
			"Cursor has no native memory primitive; add this context to `.cursorrules` or a rule with activation: always.",
			fmt.Sprintf("Add the content of memory entry %q to .cursorrules, or declare a rule with `activation: always` in .xcaf.", name),
		))
	}
	return out, notes, nil
}
