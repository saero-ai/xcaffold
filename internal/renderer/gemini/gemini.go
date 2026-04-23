// Package gemini compiles an XcaffoldConfig AST into Gemini CLI output files.
// Project instructions are written to GEMINI.md using concat-nested semantics with
// native @-import preservation. Rules are written to rules/<id>.md (relative to
// OutputDir) and referenced via @-import lines in GEMINI.md using the
// project-relative path (.gemini/rules/<id>.md).
package gemini

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/output"
	"github.com/saero-ai/xcaffold/internal/renderer"
	rendshared "github.com/saero-ai/xcaffold/internal/renderer/shared"
	"github.com/saero-ai/xcaffold/internal/resolver"
	"gopkg.in/yaml.v3"
)

const targetName = "gemini"

// Sentinel keys used to accumulate partial data across per-resource method calls.
// These keys are never written to disk — Finalize consumes and removes them.
const (
	// geminiRuleImportsKey accumulates @-import lines to be appended to GEMINI.md.
	geminiRuleImportsKey = "_gemini_rule_imports"
	// geminiHooksKey stores JSON-encoded ast.HookConfig from CompileHooks.
	geminiHooksKey = "_gemini_hooks_json"
	// geminiMCPKey stores JSON-encoded map[string]ast.MCPConfig from CompileMCP.
	geminiMCPKey = "_gemini_mcp_json"
	// geminiSettingsKey stores JSON-encoded ast.SettingsConfig from CompileSettings.
	geminiSettingsKey = "_gemini_settings_json"
)

// Renderer compiles an XcaffoldConfig AST into Gemini CLI output files.
type Renderer struct{}

// New returns a new Renderer instance.
func New() *Renderer { return &Renderer{} }

// Target returns the canonical name of this renderer.
func (r *Renderer) Target() string { return targetName }

// OutputDir returns the base output directory for Gemini CLI.
func (r *Renderer) OutputDir() string { return ".gemini" }

// Capabilities declares the resource kinds the Gemini renderer supports.
func (r *Renderer) Capabilities() renderer.CapabilitySet {
	return renderer.CapabilitySet{
		Agents:               true,
		Skills:               true,
		Rules:                true,
		Workflows:            true,
		Hooks:                true,
		Settings:             true,
		MCP:                  true,
		Memory:               false,
		ProjectInstructions:  true,
		AgentToolsField:      true,
		AgentNativeToolsOnly: false,
		// examples is intentionally absent — Gemini collapses examples into references/ at compile time.
		SkillSubdirs:    []string{"references", "scripts", "assets"},
		ModelField:      true,
		RuleActivations: []string{"always", "path-glob"},
		RuleEncoding: renderer.RuleEncodingCapabilities{
			Description: "prose",
			Activation:  "omit",
		},
		SecurityFields: renderer.SecurityFieldSupport{},
	}
}

// CompileAgents compiles all agent configs to agents/<id>.md files.
func (r *Renderer) CompileAgents(agents map[string]ast.AgentConfig, baseDir string) (map[string]string, []renderer.FidelityNote, error) {
	files := make(map[string]string)
	cfg := &ast.XcaffoldConfig{ResourceScope: ast.ResourceScope{Agents: agents}}
	notes := r.renderAgents(cfg, baseDir, files, r.Capabilities())
	return files, notes, nil
}

// CompileSkills compiles all skill configs to skills/<id>/SKILL.md files.
func (r *Renderer) CompileSkills(skills map[string]ast.SkillConfig, baseDir string) (map[string]string, []renderer.FidelityNote, error) {
	files := make(map[string]string)
	cfg := &ast.XcaffoldConfig{ResourceScope: ast.ResourceScope{Skills: skills}}
	notes := r.renderSkills(cfg, baseDir, files)
	return files, notes, nil
}

// CompileRules compiles all rule configs to rules/<id>.md files and records
// @-import lines under the geminiRuleImportsKey sentinel for Finalize.
func (r *Renderer) CompileRules(rules map[string]ast.RuleConfig, baseDir string) (map[string]string, []renderer.FidelityNote, error) {
	files := make(map[string]string)
	cfg := &ast.XcaffoldConfig{ResourceScope: ast.ResourceScope{Rules: rules}}
	notes, err := r.renderRulesToMap(cfg, files, baseDir)
	if err != nil {
		return nil, nil, err
	}
	return files, notes, nil
}

