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

// --- mergeNamedHookConfig ---

// TestMergeNamedHookConfig_ScalarReplace verifies that non-empty override scalars
// replace base values and that base-only scalars are preserved.
func TestMergeNamedHookConfig_ScalarReplace(t *testing.T) {
	base := ast.NamedHookConfig{
		Name:        "base-hook",
		Description: "base description",
	}
	override := ast.NamedHookConfig{
		Description: "override description",
	}

	got := mergeNamedHookConfig(base, override)

	if got.Name != "base-hook" {
		t.Errorf("Name: want %q, got %q", "base-hook", got.Name)
	}
	if got.Description != "override description" {
		t.Errorf("Description: want %q, got %q", "override description", got.Description)
	}
}

// TestMergeNamedHookConfig_BasePreserved verifies that a zero-value override
// leaves all base fields unchanged.
func TestMergeNamedHookConfig_BasePreserved(t *testing.T) {
	base := ast.NamedHookConfig{
		Name:        "my-hook",
		Description: "base",
		Artifacts:   []string{"scripts/"},
	}
	override := ast.NamedHookConfig{}

	got := mergeNamedHookConfig(base, override)

	if got.Name != "my-hook" {
		t.Errorf("Name: want %q, got %q", "my-hook", got.Name)
	}
	if got.Description != "base" {
		t.Errorf("Description: want %q, got %q", "base", got.Description)
	}
	if len(got.Artifacts) != 1 || got.Artifacts[0] != "scripts/" {
		t.Errorf("Artifacts: want [scripts/], got %v", got.Artifacts)
	}
}

// TestMergeNamedHookConfig_ArtifactsReplace verifies that a non-empty override
// Artifacts list replaces the base list entirely.
func TestMergeNamedHookConfig_ArtifactsReplace(t *testing.T) {
	base := ast.NamedHookConfig{
		Artifacts: []string{"scripts/", "helpers/"},
	}
	override := ast.NamedHookConfig{
		Artifacts: []string{"override-scripts/"},
	}

	got := mergeNamedHookConfig(base, override)

	if len(got.Artifacts) != 1 || got.Artifacts[0] != "override-scripts/" {
		t.Errorf("Artifacts: want [override-scripts/], got %v", got.Artifacts)
	}
}

// TestMergeNamedHookConfig_EventsDeepMerge verifies that Events are deep-merged:
// disjoint event keys are preserved, and on conflict the override wins.
func TestMergeNamedHookConfig_EventsDeepMerge(t *testing.T) {
	baseGroup := []ast.HookMatcherGroup{
		{Matcher: "base-matcher", Hooks: []ast.HookHandler{{Type: "command", Command: "base-cmd"}}},
	}
	overrideGroup := []ast.HookMatcherGroup{
		{Matcher: "override-matcher", Hooks: []ast.HookHandler{{Type: "command", Command: "override-cmd"}}},
	}

	base := ast.NamedHookConfig{
		Events: ast.HookConfig{"pre-commit": baseGroup},
	}
	override := ast.NamedHookConfig{
		Events: ast.HookConfig{"post-commit": overrideGroup},
	}

	got := mergeNamedHookConfig(base, override)

	if _, ok := got.Events["pre-commit"]; !ok {
		t.Error("Events: pre-commit from base should be present after merge")
	}
	if _, ok := got.Events["post-commit"]; !ok {
		t.Error("Events: post-commit from override should be present after merge")
	}
}

