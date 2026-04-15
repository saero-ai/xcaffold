package agentsmd_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/output"
	"github.com/saero-ai/xcaffold/internal/renderer"
	"github.com/saero-ai/xcaffold/internal/renderer/agentsmd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func boolPtr(b bool) *bool { return &b }

// Test 1
func TestAgentsMDRenderer_Target(t *testing.T) {
	r := agentsmd.New()
	assert.Equal(t, "agentsmd", r.Target())
}

// Test 2
func TestAgentsMDRenderer_OutputDir(t *testing.T) {
	r := agentsmd.New()
	assert.Equal(t, ".", r.OutputDir())
}

// Test 3
func TestAgentsMDRenderer_Render_Identity(t *testing.T) {
	r := agentsmd.New()
	files := map[string]string{
		"AGENTS.md": "hello",
	}
	out := r.Render(files)
	require.NotNil(t, out)
	assert.IsType(t, &output.Output{}, out)
	assert.Equal(t, files, out.Files)
}

// Test 4
func TestAgentsMDRenderer_Compile_EmptyConfig(t *testing.T) {
	r := agentsmd.New()
	config := &ast.XcaffoldConfig{}
	out, notes, err := r.Compile(config, t.TempDir())
	require.NoError(t, err)
	require.Len(t, out.Files, 1)
	_, ok := out.Files["AGENTS.md"]
	assert.True(t, ok, "expected AGENTS.md to be present")
	assert.Empty(t, notes)
}

// Test 5
func TestAgentsMDRenderer_Compile_ProjectSection(t *testing.T) {
	r := agentsmd.New()
	config := &ast.XcaffoldConfig{
		Project: &ast.ProjectConfig{
			Name:        "myapp",
			Description: "a test project",
		},
	}
	out, _, err := r.Compile(config, t.TempDir())
	require.NoError(t, err)
	body := out.Files["AGENTS.md"]
	assert.Contains(t, body, "myapp — a test project")
}

// Test 6
func TestAgentsMDRenderer_Compile_AgentSection(t *testing.T) {
	r := agentsmd.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"developer": {
					Description:  "Writes code.",
					Model:        "claude-opus-4",
					Instructions: "You are a developer.",
				},
			},
		},
	}
	out, _, err := r.Compile(config, t.TempDir())
	require.NoError(t, err)
	body := out.Files["AGENTS.md"]
	assert.Contains(t, body, "## Agents")
	assert.Contains(t, body, "### developer")
	assert.Contains(t, body, "**Model**:")
}

// Test 7
func TestAgentsMDRenderer_Compile_SkillSection(t *testing.T) {
	r := agentsmd.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Skills: map[string]ast.SkillConfig{
				"git": {
					Description:  "Git conventions.",
					Instructions: "Use conventional commits.",
				},
			},
		},
	}
	out, _, err := r.Compile(config, t.TempDir())
	require.NoError(t, err)
	body := out.Files["AGENTS.md"]
	assert.Contains(t, body, "## Skills")
	assert.Contains(t, body, "### git")
}

// Test 8
func TestAgentsMDRenderer_Compile_RuleSection_WithPaths(t *testing.T) {
	r := agentsmd.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"go-style": {
					Paths:        []string{"src/**/*.go"},
					Instructions: "Use gofmt.",
				},
			},
		},
	}
	out, _, err := r.Compile(config, t.TempDir())
	require.NoError(t, err)
	found := false
	for _, content := range out.Files {
		if strings.Contains(content, "**Applies to**: src/**/*.go") {
			found = true
			break
		}
	}
	assert.True(t, found, "expected **Applies to**: src/**/*.go in some output file")
}

// Test 9
func TestAgentsMDRenderer_Compile_RuleSection_NoPaths(t *testing.T) {
	r := agentsmd.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"global-lint": {
					Instructions: "Run eslint.",
				},
			},
		},
	}
	out, _, err := r.Compile(config, t.TempDir())
	require.NoError(t, err)
	body := out.Files["AGENTS.md"]
	assert.Contains(t, body, "**Applies to**: all files")
}

