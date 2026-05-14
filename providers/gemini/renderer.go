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
		AgentNativeToolsOnly: false,
		// examples collapses into references/ for Gemini.
		SkillArtifactDirs: map[string]string{
			"references": "references",
			"scripts":    "scripts",
			"assets":     "assets",
			"examples":   "references",
		},
		RuleActivations: []string{"always", "path-glob"},
		RuleEncoding: renderer.RuleEncodingCapabilities{
			Description: "prose",
			Activation:  "omit",
		},
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

// CompileWorkflows lowers workflow configs to provider-native primitives and compiles
// them. Rule and skill primitives are rendered into their standard output paths.
// Primitives with provider-native paths (custom-command, prompt-file) are written
// directly using the path set by the translator. @-import lines for any lowered
// rules are stored under the sentinel key.
func (r *Renderer) CompileWorkflows(workflows map[string]ast.WorkflowConfig, baseDir string) (map[string]string, []renderer.FidelityNote, error) {
	cfg := &ast.XcaffoldConfig{ResourceScope: ast.ResourceScope{Workflows: workflows}}
	lowered, directFiles, workflowNotes := rendshared.LowerWorkflows(cfg, targetName)

	files := make(map[string]string)
	var notes []renderer.FidelityNote
	notes = append(notes, workflowNotes...)

	// Merge direct-path files (e.g. custom-command primitives). The translator sets
	// p.Path to the full provider-prefixed path (e.g. ".gemini/commands/<name>.md")
	// because it encodes the intended on-disk location. The files map here is keyed
	// relative to OutputDir, so apply.go will prepend OutputDir again when writing.
	// Strip the OutputDir prefix to avoid the doubled path ".gemini/.gemini/..." on disk.
	prefix := r.OutputDir() + "/"
	for path, content := range directFiles {
		files[strings.TrimPrefix(path, prefix)] = content
	}

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

	// Copy workflow artifact directories. Lowered skills don't carry the original
	// workflow's Artifacts field, so handle them from the original workflow configs.
	artifactNotes := renderer.AppendWorkflowArtifacts(renderer.WorkflowArtifactArgs{
		Target: targetName, Workflows: workflows, BaseDir: baseDir, Caps: r.Capabilities(), Files: files,
	})
	notes = append(notes, artifactNotes...)
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
func (r *Renderer) CompileProjectInstructions(config *ast.XcaffoldConfig, baseDir string) (map[string]string, map[string]string, []renderer.FidelityNote, error) {
	rootFiles := make(map[string]string)
	notes := r.renderProjectInstructions(config, baseDir, rootFiles)
	return nil, rootFiles, notes, nil
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
func (r *Renderer) Finalize(files map[string]string, rootFiles map[string]string) (map[string]string, map[string]string, []renderer.FidelityNote, error) {
	// 1. Merge rule @-import lines into GEMINI.md.
	if imports, ok := files[geminiRuleImportsKey]; ok {
		if imports != "" {
			existing := rootFiles["GEMINI.md"]
			if existing != "" && !strings.HasSuffix(existing, "\n") {
				existing += "\n"
			}
			rootFiles["GEMINI.md"] = existing + imports
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
		return nil, nil, nil, fmt.Errorf("Finalize: gemini settings: %w", err)
	}
	if settingsJSON != "" {
		files["settings.json"] = settingsJSON
	}

	return files, rootFiles, settingsNotes, nil
}

// renderProjectInstructions writes project root instructions to GEMINI.md and
// emits per-scope nested GEMINI.md files. @-import lines are preserved verbatim
// since Gemini natively supports them.
func (r *Renderer) renderProjectInstructions(config *ast.XcaffoldConfig, baseDir string, files map[string]string) []renderer.FidelityNote {
	rootContent := renderer.ResolveContextBody(config, targetName)
	if rootContent == "" {
		return nil
	}

	files["GEMINI.md"] = rootContent
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
			notes = append(notes, renderer.FidelityNote{
				Level:      renderer.LevelWarning,
				Target:     targetName,
				Kind:       "rule",
				Resource:   id,
				Field:      "activation",
				Code:       renderer.CodeRuleActivationUnsupported,
				Reason:     fmt.Sprintf("rule %q activation %q has no native equivalent in Gemini; rule will be always-loaded via @-import", id, activation),
				Mitigation: "Remove the activation field or use 'always' or 'path-glob' for Gemini targets",
			})
		}

		body := buildRuleBody(rule, r.Capabilities(), baseDir)
		rulePath := fmt.Sprintf("rules/%s.md", id)
		safePath := filepath.Clean(rulePath)
		files[safePath] = body
		// @-import lines use the project-relative path so the Gemini CLI can
		// locate rule files from the project root (OutputDir is .gemini).
		var importText string
		if len(rule.Paths.Values) > 0 {
			importText = fmt.Sprintf("Apply this rule when accessing %s:\n@.gemini/%s", strings.Join(rule.Paths.Values, ", "), safePath)
		} else {
			importText = fmt.Sprintf("Always apply @.gemini/%s", safePath)
		}
		importLines = append(importLines, importText)
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
	caps := r.Capabilities()

	for _, id := range renderer.SortedKeys(config.Skills) {
		skill := config.Skills[id]
		files[filepath.Clean(fmt.Sprintf("skills/%s/SKILL.md", id))] = buildGeminiSkillContent(skill)
		notes = append(notes, geminiSkillFidelityNotes(id, skill)...)
		artifactNotes := renderer.CompileArtifactsDemoted(targetName, renderer.ArtifactJob{ID: id, BaseDir: baseDir, Caps: caps, Files: files}, skill.Artifacts)
		notes = append(notes, artifactNotes...)
	}

	return notes
}

// buildGeminiSkillContent renders the SKILL.md content for a single Gemini skill.
// Gemini supports only name and description in frontmatter; all other fields are dropped.
func buildGeminiSkillContent(skill ast.SkillConfig) string {
	body := resolver.StripFrontmatter(skill.Body)
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
	return sb.String()
}

// geminiSkillFidelityNotes returns fidelity notes for Gemini-unsupported skill fields.
func geminiSkillFidelityNotes(id string, skill ast.SkillConfig) []renderer.FidelityNote {
	var notes []renderer.FidelityNote
	if len(skill.AllowedTools.Values) > 0 {
		notes = append(notes, renderer.FidelityNote{
			Level:      renderer.LevelWarning,
			Target:     targetName,
			Kind:       "skill",
			Resource:   id,
			Field:      "allowed-tools",
			Code:       renderer.CodeFieldUnsupported,
			Reason:     "Gemini CLI skills do not support allowed-tools in SKILL.md frontmatter",
			Mitigation: "Remove allowed-tools or use targets.gemini.provider pass-through",
		})
	}
	if skill.WhenToUse != "" {
		notes = append(notes, renderer.FidelityNote{
			Level:      renderer.LevelWarning,
			Target:     targetName,
			Kind:       "skill",
			Resource:   id,
			Field:      "when-to-use",
			Code:       renderer.CodeFieldUnsupported,
			Reason:     "Gemini CLI skills do not support when-to-use; use description for trigger guidance",
			Mitigation: "Move when-to-use content into description",
		})
	}
	if skill.DisableModelInvocation != nil {
		notes = append(notes, renderer.FidelityNote{
			Level:    renderer.LevelWarning,
			Target:   targetName,
			Kind:     "skill",
			Resource: id,
			Field:    "disable-model-invocation",
			Code:     renderer.CodeFieldUnsupported,
			Reason:   "Gemini CLI skills do not support disable-model-invocation",
		})
	}
	return notes
}

// renderAgents writes each agent to agents/<id>.md (relative to OutputDir) using
// YAML frontmatter (name, description, tools, model, max_turns, mcpServers) with
// a markdown body as the system prompt. Gemini-specific fields (timeout_mins,
// temperature, kind) are sourced from targets.gemini.provider pass-through.
// Unsupported fields emit fidelity notes.
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
		toolNotes, modelNotes := renderGeminiAgentFrontmatter(&sb, agent, id, caps)
		notes = append(notes, toolNotes...)
		notes = append(notes, modelNotes...)

		body := resolver.StripFrontmatter(agent.Body)
		if body != "" {
			sb.WriteString("\n")
			sb.WriteString(strings.TrimRight(body, "\n"))
			sb.WriteString("\n")
		}

		filePath := fmt.Sprintf("agents/%s.md", id)
		files[filepath.Clean(filePath)] = sb.String()

		notes = append(notes, geminiAgentFidelityNotes(agent, id)...)
	}

	return notes
}

