package agentsmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/output"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// helper returns a pointer to a bool value.
func boolPtr(b bool) *bool { return &b }

// Test 1
func TestAgentsMDRenderer_Target(t *testing.T) {
	r := New()
	assert.Equal(t, "agentsmd", r.Target())
}

// Test 2
func TestAgentsMDRenderer_OutputDir(t *testing.T) {
	r := New()
	assert.Equal(t, ".", r.OutputDir())
}

// Test 3
func TestAgentsMDRenderer_Render_Identity(t *testing.T) {
	r := New()
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
	r := New()
	config := &ast.XcaffoldConfig{}
	out, err := r.Compile(config, t.TempDir())
	require.NoError(t, err)
	require.Len(t, out.Files, 1)
	_, ok := out.Files["AGENTS.md"]
	assert.True(t, ok, "expected AGENTS.md to be present")
}

// Test 5
func TestAgentsMDRenderer_Compile_ProjectSection(t *testing.T) {
	r := New()
	config := &ast.XcaffoldConfig{
		Project: ast.ProjectConfig{
			Name:        "myapp",
			Description: "a test project",
		},
	}
	out, err := r.Compile(config, t.TempDir())
	require.NoError(t, err)
	body := out.Files["AGENTS.md"]
	assert.Contains(t, body, "myapp — a test project")
}

// Test 6
func TestAgentsMDRenderer_Compile_AgentSection(t *testing.T) {
	r := New()
	config := &ast.XcaffoldConfig{
		Agents: map[string]ast.AgentConfig{
			"developer": {
				Description:  "Writes code.",
				Model:        "claude-opus-4",
				Instructions: "You are a developer.",
			},
		},
	}
	out, err := r.Compile(config, t.TempDir())
	require.NoError(t, err)
	body := out.Files["AGENTS.md"]
	assert.Contains(t, body, "## Agents")
	assert.Contains(t, body, "### developer")
	assert.Contains(t, body, "**Model**:")
}

// Test 7
func TestAgentsMDRenderer_Compile_SkillSection(t *testing.T) {
	r := New()
	config := &ast.XcaffoldConfig{
		Skills: map[string]ast.SkillConfig{
			"git": {
				Description:  "Git conventions.",
				Instructions: "Use conventional commits.",
			},
		},
	}
	out, err := r.Compile(config, t.TempDir())
	require.NoError(t, err)
	body := out.Files["AGENTS.md"]
	assert.Contains(t, body, "## Skills")
	assert.Contains(t, body, "### git")
}

// Test 8
func TestAgentsMDRenderer_Compile_RuleSection_WithPaths(t *testing.T) {
	r := New()
	config := &ast.XcaffoldConfig{
		Rules: map[string]ast.RuleConfig{
			"go-style": {
				Paths:        []string{"src/**/*.go"},
				Instructions: "Use gofmt.",
			},
		},
	}
	out, err := r.Compile(config, t.TempDir())
	require.NoError(t, err)
	// Rule with paths goes to a nested file, not root AGENTS.md
	// Check nested file contains the Applies to line
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
	r := New()
	config := &ast.XcaffoldConfig{
		Rules: map[string]ast.RuleConfig{
			"global-lint": {
				Instructions: "Run eslint.",
			},
		},
	}
	out, err := r.Compile(config, t.TempDir())
	require.NoError(t, err)
	body := out.Files["AGENTS.md"]
	assert.Contains(t, body, "**Applies to**: all files")
}

// Test 10
func TestAgentsMDRenderer_Compile_WorkflowSection(t *testing.T) {
	r := New()
	config := &ast.XcaffoldConfig{
		Workflows: map[string]ast.WorkflowConfig{
			"release": {
				Description:  "Release workflow.",
				Instructions: "Tag and push.",
			},
		},
	}
	out, err := r.Compile(config, t.TempDir())
	require.NoError(t, err)
	body := out.Files["AGENTS.md"]
	assert.Contains(t, body, "## Workflows")
	assert.Contains(t, body, "### release")
}

// Test 11
func TestAgentsMDRenderer_Compile_EmptySectionsOmitted(t *testing.T) {
	r := New()
	config := &ast.XcaffoldConfig{
		Agents: map[string]ast.AgentConfig{
			"dev": {Instructions: "Build things."},
		},
	}
	out, err := r.Compile(config, t.TempDir())
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

	r := New()
	config := &ast.XcaffoldConfig{
		Agents: map[string]ast.AgentConfig{
			"cautious": {
				InstructionsFile: instrFile,
			},
		},
	}
	out, err := r.Compile(config, tmpDir)
	require.NoError(t, err)
	body := out.Files["AGENTS.md"]
	assert.Contains(t, body, "Read carefully before acting.")
}

// Test 13
func TestAgentsMDRenderer_Compile_FidelityWarning_Tools(t *testing.T) {
	var buf bytes.Buffer
	warningWriter = &buf

	r := New()
	config := &ast.XcaffoldConfig{
		Agents: map[string]ast.AgentConfig{
			"developer": {
				Tools:        []string{"Bash", "Read"},
				Instructions: "Do stuff.",
			},
		},
	}
	_, err := r.Compile(config, t.TempDir())
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, `WARNING (agentsmd): field "tools" on agent "developer" has no AGENTS.md equivalent and was dropped.`)

	warningWriter = os.Stderr
}

