package compiler

import (
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
)

func boolPtr(b bool) *bool { return &b }

// TestMergeAgentConfig_ScalarReplace verifies that a non-zero override scalar
// replaces the base value, while base scalars not present in the override are
// preserved.
func TestMergeAgentConfig_ScalarReplace(t *testing.T) {
	base := ast.AgentConfig{
		Name:  "my-agent",
		Model: "sonnet",
	}
	override := ast.AgentConfig{
		Model: "opus",
	}

	got := mergeAgentConfig(base, override)

	if got.Model != "opus" {
		t.Errorf("Model: want %q, got %q", "opus", got.Model)
	}
	if got.Name != "my-agent" {
		t.Errorf("Name: want %q, got %q", "my-agent", got.Name)
	}
}

// TestMergeAgentConfig_ListReplace verifies that a non-empty override list
// replaces the base list entirely.
func TestMergeAgentConfig_ListReplace(t *testing.T) {
	base := ast.AgentConfig{
		Tools: ast.ClearableList{Values: []string{"Read", "Write", "Bash"}},
	}
	override := ast.AgentConfig{
		Tools: ast.ClearableList{Values: []string{"Read"}},
	}

	got := mergeAgentConfig(base, override)

	if len(got.Tools.Values) != 1 || got.Tools.Values[0] != "Read" {
		t.Errorf("Tools: want [Read], got %v", got.Tools.Values)
	}
}

// TestMergeAgentConfig_MapDeepMerge verifies that override MCPServers entries
// are merged into (not replace) the base MCPServers map: both base-only and
// override-only keys are present in the result, and override wins on conflicts.
func TestMergeAgentConfig_MapDeepMerge(t *testing.T) {
	base := ast.AgentConfig{
		MCPServers: map[string]ast.MCPConfig{
			"base-server": {Command: "base-cmd"},
		},
	}
	override := ast.AgentConfig{
		MCPServers: map[string]ast.MCPConfig{
			"override-server": {Command: "override-cmd"},
		},
	}

	got := mergeAgentConfig(base, override)

	if _, ok := got.MCPServers["base-server"]; !ok {
		t.Error("MCPServers: base-server should be preserved after merge")
	}
	if _, ok := got.MCPServers["override-server"]; !ok {
		t.Error("MCPServers: override-server should be present after merge")
	}
	if len(got.MCPServers) != 2 {
		t.Errorf("MCPServers: want 2 entries, got %d", len(got.MCPServers))
	}
}

// TestMergeAgentConfig_BodyReplaceWhenPresent verifies that a non-empty
// override Body replaces the base Body.
func TestMergeAgentConfig_BodyReplaceWhenPresent(t *testing.T) {
	base := ast.AgentConfig{
		Body: "base instructions",
	}
	override := ast.AgentConfig{
		Body: "override instructions",
	}

	got := mergeAgentConfig(base, override)

	if got.Body != "override instructions" {
		t.Errorf("Body: want %q, got %q", "override instructions", got.Body)
	}
}

// TestMergeAgentConfig_BodyInheritWhenAbsent verifies that an empty override
// Body inherits the base Body unchanged.
func TestMergeAgentConfig_BodyInheritWhenAbsent(t *testing.T) {
	base := ast.AgentConfig{
		Body: "base instructions",
	}
	override := ast.AgentConfig{
		Body: "",
	}

	got := mergeAgentConfig(base, override)

	if got.Body != "base instructions" {
		t.Errorf("Body: want %q, got %q", "base instructions", got.Body)
	}
}

// TestMergeAgentConfig_BoolPointerReplace verifies that a non-nil override
// bool pointer replaces the base value.
func TestMergeAgentConfig_BoolPointerReplace(t *testing.T) {
	base := ast.AgentConfig{
		Readonly: boolPtr(false),
	}
	override := ast.AgentConfig{
		Readonly: boolPtr(true),
	}

	got := mergeAgentConfig(base, override)

	if got.Readonly == nil {
		t.Fatal("Readonly: want non-nil pointer, got nil")
	}
	if !*got.Readonly {
		t.Errorf("Readonly: want true, got false")
	}
}

// TestMergeAgentConfig_EmptyOverride verifies that an empty override leaves all
// base fields unchanged.
func TestMergeAgentConfig_EmptyOverride(t *testing.T) {
	base := ast.AgentConfig{
		Model:    "sonnet",
		Name:     "dev",
		Tools:    ast.ClearableList{Values: []string{"Read", "Write"}},
		Body:     "instructions",
		Readonly: boolPtr(true),
	}
	override := ast.AgentConfig{}

	got := mergeAgentConfig(base, override)

	if got.Model != "sonnet" {
		t.Errorf("Model: want %q, got %q", "sonnet", got.Model)
	}
	if got.Name != "dev" {
		t.Errorf("Name: want %q, got %q", "dev", got.Name)
	}
	if len(got.Tools.Values) != 2 || got.Tools.Values[0] != "Read" || got.Tools.Values[1] != "Write" {
		t.Errorf("Tools: want [Read Write], got %v", got.Tools.Values)
	}
	if got.Body != "instructions" {
		t.Errorf("Body: want %q, got %q", "instructions", got.Body)
	}
	if got.Readonly == nil || !*got.Readonly {
		t.Errorf("Readonly: want non-nil true, got %v", got.Readonly)
	}
}