// Test 10
func TestAgentsMDRenderer_Compile_WorkflowSection(t *testing.T) {
	r := agentsmd.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Workflows: map[string]ast.WorkflowConfig{
				"release": {
					Description:  "Release workflow.",
					Instructions: "Tag and push.",
				},
			},
		},
	}
	out, _, err := r.Compile(config, t.TempDir())
	require.NoError(t, err)
	body := out.Files["AGENTS.md"]
	assert.Contains(t, body, "## Workflows")
	assert.Contains(t, body, "### release")
}

// Test 11
func TestAgentsMDRenderer_Compile_EmptySectionsOmitted(t *testing.T) {
	r := agentsmd.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"dev": {Instructions: "Build things."},
			},
		},
	}
	out, _, err := r.Compile(config, t.TempDir())
	require.NoError(t, err)
	body := out.Files["AGENTS.md"]
	assert.NotContains(t, body, "## Skills")
	assert.NotContains(t, body, "## Rules")
	assert.NotContains(t, body, "## Workflows")
}

// Test 12
func TestAgentsMDRenderer_Compile_InstructionsFile(t *testing.T) {
	tmpDir := t.TempDir()
	instrFile := filepath.Join(tmpDir, "instructions.md")
	require.NoError(t, os.WriteFile(instrFile, []byte("Read carefully before acting."), 0600))

	r := agentsmd.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"cautious": {
					InstructionsFile: instrFile,
				},
			},
		},
	}
	out, _, err := r.Compile(config, tmpDir)
	require.NoError(t, err)
	body := out.Files["AGENTS.md"]
	assert.Contains(t, body, "Read carefully before acting.")
}

// Test 13 — notes returned for lossy agent field (tools)
func TestAgentsMDRenderer_Compile_FidelityNote_Tools(t *testing.T) {
	r := agentsmd.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"developer": {
					Tools:        []string{"Bash", "Read"},
					Instructions: "Do stuff.",
				},
			},
		},
	}
	_, notes, err := r.Compile(config, t.TempDir())
	require.NoError(t, err)
	require.NotEmpty(t, notes)

	var found bool
	for _, n := range notes {
		if n.Field == "tools" {
			found = true
			assert.Equal(t, renderer.CodeFieldUnsupported, n.Code)
			assert.Equal(t, "agentsmd", n.Target)
			assert.Equal(t, "agent", n.Kind)
			assert.Equal(t, "developer", n.Resource)
			assert.Equal(t, renderer.LevelWarning, n.Level)
		}
	}
	assert.True(t, found, "tools note must be emitted")
}

// Test 14 — suppression moved to the command layer; renderer always returns notes.
func TestAgentsMDRenderer_Compile_SuppressFidelityWarnings_NotesStillReturned(t *testing.T) {
	r := agentsmd.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"developer": {
					Tools:        []string{"Bash", "Read"},
					Instructions: "Do stuff.",
					Targets: map[string]ast.TargetOverride{
						"agentsmd": {SuppressFidelityWarnings: boolPtr(true)},
					},
				},
			},
		},
	}
	_, notes, err := r.Compile(config, t.TempDir())
	require.NoError(t, err)
	assert.NotEmpty(t, notes, "renderer returns notes regardless of suppression; suppression is applied at the command layer")
}

// Test 15 — unsupported field test that exercises the CodeFieldUnsupported path for a skill field.
func TestAgentsMDRenderer_Compile_FidelityNote_UnsupportedSkillField(t *testing.T) {
	r := agentsmd.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Skills: map[string]ast.SkillConfig{
				"setup": {
					Description: "Env setup.",
					Scripts:     []string{"scripts/install.sh"},
				},
			},
		},
	}
	_, notes, err := r.Compile(config, t.TempDir())
	require.NoError(t, err)
	require.NotEmpty(t, notes)
	assert.Equal(t, renderer.CodeFieldUnsupported, notes[0].Code)
	assert.Equal(t, "agentsmd", notes[0].Target)
}

