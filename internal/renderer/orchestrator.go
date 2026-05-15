package renderer

import (
	"fmt"
	"path/filepath"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/output"
)

// renderCtx bundles the renderer, config, baseDir, caps, and output accumulator
// so they can be passed as a single receiver rather than as individual parameters.
type renderCtx struct {
	r       TargetRenderer
	config  *ast.XcaffoldConfig
	baseDir string
	caps    CapabilitySet
	out     *output.Output
}

// Orchestrate compiles config using r by dispatching to each per-resource method
// individually. RENDERER_KIND_UNSUPPORTED fidelity notes are emitted for resource
// kinds the renderer does not support according to its Capabilities declaration.
func Orchestrate(r TargetRenderer, config *ast.XcaffoldConfig, baseDir string) (*output.Output, []FidelityNote, error) {
	ctx := renderCtx{
		r:       r,
		config:  config,
		baseDir: baseDir,
		caps:    r.Capabilities(),
		out: &output.Output{
			Files:     make(map[string]string),
			RootFiles: make(map[string]string),
		},
	}

	setMemoryRefs(r, config)

	notes, err := ctx.runAllResources()
	if err != nil {
		return nil, nil, err
	}

	finalized, finalizedRoot, finalNotes, err := r.Finalize(ctx.out.Files, ctx.out.RootFiles)
	if err != nil {
		return nil, nil, fmt.Errorf("Finalize: %w", err)
	}
	ctx.out.Files = finalized
	ctx.out.RootFiles = finalizedRoot
	return ctx.out, append(notes, finalNotes...), nil
}

// runAllResources dispatches every resource-kind renderer in the canonical order
// and collects their fidelity notes. runHookArtifacts is the only step that does
// not return notes; its error is propagated directly.
func (ctx renderCtx) runAllResources() ([]FidelityNote, error) {
	var notes []FidelityNote

	steps := []func() ([]FidelityNote, error){
		ctx.runAgents,
		ctx.runSkills,
		ctx.runRules,
		ctx.runWorkflows,
		ctx.runHooks,
		ctx.runSettings,
		ctx.runMCP,
		ctx.runProject,
		ctx.runMemory,
	}

	for _, step := range steps {
		n, err := step()
		if err != nil {
			return nil, err
		}
		notes = append(notes, n...)
	}

	if err := ctx.runHookArtifacts(); err != nil {
		return nil, err
	}

	return notes, nil
}

// setMemoryRefs pre-computes memory agent refs for renderers that implement
// MemoryAwareRenderer.
func setMemoryRefs(r TargetRenderer, config *ast.XcaffoldConfig) {
	mar, ok := r.(MemoryAwareRenderer)
	if !ok || len(config.Memory) == 0 {
		return
	}
	refs := make(map[string]bool)
	for _, m := range config.Memory {
		if m.AgentRef != "" {
			refs[m.AgentRef] = true
		}
	}
	mar.SetMemoryRefs(refs)
}

// runAgents filters agents for the target and compiles them, or emits
// RENDERER_KIND_UNSUPPORTED notes when the renderer has no agent capability.
func (ctx renderCtx) runAgents() ([]FidelityNote, error) {
	if len(ctx.config.Agents) == 0 {
		return nil, nil
	}
	filtered := filterMap(ctx.config.Agents, ctx.r.Target(), func(v ast.AgentConfig) map[string]ast.TargetOverride { return v.Targets })
	if len(filtered) == 0 {
		return nil, nil
	}
	if !ctx.caps.Agents {
		return unsupportedNotes(ctx.r.Target(), "agent", SortedKeys(filtered), "agents are not supported by this renderer"), nil
	}
	files, notes, err := ctx.r.CompileAgents(filtered, ctx.baseDir)
	if err != nil {
		return nil, fmt.Errorf("CompileAgents: %w", err)
	}
	mergeFiles(ctx.out, files)
	for id, agent := range filtered {
		present := ExtractAgentPresentFields(agent)
		suppressed := isSuppressed(agent.Targets, ctx.r.Target())
		notes = append(notes, CheckFieldSupport(FieldCheckInput{Target: ctx.r.Target(), Kind: "agent", ResourceName: id, PresentFields: present, Suppressed: suppressed})...)
	}
	return notes, nil
}

