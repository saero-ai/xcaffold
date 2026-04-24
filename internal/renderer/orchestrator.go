package renderer

import (
	"fmt"

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

	// Agents
	if len(config.Agents) > 0 {
		if caps.Agents {
			files, n, err := r.CompileAgents(config.Agents, baseDir)
			if err != nil {
				return nil, nil, fmt.Errorf("CompileAgents: %w", err)
			}
			mergeFiles(out, files)
			notes = append(notes, n...)
		} else {
			for _, id := range SortedKeys(config.Agents) {
				notes = append(notes, NewNote(
					LevelWarning, r.Target(), "agent", id, "",
					CodeRendererKindUnsupported,
					"agents are not supported by this renderer",
					"",
				))
			}
		}
	}

	// Skills
	if len(config.Skills) > 0 {
		if caps.Skills {
			files, n, err := r.CompileSkills(config.Skills, baseDir)
			if err != nil {
				return nil, nil, fmt.Errorf("CompileSkills: %w", err)
			}
			mergeFiles(out, files)
			notes = append(notes, n...)
		} else {
			for _, id := range SortedKeys(config.Skills) {
				notes = append(notes, NewNote(
					LevelWarning, r.Target(), "skill", id, "",
					CodeRendererKindUnsupported,
					"skills are not supported by this renderer",
					"",
				))
			}
		}
	}

	// Rules
	if len(config.Rules) > 0 {
		if caps.Rules {
			files, n, err := r.CompileRules(config.Rules, baseDir)
			if err != nil {
				return nil, nil, fmt.Errorf("CompileRules: %w", err)
			}
			mergeFiles(out, files)
			notes = append(notes, n...)
		} else {
			for _, id := range SortedKeys(config.Rules) {
				notes = append(notes, NewNote(
					LevelWarning, r.Target(), "rule", id, "",
					CodeRendererKindUnsupported,
					"rules are not supported by this renderer",
					"",
				))
			}
		}
	}

	// Workflows
	if len(config.Workflows) > 0 {
		if caps.Workflows {
			files, n, err := r.CompileWorkflows(config.Workflows, baseDir)
			if err != nil {
				return nil, nil, fmt.Errorf("CompileWorkflows: %w", err)
			}
			mergeFiles(out, files)
			notes = append(notes, n...)
		} else {
			for _, id := range SortedKeys(config.Workflows) {
				notes = append(notes, NewNote(
					LevelWarning, r.Target(), "workflow", id, "",
					CodeRendererKindUnsupported,
					"workflows are not supported by this renderer",
					"",
				))
			}
		}
	}

	// Hooks — XcaffoldConfig.Hooks is map[string]NamedHookConfig; the canonical
	// entry is "default". The per-resource method receives the merged HookConfig
	// (event-name → matcher-group slice) from the default named block.
	var mergedHooks ast.HookConfig
	if dh, ok := config.Hooks["default"]; ok && len(dh.Events) > 0 {
		mergedHooks = dh.Events
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

	// Settings — canonical entry is "default".
	settings, hasSettings := config.Settings["default"]
	if hasSettings {
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

	// MCP servers
	if len(config.MCP) > 0 {
		if caps.MCP {
			files, n, err := r.CompileMCP(config.MCP)
			if err != nil {
				return nil, nil, fmt.Errorf("CompileMCP: %w", err)
			}
			mergeFiles(out, files)
			notes = append(notes, n...)
		} else {
			for _, id := range SortedKeys(config.MCP) {
				notes = append(notes, NewNote(
					LevelWarning, r.Target(), "mcp", id, "",
					CodeRendererKindUnsupported,
					"MCP servers are not supported by this renderer",
					"",
				))
			}
		}
	}

	// Project instructions
	if config.Project != nil {
		if caps.ProjectInstructions {
			files, rootFiles, n, err := r.CompileProjectInstructions(config.Project, baseDir)
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
