// Package copilot compiles an XcaffoldConfig AST into GitHub Copilot instruction
// files. Rules are written as .instructions.md files under .github/instructions/
// with Copilot-compatible YAML frontmatter.
//
// Key normalizations applied during compilation:
//   - Output path: .github/instructions/<id>.instructions.md
//   - activation: always → applyTo: "**"
//   - activation: path-glob → applyTo: "<comma-joined paths>"
//   - activation: manual-mention / model-decided / explicit-invoke → applyTo: "**" + FidelityNote
//   - exclude-agents → excludeAgent: <list> (Copilot singular key name)
package copilot

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/output"
	"github.com/saero-ai/xcaffold/internal/renderer"
	rendshared "github.com/saero-ai/xcaffold/internal/renderer/shared"
	"github.com/saero-ai/xcaffold/internal/resolver"
)

const targetName = "copilot"

// Renderer compiles an XcaffoldConfig AST into GitHub Copilot instruction files.
// It targets the ".github/instructions/" directory structure.
type Renderer struct{}

// New returns a new Renderer instance.
func New() *Renderer {
	return &Renderer{}
}

// Target returns the identifier for this renderer's target platform.
func (r *Renderer) Target() string {
	return targetName
}

// OutputDir returns the output directory prefix for this renderer.
func (r *Renderer) OutputDir() string {
	return ".github"
}

// Capabilities declares the resource kinds this renderer supports.
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
		SkillArtifactDirs: map[string]string{
			"references": "references",
			"scripts":    "scripts",
			"assets":     "assets",
			"examples":   "examples",
		},
		RuleActivations: []string{"always", "path-glob"},
		RuleEncoding: renderer.RuleEncodingCapabilities{
			Description: "frontmatter",
			Activation:  "frontmatter",
		},
	}
}

// CompileAgents compiles each agent to agents/<id>.agent.md.
// If a .claude/ directory is detected in baseDir, all agents are skipped and
// a CLAUDE_NATIVE_PASSTHROUGH info note is emitted per agent — GitHub Copilot
// natively loads .claude/agents/ and re-translation is redundant.
func (r *Renderer) CompileAgents(agents map[string]ast.AgentConfig, baseDir string) (map[string]string, []renderer.FidelityNote, error) {
	files := make(map[string]string)
	if claudeDirExists(baseDir) {
		var notes []renderer.FidelityNote
		for _, id := range renderer.SortedKeys(agents) {
			notes = append(notes, renderer.NewNote(
				renderer.LevelInfo, targetName, "agent", id, "",
				renderer.CodeClaudeNativePassthrough,
				fmt.Sprintf("agent %q skipped; .claude/agents/%s.md detected and natively loaded by GitHub Copilot", id, id),
				"No action needed — GitHub Copilot reads .claude/agents/ automatically",
			))
		}
		return files, notes, nil
	}
	cfg := &ast.XcaffoldConfig{ResourceScope: ast.ResourceScope{Agents: agents}}
	notes := r.renderAgents(cfg, baseDir, files, r.Capabilities())
	return files, notes, nil
}

// CompileSkills compiles each skill to skills/<id>/SKILL.md.
// If a .claude/ directory is detected in baseDir, all skills are skipped and
// a CLAUDE_NATIVE_PASSTHROUGH info note is emitted per skill.
func (r *Renderer) CompileSkills(skills map[string]ast.SkillConfig, baseDir string) (map[string]string, []renderer.FidelityNote, error) {
	files := make(map[string]string)
	if claudeDirExists(baseDir) {
		var notes []renderer.FidelityNote
		for _, id := range renderer.SortedKeys(skills) {
			notes = append(notes, renderer.NewNote(
				renderer.LevelInfo, targetName, "skill", id, "",
				renderer.CodeClaudeNativePassthrough,
				fmt.Sprintf("skill %q skipped; .claude/skills/%s/SKILL.md detected and natively loaded by GitHub Copilot", id, id),
				"No action needed — GitHub Copilot reads .claude/skills/ automatically",
			))
		}
		return files, notes, nil
	}
	cfg := &ast.XcaffoldConfig{ResourceScope: ast.ResourceScope{Skills: skills}}
	notes := r.renderSkills(cfg, baseDir, files)
	return files, notes, nil
}

