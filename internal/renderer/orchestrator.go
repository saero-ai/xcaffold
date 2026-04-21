package renderer

import (
	"fmt"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/output"
)

// Orchestrate compiles config using r. If r also implements ResourceRenderer, the
// orchestrator calls each per-resource method individually and emits
// RENDERER_KIND_UNSUPPORTED fidelity notes for resource kinds the renderer does
// not support. If r only implements TargetRenderer, Orchestrate falls back to the
// legacy Compile() method unchanged.
func Orchestrate(r TargetRenderer, config *ast.XcaffoldConfig, baseDir string) (*output.Output, []FidelityNote, error) {
	rr, ok := r.(ResourceRenderer)
	if !ok {
		return r.Compile(config, baseDir)
	}

	caps := rr.Capabilities()
	out := &output.Output{Files: make(map[string]string)}
	var notes []FidelityNote

	// Agents
	if len(config.Agents) > 0 {
		if caps.Agents {
			files, n, err := rr.CompileAgents(config.Agents, baseDir)
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
			files, n, err := rr.CompileSkills(config.Skills, baseDir)
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
			files, n, err := rr.CompileRules(config.Rules, baseDir)
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
			files, n, err := rr.CompileWorkflows(config.Workflows, baseDir)
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
			files, n, err := rr.CompileHooks(mergedHooks, baseDir)
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
			files, n, err := rr.CompileSettings(settings)
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
			files, n, err := rr.CompileMCP(config.MCP)
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
			files, n, err := rr.CompileProjectInstructions(config.Project, baseDir)
			if err != nil {
				return nil, nil, fmt.Errorf("CompileProjectInstructions: %w", err)
			}
			mergeFiles(out, files)
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

	// Finalize: post-processing pass (path normalization, dedup, etc.)
	finalized, finalNotes, err := rr.Finalize(out.Files)
	if err != nil {
		return nil, nil, fmt.Errorf("Finalize: %w", err)
	}
	out.Files = finalized
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