// TestMergeAgentConfig_BoolPointerNilBase verifies that a nil base bool pointer
// is overridden by a non-nil override, and that a non-nil base is preserved when
// the override pointer is nil.
func TestMergeAgentConfig_BoolPointerNilBase(t *testing.T) {
	// nil base, non-nil override → override wins
	base := ast.AgentConfig{}
	override := ast.AgentConfig{Readonly: boolPtr(true)}

	got := mergeAgentConfig(base, override)

	if got.Readonly == nil {
		t.Fatal("Readonly: want non-nil pointer, got nil (nil-base case)")
	}
	if !*got.Readonly {
		t.Errorf("Readonly: want true, got false (nil-base case)")
	}

	// non-nil base, nil override → base preserved
	base2 := ast.AgentConfig{Readonly: boolPtr(false)}
	override2 := ast.AgentConfig{}

	got2 := mergeAgentConfig(base2, override2)

	if got2.Readonly == nil {
		t.Fatal("Readonly: want non-nil pointer, got nil (nil-override case)")
	}
	if *got2.Readonly {
		t.Errorf("Readonly: want false, got true (nil-override case)")
	}
}

// TestMergeAgentConfig_MapConflictOverrideWins verifies that when base and
// override both contain the same MCPServers key, the override's value wins.
func TestMergeAgentConfig_MapConflictOverrideWins(t *testing.T) {
	base := ast.AgentConfig{
		MCPServers: map[string]ast.MCPConfig{
			"shared-key": {Command: "base-cmd"},
		},
	}
	override := ast.AgentConfig{
		MCPServers: map[string]ast.MCPConfig{
			"shared-key": {Command: "override-cmd"},
		},
	}

	got := mergeAgentConfig(base, override)

	entry, ok := got.MCPServers["shared-key"]
	if !ok {
		t.Fatal("MCPServers: shared-key should be present after merge")
	}
	if entry.Command != "override-cmd" {
		t.Errorf("MCPServers[shared-key].Command: want %q, got %q", "override-cmd", entry.Command)
	}
}

// TestMergeAgentConfig_HooksDeepMerge verifies that Hooks are deep-merged:
// disjoint event keys from base and override are both present, and on conflict
// the override value wins.
func TestMergeAgentConfig_HooksDeepMerge(t *testing.T) {
	baseGroup := []ast.HookMatcherGroup{
		{Matcher: "base-matcher", Hooks: []ast.HookHandler{{Type: "command", Command: "base-hook"}}},
	}
	overrideGroup := []ast.HookMatcherGroup{
		{Matcher: "override-matcher", Hooks: []ast.HookHandler{{Type: "command", Command: "override-hook"}}},
	}

	// Disjoint keys: both should be present after merge.
	base := ast.AgentConfig{
		Hooks: ast.HookConfig{"event-a": baseGroup},
	}
	override := ast.AgentConfig{
		Hooks: ast.HookConfig{"event-b": overrideGroup},
	}

	got := mergeAgentConfig(base, override)

	if _, ok := got.Hooks["event-a"]; !ok {
		t.Error("Hooks: event-a from base should be present after merge")
	}
	if _, ok := got.Hooks["event-b"]; !ok {
		t.Error("Hooks: event-b from override should be present after merge")
	}

	// Conflict: same key in both — override wins.
	conflictGroup := []ast.HookMatcherGroup{
		{Matcher: "conflict-matcher", Hooks: []ast.HookHandler{{Type: "command", Command: "conflict-cmd"}}},
	}
	base2 := ast.AgentConfig{
		Hooks: ast.HookConfig{"event-a": baseGroup},
	}
	override2 := ast.AgentConfig{
		Hooks: ast.HookConfig{"event-a": conflictGroup},
	}

	got2 := mergeAgentConfig(base2, override2)

	groups, ok := got2.Hooks["event-a"]
	if !ok {
		t.Fatal("Hooks: event-a should be present after conflict merge")
	}
	if len(groups) == 0 || groups[0].Matcher != "conflict-matcher" {
		t.Errorf("Hooks[event-a]: want override matcher %q, got %v", "conflict-matcher", groups)
	}
}

