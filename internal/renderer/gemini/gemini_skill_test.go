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
	out, notes, err := r.Compile(config, "/tmp/test")
	require.NoError(t, err)
	content := out.Files[".gemini/skills/code-review/SKILL.md"]
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
					Name:         "tdd",
					Description:  "Test-driven development workflow.",
					Instructions: "Write the test first. Watch it fail. Write minimal code.",
				},
			},
		},
	}
	out, _, err := r.Compile(config, "/tmp/test")
	require.NoError(t, err)
	content := out.Files[".gemini/skills/tdd/SKILL.md"]
	assert.Contains(t, content, "name: tdd")
	assert.Contains(t, content, "Write the test first.")
}

func TestCompile_Gemini_Skills_FromFile(t *testing.T) {
	tmpDir := t.TempDir()
	err := os.WriteFile(filepath.Join(tmpDir, "skill-body.md"), []byte("Skill instructions from file."), 0o644)
	require.NoError(t, err)

	r := New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Skills: map[string]ast.SkillConfig{
				"file-skill": {
					Name:             "file-skill",
					Description:      "A skill loaded from file.",
					InstructionsFile: "skill-body.md",
				},
			},
		},
	}
	out, _, err := r.Compile(config, tmpDir)
	require.NoError(t, err)
	content := out.Files[".gemini/skills/file-skill/SKILL.md"]
	assert.Contains(t, content, "Skill instructions from file.")
}

func TestCompile_Gemini_Skills_UnsupportedFields(t *testing.T) {
	r := New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Skills: map[string]ast.SkillConfig{
				"full-skill": {
					Name:                   "full-skill",
					Description:            "Skill with all fields.",
					Instructions:           "Do the thing.",
					AllowedTools:           []string{"Read", "Grep"},
					WhenToUse:              "When reviewing code.",
					Scripts:                []string{"scripts/run.sh"},
					Assets:                 []string{"assets/template.md"},
					DisableModelInvocation: boolPtr(true),
				},
			},
		},
	}
	_, notes, err := r.Compile(config, "/tmp/test")
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
	out, _, err := r.Compile(config, "/tmp/test")
	require.NoError(t, err)
	assert.Contains(t, out.Files, ".gemini/skills/alpha-skill/SKILL.md")
	assert.Contains(t, out.Files, ".gemini/skills/beta-skill/SKILL.md")
}
