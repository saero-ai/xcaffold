package copilot_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/renderer"
	"github.com/saero-ai/xcaffold/internal/renderer/copilot"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCompile_Copilot_Skills_Minimal verifies that a skill with only name and
// description is written to .github/skills/<id>/SKILL.md with correct frontmatter.
func TestCompile_Copilot_Skills_Minimal(t *testing.T) {
	r := copilot.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Skills: map[string]ast.SkillConfig{
				"code-review": {
					Name:        "code-review",
					Description: "Reviews code for bugs.",
				},
			},
		},
	}
	out, notes, err := renderer.Orchestrate(r, config, "/tmp/test")
	require.NoError(t, err)

	content, ok := out.Files["skills/code-review/SKILL.md"]
	require.True(t, ok, "expected .github/skills/code-review/SKILL.md to be emitted")
	assert.Contains(t, content, "---")
	assert.Contains(t, content, "name: code-review")
	assert.Contains(t, content, "description: Reviews code for bugs.")
	assert.Empty(t, filterNotes(notes, renderer.CodeFieldUnsupported),
		"no FIELD_UNSUPPORTED notes expected for a minimal skill")
	assert.Empty(t, filterNotes(notes, renderer.CodeSkillScriptsDropped),
		"no SKILL_SCRIPTS_DROPPED notes expected for a minimal skill")
	assert.Empty(t, filterNotes(notes, renderer.CodeSkillAssetsDropped),
		"no SKILL_ASSETS_DROPPED notes expected for a minimal skill")
}

// TestCompile_Copilot_Skills_WithBody verifies that a skill with instructions
// produces a SKILL.md with both YAML frontmatter and a markdown body.
func TestCompile_Copilot_Skills_WithBody(t *testing.T) {
	r := copilot.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Skills: map[string]ast.SkillConfig{
				"tdd": {
					Name:        "tdd",
					Description: "Test-driven development workflow.",
					Body:        "Write the test first. Watch it fail. Write minimal code.",
				},
			},
		},
	}
	out, _, err := renderer.Orchestrate(r, config, "/tmp/test")
	require.NoError(t, err)

	content, ok := out.Files["skills/tdd/SKILL.md"]
	require.True(t, ok, "expected .github/skills/tdd/SKILL.md to be emitted")
	assert.Contains(t, content, "---")
	assert.Contains(t, content, "name: tdd")
	assert.Contains(t, content, "description: Test-driven development workflow.")
	assert.Contains(t, content, "Write the test first.")
}

// TestCompile_Copilot_Skills_WithAllowedTools verifies that allowed-tools is
// natively supported by Copilot and written as an allowed-tools list in frontmatter.
func TestCompile_Copilot_Skills_WithAllowedTools(t *testing.T) {
	r := copilot.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Skills: map[string]ast.SkillConfig{
				"search-skill": {
					Name:         "search-skill",
					Description:  "Skill with tool access.",
					Body:         "Use the read tool.",
					AllowedTools: []string{"Read", "Grep"},
				},
			},
		},
	}
	out, notes, err := renderer.Orchestrate(r, config, "/tmp/test")
	require.NoError(t, err)

	content, ok := out.Files["skills/search-skill/SKILL.md"]
	require.True(t, ok, "expected .github/skills/search-skill/SKILL.md to be emitted")
	assert.Contains(t, content, "allowed-tools:")
	assert.Contains(t, content, "Read")
	assert.Contains(t, content, "Grep")
	// allowed-tools is natively supported — no FIELD_UNSUPPORTED note expected for it
	for _, n := range filterNotes(notes, renderer.CodeFieldUnsupported) {
		assert.NotEqual(t, "allowed-tools", n.Field,
			"allowed-tools should not produce a fidelity note on Copilot")
	}
}

// TestCompile_Copilot_Skills_UnsupportedFields verifies that when-to-use,
// disable-model-invocation, scripts, and assets each produce the expected
// fidelity notes because Copilot has no native equivalent for them.
func TestCompile_Copilot_Skills_UnsupportedFields(t *testing.T) {
	r := copilot.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Skills: map[string]ast.SkillConfig{
				"full-skill": {
					Name:                   "full-skill",
					Description:            "Skill with all fields.",
					Body:                   "Do the thing.",
					WhenToUse:              "When reviewing code.",
					Scripts:                []string{"scripts/run.sh"},
					Assets:                 []string{"assets/template.md"},
					DisableModelInvocation: boolPtr(true),
				},
			},
		},
	}
	_, notes, err := renderer.Orchestrate(r, config, "/tmp/test")
	require.NoError(t, err)
	require.NotEmpty(t, notes, "expected fidelity notes for unsupported fields")

	codes := make(map[string]bool)
	fields := make(map[string]bool)
	for _, n := range notes {
		codes[n.Code] = true
		fields[n.Field] = true
	}

	assert.True(t, codes[renderer.CodeFieldUnsupported],
		"expected CodeFieldUnsupported for when-to-use or disable-model-invocation")
	assert.True(t, fields["when-to-use"],
		"expected fidelity note for when-to-use")
	assert.True(t, fields["disable-model-invocation"],
		"expected fidelity note for disable-model-invocation")
	assert.True(t, codes[renderer.CodeSkillScriptsDropped],
		"expected CodeSkillScriptsDropped for scripts field")
	assert.True(t, codes[renderer.CodeSkillAssetsDropped],
		"expected CodeSkillAssetsDropped for assets field")
}