// CompileWorkflows lowers workflow configs to rule+skill primitives and compiles
// them. @-import lines for any lowered rules are stored under the sentinel key.
func (r *Renderer) CompileWorkflows(workflows map[string]ast.WorkflowConfig, baseDir string) (map[string]string, []renderer.FidelityNote, error) {
	cfg := &ast.XcaffoldConfig{ResourceScope: ast.ResourceScope{Workflows: workflows}}
	lowered, workflowNotes := rendshared.LowerWorkflows(cfg, targetName)

	files := make(map[string]string)
	var notes []renderer.FidelityNote
	notes = append(notes, workflowNotes...)

	if len(lowered.Rules) > 0 {
		ruleNotes, err := r.renderRulesToMap(lowered, files, baseDir)
		if err != nil {
			return nil, nil, err
		}
		notes = append(notes, ruleNotes...)
	}

	if len(lowered.Skills) > 0 {
		skillNotes := r.renderSkills(lowered, baseDir, files)
		notes = append(notes, skillNotes...)
	}

	return files, notes, nil
}

// CompileHooks stores the hook config under a sentinel key for Finalize to merge
// into settings.json alongside MCP and settings data.
func (r *Renderer) CompileHooks(hooks ast.HookConfig, _ string) (map[string]string, []renderer.FidelityNote, error) {
	files := make(map[string]string)
	if len(hooks) == 0 {
		return files, nil, nil
	}
	b, err := json.Marshal(hooks)
	if err != nil {
		return nil, nil, fmt.Errorf("gemini: marshal hooks for sentinel: %w", err)
	}
	files[geminiHooksKey] = string(b)
	return files, nil, nil
}

// CompileSettings stores the settings config under a sentinel key for Finalize
// and emits fidelity notes for unsupported fields.
func (r *Renderer) CompileSettings(settings ast.SettingsConfig) (map[string]string, []renderer.FidelityNote, error) {
	notes := detectUnsupportedSettingsFields(settings)
	b, err := json.Marshal(settings)
	if err != nil {
		return nil, nil, fmt.Errorf("gemini: marshal settings for sentinel: %w", err)
	}
	files := map[string]string{
		geminiSettingsKey: string(b),
	}
	return files, notes, nil
}

// CompileMCP stores the MCP config under a sentinel key for Finalize to merge
// into settings.json.
func (r *Renderer) CompileMCP(servers map[string]ast.MCPConfig) (map[string]string, []renderer.FidelityNote, error) {
	files := make(map[string]string)
	if len(servers) == 0 {
		return files, nil, nil
	}
	b, err := json.Marshal(servers)
	if err != nil {
		return nil, nil, fmt.Errorf("gemini: marshal MCP for sentinel: %w", err)
	}
	files[geminiMCPKey] = string(b)
	return files, nil, nil
}

// CompileProjectInstructions writes GEMINI.md with project root instructions and
// emits per-scope GEMINI.md files.
func (r *Renderer) CompileProjectInstructions(project *ast.ProjectConfig, baseDir string) (map[string]string, []renderer.FidelityNote, error) {
	files := make(map[string]string)
	cfg := &ast.XcaffoldConfig{Project: project}
	notes := r.renderProjectInstructions(cfg, baseDir, files)
	return files, notes, nil
}

// CompileMemory delegates to MemoryRenderer, writing entries into GEMINI.md
// as provenance-marked blocks under the gemini memory section.
func (r *Renderer) CompileMemory(config *ast.XcaffoldConfig, baseDir string, opts renderer.MemoryOptions) (map[string]string, []renderer.FidelityNote, error) {
	if len(config.Memory) == 0 {
		return map[string]string{}, nil, nil
	}
	memDir := opts.OutputDir
	if memDir == "" {
		return nil, nil, fmt.Errorf("gemini CompileMemory: OutputDir required")
	}
	mr := NewMemoryRenderer(memDir)
	out, notes, err := mr.Compile(config, baseDir)
	if err != nil {
		return nil, notes, err
	}
	return out.Files, notes, nil
}

