package compiler

import (
	"fmt"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/output"
	"github.com/saero-ai/xcaffold/internal/renderer/agentsmd"
	"github.com/saero-ai/xcaffold/internal/renderer/antigravity"
	"github.com/saero-ai/xcaffold/internal/renderer/claude"
	"github.com/saero-ai/xcaffold/internal/renderer/cursor"
)

const (
	TargetClaude      = "claude"
	TargetCursor      = "cursor"
	TargetAntigravity = "antigravity"
	TargetAgentsMD    = "agentsmd"
)

// Output is an alias for output.Output, preserved for backward compatibility.
// All callers that reference compiler.Output continue to work without changes.
type Output = output.Output

// Compile translates an XcaffoldConfig AST into platform-native files.
// target selects the output platform: "claude" (default), "cursor", "antigravity".
// If target is empty, defaults to "claude" for backward compatibility.
//
// Before compilation, any resources marked as Inherited (e.g. from extends: global)
// are stripped. This ensures global configurations are not physically duplicated
// into local project directories, keeping the compiler's output strictly localized
// while allowing parser-level ASTs to retain full scope awareness (for e.g. graph).
func Compile(config *ast.XcaffoldConfig, baseDir string, target string) (*Output, error) {
	if target == "" {
		target = TargetClaude
	}

	config.StripInherited()

	switch target {
	case TargetClaude:
		r := claude.New()
		return r.Compile(config, baseDir)
	case TargetCursor:
		r := cursor.New()
		return r.Compile(config, baseDir)
	case TargetAntigravity:
		r := antigravity.New()
		return r.Compile(config, baseDir)
	case TargetAgentsMD:
		r := agentsmd.New()
		return r.Compile(config, baseDir)
	default:
		return nil, fmt.Errorf("unsupported target %q: supported targets are \"claude\", \"cursor\", \"antigravity\", \"agentsmd\"", target)
	}
}

// OutputDir returns the target-specific root directory for compilation outputs
// (e.g. .claude, .cursor, .agents).
func OutputDir(target string) string {
	if target == "" {
		target = TargetClaude
	}
	switch target {
	case TargetClaude:
		return claude.New().OutputDir()
	case TargetCursor:
		return cursor.New().OutputDir()
	case TargetAntigravity:
		return antigravity.New().OutputDir()
	case TargetAgentsMD:
		return agentsmd.New().OutputDir()
	default:
		return ".claude"
	}
}
