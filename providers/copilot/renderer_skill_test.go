package copilot_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/renderer"
	"github.com/saero-ai/xcaffold/providers/copilot"
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
					AllowedTools: ast.ClearableList{Values: []string{"Read", "Grep"}},
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
// disable-model-invocation, and argument-hint each produce fidelity notes
// because Copilot has no native equivalent for them in SKILL.md frontmatter.
// Scripts and assets are handled via Artifacts + auto-discovery and produce
// no fidelity notes when no artifact directory is declared.
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
	// Scripts and assets are now handled via Artifacts + auto-discovery.
	// No drop notes are emitted for legacy field values.
}

// TestCompile_Copilot_Skills_NoArtifactsProducesOnlySKILLMd verifies that a
// skill with no Artifacts list declared produces only SKILL.md and no subdirectory
// files, regardless of any legacy field values set. With auto-discovery, artifact
// discovery is driven entirely by skill.Artifacts.
func TestCompile_Copilot_Skills_NoArtifactsProducesOnlySKILLMd(t *testing.T) {
	r := copilot.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Skills: map[string]ast.SkillConfig{
				"test-skill": {
					Name:        "test-skill",
					Description: "A skill with no declared artifacts",
					Body:        "Do things.",
				},
			},
		},
	}
	files, _, err := renderer.Orchestrate(r, config, t.TempDir())
	require.NoError(t, err)

	assert.Contains(t, files.Files, "skills/test-skill/SKILL.md", "SKILL.md should always be emitted")
	for k := range files.Files {
		if k != "skills/test-skill/SKILL.md" {
			t.Errorf("unexpected file emitted with no artifacts: %s", k)
		}
	}
}

// TestCompile_Copilot_Skills_WithSubdirs verifies that references and scripts
// are compiled to standard subdirectories when declared in skill.Artifacts.
// Auto-discovery walks xcaf/skills/<id>/<artifactName>/ using canonical names.
func TestCompile_Copilot_Skills_WithSubdirs(t *testing.T) {
	tmpDir := t.TempDir()
	// Use canonical directory names — auto-discovery walks these exact paths.
	skillBase := filepath.Join(tmpDir, "xcaf", "skills", "my-skill")
	if err := os.MkdirAll(filepath.Join(skillBase, "references"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillBase, "references", "guide.md"), []byte("# Guide"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(skillBase, "scripts"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillBase, "scripts", "run.sh"), []byte("#!/bin/bash"), 0o644); err != nil {
		t.Fatal(err)
	}

	skills := map[string]ast.SkillConfig{
		"my-skill": {
			Description: "test",
			Body:        "Do the thing.",
			Artifacts:   []string{"references", "scripts"},
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