// renderGeminiAgentFrontmatter writes the complete YAML frontmatter block (including
// the opening and closing "---\n" delimiters) to sb. It returns tool and model
// fidelity notes collected during sanitization.
func renderGeminiAgentFrontmatter(sb *strings.Builder, agent ast.AgentConfig, id string, caps renderer.CapabilitySet) (toolNotes, modelNotes []renderer.FidelityNote) {
	sb.WriteString("---\n")

	if agent.Name != "" {
		fmt.Fprintf(sb, "name: %s\n", agent.Name)
	}
	if agent.Description != "" {
		fmt.Fprintf(sb, "description: %s\n", agent.Description)
	}

	sanitizedTools, tn := renderer.SanitizeAgentTools(agent.Tools.Values, caps, targetName, id)
	toolNotes = tn
	if len(sanitizedTools) > 0 {
		sb.WriteString("tools:\n")
		for _, tool := range sanitizedTools {
			fmt.Fprintf(sb, "  - %s\n", tool)
		}
	}

	resolvedModel, mn := renderer.SanitizeAgentModel(agent.Model, caps, targetName, id)
	modelNotes = mn
	if resolvedModel != "" {
		fmt.Fprintf(sb, "model: %s\n", resolvedModel)
	}

	if agent.MaxTurns != nil && *agent.MaxTurns > 0 {
		fmt.Fprintf(sb, "max_turns: %d\n", *agent.MaxTurns)
	}

	renderGeminiAgentMCPServers(sb, agent)
	renderGeminiAgentProviderPassthrough(sb, agent)

	sb.WriteString("---\n")
	return toolNotes, modelNotes
}

