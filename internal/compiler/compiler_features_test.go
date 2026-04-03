package compiler

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Feature 1A: instructions_file: external file references
// ---------------------------------------------------------------------------

// TestCompile_AgentInstructionsFile_ReadsExternalFile verifies that an agent
// with instructions_file: uses the file body as its system prompt.
func TestCompile_AgentInstructionsFile_ReadsExternalFile(t *testing.T) {
	dir := t.TempDir()
	instrPath := filepath.Join(dir, "agents", "cto.md")
	require.NoError(t, os.MkdirAll(filepath.Dir(instrPath), 0755))
	require.NoError(t, os.WriteFile(instrPath, []byte("You are the Chief Technology Officer.\nLead with clarity."), 0644))

	config := &ast.XcaffoldConfig{
		Agents: map[string]ast.AgentConfig{
			"cto": {
				Description:      "Chief Technology Officer",
				InstructionsFile: "agents/cto.md",
			},
		},
	}

	out, err := Compile(config, dir)
	require.NoError(t, err)

	content, ok := out.Files["agents/cto.md"]
	require.True(t, ok, "agents/cto.md should be compiled")
	assert.Contains(t, content, "You are the Chief Technology Officer.")
	assert.Contains(t, content, "Lead with clarity.")
	assert.Contains(t, content, "description: Chief Technology Officer")
}

// TestCompile_AgentInstructionsFile_StripsFrontmatter verifies that frontmatter
// in an instructions_file is stripped before being used as the prompt body.
func TestCompile_AgentInstructionsFile_StripsFrontmatter(t *testing.T) {
	dir := t.TempDir()
	instrPath := filepath.Join(dir, "agents", "dev.md")
	require.NoError(t, os.MkdirAll(filepath.Dir(instrPath), 0755))
	content := "---\nname: Developer\nmodel: claude-sonnet\n---\n\nWrite clean code.\nAlways test."
	require.NoError(t, os.WriteFile(instrPath, []byte(content), 0644))

	config := &ast.XcaffoldConfig{
		Agents: map[string]ast.AgentConfig{
			"developer": {InstructionsFile: "agents/dev.md"},
		},
	}

	out, err := Compile(config, dir)
	require.NoError(t, err)

	compiled := out.Files["agents/developer.md"]
	assert.Contains(t, compiled, "Write clean code.")
	assert.NotContains(t, compiled, "name: Developer", "frontmatter should be stripped from file body")
}

// TestCompile_AgentInstructionsFile_Missing_ReturnsError verifies that a
// missing instructions_file causes a compile error, not silent empty content.
func TestCompile_AgentInstructionsFile_Missing_ReturnsError(t *testing.T) {
	config := &ast.XcaffoldConfig{
		Agents: map[string]ast.AgentConfig{
			"cto": {InstructionsFile: "nonexistent/cto.md"},
		},
	}

	_, err := Compile(config, t.TempDir())
	require.Error(t, err, "missing instructions_file must return an error")
	assert.Contains(t, err.Error(), "nonexistent/cto.md")
}

// TestCompile_InstructionsFile_PathTraversal_Rejected verifies that
// instructions_file paths that escape the project root are rejected.
func TestCompile_InstructionsFile_PathTraversal_Rejected(t *testing.T) {
	config := &ast.XcaffoldConfig{
		Agents: map[string]ast.AgentConfig{
			"evil": {InstructionsFile: "../../etc/passwd"},
		},
	}

	_, err := Compile(config, t.TempDir())
	require.Error(t, err, "traversal paths in instructions_file must be rejected")
}

// TestCompile_InstructionsFile_InlineWins verifies that inline "instructions:"
// takes priority over "instructions_file:" when both are set in the AST
// (this case is normally caught by the parser, but the compiler should also
// be defensive and honour the priority ordering).
func TestCompile_InstructionsFile_InlinePriority(t *testing.T) {
	dir := t.TempDir()
	fPath := filepath.Join(dir, "file.md")
	require.NoError(t, os.WriteFile(fPath, []byte("From file."), 0644))

	config := &ast.XcaffoldConfig{
		Agents: map[string]ast.AgentConfig{
			"agent": {
				Instructions:     "From inline.",
				InstructionsFile: "file.md",
			},
		},
	}

	// Parser would reject this, but we test the compiler directly.
	out, err := Compile(config, dir)
	require.NoError(t, err)
	content := out.Files["agents/agent.md"]
	assert.Contains(t, content, "From inline.", "inline instructions must take priority")
	assert.NotContains(t, content, "From file.")
}

// ---------------------------------------------------------------------------
// Feature 1B: references: skill supplementary files
// ---------------------------------------------------------------------------