// runSkills filters skills for the target and compiles them, or emits
// RENDERER_KIND_UNSUPPORTED notes when the renderer has no skill capability.
func (ctx renderCtx) runSkills() ([]FidelityNote, error) {
	if len(ctx.config.Skills) == 0 {
		return nil, nil
	}
	filtered := filterMap(ctx.config.Skills, ctx.r.Target(), func(v ast.SkillConfig) map[string]ast.TargetOverride { return v.Targets })
	if len(filtered) == 0 {
		return nil, nil
	}
	if !ctx.caps.Skills {
		return unsupportedNotes(ctx.r.Target(), "skill", SortedKeys(filtered), "skills are not supported by this renderer"), nil
	}
	files, notes, err := ctx.r.CompileSkills(filtered, ctx.baseDir)
	if err != nil {
		return nil, fmt.Errorf("CompileSkills: %w", err)
	}
	mergeFiles(ctx.out, files)
	for id, skill := range filtered {
		present := ExtractSkillPresentFields(skill)
		suppressed := isSuppressed(skill.Targets, ctx.r.Target())
		notes = append(notes, CheckFieldSupport(FieldCheckInput{Target: ctx.r.Target(), Kind: "skill", ResourceName: id, PresentFields: present, Suppressed: suppressed})...)
	}
	return notes, nil
}

// runRules filters rules for the target and compiles them, or emits
// RENDERER_KIND_UNSUPPORTED notes when the renderer has no rule capability.
func (ctx renderCtx) runRules() ([]FidelityNote, error) {
	if len(ctx.config.Rules) == 0 {
		return nil, nil
	}
	filtered := filterMap(ctx.config.Rules, ctx.r.Target(), func(v ast.RuleConfig) map[string]ast.TargetOverride { return v.Targets })
	if len(filtered) == 0 {
		return nil, nil
	}
	if !ctx.caps.Rules {
		return unsupportedNotes(ctx.r.Target(), "rule", SortedKeys(filtered), "rules are not supported by this renderer"), nil
	}
	files, notes, err := ctx.r.CompileRules(filtered, ctx.baseDir)
	if err != nil {
		return nil, fmt.Errorf("CompileRules: %w", err)
	}
	mergeFiles(ctx.out, files)
	for id, rule := range filtered {
		present := ExtractRulePresentFields(rule)
		suppressed := isSuppressed(rule.Targets, ctx.r.Target())
		notes = append(notes, CheckFieldSupport(FieldCheckInput{Target: ctx.r.Target(), Kind: "rule", ResourceName: id, PresentFields: present, Suppressed: suppressed})...)
	}
	return notes, nil
}

// runWorkflows filters workflows for the target and compiles them, or emits
// RENDERER_KIND_UNSUPPORTED notes when the renderer has no workflow capability.
func (ctx renderCtx) runWorkflows() ([]FidelityNote, error) {
	if len(ctx.config.Workflows) == 0 {
		return nil, nil
	}
	filtered := filterMap(ctx.config.Workflows, ctx.r.Target(), func(v ast.WorkflowConfig) map[string]ast.TargetOverride { return v.Targets })
	if len(filtered) == 0 {
		return nil, nil
	}
	if !ctx.caps.Workflows {
		return unsupportedNotes(ctx.r.Target(), "workflow", SortedKeys(filtered), "workflows are not supported by this renderer"), nil
	}
	files, notes, err := ctx.r.CompileWorkflows(filtered, ctx.baseDir)
	if err != nil {
		return nil, fmt.Errorf("CompileWorkflows: %w", err)
	}
	mergeFiles(ctx.out, files)
	return notes, nil
}

// resolveSettingsEntry returns the active settings entry and its key.
// It prefers the "default" key. If "default" is absent but exactly one
// entry exists (e.g., after blueprint filtering), that entry is used.
// Returns zero-value and "" if no entry is available.
func resolveSettingsEntry(m map[string]ast.SettingsConfig) (ast.SettingsConfig, string) {
	if s, ok := m["default"]; ok {
		return s, "default"
	}
	if len(m) == 1 {
		for k, v := range m {
			return v, k
		}
	}
	return ast.SettingsConfig{}, ""
}