// renderGeminiAgentMCPServers writes inline mcpServers YAML entries to sb.
func renderGeminiAgentMCPServers(sb *strings.Builder, agent ast.AgentConfig) {
	if len(agent.MCPServers) == 0 {
		return
	}
	sb.WriteString("mcpServers:\n")
	for _, mcpID := range renderer.SortedKeys(agent.MCPServers) {
		mcp := agent.MCPServers[mcpID]
		fmt.Fprintf(sb, "  %s:\n", mcpID)
		if mcp.Command != "" {
			fmt.Fprintf(sb, "    command: %s\n", mcp.Command)
		}
		if len(mcp.Args) > 0 {
			sb.WriteString("    args:\n")
			for _, arg := range mcp.Args {
				fmt.Fprintf(sb, "      - %s\n", arg)
			}
		}
		if mcp.URL != "" {
			fmt.Fprintf(sb, "    url: %s\n", mcp.URL)
		}
		if mcp.Type != "" {
			fmt.Fprintf(sb, "    type: %s\n", mcp.Type)
		}
		if len(mcp.Env) > 0 {
			sb.WriteString("    env:\n")
			for _, envKey := range renderer.SortedKeys(mcp.Env) {
				fmt.Fprintf(sb, "      %s: %s\n", envKey, mcp.Env[envKey])
			}
		}
	}
}

// renderGeminiAgentProviderPassthrough writes targets.gemini.provider pass-through
// fields (kind, temperature, timeout_mins) in stable order to sb.
func renderGeminiAgentProviderPassthrough(sb *strings.Builder, agent ast.AgentConfig) {
	geminiTarget, ok := agent.Targets[targetName]
	if !ok {
		return
	}
	provider := geminiTarget.Provider
	for _, key := range []string{"kind", "temperature", "timeout_mins"} {
		val, exists := provider[key]
		if !exists {
			continue
		}
		encoded, err := yaml.Marshal(val)
		if err == nil {
			fmt.Fprintf(sb, "%s: %s", key, strings.TrimRight(string(encoded), "\n"))
			sb.WriteString("\n")
		}
	}
}

// geminiAgentFidelityNotes returns FIELD_UNSUPPORTED fidelity notes for each
// agent field that has no Gemini CLI equivalent.
func geminiAgentFidelityNotes(agent ast.AgentConfig, id string) []renderer.FidelityNote {
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
		{"skills", len(agent.Skills.Values) > 0},
		{"hooks", len(agent.Hooks) > 0},
		{"memory", len(agent.Memory) > 0},
		{"disable-model-invocation", agent.DisableModelInvocation != nil},
	}
	var notes []renderer.FidelityNote
	for _, f := range unsupported {
		if f.present {
			notes = append(notes, renderer.FidelityNote{
				Level:      renderer.LevelWarning,
				Target:     targetName,
				Kind:       "agent",
				Resource:   id,
				Field:      f.name,
				Code:       renderer.CodeFieldUnsupported,
				Reason:     fmt.Sprintf("agent %q field %q has no Gemini CLI equivalent and was dropped", id, f.name),
				Mitigation: "Remove the field or use targets.gemini.provider pass-through",
			})
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
	body := strings.TrimRight(resolver.StripFrontmatter(rule.Body), "\n")
	if body != "" {
		sb.WriteString(body)
		sb.WriteString("\n")
	}
	return sb.String()
}
