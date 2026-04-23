package copilot_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/parser"
	"github.com/saero-ai/xcaffold/internal/renderer"
	"github.com/saero-ai/xcaffold/internal/renderer/copilot"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCopilotRenderer_ProjectInstructions_FlatSingleton(t *testing.T) {
	r := copilot.New()
	config := &ast.XcaffoldConfig{
		Project: &ast.ProjectConfig{
			Name:         "test-project",
			Instructions: "Root content.",
			InstructionsScopes: []ast.InstructionsScope{
				{Path: "packages/worker", Instructions: "Worker content.", MergeStrategy: "concat"},
			},
		},
	}
	out, _, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)
	root := out.Files["copilot-instructions.md"]
	require.NotEmpty(t, root)
	require.Contains(t, root, "Root content.")
	require.Contains(t, root, `<!-- xcaffold:scope path="packages/worker"`)
	require.Contains(t, root, "Worker content.")
	require.Contains(t, root, `<!-- xcaffold:/scope -->`)
}

func TestCopilotRenderer_ProjectInstructions_FidelityNote(t *testing.T) {
	r := copilot.New()
	config := &ast.XcaffoldConfig{
		Project: &ast.ProjectConfig{
			Name:         "test",
			Instructions: "Root.",
			InstructionsScopes: []ast.InstructionsScope{
				{Path: "packages/worker", Instructions: "Worker.", MergeStrategy: "concat"},
			},
		},
	}
	_, notes, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)
	require.NotEmpty(t, notes)
	var found *renderer.FidelityNote
	for i := range notes {
		if notes[i].Code == "INSTRUCTIONS_FLATTENED" {
			found = &notes[i]
			break
		}
	}
	require.NotNil(t, found, "expected INSTRUCTIONS_FLATTENED note")
	require.Equal(t, renderer.LevelInfo, found.Level)
}

func TestCopilotRenderer_ProjectInstructions_NoNestedOutput(t *testing.T) {
	// Copilot flat-mode renderer must NEVER emit nested AGENTS.md.
	r := copilot.New()
	config := &ast.XcaffoldConfig{
		Project: &ast.ProjectConfig{
			Name:         "test",
			Instructions: "Root.",
			InstructionsScopes: []ast.InstructionsScope{
				{Path: "packages/worker", Instructions: "Worker.", MergeStrategy: "concat"},
			},
		},
	}
	out, _, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)
	for path := range out.Files {
		require.NotEqual(t, "AGENTS.md", path, "copilot renderer must not emit AGENTS.md")
		require.False(t, strings.HasSuffix(path, "/AGENTS.md"),
			"copilot renderer must not emit nested AGENTS.md; got %s", path)
	}
}

// TestCopilotRenderer_FlatMode_ExplicitFlag verifies that setting
// target-options.copilot.instructions-mode: flat produces flat singleton output
// (identical to the default).
func TestCopilotRenderer_FlatMode_ExplicitFlag(t *testing.T) {
	r := copilot.New()
	config := &ast.XcaffoldConfig{
		Project: &ast.ProjectConfig{
			Name:         "test",
			Instructions: "Root.",
			TargetOptions: map[string]ast.TargetOverride{
				"copilot": {InstructionsMode: "flat"},
			},
			InstructionsScopes: []ast.InstructionsScope{
				{Path: "packages/worker", Instructions: "Worker.", MergeStrategy: "flat"},
			},
		},
	}
	out, _, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)
	_, hasCopilotInstructions := out.Files["copilot-instructions.md"]
	require.True(t, hasCopilotInstructions, "flat mode must emit .github/copilot-instructions.md")
	for path := range out.Files {
		require.False(t, strings.HasSuffix(path, "/AGENTS.md"),
			"flat mode must not emit AGENTS.md; got %s", path)
	}
}

// TestCopilotRenderer_NestedMode_EmitsNestedDirs verifies that setting
// target-options.copilot.instructions-mode: nested produces per-directory AGENTS.md
// files instead of the flat singleton.
func TestCopilotRenderer_NestedMode_EmitsNestedDirs(t *testing.T) {
	r := copilot.New()
	config := &ast.XcaffoldConfig{
		Project: &ast.ProjectConfig{
			Name:         "test",
			Instructions: "Root.",
			TargetOptions: map[string]ast.TargetOverride{
				"copilot": {InstructionsMode: "nested"},
			},
			InstructionsScopes: []ast.InstructionsScope{
				{Path: "packages/worker", Instructions: "Worker.", MergeStrategy: "closest-wins"},
			},
		},
	}
	out, _, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)
	// Nested mode must NOT emit flat singleton.
	_, hasFlatFile := out.Files["copilot-instructions.md"]
	require.False(t, hasFlatFile, "nested mode must not emit .github/copilot-instructions.md")
	// Nested mode must emit root AGENTS.md and per-scope AGENTS.md.
	_, hasRoot := out.Files["AGENTS.md"]
	require.True(t, hasRoot, "nested mode must emit root AGENTS.md")
	_, hasScope := out.Files["packages/worker/AGENTS.md"]
	require.True(t, hasScope, "nested mode must emit packages/worker/AGENTS.md")
}

