package renderer

import (
	"fmt"
	"path/filepath"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/output"
)

// Orchestrate compiles config using r by dispatching to each per-resource method
// individually. RENDERER_KIND_UNSUPPORTED fidelity notes are emitted for resource
// kinds the renderer does not support according to its Capabilities declaration.
func Orchestrate(r TargetRenderer, config *ast.XcaffoldConfig, baseDir string) (*output.Output, []FidelityNote, error) {
	caps := r.Capabilities()
	out := &output.Output{
		Files:     make(map[string]string),
		RootFiles: make(map[string]string),
	}
	var notes []FidelityNote

	// Pre-compute memory agent refs for renderers that support MemoryAwareRenderer.
	if mar, ok := r.(MemoryAwareRenderer); ok && len(config.Memory) > 0 {
		refs := make(map[string]bool)
		for _, m := range config.Memory {
			if m.AgentRef != "" {
				refs[m.AgentRef] = true
			}
		}
		mar.SetMemoryRefs(refs)
	}

	// Agents
	if len(config.Agents) > 0 {
		filtered := make(map[string]ast.AgentConfig)
		for id, agent := range config.Agents {
			if !isSkipSynthesis(agent.Targets, r.Target()) {
				filtered[id] = agent
			}
		}
		if len(filtered) > 0 {
			if caps.Agents {
				files, n, err := r.CompileAgents(filtered, baseDir)
				if err != nil {
					return nil, nil, fmt.Errorf("CompileAgents: %w", err)
				}
				mergeFiles(out, files)
				notes = append(notes, n...)
				for id, agent := range filtered {
					present := ExtractAgentPresentFields(agent)
					suppressed := isSuppressed(agent.Targets, r.Target())
					notes = append(notes, CheckFieldSupport(r.Target(), "agent", id, present, suppressed)...)
				}
			} else {
				for _, id := range SortedKeys(filtered) {
					notes = append(notes, NewNote(
						LevelWarning, r.Target(), "agent", id, "",
						CodeRendererKindUnsupported,
						"agents are not supported by this renderer",
						"",
					))
				}
			}
		}
	}

	// Skills
	if len(config.Skills) > 0 {
		filtered := make(map[string]ast.SkillConfig)
		for id, skill := range config.Skills {
			if !isSkipSynthesis(skill.Targets, r.Target()) {
				filtered[id] = skill
			}
		}
		if len(filtered) > 0 {
			if caps.Skills {
				files, n, err := r.CompileSkills(filtered, baseDir)
				if err != nil {
					return nil, nil, fmt.Errorf("CompileSkills: %w", err)
				}
				mergeFiles(out, files)
				notes = append(notes, n...)
				for id, skill := range filtered {
					present := ExtractSkillPresentFields(skill)
					suppressed := isSuppressed(skill.Targets, r.Target())
					notes = append(notes, CheckFieldSupport(r.Target(), "skill", id, present, suppressed)...)
				}
			} else {
				for _, id := range SortedKeys(filtered) {
					notes = append(notes, NewNote(
						LevelWarning, r.Target(), "skill", id, "",
						CodeRendererKindUnsupported,
						"skills are not supported by this renderer",
						"",
					))
				}
			}
		}
	}

	// Rules
	if len(config.Rules) > 0 {
		filtered := make(map[string]ast.RuleConfig)
		for id, rule := range config.Rules {
			if !isSkipSynthesis(rule.Targets, r.Target()) {
				filtered[id] = rule
			}
		}
		if len(filtered) > 0 {
			if caps.Rules {
				files, n, err := r.CompileRules(filtered, baseDir)
				if err != nil {
					return nil, nil, fmt.Errorf("CompileRules: %w", err)
				}
				mergeFiles(out, files)
				notes = append(notes, n...)
				for id, rule := range filtered {
					present := ExtractRulePresentFields(rule)
					suppressed := isSuppressed(rule.Targets, r.Target())
					notes = append(notes, CheckFieldSupport(r.Target(), "rule", id, present, suppressed)...)
				}
			} else {
				for _, id := range SortedKeys(filtered) {
					notes = append(notes, NewNote(
						LevelWarning, r.Target(), "rule", id, "",
						CodeRendererKindUnsupported,
						"rules are not supported by this renderer",
						"",
					))
				}
			}
		}
	}

	// Workflows
	if len(config.Workflows) > 0 {
		filtered := make(map[string]ast.WorkflowConfig)
		for id, wf := range config.Workflows {
			if !isSkipSynthesis(wf.Targets, r.Target()) {
				filtered[id] = wf
			}
		}
		if len(filtered) > 0 {
			if caps.Workflows {
				files, n, err := r.CompileWorkflows(filtered, baseDir)
				if err != nil {
					return nil, nil, fmt.Errorf("CompileWorkflows: %w", err)
				}
				mergeFiles(out, files)
				notes = append(notes, n...)
			} else {
				for _, id := range SortedKeys(filtered) {
					notes = append(notes, NewNote(
						LevelWarning, r.Target(), "workflow", id, "",
						CodeRendererKindUnsupported,
						"workflows are not supported by this renderer",
						"",
					))
				}
			}
		}
	}

	// Hooks — XcaffoldConfig.Hooks is map[string]NamedHookConfig; the canonical
	// entry is "default". The per-resource method receives the merged HookConfig
	// (event-name → matcher-group slice) from the default named block.
	var mergedHooks ast.HookConfig
	if dh, ok := config.Hooks["default"]; ok && len(dh.Events) > 0 {
		// NamedHookConfig also has Targets field.
		if !isSkipSynthesis(dh.Targets, r.Target()) {
			mergedHooks = dh.Events
		}
	}
	if len(mergedHooks) > 0 {
		if caps.Hooks {
			files, n, err := r.CompileHooks(mergedHooks, baseDir)
			if err != nil {
				return nil, nil, fmt.Errorf("CompileHooks: %w", err)
			}
			mergeFiles(out, files)
			notes = append(notes, n...)
		} else {
			for _, event := range SortedKeys(mergedHooks) {
				notes = append(notes, NewNote(
					LevelWarning, r.Target(), "hook", string(event), "",
					CodeRendererKindUnsupported,
					"hooks are not supported by this renderer",
					"",
				))
			}
		}
	}

	// Hook artifacts — copy script files from xcaf/hooks/<name>/ to provider output.
	if caps.Hooks {
		for hookKey, hook := range config.Hooks {
			if len(hook.Artifacts) == 0 {
				continue
			}
			if isSkipSynthesis(hook.Targets, r.Target()) {
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
				return nil, nil, fmt.Errorf("hook artifacts %s: %w", name, err)
			}
			mergeFiles(out, artifactFiles)
		}
	}

	// Settings — canonical entry is "default".
	settings, hasSettings := config.Settings["default"]
	if hasSettings {
		if !isSkipSynthesis(settings.Targets, r.Target()) {
			if caps.Settings {
				files, n, err := r.CompileSettings(settings)
				if err != nil {
					return nil, nil, fmt.Errorf("CompileSettings: %w", err)
				}
				mergeFiles(out, files)
				notes = append(notes, n...)
			} else {
				notes = append(notes, NewNote(
					LevelWarning, r.Target(), "settings", "default", "",
					CodeRendererKindUnsupported,
					"settings are not supported by this renderer",
					"",
				))
			}
		}
	}

	// MCP servers
	if len(config.MCP) > 0 {
		filtered := make(map[string]ast.MCPConfig)
		for id, mcp := range config.MCP {
			if !isSkipSynthesis(mcp.Targets, r.Target()) {
				filtered[id] = mcp
			}
		}
		if len(filtered) > 0 {
			if caps.MCP {
				files, n, err := r.CompileMCP(filtered)
				if err != nil {
					return nil, nil, fmt.Errorf("CompileMCP: %w", err)
				}
				mergeFiles(out, files)
				notes = append(notes, n...)
			} else {
				for _, id := range SortedKeys(filtered) {
					notes = append(notes, NewNote(
						LevelWarning, r.Target(), "mcp", id, "",
						CodeRendererKindUnsupported,
						"MCP servers are not supported by this renderer",
						"",
					))
				}
			}
		}
	}

	// Project instructions
	if config.Project != nil {
		if caps.ProjectInstructions {
			files, rootFiles, n, err := r.CompileProjectInstructions(config, baseDir)
			if err != nil {
				return nil, nil, fmt.Errorf("CompileProjectInstructions: %w", err)
			}
			mergeFiles(out, files)
			mergeRootFiles(out, rootFiles)
			notes = append(notes, n...)
		} else {
			notes = append(notes, NewNote(
				LevelWarning, r.Target(), "project", config.Project.Name, "",
				CodeRendererKindUnsupported,
				"project instructions are not supported by this renderer",
				"",
			))
		}
	}

	// Memory
	if len(config.Memory) > 0 {
		if caps.Memory {
			files, n, err := r.CompileMemory(config, baseDir, MemoryOptions{})
			if err != nil {
				return nil, nil, fmt.Errorf("CompileMemory: %w", err)
			}
			mergeFiles(out, files)
			notes = append(notes, n...)
		} else {
			for _, id := range SortedKeys(config.Memory) {
				notes = append(notes, NewNote(
					LevelWarning, r.Target(), "memory", id, "",
					CodeRendererKindUnsupported,
					"memory entries are not supported by this renderer",
					"",
				))
			}
		}
	}

	// Finalize: post-processing pass (path normalization, dedup, etc.)
	finalized, finalizedRoot, finalNotes, err := r.Finalize(out.Files, out.RootFiles)
	if err != nil {
		return nil, nil, fmt.Errorf("Finalize: %w", err)
	}
	out.Files = finalized
	out.RootFiles = finalizedRoot
	notes = append(notes, finalNotes...)

	return out, notes, nil
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