// TestMergeNamedHookConfig_TargetsDeepMerge verifies that Targets are deep-merged.
func TestMergeNamedHookConfig_TargetsDeepMerge(t *testing.T) {
	base := ast.NamedHookConfig{
		Targets: map[string]ast.TargetOverride{"claude": {}},
	}
	override := ast.NamedHookConfig{
		Targets: map[string]ast.TargetOverride{"gemini": {}},
	}

	got := mergeNamedHookConfig(base, override)

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

// TestMergeNamedHookConfig_ProvenanceNotMerged verifies that Inherited and
// SourceProvider are not merged.
func TestMergeNamedHookConfig_ProvenanceNotMerged(t *testing.T) {
	base := ast.NamedHookConfig{
		Name:           "hook",
		Inherited:      true,
		SourceProvider: "claude",
	}
	override := ast.NamedHookConfig{
		Inherited:      false,
		SourceProvider: "gemini",
	}

	got := mergeNamedHookConfig(base, override)

	if !got.Inherited {
		t.Error("Inherited: base provenance should be preserved (not merged from override)")
	}
	if got.SourceProvider != "claude" {
		t.Errorf("SourceProvider: want %q (base preserved), got %q", "claude", got.SourceProvider)
	}
}

// --- mergePolicyConfig ---

// TestMergePolicyConfig_ScalarReplace verifies that non-empty override scalars
// replace base values.
func TestMergePolicyConfig_ScalarReplace(t *testing.T) {
	base := ast.PolicyConfig{
		Name:     "my-policy",
		Severity: "warning",
		Target:   "agent",
	}
	override := ast.PolicyConfig{
		Severity: "error",
	}

	got := mergePolicyConfig(base, override)

	if got.Severity != "error" {
		t.Errorf("Severity: want %q, got %q", "error", got.Severity)
	}
	if got.Name != "my-policy" {
		t.Errorf("Name: want %q, got %q", "my-policy", got.Name)
	}
	if got.Target != "agent" {
		t.Errorf("Target: want %q, got %q", "agent", got.Target)
	}
}

// TestMergePolicyConfig_BasePreserved verifies that a zero-value override
// leaves all base fields unchanged.
func TestMergePolicyConfig_BasePreserved(t *testing.T) {
	require := []ast.PolicyRequire{{Field: "model"}}
	base := ast.PolicyConfig{
		Name:     "my-policy",
		Severity: "error",
		Target:   "agent",
		Require:  require,
	}
	override := ast.PolicyConfig{}

	got := mergePolicyConfig(base, override)

	if got.Name != "my-policy" {
		t.Errorf("Name: want %q, got %q", "my-policy", got.Name)
	}
	if got.Severity != "error" {
		t.Errorf("Severity: want %q, got %q", "error", got.Severity)
	}
	if len(got.Require) != 1 {
		t.Errorf("Require: want 1 entry, got %d", len(got.Require))
	}
}

// TestMergePolicyConfig_MatchReplaceOnNonNil verifies that a non-nil override Match
// replaces the base Match.
func TestMergePolicyConfig_MatchReplaceOnNonNil(t *testing.T) {
	base := ast.PolicyConfig{
		Match: &ast.PolicyMatch{HasTool: "Bash"},
	}
	override := ast.PolicyConfig{
		Match: &ast.PolicyMatch{HasTool: "Read"},
	}

	got := mergePolicyConfig(base, override)

	if got.Match == nil {
		t.Fatal("Match: want non-nil, got nil")
	}
	if got.Match.HasTool != "Read" {
		t.Errorf("Match.HasTool: want %q, got %q", "Read", got.Match.HasTool)
	}
}

// TestMergePolicyConfig_RequireReplace verifies that a non-empty override Require
// slice replaces the base slice.
func TestMergePolicyConfig_RequireReplace(t *testing.T) {
	base := ast.PolicyConfig{
		Require: []ast.PolicyRequire{{Field: "model"}, {Field: "description"}},
	}
	override := ast.PolicyConfig{
		Require: []ast.PolicyRequire{{Field: "name"}},
	}

	got := mergePolicyConfig(base, override)

	if len(got.Require) != 1 || got.Require[0].Field != "name" {
		t.Errorf("Require: want [{name}], got %v", got.Require)
	}
}

// TestMergePolicyConfig_ProvenanceNotMerged verifies that provenance fields are
// not merged.
func TestMergePolicyConfig_ProvenanceNotMerged(t *testing.T) {
	base := ast.PolicyConfig{
		Name:           "policy",
		Inherited:      true,
		SourceProvider: "claude",
	}
	override := ast.PolicyConfig{
		Inherited:      false,
		SourceProvider: "gemini",
	}

	got := mergePolicyConfig(base, override)

	if !got.Inherited {
		t.Error("Inherited: base provenance should be preserved")
	}
	if got.SourceProvider != "claude" {
		t.Errorf("SourceProvider: want %q (base), got %q", "claude", got.SourceProvider)
	}
}

// --- mergeTemplateConfig ---

// TestMergeTemplateConfig_ScalarReplace verifies that non-empty override scalars
// replace base values and base-only scalars are preserved.
func TestMergeTemplateConfig_ScalarReplace(t *testing.T) {
	base := ast.TemplateConfig{
		Name:          "my-template",
		DefaultTarget: "claude",
	}
	override := ast.TemplateConfig{
		DefaultTarget: "gemini",
	}

	got := mergeTemplateConfig(base, override)

	if got.DefaultTarget != "gemini" {
		t.Errorf("DefaultTarget: want %q, got %q", "gemini", got.DefaultTarget)
	}
	if got.Name != "my-template" {
		t.Errorf("Name: want %q, got %q", "my-template", got.Name)
	}
}

// TestMergeTemplateConfig_BasePreserved verifies that a zero-value override
// leaves all base fields unchanged.
func TestMergeTemplateConfig_BasePreserved(t *testing.T) {
	base := ast.TemplateConfig{
		Name:          "tmpl",
		Description:   "desc",
		DefaultTarget: "claude",
		Body:          "template body",
	}
	override := ast.TemplateConfig{}

	got := mergeTemplateConfig(base, override)

	if got.Name != "tmpl" {
		t.Errorf("Name: want %q, got %q", "tmpl", got.Name)
	}
	if got.Description != "desc" {
		t.Errorf("Description: want %q, got %q", "desc", got.Description)
	}
	if got.DefaultTarget != "claude" {
		t.Errorf("DefaultTarget: want %q, got %q", "claude", got.DefaultTarget)
	}
	if got.Body != "template body" {
		t.Errorf("Body: want %q, got %q", "template body", got.Body)
	}
}

// TestMergeTemplateConfig_BodyReplace verifies that a non-empty override Body
// replaces the base Body.
func TestMergeTemplateConfig_BodyReplace(t *testing.T) {
	base := ast.TemplateConfig{Body: "base body"}
	override := ast.TemplateConfig{Body: "override body"}

	got := mergeTemplateConfig(base, override)

	if got.Body != "override body" {
		t.Errorf("Body: want %q, got %q", "override body", got.Body)
	}
}

// TestMergeTemplateConfig_ProvenanceNotMerged verifies that provenance fields are
// not merged.
func TestMergeTemplateConfig_ProvenanceNotMerged(t *testing.T) {
	base := ast.TemplateConfig{
		Name:           "tmpl",
		Inherited:      true,
		SourceProvider: "claude",
	}
	override := ast.TemplateConfig{
		Inherited:      false,
		SourceProvider: "gemini",
	}

	got := mergeTemplateConfig(base, override)

	if !got.Inherited {
		t.Error("Inherited: base provenance should be preserved")
	}
	if got.SourceProvider != "claude" {
		t.Errorf("SourceProvider: want %q (base), got %q", "claude", got.SourceProvider)
	}
}

// --- mergeContextConfig ---

// TestMergeContextConfig_ScalarReplace verifies that non-empty override scalars
// replace base values and base-only scalars are preserved.
func TestMergeContextConfig_ScalarReplace(t *testing.T) {
	base := ast.ContextConfig{
		Name:        "my-context",
		Description: "base description",
	}
	override := ast.ContextConfig{
		Description: "override description",
	}

	got := mergeContextConfig(base, override)

	if got.Description != "override description" {
		t.Errorf("Description: want %q, got %q", "override description", got.Description)
	}
	if got.Name != "my-context" {
		t.Errorf("Name: want %q, got %q", "my-context", got.Name)
	}
}

// TestMergeContextConfig_BasePreserved verifies that a zero-value override
// leaves all base fields unchanged.
func TestMergeContextConfig_BasePreserved(t *testing.T) {
	base := ast.ContextConfig{
		Name:        "ctx",
		Description: "desc",
		Default:     true,
		Targets:     []string{"claude", "gemini"},
		Body:        "context body",
	}
	override := ast.ContextConfig{}

	got := mergeContextConfig(base, override)

	if got.Name != "ctx" {
		t.Errorf("Name: want %q, got %q", "ctx", got.Name)
	}
	if !got.Default {
		t.Error("Default: want true, got false")
	}
	if len(got.Targets) != 2 {
		t.Errorf("Targets: want 2 entries, got %d", len(got.Targets))
	}
	if got.Body != "context body" {
		t.Errorf("Body: want %q, got %q", "context body", got.Body)
	}
}

// TestMergeContextConfig_TargetsReplace verifies that a non-empty override Targets
// slice replaces the base slice entirely (list semantics).
func TestMergeContextConfig_TargetsReplace(t *testing.T) {
	base := ast.ContextConfig{
		Targets: []string{"claude", "gemini"},
	}
	override := ast.ContextConfig{
		Targets: []string{"copilot"},
	}

	got := mergeContextConfig(base, override)

	if len(got.Targets) != 1 || got.Targets[0] != "copilot" {
		t.Errorf("Targets: want [copilot], got %v", got.Targets)
	}
}

// TestMergeContextConfig_ProvenanceNotMerged verifies that provenance fields are
// not merged.
func TestMergeContextConfig_ProvenanceNotMerged(t *testing.T) {
	base := ast.ContextConfig{
		Name:           "ctx",
		Inherited:      true,
		SourceProvider: "claude",
	}
	override := ast.ContextConfig{
		Inherited:      false,
		SourceProvider: "gemini",
	}

	got := mergeContextConfig(base, override)

	if !got.Inherited {
		t.Error("Inherited: base provenance should be preserved")
	}
	if got.SourceProvider != "claude" {
		t.Errorf("SourceProvider: want %q (base), got %q", "claude", got.SourceProvider)
	}
}

// --- mergeSettingsConfig ---

// TestMergeSettingsConfig_ScalarReplace verifies that non-empty override scalars
// replace base values and base-only scalars are preserved.
func TestMergeSettingsConfig_ScalarReplace(t *testing.T) {
	base := ast.SettingsConfig{
		Name:  "my-settings",
		Model: "sonnet",
	}
	override := ast.SettingsConfig{
		Model: "opus",
	}

	got := mergeSettingsConfig(base, override)

	if got.Model != "opus" {
		t.Errorf("Model: want %q, got %q", "opus", got.Model)
	}
	if got.Name != "my-settings" {
		t.Errorf("Name: want %q, got %q", "my-settings", got.Name)
	}
}

// TestMergeSettingsConfig_BasePreserved verifies that a zero-value override
// leaves all base fields unchanged.
func TestMergeSettingsConfig_BasePreserved(t *testing.T) {
	base := ast.SettingsConfig{
		Name:        "settings",
		Model:       "sonnet",
		EffortLevel: "high",
	}
	override := ast.SettingsConfig{}

	got := mergeSettingsConfig(base, override)

	if got.Name != "settings" {
		t.Errorf("Name: want %q, got %q", "settings", got.Name)
	}
	if got.Model != "sonnet" {
		t.Errorf("Model: want %q, got %q", "sonnet", got.Model)
	}
	if got.EffortLevel != "high" {
		t.Errorf("EffortLevel: want %q, got %q", "high", got.EffortLevel)
	}
}

// TestMergeSettingsConfig_BoolPtrReplace verifies that a non-nil override bool
// pointer replaces the base value.
func TestMergeSettingsConfig_BoolPtrReplace(t *testing.T) {
	base := ast.SettingsConfig{
		AutoMemoryEnabled: boolPtr(false),
	}
	override := ast.SettingsConfig{
		AutoMemoryEnabled: boolPtr(true),
	}

	got := mergeSettingsConfig(base, override)

	if got.AutoMemoryEnabled == nil {
		t.Fatal("AutoMemoryEnabled: want non-nil, got nil")
	}
	if !*got.AutoMemoryEnabled {
		t.Error("AutoMemoryEnabled: want true, got false")
	}
}

// TestMergeSettingsConfig_CleanupPeriodDays verifies that a non-nil
// CleanupPeriodDays *int is copied correctly.
func TestMergeSettingsConfig_CleanupPeriodDays(t *testing.T) {
	base := ast.SettingsConfig{
		CleanupPeriodDays: intPtr(30),
	}
	override := ast.SettingsConfig{
		CleanupPeriodDays: intPtr(7),
	}

	got := mergeSettingsConfig(base, override)

	if got.CleanupPeriodDays == nil {
		t.Fatal("CleanupPeriodDays: want non-nil, got nil")
	}
	if *got.CleanupPeriodDays != 7 {
		t.Errorf("CleanupPeriodDays: want 7, got %d", *got.CleanupPeriodDays)
	}
}

// TestMergeSettingsConfig_EnvDeepMerge verifies that Env maps are deep-merged:
// both keys are preserved, and on conflict the override wins.
func TestMergeSettingsConfig_EnvDeepMerge(t *testing.T) {
	base := ast.SettingsConfig{
		Env: map[string]string{"BASE_KEY": "base-val", "SHARED": "base"},
	}
	override := ast.SettingsConfig{
		Env: map[string]string{"OVERRIDE_KEY": "override-val", "SHARED": "override"},
	}

	got := mergeSettingsConfig(base, override)

	if got.Env["BASE_KEY"] != "base-val" {
		t.Errorf("Env[BASE_KEY]: want %q, got %q", "base-val", got.Env["BASE_KEY"])
	}
	if got.Env["OVERRIDE_KEY"] != "override-val" {
		t.Errorf("Env[OVERRIDE_KEY]: want %q, got %q", "override-val", got.Env["OVERRIDE_KEY"])
	}
	if got.Env["SHARED"] != "override" {
		t.Errorf("Env[SHARED]: want %q (override wins), got %q", "override", got.Env["SHARED"])
	}
}

// TestMergeSettingsConfig_MCPServersDeepMerge verifies that MCPServers are
// deep-merged: disjoint keys are preserved, and on conflict override wins.
func TestMergeSettingsConfig_MCPServersDeepMerge(t *testing.T) {
	base := ast.SettingsConfig{
		MCPServers: map[string]ast.MCPConfig{
			"base-server": {Command: "base-cmd"},
		},
	}
	override := ast.SettingsConfig{
		MCPServers: map[string]ast.MCPConfig{
			"override-server": {Command: "override-cmd"},
		},
	}

	got := mergeSettingsConfig(base, override)

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

func TestMergeAgentConfig_MaxTurnsPointer_ZeroOverride(t *testing.T) {
	base := ast.AgentConfig{MaxTurns: intPtr(10)}
	override := ast.AgentConfig{MaxTurns: intPtr(0)}

	got := mergeAgentConfig(base, override)

	if got.MaxTurns == nil || *got.MaxTurns != 0 {
		t.Errorf("MaxTurns: want ptr(0), got %v", got.MaxTurns)
	}
}

func TestMergeAgentConfig_MaxTurnsPointer_NilPreservesBase(t *testing.T) {
	base := ast.AgentConfig{MaxTurns: intPtr(10)}
	override := ast.AgentConfig{}

	got := mergeAgentConfig(base, override)

	if got.MaxTurns == nil || *got.MaxTurns != 10 {
		t.Errorf("MaxTurns: want ptr(10), got %v", got.MaxTurns)
	}
}

func TestMergeSettingsConfig_ProvenanceNotMerged(t *testing.T) {
	base := ast.SettingsConfig{
		Name:           "settings",
		Inherited:      true,
		SourceProvider: "claude",
	}
	override := ast.SettingsConfig{
		Inherited:      false,
		SourceProvider: "gemini",
	}

	got := mergeSettingsConfig(base, override)

	if !got.Inherited {
		t.Error("Inherited: base provenance should be preserved")
	}
	if got.SourceProvider != "claude" {
		t.Errorf("SourceProvider: want %q (base), got %q", "claude", got.SourceProvider)
	}
}