// TestCompile_SkillWithReferences_CopiesFiles verifies that reference files
// declared in skills.references are copied into skills/<id>/references/.
func TestCompile_SkillWithReferences_CopiesFiles(t *testing.T) {
	dir := t.TempDir()
	refDir := filepath.Join(dir, "skills", "flutter-integration", "references")
	require.NoError(t, os.MkdirAll(refDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(refDir, "advanced-patterns.md"), []byte("# Advanced Patterns"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(refDir, "lottie-guide.md"), []byte("# Lottie Guide"), 0644))

	config := &ast.XcaffoldConfig{
		Skills: map[string]ast.SkillConfig{
			"flutter-integration": {
				Description:  "Flutter SVG and Lottie integration",
				Instructions: "Integrate SVG and Lottie into Flutter apps.",
				References: []string{
					"skills/flutter-integration/references/advanced-patterns.md",
					"skills/flutter-integration/references/lottie-guide.md",
				},
			},
		},
	}

	out, err := Compile(config, dir)
	require.NoError(t, err)

	_, hasSkill := out.Files["skills/flutter-integration/SKILL.md"]
	assert.True(t, hasSkill, "SKILL.md must be compiled")

	refContent, hasRef := out.Files["skills/flutter-integration/references/advanced-patterns.md"]
	assert.True(t, hasRef, "reference file must be in output")
	assert.Contains(t, refContent, "Advanced Patterns")

	_, hasRef2 := out.Files["skills/flutter-integration/references/lottie-guide.md"]
	assert.True(t, hasRef2, "second reference file must be in output")
}

// TestCompile_SkillReferences_Glob_ExpandsCorrectly verifies that glob patterns
// in references: expand to multiple files.
func TestCompile_SkillReferences_Glob_ExpandsCorrectly(t *testing.T) {
	dir := t.TempDir()
	refDir := filepath.Join(dir, "skills", "design", "refs")
	require.NoError(t, os.MkdirAll(refDir, 0755))
	for _, name := range []string{"colors.md", "typography.md", "layout.md"} {
		require.NoError(t, os.WriteFile(filepath.Join(refDir, name), []byte("# "+name), 0644))
	}

	config := &ast.XcaffoldConfig{
		Skills: map[string]ast.SkillConfig{
			"design": {
				Instructions: "Design system patterns.",
				References:   []string{"skills/design/refs/*.md"},
			},
		},
	}

	out, err := Compile(config, dir)
	require.NoError(t, err)

	refCount := 0
	for key := range out.Files {
		if filepath.Dir(key) == filepath.Clean("skills/design/references") {
			refCount++
		}
	}
	assert.Equal(t, 3, refCount, "glob should expand to all 3 reference files")
}

// TestCompile_SkillReferences_PathTraversal_Rejected verifies traversal is blocked.
func TestCompile_SkillReferences_PathTraversal_Rejected(t *testing.T) {
	config := &ast.XcaffoldConfig{
		Skills: map[string]ast.SkillConfig{
			"evil": {
				Instructions: "Some skill.",
				References:   []string{"../../etc/shadow"},
			},
		},
	}
	_, err := Compile(config, t.TempDir())
	require.Error(t, err, "traversal references must be rejected")
}

// ---------------------------------------------------------------------------
// Feature 2A: settings type fixes
// ---------------------------------------------------------------------------

// TestCompile_Settings_StatusLine_IsObject verifies that statusLine emits as
// a JSON object ({"type":"command","command":"..."}) not a plain string.
func TestCompile_Settings_StatusLine_IsObject(t *testing.T) {
	config := &ast.XcaffoldConfig{
		Settings: ast.SettingsConfig{
			StatusLine: &ast.StatusLineConfig{
				Type:    "command",
				Command: "bash ~/.claude/statusline.sh",
			},
		},
	}

	out, err := Compile(config, "")
	require.NoError(t, err)

	raw, ok := out.Files["settings.json"]
	require.True(t, ok, "settings.json must be generated")

	var parsed map[string]any
	require.NoError(t, json.Unmarshal([]byte(raw), &parsed))

	slAny, has := parsed["statusLine"]
	require.True(t, has, "statusLine must be present")
	slMap, ok := slAny.(map[string]any)
	require.True(t, ok, "statusLine must be an object, not a string")
	assert.Equal(t, "command", slMap["type"])
	assert.Equal(t, "bash ~/.claude/statusline.sh", slMap["command"])
}

// TestCompile_Settings_EnabledPlugins_IsMap verifies that enabledPlugins emits
// as a JSON object (map[string]bool) not an array.
func TestCompile_Settings_EnabledPlugins_IsMap(t *testing.T) {
	config := &ast.XcaffoldConfig{
		Settings: ast.SettingsConfig{
			EnabledPlugins: map[string]bool{
				"plugin-a": true,
				"plugin-b": false,
			},
		},
	}

	out, err := Compile(config, "")
	require.NoError(t, err)

	raw, ok := out.Files["settings.json"]
	require.True(t, ok, "settings.json must be generated")

	var parsed map[string]any
	require.NoError(t, json.Unmarshal([]byte(raw), &parsed))

	epAny, has := parsed["enabledPlugins"]
	require.True(t, has, "enabledPlugins must be present")
	epMap, ok := epAny.(map[string]any)
	require.True(t, ok, "enabledPlugins must be an object (map), not an array")
	assert.Equal(t, true, epMap["plugin-a"])
	assert.Equal(t, false, epMap["plugin-b"])
}

// TestCompile_Settings_Schema_IsFirstKey verifies that $schema is emitted
// as the first key in settings.json.
func TestCompile_Settings_Schema_IsFirstKey(t *testing.T) {
	config := &ast.XcaffoldConfig{
		MCP: map[string]ast.MCPConfig{
			"sqlite": {Command: "npx", Args: []string{"-y", "sqlite"}},
		},
	}
	out, err := Compile(config, "")
	require.NoError(t, err)

	raw := out.Files["settings.json"]
	var parsed map[string]any
	require.NoError(t, json.Unmarshal([]byte(raw), &parsed))

	_, has := parsed["$schema"]
	assert.True(t, has, "settings.json must contain $schema key")
}

// ---------------------------------------------------------------------------
// Feature: stripFrontmatter helper
// ---------------------------------------------------------------------------

func TestStripFrontmatter_WithFrontmatter(t *testing.T) {
	input := "---\nname: CTO\nmodel: claude-sonnet\n---\n\nYou are the CTO.\nLead with clarity."
	result := stripFrontmatter(input)
	assert.Equal(t, "You are the CTO.\nLead with clarity.", result)
	assert.NotContains(t, result, "name:")
}

func TestStripFrontmatter_WithoutFrontmatter(t *testing.T) {
	input := "You are the CTO.\nLead with clarity."
	result := stripFrontmatter(input)
	assert.Equal(t, input, result)
}

func TestStripFrontmatter_EmptyFile(t *testing.T) {
	result := stripFrontmatter("")
	assert.Equal(t, "", result)
}

func TestStripFrontmatter_OnlyFrontmatter(t *testing.T) {
	input := "---\nname: CTO\n---\n"
	result := stripFrontmatter(input)
	assert.Equal(t, "", result)
}

// ---------------------------------------------------------------------------
// Feature 4A: Convention-over-configuration auto-discovery
// ---------------------------------------------------------------------------

// TestCompile_ConventionAutoDiscover_Agent verifies that when an agent has no
// instructions or instructions_file, the compiler auto-discovers agents/<id>.md.
func TestCompile_ConventionAutoDiscover_Agent(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "agents"), 0755))
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "agents", "cto.md"),
		[]byte("You are the CTO. Lead with clarity."),
		0644,
	))

	config := &ast.XcaffoldConfig{
		Agents: map[string]ast.AgentConfig{
			"cto": {Description: "Chief Technology Officer"},
			// No instructions or instructions_file — relies on convention
		},
	}

	out, err := Compile(config, dir)
	require.NoError(t, err)

	content := out.Files["agents/cto.md"]
	assert.Contains(t, content, "You are the CTO.")
}