// CompileRules compiles each rule to instructions/<id>.instructions.md.
// If a .claude/ directory is detected in baseDir, all rules are skipped and
// a CLAUDE_NATIVE_PASSTHROUGH info note is emitted per rule.
func (r *Renderer) CompileRules(rules map[string]ast.RuleConfig, baseDir string) (map[string]string, []renderer.FidelityNote, error) {
	files := make(map[string]string)
	if claudeDirExists(baseDir) {
		var notes []renderer.FidelityNote
		for _, id := range renderer.SortedKeys(rules) {
			notes = append(notes, renderer.NewNote(
				renderer.LevelInfo, targetName, "rule", id, "",
				renderer.CodeClaudeNativePassthrough,
				fmt.Sprintf("rule %q skipped; .claude/rules/ detected and natively loaded by GitHub Copilot", id),
				"No action needed — GitHub Copilot reads .claude/rules/ automatically",
			))
		}
		return files, notes, nil
	}
	var notes []renderer.FidelityNote
	for id, rule := range rules {
		md, ruleNotes, err := compileCopilotRule(id, rule, r.Capabilities(), baseDir)
		if err != nil {
			return nil, nil, fmt.Errorf("copilot: failed to compile rule %q: %w", id, err)
		}
		safePath := filepath.Clean(fmt.Sprintf("instructions/%s.instructions.md", id))
		files[safePath] = md
		notes = append(notes, ruleNotes...)
	}
	return files, notes, nil
}

// CompileWorkflows lowers workflow configs to rule+skill primitives and compiles
// them. Rules are emitted as instructions/<id>.instructions.md files; skills
// are emitted as skills/<id>/SKILL.md files. If a .claude/ directory is
// present, the lowered rules will be seamlessly skipped by CompileRules.
func (r *Renderer) CompileWorkflows(workflows map[string]ast.WorkflowConfig, baseDir string) (map[string]string, []renderer.FidelityNote, error) {
	cfg := &ast.XcaffoldConfig{ResourceScope: ast.ResourceScope{Workflows: workflows}}
	lowered, workflowNotes := rendshared.LowerWorkflows(cfg, targetName)

	files := make(map[string]string)
	var notes []renderer.FidelityNote
	notes = append(notes, workflowNotes...)

	if len(lowered.Rules) > 0 {
		ruleFiles, ruleNotes, err := r.CompileRules(lowered.Rules, baseDir)
		if err != nil {
			return nil, nil, err
		}
		for k, v := range ruleFiles {
			files[k] = v
		}
		notes = append(notes, ruleNotes...)
	}

	if len(lowered.Skills) > 0 {
		skillNotes := r.renderSkills(lowered, baseDir, files)
		notes = append(notes, skillNotes...)
	}

	return files, notes, nil
}

// CompileHooks compiles hooks to hooks/xcaffold-hooks.json.
func (r *Renderer) CompileHooks(hooks ast.HookConfig, baseDir string) (map[string]string, []renderer.FidelityNote, error) {
	files, notes, err := compileCopilotSettings(hooks, nil, nil)
	return files, notes, err
}

// CompileSettings emits fidelity notes for unsupported settings fields.
// Copilot settings themselves are not written to a file.
func (r *Renderer) CompileSettings(settings ast.SettingsConfig) (map[string]string, []renderer.FidelityNote, error) {
	notes := detectUnsupportedCopilotSettings(&settings)
	return make(map[string]string), notes, nil
}

// CompileMCP emits a fidelity note for MCP servers that require manual placement.
func (r *Renderer) CompileMCP(servers map[string]ast.MCPConfig) (map[string]string, []renderer.FidelityNote, error) {
	mcpJSON, mcpNotes, err := compileCopilotMCP(servers)
	if err != nil {
		return nil, nil, err
	}
	files := make(map[string]string)
	if mcpJSON != "" {
		files[".vscode/mcp.json"] = mcpJSON
	}
	return files, mcpNotes, nil
}

