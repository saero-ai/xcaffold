// Package gemini compiles an XcaffoldConfig AST into Gemini CLI output files.
// Project instructions are written to GEMINI.md using concat-nested semantics with
// native @-import preservation. Rules are written to .gemini/rules/<id>.md and
// referenced via @-import lines in GEMINI.md.
package gemini

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/output"
	"github.com/saero-ai/xcaffold/internal/renderer"
	"github.com/saero-ai/xcaffold/internal/resolver"
	"github.com/saero-ai/xcaffold/internal/translator"
	"gopkg.in/yaml.v3"
)

const targetName = "gemini"

// Renderer compiles an XcaffoldConfig AST into Gemini CLI output files.
type Renderer struct{}

// New returns a new Renderer instance.
func New() *Renderer { return &Renderer{} }

// Target returns the canonical name of this renderer.
func (r *Renderer) Target() string { return targetName }

// OutputDir returns the base output directory for Gemini CLI.
func (r *Renderer) OutputDir() string { return ".gemini" }

// Render wraps a files map in an output.Output. This is an identity
// operation — no additional path rewriting is needed at this layer.
func (r *Renderer) Render(files map[string]string) *output.Output {
	return &output.Output{Files: files}
}

// Compile translates an XcaffoldConfig AST into Gemini CLI output files.
// baseDir is the directory containing the scaffold.xcf file; it is used to
// resolve instructions-file paths. Compile returns an error if any resource
// fails to compile. It never panics.
func (r *Renderer) Compile(config *ast.XcaffoldConfig, baseDir string) (*output.Output, []renderer.FidelityNote, error) {
	out := &output.Output{Files: make(map[string]string)}
	var notes []renderer.FidelityNote

	if config.Project != nil {
		instrNotes := r.renderProjectInstructions(config, baseDir, out.Files)
		notes = append(notes, instrNotes...)
	}

	// Lower workflows to rule+skill primitives before rendering rules and skills.
	// Lowered primitives are merged into the config copies used for rendering.
	config, workflowNotes := r.lowerWorkflows(config)
	notes = append(notes, workflowNotes...)

	ruleNotes, err := r.renderRules(config, out.Files, baseDir)
	if err != nil {
		return nil, nil, err
	}
	notes = append(notes, ruleNotes...)

	skillNotes := r.renderSkills(config, baseDir, out.Files)
	notes = append(notes, skillNotes...)

	agentNotes := r.renderAgents(config, baseDir, out.Files)
	notes = append(notes, agentNotes...)

	settingsJSON, settingsNotes, err := compileGeminiSettings(config.Hooks, config.MCP, config.Settings)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to compile gemini settings: %w", err)
	}
	notes = append(notes, settingsNotes...)
	if settingsJSON != "" {
		out.Files[".gemini/settings.json"] = settingsJSON
	}

	return out, notes, nil
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

	files["GEMINI.md"] = sb.String()

	// Emit per-scope GEMINI.md files.
	for _, scope := range p.InstructionsScopes {
		scopeContent := renderer.ResolveScopeContent(scope, targetName, baseDir)
		if scopeContent == "" {
			continue
		}
		scopePath := filepath.Join(scope.Path, "GEMINI.md")
		safePath := filepath.Clean(scopePath)
		files[safePath] = scopeContent
	}

	return nil
}

