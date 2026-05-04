package gemini

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/renderer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompile_Gemini_Skills_Minimal(t *testing.T) {
	r := New()
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
	out, notes, err := renderer.Orchestrate(r, config, t.TempDir())
	require.NoError(t, err)
	content := out.Files["skills/code-review/SKILL.md"]
	assert.Contains(t, content, "---")
	assert.Contains(t, content, "name: code-review")
	assert.Contains(t, content, "description: Reviews code for bugs.")
	assert.Empty(t, notes)
}

func TestCompile_Gemini_Skills_WithBody(t *testing.T) {
	r := New()
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
	out, _, err := renderer.Orchestrate(r, config, t.TempDir())
	require.NoError(t, err)
	content := out.Files["skills/tdd/SKILL.md"]
	assert.Contains(t, content, "name: tdd")
	assert.Contains(t, content, "Write the test first.")
}

func TestCompile_Gemini_Skills_UnsupportedFields(t *testing.T) {
	r := New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Skills: map[string]ast.SkillConfig{
				"full-skill": {
					Name:                   "full-skill",
					Description:            "Skill with all fields.",
					Body:                   "Do the thing.",
					AllowedTools:           []string{"Read", "Grep"},
					WhenToUse:              "When reviewing code.",
					Scripts:                []string{"scripts/run.sh"},
					Assets:                 []string{"assets/template.md"},
					DisableModelInvocation: boolPtr(true),
				},
			},
		},
	}
	_, notes, err := renderer.Orchestrate(r, config, t.TempDir())
	require.NoError(t, err)

	// Should have fidelity notes for unsupported fields
	codes := make(map[string]bool)
	fields := make(map[string]bool)
	for _, n := range notes {
		codes[n.Code] = true
		fields[n.Field] = true
	}
	assert.True(t, codes[renderer.CodeFieldUnsupported], "expected CodeFieldUnsupported for allowed-tools, when-to-use, or disable-model-invocation")
	assert.True(t, fields["disable-model-invocation"], "expected fidelity note for disable-model-invocation")
	assert.True(t, codes[renderer.CodeSkillScriptsDropped], "expected CodeSkillScriptsDropped")
	assert.True(t, codes[renderer.CodeSkillAssetsDropped], "expected CodeSkillAssetsDropped")
}

func TestCompile_Gemini_Skills_ReferencesDropped(t *testing.T) {
	r := New()
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

func TestCompile_Gemini_Skills_WithSubdirs(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "refs"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "refs", "guide.md"), []byte("# Guide"), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "examples"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "examples", "sample.md"), []byte("# Sample"), 0o644))

	skills := map[string]ast.SkillConfig{
		"my-skill": {
			Description: "test",
			Body:        "Do the thing.",
			References:  []string{"refs/guide.md"},
			Examples:    []string{"examples/sample.md"},
		},
	}

	r := New()
	files, notes, err := r.CompileSkills(skills, tmpDir)
	require.NoError(t, err)

	if _, ok := files["skills/my-skill/references/guide.md"]; !ok {
		t.Error("expected references/guide.md to be compiled")
	}
	// Examples collapse into references for Gemini
	if _, ok := files["skills/my-skill/references/sample.md"]; !ok {
		t.Error("expected examples collapsed into references/sample.md")
	}

	// No drop notes should be emitted when files exist
	for _, n := range notes {
		if n.Code == renderer.CodeSkillReferencesDropped {
			t.Errorf("unexpected SKILL_REFERENCES_DROPPED note: %s", n.Reason)
		}
	}
}

func TestCompile_Gemini_Skills_Multiple(t *testing.T) {
	r := New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Skills: map[string]ast.SkillConfig{
				"beta-skill":  {Name: "beta-skill", Description: "Second."},
				"alpha-skill": {Name: "alpha-skill", Description: "First."},
			},
		},
	}
	out, _, err := renderer.Orchestrate(r, config, t.TempDir())
	require.NoError(t, err)
	assert.Contains(t, out.Files, "skills/alpha-skill/SKILL.md")
	assert.Contains(t, out.Files, "skills/beta-skill/SKILL.md")
}
