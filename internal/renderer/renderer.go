// Package renderer defines the TargetRenderer interface implemented by each
// output target (e.g. claude, cursor, antigravity).
package renderer

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/output"
)

// ResolveInstructionsContent returns inline instructions or reads InstructionsFile
// relative to baseDir. Returns an empty string on any read error or when both
// are empty. This is the shared low-level helper used by all renderers; it
// intentionally swallows file read errors (missing files are treated as empty).
func ResolveInstructionsContent(inline, file, baseDir string) string {
	if inline != "" {
		return inline
	}
	if file == "" {
		return ""
	}
	data, err := os.ReadFile(filepath.Join(baseDir, file))
	if err != nil {
		return ""
	}
	content := string(data)
	if strings.HasSuffix(file, ".xcf") {
		content = ExtractXCFInstructions(data)
	}
	return content
}

// ExtractXCFInstructions removes YAML frontmatter delimited by "---" from the start
// of a byte slice, returning only the body content with leading whitespace trimmed.
func ExtractXCFInstructions(data []byte) string {
	b := data
	if !bytes.HasPrefix(b, []byte("---\n")) && !bytes.HasPrefix(b, []byte("---\r\n")) {
		return string(b)
	}

	// Find the end of the frontmatter block
	endIdx := bytes.Index(b[4:], []byte("\n---"))
	if endIdx == -1 {
		return string(b)
	}

	startOfContent := 4 + endIdx + 4
	return string(bytes.TrimSpace(b[startOfContent:]))
}

// ResolveScopeContent returns the effective content for an InstructionsScope,
// preferring a provider-specific variant when one is declared under
// scope.Variants[provider]. Falls back to the scope's own Instructions /
// InstructionsFile pair.
func ResolveScopeContent(scope ast.InstructionsScope, provider, baseDir string) string {
	if v, ok := scope.Variants[provider]; ok {
		return ResolveInstructionsContent("", v.InstructionsFile, baseDir)
	}
	return ResolveInstructionsContent(scope.Instructions, scope.InstructionsFile, baseDir)
}

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