// TestMergeAgentConfig_TargetsDeepMerge verifies that Targets are deep-merged:
// disjoint provider keys from base and override are both present after merge.
func TestMergeAgentConfig_TargetsDeepMerge(t *testing.T) {
	base := ast.AgentConfig{
		Targets: map[string]ast.TargetOverride{
			"claude": {Provider: map[string]any{"key": "claude-val"}},
		},
	}
	override := ast.AgentConfig{
		Targets: map[string]ast.TargetOverride{
			"gemini": {Provider: map[string]any{"key": "gemini-val"}},
		},
	}

	got := mergeAgentConfig(base, override)

	if _, ok := got.Targets["claude"]; !ok {
		t.Error("Targets: claude from base should be present after merge")
	}
	if _, ok := got.Targets["gemini"]; !ok {
		t.Error("Targets: gemini from override should be present after merge")
	}
	if len(got.Targets) != 2 {
		t.Errorf("Targets: want 2 entries, got %d", len(got.Targets))
	}
}

// TestMergeAgentConfig_BodyOnlyOverride verifies that when the override contains
// only a Body (all other fields are zero), the Body is replaced while all other
// base fields — Model, Tools, etc. — are preserved unchanged.
func TestMergeAgentConfig_BodyOnlyOverride(t *testing.T) {
	base := ast.AgentConfig{
		Name:  "developer",
		Model: "sonnet",
		Tools: ast.ClearableList{Values: []string{"Bash", "Read", "Write"}},
		Body:  "Universal instructions.",
	}
	override := ast.AgentConfig{
		Body: "Provider-specific instructions.",
	}

	got := mergeAgentConfig(base, override)

	if got.Model != "sonnet" {
		t.Errorf("Model: want %q, got %q", "sonnet", got.Model)
	}
	if len(got.Tools.Values) != 3 {
		t.Errorf("Tools: want 3 elements, got %d: %v", len(got.Tools.Values), got.Tools.Values)
	}
	if got.Body != "Provider-specific instructions." {
		t.Errorf("Body: want %q, got %q", "Provider-specific instructions.", got.Body)
	}
}

// TestMergeAgentConfig_FrontmatterOnlyOverride verifies that when the override
// contains only a frontmatter scalar (Model) with an empty Body, the Model is
// replaced while the base Body is inherited unchanged.
func TestMergeAgentConfig_FrontmatterOnlyOverride(t *testing.T) {
	base := ast.AgentConfig{
		Name:  "developer",
		Model: "sonnet",
		Body:  "Universal instructions.",
	}
	override := ast.AgentConfig{
		Model: "opus",
	}

	got := mergeAgentConfig(base, override)

	if got.Model != "opus" {
		t.Errorf("Model: want %q, got %q", "opus", got.Model)
	}
	if got.Body != "Universal instructions." {
		t.Errorf("Body: want %q, got %q", "Universal instructions.", got.Body)
	}
}

// TestMergeSkillConfig_ScalarReplace verifies that a non-zero override scalar
// replaces the base value, while base scalars not present in the override are
// preserved.
func TestMergeSkillConfig_ScalarReplace(t *testing.T) {
	base := ast.SkillConfig{
		Name:      "my-skill",
		WhenToUse: "base description of when to use",
	}
	override := ast.SkillConfig{
		WhenToUse: "override description of when to use",
	}

	got := mergeSkillConfig(base, override)

	if got.WhenToUse != "override description of when to use" {
		t.Errorf("WhenToUse: want %q, got %q", "override description of when to use", got.WhenToUse)
	}
	if got.Name != "my-skill" {
		t.Errorf("Name: want %q, got %q", "my-skill", got.Name)
	}
}

// TestMergeRuleConfig_BodyReplace verifies that a non-empty override Body
// replaces the base Body.
func TestMergeRuleConfig_BodyReplace(t *testing.T) {
	base := ast.RuleConfig{
		Body: "base rule body",
	}
	override := ast.RuleConfig{
		Body: "override rule body",
	}

	got := mergeRuleConfig(base, override)

	if got.Body != "override rule body" {
		t.Errorf("Body: want %q, got %q", "override rule body", got.Body)
	}
}

// TestMergeWorkflowConfig_StepsReplace verifies that a non-empty override Steps
// list replaces the base Steps list entirely (list semantics, not append).
func TestMergeWorkflowConfig_StepsReplace(t *testing.T) {
	base := ast.WorkflowConfig{
		Steps: []ast.WorkflowStep{
			{Name: "step-a"},
			{Name: "step-b"},
		},
	}
	override := ast.WorkflowConfig{
		Steps: []ast.WorkflowStep{
			{Name: "step-override"},
		},
	}

	got := mergeWorkflowConfig(base, override)

	if len(got.Steps) != 1 {
		t.Fatalf("Steps: want 1 step, got %d", len(got.Steps))
	}
	if got.Steps[0].Name != "step-override" {
		t.Errorf("Steps[0].Name: want %q, got %q", "step-override", got.Steps[0].Name)
	}
}

