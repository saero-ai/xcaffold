// Package renderer defines the TargetRenderer interface implemented by each
// output target (e.g. claude, cursor, antigravity).
package renderer

import (
	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/output"
)

// TargetRenderer renders a compiled file map for a specific target environment.
type TargetRenderer interface {
	// Target returns the canonical name of this renderer (e.g. "claude").
	Target() string

	// OutputDir returns the base output directory for this target
	// (e.g. ".claude", ".cursor/rules").
	OutputDir() string

	// Compile translates an XcaffoldConfig into a compiler Output and a slice
	// of fidelity notes describing any information loss or transformation
	// that occurred. Compile is the semantic entry point consumed by the
	// top-level compiler. A non-empty notes slice does not indicate failure;
	// callers decide whether to promote notes to errors based on the
	// --fidelity mode.
	Compile(config *ast.XcaffoldConfig, baseDir string) (*output.Output, []FidelityNote, error)

	// Render wraps a file map in an output.Output. It is retained for
	// backward compatibility with callers that have already assembled a
	// file map and need the renderer's Output envelope. The canonical
	// compilation entry point is Compile; Render is a thin passthrough.
	Render(files map[string]string) *output.Output
}