// resolveHooksEntry returns the active hooks entry and its key.
// Prefers "default"; falls back to the sole entry if exactly one exists.
func resolveHooksEntry(m map[string]ast.NamedHookConfig) (ast.NamedHookConfig, string) {
	if h, ok := m["default"]; ok {
		return h, "default"
	}
	if len(m) == 1 {
		for k, v := range m {
			return v, k
		}
	}
	return ast.NamedHookConfig{}, ""
}

// runHooks extracts the active hook block and compiles it, or emits
// RENDERER_KIND_UNSUPPORTED notes per event when the renderer has no hook capability.
func (ctx renderCtx) runHooks() ([]FidelityNote, error) {
	dh, key := resolveHooksEntry(ctx.config.Hooks)
	if key == "" || len(dh.Events) == 0 || isSkipSynthesis(dh.Targets, ctx.r.Target()) {
		return nil, nil
	}
	mergedHooks := dh.Events
	if !ctx.caps.Hooks {
		var notes []FidelityNote
		for _, event := range SortedKeys(mergedHooks) {
			notes = append(notes, FidelityNote{
				Level:    LevelWarning,
				Target:   ctx.r.Target(),
				Kind:     "hook",
				Resource: event,
				Code:     CodeRendererKindUnsupported,
				Reason:   "hooks are not supported by this renderer",
			})
		}
		return notes, nil
	}
	files, notes, err := ctx.r.CompileHooks(mergedHooks, ctx.baseDir)
	if err != nil {
		return nil, fmt.Errorf("CompileHooks: %w", err)
	}
	mergeFiles(ctx.out, files)
	return notes, nil
}

// runHookArtifacts copies script files from xcaf/hooks/<name>/ to provider output
// for renderers that support hooks.
func (ctx renderCtx) runHookArtifacts() error {
	if !ctx.caps.Hooks {
		return nil
	}
	for hookKey, hook := range ctx.config.Hooks {
		if len(hook.Artifacts) == 0 || isSkipSynthesis(hook.Targets, ctx.r.Target()) {
			continue
		}
		name := hook.Name
		if name == "" {
			name = hookKey
		}
		hookSrcDir := filepath.Join(ctx.baseDir, "xcaf", "hooks", name)
		hookDstDir := filepath.Join(ctx.r.OutputDir(), "hooks")
		artifactFiles, err := CompileHookArtifacts(name, hook.Artifacts, hookSrcDir, hookDstDir)
		if err != nil {
			return fmt.Errorf("hook artifacts %s: %w", name, err)
		}
		mergeFiles(ctx.out, artifactFiles)
	}
	return nil
}

// runSettings compiles the active settings block, or emits a
// RENDERER_KIND_UNSUPPORTED note when the renderer has no settings capability.
func (ctx renderCtx) runSettings() ([]FidelityNote, error) {
	settings, key := resolveSettingsEntry(ctx.config.Settings)
	if key == "" || isSkipSynthesis(settings.Targets, ctx.r.Target()) {
		return nil, nil
	}
	if !ctx.caps.Settings {
		return []FidelityNote{{
			Level:    LevelWarning,
			Target:   ctx.r.Target(),
			Kind:     "settings",
			Resource: key,
			Code:     CodeRendererKindUnsupported,
			Reason:   "settings are not supported by this renderer",
		}}, nil
	}
	files, notes, err := ctx.r.CompileSettings(settings)
	if err != nil {
		return nil, fmt.Errorf("CompileSettings: %w", err)
	}
	mergeFiles(ctx.out, files)
	return notes, nil
}