// Test 16
func TestAgentsMDRenderer_Compile_GeneratedComment(t *testing.T) {
	r := agentsmd.New()
	config := &ast.XcaffoldConfig{}
	out, _, err := r.Compile(config, t.TempDir())
	require.NoError(t, err)
	body := out.Files["AGENTS.md"]
	assert.Contains(t, body, "<!-- Generated by xcaffold. Do not edit manually. -->")
}

// Test 17
func TestAgentsMDRenderer_Compile_NestedRule_SingleDir(t *testing.T) {
	r := agentsmd.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"api-conventions": {
					Paths:        []string{"src/api/**/*.ts"},
					Instructions: "Use REST conventions.",
				},
			},
		},
	}
	out, _, err := r.Compile(config, t.TempDir())
	require.NoError(t, err)

	_, hasRoot := out.Files["AGENTS.md"]
	assert.True(t, hasRoot, "root AGENTS.md should be present")

	_, hasNested := out.Files["src/api/AGENTS.md"]
	assert.True(t, hasNested, "src/api/AGENTS.md should be present")

	assert.NotContains(t, out.Files["AGENTS.md"], "api-conventions")
}

// Test 18
func TestAgentsMDRenderer_Compile_NestedRule_MultipleDir(t *testing.T) {
	r := agentsmd.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"api-conventions": {
					Paths:        []string{"src/api/**/*.ts"},
					Instructions: "Use REST conventions.",
				},
				"frontend-style": {
					Paths:        []string{"src/ui/**/*.tsx"},
					Instructions: "Use functional components.",
				},
			},
		},
	}
	out, _, err := r.Compile(config, t.TempDir())
	require.NoError(t, err)

	assert.Len(t, out.Files, 3, "expected root AGENTS.md + two directory-scoped files")
	_, hasRoot := out.Files["AGENTS.md"]
	assert.True(t, hasRoot)
	_, hasAPI := out.Files["src/api/AGENTS.md"]
	assert.True(t, hasAPI)
	_, hasUI := out.Files["src/ui/AGENTS.md"]
	assert.True(t, hasUI)
}

// Test 19
func TestAgentsMDRenderer_Compile_NestedRule_GlobalFallback(t *testing.T) {
	r := agentsmd.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"ts-global": {
					Paths:        []string{"**/*.ts"},
					Instructions: "Prefer type aliases.",
				},
			},
		},
	}
	out, _, err := r.Compile(config, t.TempDir())
	require.NoError(t, err)

	assert.Len(t, out.Files, 1)
	body := out.Files["AGENTS.md"]
	assert.Contains(t, body, "ts-global")
}

// Test 20
func TestAgentsMDRenderer_Compile_NestedRule_MixedGlobalAndScoped(t *testing.T) {
	r := agentsmd.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"global-lint": {
					Instructions: "Run linter.",
				},
				"api-conventions": {
					Paths:        []string{"src/api/**/*.ts"},
					Instructions: "Use REST conventions.",
				},
			},
		},
	}
	out, _, err := r.Compile(config, t.TempDir())
	require.NoError(t, err)

	assert.Contains(t, out.Files["AGENTS.md"], "global-lint")
	assert.NotContains(t, out.Files["AGENTS.md"], "api-conventions")
	assert.Contains(t, out.Files["src/api/AGENTS.md"], "api-conventions")
}

// Test 21
func TestAgentsMDRenderer_Compile_NestedLockTracking(t *testing.T) {
	r := agentsmd.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"api-conventions": {
					Paths:        []string{"src/api/**/*.ts"},
					Instructions: "Use REST conventions.",
				},
				"frontend-style": {
					Paths:        []string{"src/ui/**/*.tsx"},
					Instructions: "Use functional components.",
				},
			},
		},
	}
	out, _, err := r.Compile(config, t.TempDir())
	require.NoError(t, err)

	assert.Contains(t, out.Files, "AGENTS.md")
	assert.Contains(t, out.Files, "src/api/AGENTS.md")
	assert.Contains(t, out.Files, "src/ui/AGENTS.md")
}
