// Package renderer defines the TargetRenderer interface implemented by each
// output target (e.g. claude, cursor, antigravity).
package renderer

import (
	"github.com/saero-ai/xcaffold/internal/output"
)

// TargetRenderer renders a compiled file map for a specific target environment.
type TargetRenderer interface {
	// Target returns the canonical name of this renderer (e.g. "claude").
	Target() string

	// OutputDir returns the base output directory for this target
	// (e.g. ".claude", ".cursor/rules").
	OutputDir() string

	// Render takes a map of relative path → content and returns a compiler
	// Output containing the files as they should be written to disk.
	Render(files map[string]string) *output.Output
}
