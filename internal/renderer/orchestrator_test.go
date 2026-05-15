package renderer_test

import (
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/renderer"
	"github.com/saero-ai/xcaffold/providers/claude"
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
					Name:        "Tester",
					Description: "A test agent",
					Body:        "Do the test.",
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
func (m *mockRenderer) CompileProjectInstructions(_ *ast.XcaffoldConfig, _ string) (map[string]string, map[string]string, []renderer.FidelityNote, error) {
	return nil, nil, nil, nil
}
func (m *mockRenderer) CompileMemory(_ *ast.XcaffoldConfig, _ string, _ renderer.MemoryOptions) (map[string]string, []renderer.FidelityNote, error) {
	return map[string]string{}, nil, nil
}
func (m *mockRenderer) Finalize(files map[string]string, rootFiles map[string]string) (map[string]string, map[string]string, []renderer.FidelityNote, error) {
	return files, rootFiles, nil, nil
}

// TestOrchestrate_PerResourceDispatch verifies that Orchestrate dispatches to
// CompileAgents for a config with agents.
func TestOrchestrate_PerResourceDispatch(t *testing.T) {
	m := &mockRenderer{}
	config := &ast.XcaffoldConfig{
		Version: "1",
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"tester": {Name: "Tester", Description: "test agent"},
			},
		},
	}

	out, _, err := renderer.Orchestrate(m, config, ".")
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Equal(t, 1, m.agentsCallCount, "CompileAgents should be called once")
	assert.Equal(t, "mock content", out.Files["mock.txt"])
}

// TestOrchestrate_ErrorsOnMissingRequiredField verifies that Orchestrate emits a
// FIELD_REQUIRED_FOR_TARGET error-level note when a required field (description)
// is absent from an agent targeting claude.
func TestOrchestrate_ErrorsOnMissingRequiredField(t *testing.T) {
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"test": {Name: "test"},
			},
		},
	}
	r := claude.New()
	_, notes, err := renderer.Orchestrate(r, config, ".")
	if err != nil {
		t.Fatalf("Orchestrate failed: %v", err)
	}
	found := false
	for _, n := range notes {
		if n.Code == renderer.CodeFieldRequiredForTarget && n.Field == "description" {
			found = true
			if n.Level != renderer.LevelError {
				t.Errorf("expected LevelError, got %s", n.Level)
			}
		}
	}
	if !found {
		t.Error("expected FIELD_REQUIRED_FOR_TARGET for missing description on agent")
	}
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

// settingsTrackingRenderer wraps mockRenderer and records CompileSettings calls.
type settingsTrackingRenderer struct {
	mockRenderer
	settingsCallCount int
	lastSettings      ast.SettingsConfig
}

func (s *settingsTrackingRenderer) CompileSettings(cfg ast.SettingsConfig) (map[string]string, []renderer.FidelityNote, error) {
	s.settingsCallCount++
	s.lastSettings = cfg
	return map[string]string{"settings.json": "{}"}, nil, nil
}

func (s *settingsTrackingRenderer) Capabilities() renderer.CapabilitySet {
	return renderer.CapabilitySet{Settings: true}
}

// hooksTrackingRenderer wraps mockRenderer and records CompileHooks calls.
type hooksTrackingRenderer struct {
	mockRenderer
	hooksCallCount int
}

func (h *hooksTrackingRenderer) CompileHooks(cfg ast.HookConfig, _ string) (map[string]string, []renderer.FidelityNote, error) {
	h.hooksCallCount++
	return map[string]string{"hooks.sh": "#!/bin/sh"}, nil, nil
}

func (h *hooksTrackingRenderer) Capabilities() renderer.CapabilitySet {
	return renderer.CapabilitySet{Hooks: true}
}

// TestResolveSettingsEntry_NonDefaultKey_IsCompiled verifies that when the settings
// map has a single non-"default" key (as happens after blueprint filtering), that
// entry is resolved and CompileSettings is called.
func TestResolveSettingsEntry_NonDefaultKey_IsCompiled(t *testing.T) {
	m := &settingsTrackingRenderer{}
	config := &ast.XcaffoldConfig{
		Version: "1",
		Settings: map[string]ast.SettingsConfig{
			"blueprint-settings": {Name: "blueprint-settings"},
		},
	}

	_, _, err := renderer.Orchestrate(m, config, t.TempDir())
	require.NoError(t, err)
	assert.Equal(t, 1, m.settingsCallCount, "CompileSettings should be called once for a single non-default entry")
}

// TestResolveSettingsEntry_DefaultKey_IsPreferred verifies that when both "default"
// and another key exist, the "default" entry is used.
func TestResolveSettingsEntry_DefaultKey_IsPreferred(t *testing.T) {
	m := &settingsTrackingRenderer{}
	wantName := "main-settings"
	config := &ast.XcaffoldConfig{
		Version: "1",
		Settings: map[string]ast.SettingsConfig{
			"default":     {Name: wantName},
			"other-entry": {Name: "should-not-be-used"},
		},
	}

	_, _, err := renderer.Orchestrate(m, config, t.TempDir())
	require.NoError(t, err)
	assert.Equal(t, 1, m.settingsCallCount, "CompileSettings should be called exactly once")
	assert.Equal(t, wantName, m.lastSettings.Name, "should compile the 'default' entry's settings")
}