// Finalize merges all sentinel data into their final output files:
//  1. Appends rule @-import lines from geminiRuleImportsKey to GEMINI.md.
//  2. Deserializes hooks, MCP, and settings sentinels and regenerates settings.json
//     via compileGeminiSettings so hooks and MCP are always combined in one file.
//
// All sentinel keys are removed before returning.
func (r *Renderer) Finalize(files map[string]string) (map[string]string, []renderer.FidelityNote, error) {
	// 1. Merge rule @-import lines into GEMINI.md.
	if imports, ok := files[geminiRuleImportsKey]; ok {
		if imports != "" {
			existing := files["../GEMINI.md"]
			if existing != "" && !strings.HasSuffix(existing, "\n") {
				existing += "\n"
			}
			files["../GEMINI.md"] = existing + imports
		}
		delete(files, geminiRuleImportsKey)
	}

	// 2. Regenerate settings.json from sentinel data.
	var hooks ast.HookConfig
	if raw, ok := files[geminiHooksKey]; ok {
		_ = json.Unmarshal([]byte(raw), &hooks)
		delete(files, geminiHooksKey)
	}

	var mcp map[string]ast.MCPConfig
	if raw, ok := files[geminiMCPKey]; ok {
		_ = json.Unmarshal([]byte(raw), &mcp)
		delete(files, geminiMCPKey)
	}

	var settings ast.SettingsConfig
	if raw, ok := files[geminiSettingsKey]; ok {
		_ = json.Unmarshal([]byte(raw), &settings)
		delete(files, geminiSettingsKey)
	}

	// Remove any stale settings.json written before Finalize.
	delete(files, "settings.json")

	// Regenerate the combined settings.json with all three data sources.
	settingsJSON, settingsNotes, err := compileGeminiSettings(hooks, mcp, settings)
	if err != nil {
		return nil, nil, fmt.Errorf("Finalize: gemini settings: %w", err)
	}
	if settingsJSON != "" {
		files["settings.json"] = settingsJSON
	}

	return files, settingsNotes, nil
}

// renderProjectInstructions writes project root instructions to GEMINI.md and
// emits per-scope nested GEMINI.md files. @-import lines are preserved verbatim
// since Gemini natively supports them.
func (r *Renderer) renderProjectInstructions(config *ast.XcaffoldConfig, baseDir string, files map[string]string) []renderer.FidelityNote {
	p := config.Project
	if p.Instructions == "" && p.InstructionsFile == "" {
		return nil
	}

	rootContent := renderer.ResolveInstructionsContent(p.Instructions, p.InstructionsFile, baseDir)

	var sb strings.Builder
	sb.WriteString(rootContent)

	// Append @-import lines — Gemini supports native @-imports.
	for _, imp := range p.InstructionsImports {
		if !strings.HasSuffix(sb.String(), "\n") {
			sb.WriteString("\n")
		}
		fmt.Fprintf(&sb, "@%s\n", imp)
	}

	files["../GEMINI.md"] = sb.String()

	// Emit per-scope GEMINI.md files.
	for _, scope := range p.InstructionsScopes {
		scopeContent := renderer.ResolveScopeContent(scope, targetName, baseDir)
		if scopeContent == "" {
			continue
		}
		scopePath := filepath.Join("../"+scope.Path, "GEMINI.md")
		safePath := filepath.Clean(scopePath)
		files[safePath] = scopeContent
	}

	return nil
}