// TestCompile_Copilot_Skills_ReferencesDropped verifies that a skill with
// references produces a SKILL_REFERENCES_DROPPED fidelity note because Copilot
// has no native support for skill references/ directories.
func TestCompile_Copilot_Skills_ReferencesDropped(t *testing.T) {
	r := copilot.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Skills: map[string]ast.SkillConfig{
				"test-skill": {
					Name:        "test-skill",
					Description: "A skill with references",
					Body:        "Do things.",
					References:  []string{"refs/doc.md"},
				},
			},
		},
	}
	_, notes, err := renderer.Orchestrate(r, config, t.TempDir())
	require.NoError(t, err)

	found := false
	for _, n := range notes {
		if n.Code == renderer.CodeSkillReferencesDropped {
			found = true
		}
	}
	assert.True(t, found, "expected SKILL_REFERENCES_DROPPED fidelity note for skill with references")
}

// TestCompile_Copilot_Skills_WithSubdirs verifies that references and scripts
// are flattened co-located alongside SKILL.md under skills/<id>/<filename>.
func TestCompile_Copilot_Skills_WithSubdirs(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmpDir, "refs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "refs", "guide.md"), []byte("# Guide"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(tmpDir, "scripts"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "scripts", "run.sh"), []byte("#!/bin/bash"), 0o644); err != nil {
		t.Fatal(err)
	}

	skills := map[string]ast.SkillConfig{
		"my-skill": {
			Description: "test",
			Body:        "Do the thing.",
			References:  []string{"refs/guide.md"},
			Scripts:     []string{"scripts/run.sh"},
		},
	}

	r := copilot.New()
	files, _, err := r.CompileSkills(skills, tmpDir)
	require.NoError(t, err)

	// Copilot: files are kept in standard subdirectories under skill root
	if _, ok := files["skills/my-skill/references/guide.md"]; !ok {
		t.Error("expected references copied to skills/my-skill/references/guide.md")
	}
	if _, ok := files["skills/my-skill/scripts/run.sh"]; !ok {
		t.Error("expected scripts copied to skills/my-skill/scripts/run.sh")
	}
}

// TestCompile_Copilot_Skills_Multiple verifies that two skills each produce a
// separate SKILL.md file under .github/skills/<id>/SKILL.md.
func TestCompile_Copilot_Skills_Multiple(t *testing.T) {
	r := copilot.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Skills: map[string]ast.SkillConfig{
				"beta-skill":  {Name: "beta-skill", Description: "Second skill."},
				"alpha-skill": {Name: "alpha-skill", Description: "First skill."},
			},
		},
	}
	out, _, err := renderer.Orchestrate(r, config, "/tmp/test")
	require.NoError(t, err)

	assert.Contains(t, out.Files, "skills/alpha-skill/SKILL.md",
		"expected .github/skills/alpha-skill/SKILL.md")
	assert.Contains(t, out.Files, "skills/beta-skill/SKILL.md",
		"expected .github/skills/beta-skill/SKILL.md")
}

// TestCompileSkills_Copilot_ClaudeDirPresent_EmitsPassthroughNotes verifies that
// when a .claude/ directory is present, CompileSkills returns no output files and
// emits one CLAUDE_NATIVE_PASSTHROUGH info note per skill.
func TestCompileSkills_Copilot_ClaudeDirPresent_EmitsPassthroughNotes(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".claude"), 0o755))

	r := copilot.New()
	skills := map[string]ast.SkillConfig{
		"document-feature": {Name: "document-feature", Description: "Generates docs."},
	}
	files, notes, err := r.CompileSkills(skills, dir)
	require.NoError(t, err)
	assert.Empty(t, files, "no .github/skills/ files should be written when .claude/ is present")
	require.Len(t, notes, 1)
	assert.Equal(t, renderer.CodeClaudeNativePassthrough, notes[0].Code)
	assert.Equal(t, renderer.LevelInfo, notes[0].Level)
	assert.Equal(t, "document-feature", notes[0].Resource)
	assert.Contains(t, notes[0].Reason, ".claude/skills/document-feature/SKILL.md")
}

// TestCompileSkills_Copilot_NoClaude_FullTranslation verifies that when no
// .claude/ directory exists, CompileSkills writes .github/skills/<id>/SKILL.md.
func TestCompileSkills_Copilot_NoClaude_FullTranslation(t *testing.T) {
	dir := t.TempDir()

	r := copilot.New()
	skills := map[string]ast.SkillConfig{
		"my-skill": {Name: "my-skill", Description: "A skill."},
	}
	files, notes, err := r.CompileSkills(skills, dir)
	require.NoError(t, err)
	assert.Contains(t, files, "skills/my-skill/SKILL.md",
		"full translation must write .github/skills/ when .claude/ is absent")
	for _, n := range notes {
		assert.NotEqual(t, renderer.CodeClaudeNativePassthrough, n.Code,
			"no CLAUDE_NATIVE_PASSTHROUGH notes expected when .claude/ is absent")
	}
}