// TestCopilotParser_InstructionsModeInvalidValue verifies that an unknown
// instructions-mode value on a project.target-options.copilot block is a parse error.
func TestCopilotParser_InstructionsModeInvalidValue(t *testing.T) {
	yml := `
kind: project
version: "1.0"
name: test
instructions: "Root."
target-options:
  copilot:
    instructions-mode: sideways
`
	tmp := t.TempDir()
	path := filepath.Join(tmp, "project.xcf")
	require.NoError(t, os.WriteFile(path, []byte(yml), 0o600))
	_, err := parser.ParseFile(path)
	require.Error(t, err)
	require.Contains(t, err.Error(), "instructions-mode")
}

func TestCopilotRenderer_ProjectInstructions_XCFExtractsBody(t *testing.T) {
	dir := t.TempDir()
	xcfContent := `kind: project
instructions: |
  This is Copilot root extracted from XCF.`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "root.xcf"), []byte(xcfContent), 0600))

	r := copilot.New()
	config := &ast.XcaffoldConfig{
		Project: &ast.ProjectConfig{
			Name:             "test-project",
			InstructionsFile: "root.xcf",
		},
	}

	out, _, err := renderer.Orchestrate(r, config, dir)
	require.NoError(t, err)

	rootMD, ok := out.Files["copilot-instructions.md"]
	require.True(t, ok, "copilot-instructions.md must be present in output")
	require.Contains(t, rootMD, "This is Copilot root extracted from XCF.", "Must extract body")
	require.NotContains(t, rootMD, "kind: project", "Must strip XCF frontmatter")
}

func TestRenderProjectInstructions_Copilot_ScopeFiles_WithApplyTo(t *testing.T) {
	r := copilot.New()
	config := &ast.XcaffoldConfig{
		Project: &ast.ProjectConfig{
			Name:         "test-project",
			Instructions: "Root",
			InstructionsScopes: []ast.InstructionsScope{
				{
					Path:         "dummy_platform",
					Instructions: "Dummy scope content.",
				},
			},
		},
	}

	out, _, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)

	scopeFile, ok := out.Files["instructions/dummy_platform.instructions.md"]
	require.True(t, ok, "instructions/dummy_platform.instructions.md must be present")
	require.Contains(t, scopeFile, "applyTo: \"dummy_platform/**\"", "Must contain applyTo frontmatter")
	require.Contains(t, scopeFile, "Dummy scope content.", "Must contain extracted content")
}

// TestCompileProjectInstructions_Copilot_ClaudeDirPresent_SkipsRoot_WritesNestedScopes
// verifies that when .claude/ is present the root copilot-instructions.md is NOT
// written (root CLAUDE.md auto-loads), but nested scope instruction files ARE still
// written because Copilot does not auto-load subdirectory CLAUDE.md files.
func TestCompileProjectInstructions_Copilot_ClaudeDirPresent_SkipsRoot_WritesNestedScopes(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".claude"), 0o755))

	r := copilot.New()
	project := &ast.ProjectConfig{
		Name:         "test-project",
		Instructions: "Root content.",
		InstructionsScopes: []ast.InstructionsScope{
			{Path: "dummy_platform", Instructions: "Platform scope content."},
		},
	}
	files, notes, err := r.CompileProjectInstructions(project, dir)
	require.NoError(t, err)

	// Root copilot-instructions.md must NOT be written.
	_, hasRoot := files["copilot-instructions.md"]
	assert.False(t, hasRoot, "root copilot-instructions.md must NOT be written when .claude/ is present")

	// Nested scope file MUST still be written.
	scopeFile, hasScopeFile := files["instructions/dummy_platform.instructions.md"]
	assert.True(t, hasScopeFile, "instructions/dummy_platform.instructions.md must be written in passthrough mode")
	assert.Contains(t, scopeFile, `applyTo: "dummy_platform/**"`, "scope file must include applyTo frontmatter")
	assert.Contains(t, scopeFile, "Platform scope content.")

	// A CLAUDE_NATIVE_PASSTHROUGH info note must be emitted for the root.
	var found bool
	for _, n := range notes {
		if n.Code == renderer.CodeClaudeNativePassthrough && n.Resource == "root" {
			found = true
			break
		}
	}
	assert.True(t, found, "expected CLAUDE_NATIVE_PASSTHROUGH note for root instructions")
}

// TestCompileProjectInstructions_Copilot_NoClaude_WritesAll verifies that when
// no .claude/ directory is present, both the root copilot-instructions.md and
// any nested scope instruction files are written normally.
func TestCompileProjectInstructions_Copilot_NoClaude_WritesAll(t *testing.T) {
	dir := t.TempDir() // no .claude/

	r := copilot.New()
	project := &ast.ProjectConfig{
		Name:         "test-project",
		Instructions: "Root content.",
		InstructionsScopes: []ast.InstructionsScope{
			{Path: "dummy_platform", Instructions: "Platform scope content."},
		},
	}
	files, notes, err := r.CompileProjectInstructions(project, dir)
	require.NoError(t, err)

	_, hasRoot := files["copilot-instructions.md"]
	assert.True(t, hasRoot, "copilot-instructions.md must be written when .claude/ is absent")

	_, hasScopeFile := files["instructions/dummy_platform.instructions.md"]
	assert.True(t, hasScopeFile, "instructions/dummy_platform.instructions.md must still be written")

	for _, n := range notes {
		assert.NotEqual(t, renderer.CodeClaudeNativePassthrough, n.Code,
			"no CLAUDE_NATIVE_PASSTHROUGH notes expected when .claude/ is absent")
	}
}
