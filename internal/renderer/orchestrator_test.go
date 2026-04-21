package renderer_test

import (
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/output"
	"github.com/saero-ai/xcaffold/internal/renderer"
	"github.com/saero-ai/xcaffold/internal/renderer/claude"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestOrchestrate_PerResource_ProducesFiles verifies that Orchestrate dispatches to
// per-resource methods and produces at least one file for non-empty configs.
func TestOrchestrate_PerResource_ProducesFiles(t *testing.T) {
	r := claude.New()
	config := &ast.XcaffoldConfig{
		Version: "1",
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"tester": {
					Name:         "Tester",
					Description:  "A test agent",
					Instructions: "Do the test.",
				},
			},
		},
	}

	out, notes, err := renderer.Orchestrate(r, config, t.TempDir())
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.NotEmpty(t, out.Files, "per-resource dispatch should produce at least one file")
	_ = notes
}

// TestOrchestrate_EmptyConfig_ReturnsNonNilOutput verifies Orchestrate returns a
// non-nil output even when config has no resources.
func TestOrchestrate_EmptyConfig_ReturnsNonNilOutput(t *testing.T) {
	r := claude.New()
	config := &ast.XcaffoldConfig{
		Version: "1",
	}

	out, notes, err := renderer.Orchestrate(r, config, t.TempDir())
	require.NoError(t, err)
	require.NotNil(t, out)
	_ = notes
}

// TestOrchestrate_SignatureCompiles verifies the function signature matches
// the expected contract: (TargetRenderer, *ast.XcaffoldConfig, string) → (*output.Output, []FidelityNote, error).
// This is a compile-time check encoded as a runtime smoke test.
func TestOrchestrate_SignatureCompiles(t *testing.T) {
	r := claude.New()
	config := &ast.XcaffoldConfig{Version: "1"}

	out, _, err := renderer.Orchestrate(r, config, ".")
	require.NoError(t, err)
	assert.NotNil(t, out)
}

// mockRenderer implements TargetRenderer with all per-resource methods.
// Used to verify the orchestrator calls each per-resource method.
type mockRenderer struct {
	agentsCallCount int
	returnFiles     map[string]string
	capabilities    *renderer.CapabilitySet // nil means default {Agents: true}
}

func (m *mockRenderer) Target() string    { return "mock" }
func (m *mockRenderer) OutputDir() string { return ".mock" }
func (m *mockRenderer) Compile(_ *ast.XcaffoldConfig, _ string) (*output.Output, []renderer.FidelityNote, error) {
	panic("Compile should not be called on mockRenderer; use Orchestrate instead")
}
func (m *mockRenderer) Render(files map[string]string) *output.Output {
	return &output.Output{Files: files}
}
func (m *mockRenderer) Capabilities() renderer.CapabilitySet {
	if m.capabilities != nil {
		return *m.capabilities
	}
	return renderer.CapabilitySet{Agents: true}
}
func (m *mockRenderer) CompileAgents(_ map[string]ast.AgentConfig, _ string) (map[string]string, []renderer.FidelityNote, error) {
	m.agentsCallCount++
	if m.returnFiles != nil {
		return m.returnFiles, nil, nil
	}
	return map[string]string{"mock.txt": "mock content"}, nil, nil
}
func (m *mockRenderer) CompileSkills(_ map[string]ast.SkillConfig, _ string) (map[string]string, []renderer.FidelityNote, error) {
	return nil, nil, nil
}
func (m *mockRenderer) CompileRules(_ map[string]ast.RuleConfig, _ string) (map[string]string, []renderer.FidelityNote, error) {
	return nil, nil, nil
}
func (m *mockRenderer) CompileWorkflows(_ map[string]ast.WorkflowConfig, _ string) (map[string]string, []renderer.FidelityNote, error) {
	return nil, nil, nil
}
func (m *mockRenderer) CompileHooks(_ ast.HookConfig, _ string) (map[string]string, []renderer.FidelityNote, error) {
	return nil, nil, nil
}
func (m *mockRenderer) CompileSettings(_ ast.SettingsConfig) (map[string]string, []renderer.FidelityNote, error) {
	return nil, nil, nil
}
func (m *mockRenderer) CompileMCP(_ map[string]ast.MCPConfig) (map[string]string, []renderer.FidelityNote, error) {
	return nil, nil, nil
}
func (m *mockRenderer) CompileProjectInstructions(_ *ast.ProjectConfig, _ string) (map[string]string, []renderer.FidelityNote, error) {
	return nil, nil, nil
}
func (m *mockRenderer) Finalize(files map[string]string) (map[string]string, []renderer.FidelityNote, error) {
	return files, nil, nil
}

// TestOrchestrate_PerResourceDispatch verifies that Orchestrate dispatches to
// CompileAgents for a config with agents.
func TestOrchestrate_PerResourceDispatch(t *testing.T) {
	m := &mockRenderer{}
	config := &ast.XcaffoldConfig{
		Version: "1",
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"tester": {Name: "Tester"},
			},
		},
	}

	out, _, err := renderer.Orchestrate(m, config, ".")
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Equal(t, 1, m.agentsCallCount, "CompileAgents should be called once")
	assert.Equal(t, "mock content", out.Files["mock.txt"])
}

// TestOrchestrate_UnsupportedCapability_EmitsNote verifies that when a renderer
// does not support a resource kind, Orchestrate emits a RENDERER_KIND_UNSUPPORTED
// fidelity note for each resource of that kind.
func TestOrchestrate_UnsupportedCapability_EmitsNote(t *testing.T) {
	caps := renderer.CapabilitySet{Agents: true, Skills: false}
	m := &mockRenderer{capabilities: &caps}
	config := &ast.XcaffoldConfig{
		Version: "1",
		ResourceScope: ast.ResourceScope{
			Skills: map[string]ast.SkillConfig{
				"my-skill": {Name: "My Skill"},
			},
		},
	}

	_, notes, err := renderer.Orchestrate(m, config, ".")
	require.NoError(t, err)
	require.NotEmpty(t, notes, "expected at least one fidelity note for unsupported skill kind")

	var found bool
	for _, n := range notes {
		if n.Code == renderer.CodeRendererKindUnsupported && n.Kind == "skill" && n.Resource == "my-skill" {
			found = true
			break
		}
	}
	assert.True(t, found, "expected a RENDERER_KIND_UNSUPPORTED note with Kind=skill and Resource=my-skill")
}