// renderRulesToMap writes each rule to rules/<id>.md and stores @-import lines
// under geminiRuleImportsKey. This allows Finalize to append them to GEMINI.md
// after project instructions have been assembled. baseDir is used to resolve
// instructions-file paths.
func (r *Renderer) renderRulesToMap(config *ast.XcaffoldConfig, files map[string]string, baseDir string) ([]renderer.FidelityNote, error) {
	if len(config.Rules) == 0 {
		return nil, nil
	}

	var notes []renderer.FidelityNote
	var importLines []string

	for _, id := range renderer.SortedKeys(config.Rules) {
		rule := config.Rules[id]
		activation := renderer.ResolvedActivation(rule)

		if !renderer.ValidateRuleActivation(rule, r.Capabilities()) {
			notes = append(notes, renderer.NewNote(
				renderer.LevelWarning,
				targetName,
				"rule",
				id,
				"activation",
				renderer.CodeRuleActivationUnsupported,
				fmt.Sprintf("rule %q activation %q has no native equivalent in Gemini; rule will be always-loaded via @-import", id, activation),
				"Remove the activation field or use 'always' or 'path-glob' for Gemini targets",
			))
		}

		body := buildRuleBody(rule, r.Capabilities(), baseDir)
		rulePath := fmt.Sprintf("rules/%s.md", id)
		safePath := filepath.Clean(rulePath)
		files[safePath] = body
		// @-import lines use the project-relative path so the Gemini CLI can
		// locate rule files from the project root (OutputDir is .gemini).
		importLines = append(importLines, fmt.Sprintf("@.gemini/%s", safePath))
	}

	if len(importLines) > 0 {
		existing := files[geminiRuleImportsKey]
		if existing != "" && !strings.HasSuffix(existing, "\n") {
			existing += "\n"
		}
		files[geminiRuleImportsKey] = existing + strings.Join(importLines, "\n") + "\n"
	}

	return notes, nil
}

// renderSkills writes each skill to skills/<id>/SKILL.md (relative to OutputDir)
// using the agentskills.io format: YAML frontmatter (name + description) + markdown body.
func (r *Renderer) renderSkills(config *ast.XcaffoldConfig, baseDir string, files map[string]string) []renderer.FidelityNote {
	if len(config.Skills) == 0 {
		return nil
	}

	var notes []renderer.FidelityNote

	for _, id := range renderer.SortedKeys(config.Skills) {
		skill := config.Skills[id]

		body, _ := resolver.ResolveInstructions(
			skill.Instructions, skill.InstructionsFile,
			fmt.Sprintf("skills/%s/SKILL.md", id), baseDir,
		)

		var sb strings.Builder
		sb.WriteString("---\n")
		if skill.Name != "" {
			fmt.Fprintf(&sb, "name: %s\n", skill.Name)
		}
		if skill.Description != "" {
			fmt.Fprintf(&sb, "description: %s\n", skill.Description)
		}
		sb.WriteString("---\n")

		if body != "" {
			sb.WriteString("\n")
			sb.WriteString(strings.TrimRight(body, "\n"))
			sb.WriteString("\n")
		}

		filePath := fmt.Sprintf("skills/%s/SKILL.md", id)
		files[filepath.Clean(filePath)] = sb.String()

		// Emit fidelity notes for unsupported fields.
		if len(skill.AllowedTools) > 0 {
			notes = append(notes, renderer.NewNote(
				renderer.LevelWarning, targetName, "skill", id, "allowed-tools",
				renderer.CodeFieldUnsupported,
				"Gemini CLI skills do not support allowed-tools in SKILL.md frontmatter",
				"Remove allowed-tools or use targets.gemini.provider pass-through",
			))
		}
		if skill.WhenToUse != "" {
			notes = append(notes, renderer.NewNote(
				renderer.LevelWarning, targetName, "skill", id, "when-to-use",
				renderer.CodeFieldUnsupported,
				"Gemini CLI skills do not support when-to-use; use description for trigger guidance",
				"Move when-to-use content into description",
			))
		}
		// Skill subdirs are best-effort for Gemini: compilation errors are
		// demoted to fidelity notes so the rest of the skill still compiles.
		// Examples collapse into references for Gemini.
		subOut := &output.Output{Files: make(map[string]string)}
		if err := renderer.CompileSkillSubdir(id, "references", "references", skill.References, baseDir, subOut); err != nil {
			notes = append(notes, renderer.NewNote(
				renderer.LevelWarning, targetName, "skill", id, "references",
				renderer.CodeSkillReferencesDropped,
				err.Error(),
				"Check file paths in references",
			))
		}
		if err := renderer.CompileSkillSubdir(id, "scripts", "scripts", skill.Scripts, baseDir, subOut); err != nil {
			notes = append(notes, renderer.NewNote(
				renderer.LevelWarning, targetName, "skill", id, "scripts",
				renderer.CodeSkillScriptsDropped,
				err.Error(),
				"Check file paths in scripts",
			))
		}
		if err := renderer.CompileSkillSubdir(id, "assets", "assets", skill.Assets, baseDir, subOut); err != nil {
			notes = append(notes, renderer.NewNote(
				renderer.LevelWarning, targetName, "skill", id, "assets",
				renderer.CodeSkillAssetsDropped,
				err.Error(),
				"Check file paths in assets",
			))
		}
		if err := renderer.CompileSkillSubdir(id, "examples", "references", skill.Examples, baseDir, subOut); err != nil {
			notes = append(notes, renderer.NewNote(
				renderer.LevelWarning, targetName, "skill", id, "examples",
				renderer.CodeSkillExamplesDropped,
				err.Error(),
				"Check file paths in examples",
			))
		}
		for k, v := range subOut.Files {
			files[k] = v
		}
		if skill.DisableModelInvocation != nil {
			notes = append(notes, renderer.NewNote(
				renderer.LevelWarning, targetName, "skill", id, "disable-model-invocation",
				renderer.CodeFieldUnsupported,
				"Gemini CLI skills do not support disable-model-invocation",
				"",
			))
		}
	}

	return notes
}