// CompileProjectInstructions emits copilot-instructions.md (flat) or AGENTS.md
// files (nested) based on the effective instructions-mode.
//
// If a .claude/ directory is detected in baseDir, the root project instruction
// file is skipped (root CLAUDE.md is natively loaded by Copilot) and a
// CLAUDE_NATIVE_PASSTHROUGH info note is emitted. Nested InstructionsScopes
// are still written as .github/instructions/<scope>.instructions.md with
// applyTo: frontmatter because Copilot does NOT natively load subdirectory
// CLAUDE.md files.
func (r *Renderer) CompileProjectInstructions(config *ast.XcaffoldConfig, baseDir string) (map[string]string, map[string]string, []renderer.FidelityNote, error) {
	files := make(map[string]string)
	if claudeDirExists(baseDir) {
		var notes []renderer.FidelityNote
		notes = append(notes, renderer.NewNote(
			renderer.LevelInfo, targetName, "instructions", "root", "",
			renderer.CodeClaudeNativePassthrough,
			"root project instructions skipped; CLAUDE.md detected and natively loaded by GitHub Copilot",
			"No action needed — GitHub Copilot reads root CLAUDE.md automatically",
		))
		// Nested scope instruction files are no longer supported natively via scope objects;
		// use targeted Rule configurations instead.
		return files, nil, notes, nil
	}
	notes := r.renderProjectInstructions(config, baseDir, files)
	return files, nil, notes, nil
}

// CompileMemory delegates to MemoryRenderer. Copilot has no native per-file
// memory primitive; the renderer emits FidelityNotes advising use of
// .github/copilot-instructions.md.
func (r *Renderer) CompileMemory(config *ast.XcaffoldConfig, baseDir string, opts renderer.MemoryOptions) (map[string]string, []renderer.FidelityNote, error) {
	if len(config.Memory) == 0 {
		return map[string]string{}, nil, nil
	}
	mr := NewMemoryRenderer()
	out, notes, err := mr.Compile(config, baseDir)
	if err != nil {
		return nil, notes, err
	}
	return out.Files, notes, nil
}

// Finalize is a no-op for the Copilot renderer — no post-processing is required.
func (r *Renderer) Finalize(files map[string]string, rootFiles map[string]string) (map[string]string, map[string]string, []renderer.FidelityNote, error) {
	return files, rootFiles, nil, nil
}

// instructionsMode returns the effective instructions-mode for the Copilot renderer.
// Always returns "flat" now that InstructionsMode has been deprecated.
func instructionsMode(config *ast.XcaffoldConfig) string {
	return "flat"
}

// claudeDirExists reports whether a .claude/ directory exists in baseDir.
// It is used to determine whether to skip full translation and emit passthrough
// fidelity notes instead. GitHub Copilot natively loads .claude/agents/,
// .claude/skills/, .claude/rules/, and root CLAUDE.md automatically.
func claudeDirExists(baseDir string) bool {
	_, err := os.Stat(filepath.Join(baseDir, ".claude"))
	return err == nil
}

// renderProjectInstructions emits a single copilot-instructions.md file.
func (r *Renderer) renderProjectInstructions(config *ast.XcaffoldConfig, baseDir string, files map[string]string) []renderer.FidelityNote {
	rootContent := renderer.ResolveContextBody(config, targetName)

	if rootContent == "" {
		return nil
	}

	mode := instructionsMode(config)
	if mode == "nested" {
		files[filepath.Clean("AGENTS.md")] = resolver.StripFrontmatter(rootContent)
	} else {
		files[filepath.Clean("copilot-instructions.md")] = resolver.StripFrontmatter(rootContent)
	}
	return nil
}