// TestMergeSkillConfig_AllowedToolsReplace verifies that a non-empty override
// AllowedTools list replaces the base AllowedTools list entirely (list semantics,
// not append).
func TestMergeSkillConfig_AllowedToolsReplace(t *testing.T) {
	base := ast.SkillConfig{
		Name:         "my-skill",
		AllowedTools: ast.ClearableList{Values: []string{"Read", "Write", "Bash"}},
	}
	override := ast.SkillConfig{
		AllowedTools: ast.ClearableList{Values: []string{"Read"}},
	}

	got := mergeSkillConfig(base, override)

	if len(got.AllowedTools.Values) != 1 || got.AllowedTools.Values[0] != "Read" {
		t.Errorf("AllowedTools: want [Read], got %v", got.AllowedTools.Values)
	}
	if got.Name != "my-skill" {
		t.Errorf("Name: want %q, got %q", "my-skill", got.Name)
	}
}

// TestMergeRuleConfig_AlwaysApplyBoolPointer verifies that a non-nil override
// AlwaysApply bool pointer replaces the base value.
func TestMergeRuleConfig_AlwaysApplyBoolPointer(t *testing.T) {
	base := ast.RuleConfig{
		Name:        "security",
		AlwaysApply: boolPtr(false),
	}
	override := ast.RuleConfig{
		AlwaysApply: boolPtr(true),
	}

	got := mergeRuleConfig(base, override)

	if got.AlwaysApply == nil {
		t.Fatal("AlwaysApply: want non-nil pointer, got nil")
	}
	if !*got.AlwaysApply {
		t.Errorf("AlwaysApply: want true, got false")
	}
	if got.Name != "security" {
		t.Errorf("Name: want %q, got %q", "security", got.Name)
	}
}

// TestMergeMCPConfig_HeadersDeepMerge verifies that base.Headers and
// override.Headers are deep merged: both keys are preserved, and on conflict
// the override value wins.
func TestMergeMCPConfig_HeadersDeepMerge(t *testing.T) {
	base := ast.MCPConfig{
		Headers: map[string]string{
			"X-Base-Header":   "base-val",
			"X-Shared-Header": "base-shared",
		},
	}
	override := ast.MCPConfig{
		Headers: map[string]string{
			"X-Override-Header": "override-val",
			"X-Shared-Header":   "override-shared",
		},
	}

	got := mergeMCPConfig(base, override)

	if got.Headers["X-Base-Header"] != "base-val" {
		t.Errorf("Headers[X-Base-Header]: want %q, got %q", "base-val", got.Headers["X-Base-Header"])
	}
	if got.Headers["X-Override-Header"] != "override-val" {
		t.Errorf("Headers[X-Override-Header]: want %q, got %q", "override-val", got.Headers["X-Override-Header"])
	}
	if got.Headers["X-Shared-Header"] != "override-shared" {
		t.Errorf("Headers[X-Shared-Header]: want %q (override wins), got %q", "override-shared", got.Headers["X-Shared-Header"])
	}
	if len(got.Headers) != 3 {
		t.Errorf("Headers: want 3 entries, got %d", len(got.Headers))
	}
}

// TestMergeMCPConfig_EnvDeepMerge verifies that base.Env and override.Env are
// deep merged: both keys are preserved, and on conflict the override value wins.
func TestMergeMCPConfig_EnvDeepMerge(t *testing.T) {
	base := ast.MCPConfig{
		Env: map[string]string{
			"BASE_KEY":   "base-val",
			"SHARED_KEY": "base-shared",
		},
	}
	override := ast.MCPConfig{
		Env: map[string]string{
			"OVERRIDE_KEY": "override-val",
			"SHARED_KEY":   "override-shared",
		},
	}

	got := mergeMCPConfig(base, override)

	if got.Env["BASE_KEY"] != "base-val" {
		t.Errorf("Env[BASE_KEY]: want %q, got %q", "base-val", got.Env["BASE_KEY"])
	}
	if got.Env["OVERRIDE_KEY"] != "override-val" {
		t.Errorf("Env[OVERRIDE_KEY]: want %q, got %q", "override-val", got.Env["OVERRIDE_KEY"])
	}
	if got.Env["SHARED_KEY"] != "override-shared" {
		t.Errorf("Env[SHARED_KEY]: want %q (override wins), got %q", "override-shared", got.Env["SHARED_KEY"])
	}
	if len(got.Env) != 3 {
		t.Errorf("Env: want 3 entries, got %d", len(got.Env))
	}
}