// renderAgents writes each agent to agents/<id>.md (relative to OutputDir) using
// YAML frontmatter (name, description, tools, model, max_turns, mcpServers) with
// a markdown body as the system prompt. Gemini-specific fields (timeout_mins,
// temperature, kind) are sourced from targets.gemini.provider pass-through.
// Unsupported fields emit fidelity notes.
// unsupported fields emit fidelity notes.
func (r *Renderer) renderAgents(config *ast.XcaffoldConfig, baseDir string, files map[string]string, caps renderer.CapabilitySet) []renderer.FidelityNote {
	agents := config.Agents
	if len(agents) == 0 {
		return nil
	}

	var notes []renderer.FidelityNote

	for _, id := range renderer.SortedKeys(agents) {
		agent := agents[id]
		if agent.Inherited {
			continue
		}

		var sb strings.Builder
		sb.WriteString("---\n")

		// Required fields.
		if agent.Name != "" {
			fmt.Fprintf(&sb, "name: %s\n", agent.Name)
		}
		if agent.Description != "" {
			fmt.Fprintf(&sb, "description: %s\n", agent.Description)
		}

		// Optional supported fields.
		sanitizedTools, toolNotes := renderer.SanitizeAgentTools(agent.Tools, caps, targetName, id)
		notes = append(notes, toolNotes...)
		if len(sanitizedTools) > 0 {
			sb.WriteString("tools:\n")
			for _, tool := range sanitizedTools {
				fmt.Fprintf(&sb, "  - %s\n", tool)
			}
		}

		resolvedModel, modelNotes := renderer.SanitizeAgentModel(agent.Model, caps, targetName, id)
		notes = append(notes, modelNotes...)
		if resolvedModel != "" {
			fmt.Fprintf(&sb, "model: %s\n", resolvedModel)
		}

		if agent.MaxTurns > 0 {
			fmt.Fprintf(&sb, "max_turns: %d\n", agent.MaxTurns)
		}

		// Inline MCP servers.
		if len(agent.MCPServers) > 0 {
			sb.WriteString("mcpServers:\n")
			for _, mcpID := range renderer.SortedKeys(agent.MCPServers) {
				mcp := agent.MCPServers[mcpID]
				fmt.Fprintf(&sb, "  %s:\n", mcpID)
				if mcp.Command != "" {
					fmt.Fprintf(&sb, "    command: %s\n", mcp.Command)
				}
				if len(mcp.Args) > 0 {
					sb.WriteString("    args:\n")
					for _, arg := range mcp.Args {
						fmt.Fprintf(&sb, "      - %s\n", arg)
					}
				}
				if mcp.URL != "" {
					fmt.Fprintf(&sb, "    url: %s\n", mcp.URL)
				}
				if mcp.Type != "" {
					fmt.Fprintf(&sb, "    type: %s\n", mcp.Type)
				}
				if len(mcp.Env) > 0 {
					sb.WriteString("    env:\n")
					for _, envKey := range renderer.SortedKeys(mcp.Env) {
						fmt.Fprintf(&sb, "      %s: %s\n", envKey, mcp.Env[envKey])
					}
				}
			}
		}

		// Provider pass-through fields from targets.gemini.provider.
		if geminiTarget, ok := agent.Targets[targetName]; ok {
			provider := geminiTarget.Provider
			// Emit known pass-through keys in stable order.
			for _, key := range []string{"kind", "temperature", "timeout_mins"} {
				if val, exists := provider[key]; exists {
					encoded, err := yaml.Marshal(val)
					if err == nil {
						fmt.Fprintf(&sb, "%s: %s", key, strings.TrimRight(string(encoded), "\n"))
						sb.WriteString("\n")
					}
				}
			}
		}

		sb.WriteString("---\n")

		// Markdown body — system prompt.
		body, _ := resolver.ResolveInstructions(
			agent.Instructions, agent.InstructionsFile,
			fmt.Sprintf("agents/%s.md", id), baseDir,
		)
		if body != "" {
			sb.WriteString("\n")
			sb.WriteString(strings.TrimRight(body, "\n"))
			sb.WriteString("\n")
		}

		filePath := fmt.Sprintf("agents/%s.md", id)
		files[filepath.Clean(filePath)] = sb.String()

		// Fidelity notes for security fields with no Gemini equivalent.
		hasSecurityDrop := agent.PermissionMode != "" ||
			len(agent.DisallowedTools) > 0 || agent.Isolation != ""
		if hasSecurityDrop {
			var dropped []string
			if agent.PermissionMode != "" {
				dropped = append(dropped, "permission-mode")
			}
			if len(agent.DisallowedTools) > 0 {
				dropped = append(dropped, "disallowed-tools")
			}
			if agent.Isolation != "" {
				dropped = append(dropped, "isolation")
			}
			notes = append(notes, renderer.NewNote(
				renderer.LevelWarning, targetName, "agent", id,
				strings.Join(dropped, ","),
				renderer.CodeAgentSecurityFieldsDropped,
				fmt.Sprintf("agent %q fields [%s] have no Gemini equivalent and were dropped; security constraints will NOT be enforced", id, strings.Join(dropped, ", ")),
				"Review agent security requirements manually for this target",
			))
		}

		// Fidelity notes for other unsupported fields.
		type unsupportedField struct {
			name    string
			present bool
		}
		unsupported := []unsupportedField{
			{"effort", agent.Effort != ""},
			{"background", agent.Background != nil},
			{"color", agent.Color != ""},
			{"initial-prompt", agent.InitialPrompt != ""},
			{"readonly", agent.Readonly != nil},
			{"user-invocable", agent.UserInvocable != nil},
			{"skills", len(agent.Skills) > 0},
			{"hooks", len(agent.Hooks) > 0},
			{"memory", agent.Memory != ""},
			{"disable-model-invocation", agent.DisableModelInvocation != nil},
			{"when", agent.When != ""},
			{"mode", agent.Mode != ""},
		}
		for _, f := range unsupported {
			if f.present {
				notes = append(notes, renderer.NewNote(
					renderer.LevelWarning, targetName, "agent", id, f.name,
					renderer.CodeFieldUnsupported,
					fmt.Sprintf("agent %q field %q has no Gemini CLI equivalent and was dropped", id, f.name),
					"Remove the field or use targets.gemini.provider pass-through",
				))
			}
		}
	}

	return notes
}

// buildRuleBody constructs the markdown content for a rule file.
// baseDir is used to resolve instructions-file paths; pass "" when no
// file resolution is needed.
func buildRuleBody(rule ast.RuleConfig, caps renderer.CapabilitySet, baseDir string) string {
	var sb strings.Builder
	sb.WriteString(renderer.BuildRuleProsePrefix(rule, caps))
	instructions := rule.Instructions
	if instructions == "" && rule.InstructionsFile != "" && baseDir != "" {
		instructions = renderer.ResolveInstructionsContent("", rule.InstructionsFile, baseDir)
	}
	body := strings.TrimRight(instructions, "\n")
	if body != "" {
		sb.WriteString(body)
		sb.WriteString("\n")
	}
	return sb.String()
}
