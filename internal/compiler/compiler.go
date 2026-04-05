package compiler

import (
	"fmt"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/output"
	"github.com/saero-ai/xcaffold/internal/renderer/claude"
	"github.com/saero-ai/xcaffold/internal/renderer/cursor"
	"github.com/saero-ai/xcaffold/internal/renderer/gemini"
)

// Output is an alias for output.Output, preserved for backward compatibility.
// All callers that reference compiler.Output continue to work without changes.
type Output = output.Output

// Compile translates an XcaffoldConfig AST into platform-native files.
// target selects the output platform: "claude" (default), "cursor", "gemini".
// If target is empty, defaults to "claude" for backward compatibility.
func Compile(config *ast.XcaffoldConfig, baseDir string, target string) (*Output, error) {
	if target == "" {
		target = "claude"
	}

	switch target {
	case "claude":
		r := claude.New()
		return r.Compile(config, baseDir)
	case "cursor":
		r := cursor.New()
		return r.Compile(config, baseDir)
	case "gemini":
		r := gemini.New()
		return r.Compile(config, baseDir)
	default:
		return nil, fmt.Errorf("unsupported target %q: supported targets are \"claude\", \"cursor\", \"gemini\"", target)
	}
}

// OutputDir returns the target-specific root directory for compilation outputs
// (e.g. .claude, .cursor, .agents).
func OutputDir(target string) string {
	if target == "" {
		target = "claude"
	}
	switch target {
	case "claude":
		return claude.New().OutputDir()
	case "cursor":
		return cursor.New().OutputDir()
	case "gemini":
		return gemini.New().OutputDir()
	default:
		return ".claude"
	}
}