// Test 14
func TestAgentsMDRenderer_Compile_SuppressFidelityWarnings(t *testing.T) {
	var buf bytes.Buffer
	warningWriter = &buf

	r := New()
	config := &ast.XcaffoldConfig{
		Agents: map[string]ast.AgentConfig{
			"developer": {
				Tools:        []string{"Bash", "Read"},
				Instructions: "Do stuff.",
				Targets: map[string]ast.TargetOverride{
					"agentsmd": {SuppressFidelityWarnings: boolPtr(true)},
				},
			},
		},
	}
	_, err := r.Compile(config, t.TempDir())
	require.NoError(t, err)

	assert.Empty(t, buf.String(), "expected no warnings when SuppressFidelityWarnings is true")

	warningWriter = os.Stderr
}

// Test 15
func TestAgentsMDRenderer_Compile_GeneratedComment(t *testing.T) {
	r := New()
	config := &ast.XcaffoldConfig{}
	out, err := r.Compile(config, t.TempDir())
	require.NoError(t, err)
	body := out.Files["AGENTS.md"]
	assert.Contains(t, body, "<!-- Generated by xcaffold. Do not edit manually. -->")
}

// Test 16
func TestAgentsMDRenderer_Compile_NestedRule_SingleDir(t *testing.T) {
	r := New()
	config := &ast.XcaffoldConfig{
		Rules: map[string]ast.RuleConfig{
			"api-conventions": {
				Paths:        []string{"src/api/**/*.ts"},
				Instructions: "Use REST conventions.",
			},
		},
	}
	out, err := r.Compile(config, t.TempDir())
	require.NoError(t, err)

	_, hasRoot := out.Files["AGENTS.md"]
	assert.True(t, hasRoot, "root AGENTS.md should be present")

	_, hasNested := out.Files["src/api/AGENTS.md"]
	assert.True(t, hasNested, "src/api/AGENTS.md should be present")

	// Root should NOT contain this scoped rule
	assert.NotContains(t, out.Files["AGENTS.md"], "api-conventions")
}

// Test 17
func TestAgentsMDRenderer_Compile_NestedRule_MultipleDir(t *testing.T) {
	r := New()
	config := &ast.XcaffoldConfig{
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
	}
	out, err := r.Compile(config, t.TempDir())
	require.NoError(t, err)

	assert.Len(t, out.Files, 3, "expected root AGENTS.md + two directory-scoped files")
	_, hasRoot := out.Files["AGENTS.md"]
	assert.True(t, hasRoot)
	_, hasAPI := out.Files["src/api/AGENTS.md"]
	assert.True(t, hasAPI)
	_, hasUI := out.Files["src/ui/AGENTS.md"]
	assert.True(t, hasUI)
}

// Test 18
func TestAgentsMDRenderer_Compile_NestedRule_GlobalFallback(t *testing.T) {
	r := New()
	config := &ast.XcaffoldConfig{
		Rules: map[string]ast.RuleConfig{
			"ts-global": {
				Paths:        []string{"**/*.ts"},
				Instructions: "Prefer type aliases.",
			},
		},
	}
	out, err := r.Compile(config, t.TempDir())
	require.NoError(t, err)

	// Should only have root AGENTS.md (no common dir prefix for **/*.ts)
	assert.Len(t, out.Files, 1)
	body := out.Files["AGENTS.md"]
	assert.Contains(t, body, "ts-global")
}

// Test 19
func TestAgentsMDRenderer_Compile_NestedRule_MixedGlobalAndScoped(t *testing.T) {
	r := New()
	config := &ast.XcaffoldConfig{
		Rules: map[string]ast.RuleConfig{
			"global-lint": {
				Instructions: "Run linter.",
			},
			"api-conventions": {
				Paths:        []string{"src/api/**/*.ts"},
				Instructions: "Use REST conventions.",
			},
		},
	}
	out, err := r.Compile(config, t.TempDir())
	require.NoError(t, err)

	// Root contains global rule
	assert.Contains(t, out.Files["AGENTS.md"], "global-lint")
	// Root does NOT contain scoped rule
	assert.NotContains(t, out.Files["AGENTS.md"], "api-conventions")
	// Nested file contains scoped rule
	assert.Contains(t, out.Files["src/api/AGENTS.md"], "api-conventions")
}

// Test 20
func TestAgentsMDRenderer_Compile_NestedLockTracking(t *testing.T) {
	r := New()
	config := &ast.XcaffoldConfig{
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
	}
	out, err := r.Compile(config, t.TempDir())
	require.NoError(t, err)

	// All emitted files must appear as keys in Output.Files
	assert.Contains(t, out.Files, "AGENTS.md")
	assert.Contains(t, out.Files, "src/api/AGENTS.md")
	assert.Contains(t, out.Files, "src/ui/AGENTS.md")
}