// runMCP filters MCP servers for the target and compiles them, or emits
// RENDERER_KIND_UNSUPPORTED notes when the renderer has no MCP capability.
func (ctx renderCtx) runMCP() ([]FidelityNote, error) {
	if len(ctx.config.MCP) == 0 {
		return nil, nil
	}
	filtered := filterMap(ctx.config.MCP, ctx.r.Target(), func(v ast.MCPConfig) map[string]ast.TargetOverride { return v.Targets })
	if len(filtered) == 0 {
		return nil, nil
	}
	if !ctx.caps.MCP {
		return unsupportedNotes(ctx.r.Target(), "mcp", SortedKeys(filtered), "MCP servers are not supported by this renderer"), nil
	}
	files, notes, err := ctx.r.CompileMCP(filtered)
	if err != nil {
		return nil, fmt.Errorf("CompileMCP: %w", err)
	}
	mergeFiles(ctx.out, files)
	return notes, nil
}

// runProject compiles project instructions, or emits a RENDERER_KIND_UNSUPPORTED
// note when the renderer has no project-instructions capability.
func (ctx renderCtx) runProject() ([]FidelityNote, error) {
	if ctx.config.Project == nil {
		return nil, nil
	}
	if !ctx.caps.ProjectInstructions {
		return []FidelityNote{{
			Level:    LevelWarning,
			Target:   ctx.r.Target(),
			Kind:     "project",
			Resource: ctx.config.Project.Name,
			Code:     CodeRendererKindUnsupported,
			Reason:   "project instructions are not supported by this renderer",
		}}, nil
	}
	files, rootFiles, notes, err := ctx.r.CompileProjectInstructions(ctx.config, ctx.baseDir)
	if err != nil {
		return nil, fmt.Errorf("CompileProjectInstructions: %w", err)
	}
	mergeFiles(ctx.out, files)
	mergeRootFiles(ctx.out, rootFiles)
	return notes, nil
}

// runMemory compiles memory entries, or emits RENDERER_KIND_UNSUPPORTED notes
// when the renderer has no memory capability.
func (ctx renderCtx) runMemory() ([]FidelityNote, error) {
	if len(ctx.config.Memory) == 0 {
		return nil, nil
	}
	if !ctx.caps.Memory {
		return unsupportedNotes(ctx.r.Target(), "memory", SortedKeys(ctx.config.Memory), "memory entries are not supported by this renderer"), nil
	}
	files, notes, err := ctx.r.CompileMemory(ctx.config, ctx.baseDir, MemoryOptions{})
	if err != nil {
		return nil, fmt.Errorf("CompileMemory: %w", err)
	}
	mergeFiles(ctx.out, files)
	return notes, nil
}

// filterMap returns a new map containing only the entries in src whose target
// override does not have skip-synthesis set for the given target.
func filterMap[V any](src map[string]V, target string, getTargets func(V) map[string]ast.TargetOverride) map[string]V {
	result := make(map[string]V, len(src))
	for id, v := range src {
		if !isSkipSynthesis(getTargets(v), target) {
			result[id] = v
		}
	}
	return result
}

// unsupportedNotes builds RENDERER_KIND_UNSUPPORTED fidelity notes for each id.
func unsupportedNotes(target, kind string, ids []string, msg string) []FidelityNote {
	notes := make([]FidelityNote, 0, len(ids))
	for _, id := range ids {
		notes = append(notes, FidelityNote{
			Level:    LevelWarning,
			Target:   target,
			Kind:     kind,
			Resource: id,
			Code:     CodeRendererKindUnsupported,
			Reason:   msg,
		})
	}
	return notes
}

// isSkipSynthesis returns true when the resource's target override for the given
// provider has skip-synthesis set to true.
func isSkipSynthesis(targets map[string]ast.TargetOverride, target string) bool {
	if targets == nil {
		return false
	}
	to, ok := targets[target]
	if !ok {
		return false
	}
	return to.SkipSynthesis != nil && *to.SkipSynthesis
}

// mergeFiles copies all entries from files into out.Files. Existing keys are
// overwritten; callers must ensure method ordering is deterministic.
func mergeFiles(out *output.Output, files map[string]string) {
	for k, v := range files {
		out.Files[k] = v
	}
}

// mergeRootFiles copies all entries from rootFiles into out.RootFiles. Existing
// keys are overwritten; callers must ensure method ordering is deterministic.
func mergeRootFiles(out *output.Output, rootFiles map[string]string) {
	for k, v := range rootFiles {
		out.RootFiles[k] = v
	}
}
