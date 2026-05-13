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
	r, config, baseDir, caps, out := ctx.r, ctx.config, ctx.baseDir, ctx.caps, ctx.out

	steps := []func() ([]FidelityNote, error){
		func() ([]FidelityNote, error) { return runAgents(r, config, baseDir, caps, out) },
		func() ([]FidelityNote, error) { return runSkills(r, config, baseDir, caps, out) },
		func() ([]FidelityNote, error) { return runRules(r, config, baseDir, caps, out) },
		func() ([]FidelityNote, error) { return runWorkflows(r, config, baseDir, caps, out) },
		func() ([]FidelityNote, error) { return runHooks(r, config, baseDir, caps, out) },
		func() ([]FidelityNote, error) { return runSettings(r, config, caps, out) },
		func() ([]FidelityNote, error) { return runMCP(r, config, caps, out) },
		func() ([]FidelityNote, error) { return runProject(r, config, baseDir, caps, out) },
		func() ([]FidelityNote, error) { return runMemory(r, config, baseDir, caps, out) },
	}

	for _, step := range steps {
		n, err := step()
		if err != nil {
			return nil, err
		}
		notes = append(notes, n...)
	}

	if err := runHookArtifacts(r, config, baseDir, caps, out); err != nil {
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
func runAgents(r TargetRenderer, config *ast.XcaffoldConfig, baseDir string, caps CapabilitySet, out *output.Output) ([]FidelityNote, error) {
	if len(config.Agents) == 0 {
		return nil, nil
	}
	filtered := filterMap(config.Agents, r.Target(), func(v ast.AgentConfig) map[string]ast.TargetOverride { return v.Targets })
	if len(filtered) == 0 {
		return nil, nil
	}
	if !caps.Agents {
		return unsupportedNotes(r.Target(), "agent", SortedKeys(filtered), "agents are not supported by this renderer"), nil
	}
	files, notes, err := r.CompileAgents(filtered, baseDir)
	if err != nil {
		return nil, fmt.Errorf("CompileAgents: %w", err)
	}
	mergeFiles(out, files)
	for id, agent := range filtered {
		present := ExtractAgentPresentFields(agent)
		suppressed := isSuppressed(agent.Targets, r.Target())
		notes = append(notes, CheckFieldSupport(r.Target(), "agent", id, present, suppressed)...)
	}
	return notes, nil
}

// runSkills filters skills for the target and compiles them, or emits
// RENDERER_KIND_UNSUPPORTED notes when the renderer has no skill capability.
func runSkills(r TargetRenderer, config *ast.XcaffoldConfig, baseDir string, caps CapabilitySet, out *output.Output) ([]FidelityNote, error) {
	if len(config.Skills) == 0 {
		return nil, nil
	}
	filtered := filterMap(config.Skills, r.Target(), func(v ast.SkillConfig) map[string]ast.TargetOverride { return v.Targets })
	if len(filtered) == 0 {
		return nil, nil
	}
	if !caps.Skills {
		return unsupportedNotes(r.Target(), "skill", SortedKeys(filtered), "skills are not supported by this renderer"), nil
	}
	files, notes, err := r.CompileSkills(filtered, baseDir)
	if err != nil {
		return nil, fmt.Errorf("CompileSkills: %w", err)
	}
	mergeFiles(out, files)
	for id, skill := range filtered {
		present := ExtractSkillPresentFields(skill)
		suppressed := isSuppressed(skill.Targets, r.Target())
		notes = append(notes, CheckFieldSupport(r.Target(), "skill", id, present, suppressed)...)
	}
	return notes, nil
}

// runRules filters rules for the target and compiles them, or emits
// RENDERER_KIND_UNSUPPORTED notes when the renderer has no rule capability.
func runRules(r TargetRenderer, config *ast.XcaffoldConfig, baseDir string, caps CapabilitySet, out *output.Output) ([]FidelityNote, error) {
	if len(config.Rules) == 0 {
		return nil, nil
	}
	filtered := filterMap(config.Rules, r.Target(), func(v ast.RuleConfig) map[string]ast.TargetOverride { return v.Targets })
	if len(filtered) == 0 {
		return nil, nil
	}
	if !caps.Rules {
		return unsupportedNotes(r.Target(), "rule", SortedKeys(filtered), "rules are not supported by this renderer"), nil
	}
	files, notes, err := r.CompileRules(filtered, baseDir)
	if err != nil {
		return nil, fmt.Errorf("CompileRules: %w", err)
	}
	mergeFiles(out, files)
	for id, rule := range filtered {
		present := ExtractRulePresentFields(rule)
		suppressed := isSuppressed(rule.Targets, r.Target())
		notes = append(notes, CheckFieldSupport(r.Target(), "rule", id, present, suppressed)...)
	}
	return notes, nil
}

// runWorkflows filters workflows for the target and compiles them, or emits
// RENDERER_KIND_UNSUPPORTED notes when the renderer has no workflow capability.
func runWorkflows(r TargetRenderer, config *ast.XcaffoldConfig, baseDir string, caps CapabilitySet, out *output.Output) ([]FidelityNote, error) {
	if len(config.Workflows) == 0 {
		return nil, nil
	}
	filtered := filterMap(config.Workflows, r.Target(), func(v ast.WorkflowConfig) map[string]ast.TargetOverride { return v.Targets })
	if len(filtered) == 0 {
		return nil, nil
	}
	if !caps.Workflows {
		return unsupportedNotes(r.Target(), "workflow", SortedKeys(filtered), "workflows are not supported by this renderer"), nil
	}
	files, notes, err := r.CompileWorkflows(filtered, baseDir)
	if err != nil {
		return nil, fmt.Errorf("CompileWorkflows: %w", err)
	}
	mergeFiles(out, files)
	return notes, nil
}

// runHooks extracts the default hook block and compiles it, or emits
// RENDERER_KIND_UNSUPPORTED notes per event when the renderer has no hook capability.
func runHooks(r TargetRenderer, config *ast.XcaffoldConfig, baseDir string, caps CapabilitySet, out *output.Output) ([]FidelityNote, error) {
	dh, ok := config.Hooks["default"]
	if !ok || len(dh.Events) == 0 || isSkipSynthesis(dh.Targets, r.Target()) {
		return nil, nil
	}
	mergedHooks := dh.Events
	if !caps.Hooks {
		var notes []FidelityNote
		for _, event := range SortedKeys(mergedHooks) {
			notes = append(notes, NewNote(
				LevelWarning, r.Target(), "hook", string(event), "",
				CodeRendererKindUnsupported,
				"hooks are not supported by this renderer",
				"",
			))
		}
		return notes, nil
	}
	files, notes, err := r.CompileHooks(mergedHooks, baseDir)
	if err != nil {
		return nil, fmt.Errorf("CompileHooks: %w", err)
	}
	mergeFiles(out, files)
	return notes, nil
}

// runHookArtifacts copies script files from xcaf/hooks/<name>/ to provider output
// for renderers that support hooks.
func runHookArtifacts(r TargetRenderer, config *ast.XcaffoldConfig, baseDir string, caps CapabilitySet, out *output.Output) error {
	if !caps.Hooks {
		return nil
	}
	for hookKey, hook := range config.Hooks {
		if len(hook.Artifacts) == 0 || isSkipSynthesis(hook.Targets, r.Target()) {
			continue
		}
		name := hook.Name
		if name == "" {
			name = hookKey
		}
		hookSrcDir := filepath.Join(baseDir, "xcaf", "hooks", name)
		hookDstDir := filepath.Join(r.OutputDir(), "hooks")
		artifactFiles, err := CompileHookArtifacts(name, hook.Artifacts, hookSrcDir, hookDstDir)
		if err != nil {
			return fmt.Errorf("hook artifacts %s: %w", name, err)
		}
		mergeFiles(out, artifactFiles)
	}
	return nil
}

// runSettings compiles the default settings block, or emits a
// RENDERER_KIND_UNSUPPORTED note when the renderer has no settings capability.
func runSettings(r TargetRenderer, config *ast.XcaffoldConfig, caps CapabilitySet, out *output.Output) ([]FidelityNote, error) {
	settings, ok := config.Settings["default"]
	if !ok || isSkipSynthesis(settings.Targets, r.Target()) {
		return nil, nil
	}
	if !caps.Settings {
		return []FidelityNote{NewNote(
			LevelWarning, r.Target(), "settings", "default", "",
			CodeRendererKindUnsupported,
			"settings are not supported by this renderer",
			"",
		)}, nil
	}
	files, notes, err := r.CompileSettings(settings)
	if err != nil {
		return nil, fmt.Errorf("CompileSettings: %w", err)
	}
	mergeFiles(out, files)
	return notes, nil
}

// runMCP filters MCP servers for the target and compiles them, or emits
// RENDERER_KIND_UNSUPPORTED notes when the renderer has no MCP capability.
func runMCP(r TargetRenderer, config *ast.XcaffoldConfig, caps CapabilitySet, out *output.Output) ([]FidelityNote, error) {
	if len(config.MCP) == 0 {
		return nil, nil
	}
	filtered := filterMap(config.MCP, r.Target(), func(v ast.MCPConfig) map[string]ast.TargetOverride { return v.Targets })
	if len(filtered) == 0 {
		return nil, nil
	}
	if !caps.MCP {
		return unsupportedNotes(r.Target(), "mcp", SortedKeys(filtered), "MCP servers are not supported by this renderer"), nil
	}
	files, notes, err := r.CompileMCP(filtered)
	if err != nil {
		return nil, fmt.Errorf("CompileMCP: %w", err)
	}
	mergeFiles(out, files)
	return notes, nil
}

// runProject compiles project instructions, or emits a RENDERER_KIND_UNSUPPORTED
// note when the renderer has no project-instructions capability.
func runProject(r TargetRenderer, config *ast.XcaffoldConfig, baseDir string, caps CapabilitySet, out *output.Output) ([]FidelityNote, error) {
	if config.Project == nil {
		return nil, nil
	}
	if !caps.ProjectInstructions {
		return []FidelityNote{NewNote(
			LevelWarning, r.Target(), "project", config.Project.Name, "",
			CodeRendererKindUnsupported,
			"project instructions are not supported by this renderer",
			"",
		)}, nil
	}
	files, rootFiles, notes, err := r.CompileProjectInstructions(config, baseDir)
	if err != nil {
		return nil, fmt.Errorf("CompileProjectInstructions: %w", err)
	}
	mergeFiles(out, files)
	mergeRootFiles(out, rootFiles)
	return notes, nil
}

// runMemory compiles memory entries, or emits RENDERER_KIND_UNSUPPORTED notes
// when the renderer has no memory capability.
func runMemory(r TargetRenderer, config *ast.XcaffoldConfig, baseDir string, caps CapabilitySet, out *output.Output) ([]FidelityNote, error) {
	if len(config.Memory) == 0 {
		return nil, nil
	}
	if !caps.Memory {
		return unsupportedNotes(r.Target(), "memory", SortedKeys(config.Memory), "memory entries are not supported by this renderer"), nil
	}
	files, notes, err := r.CompileMemory(config, baseDir, MemoryOptions{})
	if err != nil {
		return nil, fmt.Errorf("CompileMemory: %w", err)
	}
	mergeFiles(out, files)
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
		notes = append(notes, NewNote(
			LevelWarning, target, kind, id, "",
			CodeRendererKindUnsupported,
			msg,
			"",
		))
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