// renderRules writes each rule to .gemini/rules/<id>.md and appends @-import
// lines to GEMINI.md. Rules with unsupported activation modes emit a fidelity note
// but are still written (Gemini treats all imported rules as always-active).
// baseDir is used to resolve instructions-file paths on rules; pass "" when no
// file resolution is needed.
func (r *Renderer) renderRules(config *ast.XcaffoldConfig, files map[string]string, baseDir string) ([]renderer.FidelityNote, error) {
	if len(config.Rules) == 0 {
		return nil, nil
	}

	var notes []renderer.FidelityNote
	var importLines []string

	for _, id := range sortedKeys(config.Rules) {
		rule := config.Rules[id]

		activation := renderer.ResolvedActivation(rule)
		if activation != ast.RuleActivationAlways && activation != ast.RuleActivationPathGlob {
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

		body := buildRuleBody(rule, baseDir)
		rulePath := fmt.Sprintf(".gemini/rules/%s.md", id)
		safePath := filepath.Clean(rulePath)
		files[safePath] = body
		importLines = append(importLines, fmt.Sprintf("@%s", safePath))
	}

	if len(importLines) > 0 {
		existing := files["GEMINI.md"]
		if existing != "" && !strings.HasSuffix(existing, "\n") {
			existing += "\n"
		}
		files["GEMINI.md"] = existing + strings.Join(importLines, "\n") + "\n"
	}

	return notes, nil
}

// renderSkills writes each skill to .gemini/skills/<id>/SKILL.md using the
// agentskills.io format: YAML frontmatter (name + description) + markdown body.
func (r *Renderer) renderSkills(config *ast.XcaffoldConfig, baseDir string, files map[string]string) []renderer.FidelityNote {
	if len(config.Skills) == 0 {
		return nil
	}

	var notes []renderer.FidelityNote

	for _, id := range sortedKeys(config.Skills) {
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

		filePath := fmt.Sprintf(".gemini/skills/%s/SKILL.md", id)
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
		if len(skill.Scripts) > 0 {
			notes = append(notes, renderer.NewNote(
				renderer.LevelWarning, targetName, "skill", id, "scripts",
				renderer.CodeSkillScriptsDropped,
				fmt.Sprintf("skill %q scripts dropped; Gemini does not support skill scripts/ directories", id),
				"Copy scripts into .gemini/skills/"+id+"/scripts/ manually",
			))
		}
		if len(skill.Assets) > 0 {
			notes = append(notes, renderer.NewNote(
				renderer.LevelWarning, targetName, "skill", id, "assets",
				renderer.CodeSkillAssetsDropped,
				fmt.Sprintf("skill %q assets dropped; Gemini does not support skill assets/ directories", id),
				"Copy assets into .gemini/skills/"+id+"/assets/ manually",
			))
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

// renderAgents writes each agent to .gemini/agents/<id>.md using YAML
// frontmatter (name, description, tools, model, max_turns, mcpServers) with
// a markdown body as the system prompt. Gemini-specific fields (timeout_mins,
// temperature, kind) are sourced from targets.gemini.provider pass-through.
// Unsupported fields emit fidelity notes.
func (r *Renderer) renderAgents(config *ast.XcaffoldConfig, baseDir string, files map[string]string) []renderer.FidelityNote {
	agents := config.Agents
	if len(agents) == 0 {
		return nil
	}

	var notes []renderer.FidelityNote

	for _, id := range sortedKeys(agents) {
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
		if len(agent.Tools) > 0 {
			sb.WriteString("tools:\n")
			for _, tool := range agent.Tools {
				fmt.Fprintf(&sb, "  - %s\n", tool)
			}
		}
		if agent.Model != "" {
			fmt.Fprintf(&sb, "model: %s\n", agent.Model)
		}
		if agent.MaxTurns > 0 {
			fmt.Fprintf(&sb, "max_turns: %d\n", agent.MaxTurns)
		}

		// Inline MCP servers.
		if len(agent.MCPServers) > 0 {
			sb.WriteString("mcpServers:\n")
			for _, mcpID := range sortedKeys(agent.MCPServers) {
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
					for _, envKey := range sortedKeys(mcp.Env) {
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

		filePath := fmt.Sprintf(".gemini/agents/%s.md", id)
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

// lowerWorkflows translates each workflow in config into rule and skill
// primitives via translator.TranslateWorkflow, then returns a shallow copy of
// config with the lowered primitives merged into Rules and Skills. The original
// config is never mutated. Fidelity notes from the lowering are also returned.
func (r *Renderer) lowerWorkflows(config *ast.XcaffoldConfig) (*ast.XcaffoldConfig, []renderer.FidelityNote) {
	if len(config.Workflows) == 0 {
		return config, nil
	}

	// Shallow-copy ResourceScope so we can merge without mutating the input.
	merged := *config
	rs := config.ResourceScope

	mergedRules := make(map[string]ast.RuleConfig, len(rs.Rules))
	for k, v := range rs.Rules {
		mergedRules[k] = v
	}

	mergedSkills := make(map[string]ast.SkillConfig, len(rs.Skills))
	for k, v := range rs.Skills {
		mergedSkills[k] = v
	}

	var notes []renderer.FidelityNote

	for _, id := range sortedKeys(rs.Workflows) {
		wf := rs.Workflows[id]
		if wf.Name == "" {
			wf.Name = id
		}
		primitives, wfNotes := translator.TranslateWorkflow(&wf, targetName)
		notes = append(notes, wfNotes...)

		for _, p := range primitives {
			switch p.Kind {
			case "rule":
				body := p.Content
				if body == "" {
					body = p.Body
				}
				mergedRules[p.ID] = ast.RuleConfig{
					Description:  wf.Description,
					Instructions: body,
				}
			case "skill":
				body := p.Content
				if body == "" {
					body = p.Body
				}
				mergedSkills[p.ID] = ast.SkillConfig{
					Name:         p.ID,
					Instructions: body,
				}
			}
		}
	}

	rs.Rules = mergedRules
	rs.Skills = mergedSkills
	merged.ResourceScope = rs
	return &merged, notes
}

// buildRuleBody constructs the markdown content for a rule file.
// baseDir is used to resolve instructions-file paths; pass "" when no
// file resolution is needed.
func buildRuleBody(rule ast.RuleConfig, baseDir string) string {
	var sb strings.Builder
	if rule.Description != "" {
		fmt.Fprintf(&sb, "# %s\n\n", rule.Description)
	}
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

// sortedKeys returns a sorted slice of keys from a map.
func sortedKeys[K ~string, V any](m map[K]V) []K {
	keys := make([]K, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	return keys
}