// TestResolveSettingsEntry_EmptyMap_IsSkipped verifies that an empty settings map
// results in no CompileSettings call.
func TestResolveSettingsEntry_EmptyMap_IsSkipped(t *testing.T) {
	m := &settingsTrackingRenderer{}
	config := &ast.XcaffoldConfig{
		Version: "1",
	}

	_, _, err := renderer.Orchestrate(m, config, t.TempDir())
	require.NoError(t, err)
	assert.Equal(t, 0, m.settingsCallCount, "CompileSettings should not be called when settings map is empty")
}

// TestResolveSettingsEntry_MultipleNonDefaultEntries_IsSkipped verifies that when
// the settings map has more than one non-"default" entry, the resolver cannot
// determine which one to use and skips compilation (ambiguous case).
func TestResolveSettingsEntry_MultipleNonDefaultEntries_IsSkipped(t *testing.T) {
	m := &settingsTrackingRenderer{}
	config := &ast.XcaffoldConfig{
		Version: "1",
		Settings: map[string]ast.SettingsConfig{
			"entry-a": {Name: "entry-a"},
			"entry-b": {Name: "entry-b"},
		},
	}

	_, _, err := renderer.Orchestrate(m, config, t.TempDir())
	require.NoError(t, err)
	assert.Equal(t, 0, m.settingsCallCount, "CompileSettings should not be called when multiple non-default entries are ambiguous")
}

// TestResolveHooksEntry_NonDefaultKey_IsCompiled verifies that when the hooks map
// has a single non-"default" key, that entry is resolved and CompileHooks is called.
func TestResolveHooksEntry_NonDefaultKey_IsCompiled(t *testing.T) {
	m := &hooksTrackingRenderer{}
	config := &ast.XcaffoldConfig{
		Version: "1",
		Hooks: map[string]ast.NamedHookConfig{
			"blueprint-hooks": {
				Name:   "blueprint-hooks",
				Events: ast.HookConfig{"PreToolUse": nil},
			},
		},
	}

	_, _, err := renderer.Orchestrate(m, config, t.TempDir())
	require.NoError(t, err)
	assert.Equal(t, 1, m.hooksCallCount, "CompileHooks should be called once for a single non-default entry")
}

// TestResolveHooksEntry_DefaultKey_IsPreferred verifies that when both "default"
// and another key exist, the "default" hook entry is used.
func TestResolveHooksEntry_DefaultKey_IsPreferred(t *testing.T) {
	m := &hooksTrackingRenderer{}
	config := &ast.XcaffoldConfig{
		Version: "1",
		Hooks: map[string]ast.NamedHookConfig{
			"default": {
				Name:   "default",
				Events: ast.HookConfig{"PreToolUse": nil},
			},
			"other-hooks": {
				Name:   "other-hooks",
				Events: ast.HookConfig{"PostToolUse": nil},
			},
		},
	}

	_, _, err := renderer.Orchestrate(m, config, t.TempDir())
	require.NoError(t, err)
	assert.Equal(t, 1, m.hooksCallCount, "CompileHooks should be called exactly once for the default entry")
}

// TestResolveHooksEntry_EmptyMap_IsSkipped verifies that an empty hooks map
// results in no CompileHooks call.
func TestResolveHooksEntry_EmptyMap_IsSkipped(t *testing.T) {
	m := &hooksTrackingRenderer{}
	config := &ast.XcaffoldConfig{
		Version: "1",
	}

	_, _, err := renderer.Orchestrate(m, config, t.TempDir())
	require.NoError(t, err)
	assert.Equal(t, 0, m.hooksCallCount, "CompileHooks should not be called when hooks map is empty")
}

// TestResolveHooksEntry_MultipleNonDefaultEntries_IsSkipped verifies that when
// the hooks map has multiple non-"default" entries, the resolver skips compilation.
func TestResolveHooksEntry_MultipleNonDefaultEntries_IsSkipped(t *testing.T) {
	m := &hooksTrackingRenderer{}
	config := &ast.XcaffoldConfig{
		Version: "1",
		Hooks: map[string]ast.NamedHookConfig{
			"hooks-a": {Name: "hooks-a", Events: ast.HookConfig{"PreToolUse": nil}},
			"hooks-b": {Name: "hooks-b", Events: ast.HookConfig{"PostToolUse": nil}},
		},
	}

	_, _, err := renderer.Orchestrate(m, config, t.TempDir())
	require.NoError(t, err)
	assert.Equal(t, 0, m.hooksCallCount, "CompileHooks should not be called when multiple non-default entries are ambiguous")
}