// renderAgents writes each agent to .github/agents/<id>.agent.md using YAML
// frontmatter (name, description, tools, model, disable-model-invocation,
// user-invocable, mcp-servers) with a markdown body as the system prompt.
// Provider pass-through keys (target, metadata) are sourced from
// targets.copilot.provider. Unsupported fields emit fidelity notes.
func (r *Renderer) renderAgents(config *ast.XcaffoldConfig, baseDir string, files map[string]string, caps renderer.CapabilitySet) []renderer.FidelityNote {
	if len(config.Agents) == 0 {
		return nil
	}

	var notes []renderer.FidelityNote

	for _, id := range renderer.SortedKeys(config.Agents) {
		agent := config.Agents[id]
		if agent.Inherited {
			continue
		}

		var sb strings.Builder
		sb.WriteString("---\n")

		if agent.Name != "" {
			fmt.Fprintf(&sb, "name: %s\n", agent.Name)
		}
		if agent.Description != "" {
			fmt.Fprintf(&sb, "description: %s\n", agent.Description)
		}

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

		if agent.DisableModelInvocation != nil {
			fmt.Fprintf(&sb, "disable-model-invocation: %v\n", *agent.DisableModelInvocation)
		}
		if agent.UserInvocable != nil {
			fmt.Fprintf(&sb, "user-invocable: %v\n", *agent.UserInvocable)
		}

		// Inline MCP servers.
		if len(agent.MCPServers) > 0 {
			sb.WriteString("mcp-servers:\n")
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

		// Provider pass-through from targets.copilot.provider.
		if copilotTarget, ok := agent.Targets[targetName]; ok {
			for _, key := range []string{"target", "metadata"} {
				if val, exists := copilotTarget.Provider[key]; exists {
					fmt.Fprintf(&sb, "%s: %v\n", key, val)
				}
			}
		}

		sb.WriteString("---\n")

		// Markdown body — system prompt.
		body := resolver.StripFrontmatter(agent.Body)
		if body != "" {
			sb.WriteString("\n")
			sb.WriteString(strings.TrimRight(body, "\n"))
			sb.WriteString("\n")
		}

		filePath := fmt.Sprintf("agents/%s.agent.md", id)
		files[filepath.Clean(filePath)] = sb.String()

		// Fidelity notes for other unsupported fields.
		type unsupportedField struct {
			name    string
			present bool
		}
		unsupported := []unsupportedField{
			{"effort", agent.Effort != ""},
			{"max-turns", agent.MaxTurns > 0},
			{"background", agent.Background != nil},
			{"color", agent.Color != ""},
			{"initial-prompt", agent.InitialPrompt != ""},
			{"readonly", agent.Readonly != nil},
			{"memory", len(agent.Memory) > 0},
			{"skills", len(agent.Skills) > 0},
			{"hooks", len(agent.Hooks) > 0},
		}
		for _, f := range unsupported {
			if f.present {
				notes = append(notes, renderer.NewNote(
					renderer.LevelWarning, targetName, "agent", id, f.name,
					renderer.CodeFieldUnsupported,
					fmt.Sprintf("agent %q field %q has no Copilot equivalent and was dropped", id, f.name),
					"Remove the field or use targets.copilot.provider pass-through",
				))
			}
		}
	}

	return notes
}

// renderSkills writes each skill to .github/skills/<id>/SKILL.md using the
// agentskills.io format: YAML frontmatter (name, description, allowed-tools,
// license) + markdown body. Unsupported fields emit fidelity notes.
func (r *Renderer) renderSkills(config *ast.XcaffoldConfig, baseDir string, files map[string]string) []renderer.FidelityNote {
	if len(config.Skills) == 0 {
		return nil
	}

	var notes []renderer.FidelityNote

	for _, id := range renderer.SortedKeys(config.Skills) {
		skill := config.Skills[id]
		if skill.Inherited {
			continue
		}

		var sb strings.Builder
		sb.WriteString("---\n")
		if skill.Name != "" {
			fmt.Fprintf(&sb, "name: %s\n", skill.Name)
		}
		if skill.Description != "" {
			fmt.Fprintf(&sb, "description: %s\n", skill.Description)
		}
		if len(skill.AllowedTools) > 0 {
			sb.WriteString("allowed-tools:\n")
			for _, tool := range skill.AllowedTools {
				fmt.Fprintf(&sb, "  - %s\n", tool)
			}
		}
		if skill.License != "" {
			fmt.Fprintf(&sb, "license: %s\n", skill.License)
		}
		sb.WriteString("---\n")

		body := resolver.StripFrontmatter(skill.Body)
		if body != "" {
			sb.WriteString("\n")
			sb.WriteString(strings.TrimRight(body, "\n"))
			sb.WriteString("\n")
		}

		filePath := fmt.Sprintf("skills/%s/SKILL.md", id)
		files[filepath.Clean(filePath)] = sb.String()

		// Fidelity notes for unsupported fields.
		if skill.WhenToUse != "" {
			notes = append(notes, renderer.NewNote(
				renderer.LevelWarning, targetName, "skill", id, "when-to-use",
				renderer.CodeFieldUnsupported,
				fmt.Sprintf("skill %q field \"when-to-use\" has no Copilot equivalent and was dropped", id),
				"Move when-to-use content into description",
			))
		}
		if skill.DisableModelInvocation != nil {
			notes = append(notes, renderer.NewNote(
				renderer.LevelWarning, targetName, "skill", id, "disable-model-invocation",
				renderer.CodeFieldUnsupported,
				fmt.Sprintf("skill %q field \"disable-model-invocation\" has no Copilot skill equivalent and was dropped", id),
				"",
			))
		}
		if skill.UserInvocable != nil {
			notes = append(notes, renderer.NewNote(
				renderer.LevelWarning, targetName, "skill", id, "user-invocable",
				renderer.CodeFieldUnsupported,
				fmt.Sprintf("skill %q field \"user-invocable\" has no Copilot skill equivalent and was dropped", id),
				"",
			))
		}
		if skill.ArgumentHint != "" {
			notes = append(notes, renderer.NewNote(
				renderer.LevelWarning, targetName, "skill", id, "argument-hint",
				renderer.CodeFieldUnsupported,
				fmt.Sprintf("skill %q field \"argument-hint\" has no Copilot skill equivalent and was dropped", id),
				"",
			))
		}
		subOut := &output.Output{Files: make(map[string]string)}
		if err := renderer.CompileSkillSubdir(id, "references", "references", skill.References, baseDir, subOut); err != nil {
			notes = append(notes, renderer.NewNote(renderer.LevelWarning, targetName, "skill", id, "references", renderer.CodeSkillReferencesDropped, err.Error(), "Check file paths"))
		}
		if err := renderer.CompileSkillSubdir(id, "scripts", "scripts", skill.Scripts, baseDir, subOut); err != nil {
			notes = append(notes, renderer.NewNote(renderer.LevelWarning, targetName, "skill", id, "scripts", renderer.CodeSkillScriptsDropped, err.Error(), "Check file paths"))
		}
		if err := renderer.CompileSkillSubdir(id, "assets", "assets", skill.Assets, baseDir, subOut); err != nil {
			notes = append(notes, renderer.NewNote(renderer.LevelWarning, targetName, "skill", id, "assets", renderer.CodeSkillAssetsDropped, err.Error(), "Check file paths"))
		}
		if err := renderer.CompileSkillSubdir(id, "examples", "examples", skill.Examples, baseDir, subOut); err != nil {
			notes = append(notes, renderer.NewNote(renderer.LevelWarning, targetName, "skill", id, "examples", renderer.CodeSkillExamplesDropped, err.Error(), "Check file paths"))
		}
		for k, v := range subOut.Files {
			files[k] = v
		}
	}

	return notes
}

// compileCopilotRule renders a single RuleConfig as a Copilot .instructions.md file.
func compileCopilotRule(id string, rule ast.RuleConfig, caps renderer.CapabilitySet, baseDir string) (string, []renderer.FidelityNote, error) {
	var notes []renderer.FidelityNote

	activation := renderer.ResolvedActivation(rule)

	var applyTo string
	if !renderer.ValidateRuleActivation(rule, caps) {
		notes = append(notes, renderer.NewNote(
			renderer.LevelWarning,
			targetName,
			"rule",
			id,
			"activation",
			renderer.CodeRuleActivationUnsupported,
			fmt.Sprintf("rule %q: activation %q has no Copilot-native equivalent; emitted as applyTo: \"**\"", id, activation),
			"Use activation: always or activation: path-glob for full Copilot compatibility.",
		))
		applyTo = `"**"`
	} else {
		switch activation {
		case ast.RuleActivationAlways:
			applyTo = `"**"`
		case ast.RuleActivationPathGlob:
			if len(rule.Paths) > 0 {
				applyTo = fmt.Sprintf("%q", strings.Join(rule.Paths, ", "))
			} else {
				applyTo = `"**"`
			}
		default:
			applyTo = `"**"`
		}
	}

	var sb strings.Builder
	sb.WriteString("---\n")
	sb.WriteString(renderer.BuildRuleDescriptionFrontmatter(rule, caps))

	sb.WriteString(fmt.Sprintf("applyTo: %s\n", applyTo))

	if len(rule.ExcludeAgents) > 0 {
		sb.WriteString("excludeAgent:\n")
		for _, agent := range rule.ExcludeAgents {
			sb.WriteString(fmt.Sprintf("  - %s\n", agent))
		}
	}

	sb.WriteString("---\n")

	body := resolver.StripFrontmatter(rule.Body)
	if body != "" {
		sb.WriteString("\n")
		sb.WriteString(body)
		sb.WriteString("\n")
	}

	return sb.String(), notes, nil
}