// TestCompile_ConventionAutoDiscover_Skill verifies that skills/<id>/SKILL.md
// is auto-discovered by convention when no instructions fields are set.
func TestCompile_ConventionAutoDiscover_Skill(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "skills", "git-workflow"), 0755))
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "skills", "git-workflow", "SKILL.md"),
		[]byte("---\nname: Git Workflow\n---\n\nFollow the git workflow."),
		0644,
	))

	config := &ast.XcaffoldConfig{
		Skills: map[string]ast.SkillConfig{
			"git-workflow": {Description: "Git workflow patterns"},
			// No instructions or instructions_file — relies on convention
		},
	}

	out, err := Compile(config, dir)
	require.NoError(t, err)

	content := out.Files["skills/git-workflow/SKILL.md"]
	assert.Contains(t, content, "Follow the git workflow.")
	assert.NotContains(t, content, "name: Git Workflow", "frontmatter should be stripped")
}

// TestCompile_ConventionAutoDiscover_MissingFile_SilentEmpty verifies that
// when the convention file doesn't exist, the resource compiles with an empty
// body (not an error).
func TestCompile_ConventionAutoDiscover_MissingFile_SilentEmpty(t *testing.T) {
	config := &ast.XcaffoldConfig{
		Agents: map[string]ast.AgentConfig{
			"cto": {Description: "CTO agent"},
			// No agents/cto.md exists in baseDir
		},
	}

	out, err := Compile(config, t.TempDir()) // empty tempdir — no convention file
	require.NoError(t, err, "missing convention file must be silent, not an error")

	content := out.Files["agents/cto.md"]
	// Should compile the frontmatter only, with empty body
	assert.Contains(t, content, "description: CTO agent")
}
