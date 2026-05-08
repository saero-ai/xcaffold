// Package cursor compiles an XcaffoldConfig AST into Cursor rule files.
// Rules are written as .mdc files under rules/ with Cursor-compatible frontmatter.
//
// Key normalizations applied during compilation:
//   - Output extension: .md → .mdc (Rule 5)
//   - Frontmatter key: paths: → globs: (Normalization Rule 4)
//   - Rules without paths receive always-apply: true instead of a globs: field
//   - Agent field: background → is_background (Normalization Rule 6)
//   - Skills emitted to skills/<id>/SKILL.md with bundled scripts/, references/, assets/
//   - MCP field: url preserved as-is; type field omitted (Cursor infers transport)
package cursor

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/output"
	"github.com/saero-ai/xcaffold/internal/renderer"
	"github.com/saero-ai/xcaffold/internal/resolver"
	"github.com/saero-ai/xcaffold/internal/translator"
)

const targetName = "cursor"

// Renderer compiles an XcaffoldConfig AST into Cursor output files.
// It targets the ".cursor/rules/" directory structure understood by Cursor.
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
	return ".cursor"
}

// Capabilities returns the CapabilitySet for the Cursor renderer.
// Cursor supports agents, skills (with references/scripts/assets subdirs), rules,
// workflows (via rule-plus-skill lowering), hooks, MCP, and project instructions.
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
		RuleActivations:      []string{"always", "path-glob", "manual-mention"},
		RuleEncoding: renderer.RuleEncodingCapabilities{
			Description: "frontmatter",
			Activation:  "frontmatter",
		},
		SkillArtifactDirs: map[string]string{
			"references": "references",
			"scripts":    "scripts",
			"assets":     "assets",
			"examples":   "references",
		},
	}
}

// CompileAgents renders all agents to Cursor agents/<id>.md files.
func (r *Renderer) CompileAgents(agents map[string]ast.AgentConfig, baseDir string) (map[string]string, []renderer.FidelityNote, error) {
	files := make(map[string]string)
	caps := r.Capabilities()
	var notes []renderer.FidelityNote
	for id, agent := range agents {
		md, agentNotes, err := compileCursorAgent(id, agent, baseDir, caps)
		if err != nil {
			return nil, nil, fmt.Errorf("cursor: failed to compile agent %q: %w", id, err)
		}
		safePath := filepath.Clean(fmt.Sprintf("agents/%s.md", id))
		files[safePath] = md
		notes = append(notes, agentNotes...)
	}
	return files, notes, nil
}

// CompileSkills renders all skills to Cursor skills/<id>/SKILL.md files,
// including references, scripts, and assets subdirectories.
func (r *Renderer) CompileSkills(skills map[string]ast.SkillConfig, baseDir string) (map[string]string, []renderer.FidelityNote, error) {
	out := &output.Output{Files: make(map[string]string)}
	caps := r.Capabilities()
	for id, skill := range skills {
		md, err := compileCursorSkill(id, skill, baseDir)
		if err != nil {
			return nil, nil, fmt.Errorf("cursor: failed to compile skill %q: %w", id, err)
		}
		safePath := filepath.Clean(fmt.Sprintf("skills/%s/SKILL.md", id))
		out.Files[safePath] = md

		skillSourceDir := filepath.Join("xcaf", "skills", id)
		if len(skill.Artifacts) > 0 {
			if err := compileSkillArtifacts(id, skill, caps, baseDir, skillSourceDir, out); err != nil {
				return nil, nil, fmt.Errorf("cursor: skill %q: %w", id, err)
			}
		} else {
			// Legacy path: individual fields for skills that predate the artifacts field.
			if err := renderer.CompileSkillSubdir(id, "references", "references", skill.References.Values, baseDir, skillSourceDir, out); err != nil {
				return nil, nil, fmt.Errorf("cursor: skill %q references: %w", id, err)
			}
			if err := renderer.CompileSkillSubdir(id, "scripts", "scripts", skill.Scripts.Values, baseDir, skillSourceDir, out); err != nil {
				return nil, nil, fmt.Errorf("cursor: skill %q scripts: %w", id, err)
			}
			if err := renderer.CompileSkillSubdir(id, "assets", "assets", skill.Assets.Values, baseDir, skillSourceDir, out); err != nil {
				return nil, nil, fmt.Errorf("cursor: skill %q assets: %w", id, err)
			}
			if err := renderer.CompileSkillSubdir(id, "examples", "references", skill.Examples.Values, baseDir, skillSourceDir, out); err != nil {
				return nil, nil, fmt.Errorf("cursor: skill %q examples: %w", id, err)
			}
		}
	}
	return out.Files, nil, nil
}

// compileSkillArtifacts iterates skill.Artifacts and dispatches each artifact
// to the correct output subdirectory using the renderer's SkillArtifactDirs map.
func compileSkillArtifacts(id string, skill ast.SkillConfig, caps renderer.CapabilitySet, baseDir, skillSourceDir string, out *output.Output) error {
	for _, artifactName := range skill.Artifacts {
		outputSubdir, ok := caps.SkillArtifactDirs[artifactName]
		if !ok {
			outputSubdir = artifactName
		}
		var paths []string
		switch artifactName {
		case "references":
			paths = skill.References.Values
		case "scripts":
			paths = skill.Scripts.Values
		case "assets":
			paths = skill.Assets.Values
		case "examples":
			paths = skill.Examples.Values
		}
		if len(paths) == 0 {
			continue
		}
		if err := renderer.CompileSkillSubdir(id, artifactName, outputSubdir, paths, baseDir, skillSourceDir, out); err != nil {
			return fmt.Errorf("artifact %s: %w", artifactName, err)
		}
	}
	return nil
}

// CompileRules renders all rules to Cursor rules/<id>.mdc files.
func (r *Renderer) CompileRules(rules map[string]ast.RuleConfig, baseDir string) (map[string]string, []renderer.FidelityNote, error) {
	files := make(map[string]string)
	var notes []renderer.FidelityNote
	for id, rule := range rules {
		mdc, ruleNotes, err := compileCursorRule(id, rule, r.Capabilities(), baseDir)
		if err != nil {
			return nil, nil, fmt.Errorf("cursor: failed to compile rule %q: %w", id, err)
		}
		safePath := filepath.Clean(fmt.Sprintf("rules/%s.mdc", id))
		files[safePath] = mdc
		notes = append(notes, ruleNotes...)
	}
	return files, notes, nil
}

// CompileWorkflows lowers workflows via TranslateWorkflow; Cursor uses
// rule-plus-skill primitives when an explicit lowering-strategy is set.
func (r *Renderer) CompileWorkflows(workflows map[string]ast.WorkflowConfig, baseDir string) (map[string]string, []renderer.FidelityNote, error) {
	files := make(map[string]string)
	var notes []renderer.FidelityNote
	for id, wf := range workflows {
		wfCopy := wf
		if wfCopy.Name == "" {
			wfCopy.Name = id
		}
		primitives, wfNotes := translator.TranslateWorkflow(&wfCopy, targetName)
		notes = append(notes, wfNotes...)
		for _, p := range primitives {
			content := p.Content
			if content == "" {
				content = p.Body
			}
			switch p.Kind {
			case "rule":
				safePath := filepath.Clean(fmt.Sprintf("rules/%s.mdc", p.ID))
				files[safePath] = content
			case "skill":
				safePath := filepath.Clean(fmt.Sprintf("skills/%s/SKILL.md", p.ID))
				files[safePath] = content
			}
		}
	}
	return files, notes, nil
}

// CompileHooks flattens Claude's 3-level HookConfig to Cursor's 2-level format
// and writes it to hooks.json.
func (r *Renderer) CompileHooks(hooks ast.HookConfig, baseDir string) (map[string]string, []renderer.FidelityNote, error) {
	hooksJSON, notes, err := compileCursorHooks(hooks)
	if err != nil {
		return nil, nil, fmt.Errorf("cursor: failed to compile hooks: %w", err)
	}
	files := map[string]string{"hooks.json": hooksJSON}
	return files, notes, nil
}

// CompileSettings emits fidelity notes for unsupported Cursor settings fields
// (permissions, sandbox). Cursor has no settings.json equivalent.
func (r *Renderer) CompileSettings(settings ast.SettingsConfig) (map[string]string, []renderer.FidelityNote, error) {
	var notes []renderer.FidelityNote
	if settings.Permissions != nil {
		notes = append(notes, renderer.NewNote(
			renderer.LevelWarning, targetName, "settings", "global", "permissions",
			renderer.CodeSettingsFieldUnsupported,
			"settings.permissions dropped; Cursor has no permission enforcement. Declared allow/deny/ask rules will NOT apply",
			"Enforce permissions via repository tooling or remove the permissions block for this target",
		))
	}
	if settings.Sandbox != nil {
		notes = append(notes, renderer.NewNote(
			renderer.LevelWarning, targetName, "settings", "global", "sandbox",
			renderer.CodeSettingsFieldUnsupported,
			"settings.sandbox dropped; Cursor has no sandbox model. Filesystem and network restrictions will NOT apply",
			"Remove the sandbox block for this target or use a platform that supports sandboxing",
		))
	}
	return nil, notes, nil
}

// CompileMCP renders all MCP server configs to a single mcp.json file.
func (r *Renderer) CompileMCP(servers map[string]ast.MCPConfig) (map[string]string, []renderer.FidelityNote, error) {
	mcpJSON, notes, err := compileCursorMCP(servers)
	if err != nil {
		return nil, nil, fmt.Errorf("cursor: failed to compile mcp servers: %w", err)
	}
	files := map[string]string{"mcp.json": mcpJSON}
	return files, notes, nil
}

// CompileProjectInstructions emits AGENTS.md at root and one AGENTS.md per scope.
func (r *Renderer) CompileProjectInstructions(config *ast.XcaffoldConfig, baseDir string) (map[string]string, map[string]string, []renderer.FidelityNote, error) {
	// Build a minimal config shell to reuse renderProjectInstructions.
	rootFiles := make(map[string]string)
	notes := r.renderProjectInstructions(config, baseDir, rootFiles)
	return nil, rootFiles, notes, nil
}

// CompileMemory delegates to MemoryRenderer. Cursor has no native per-file
// memory primitive; the renderer emits FidelityNotes advising use of rules.
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

// Finalize is a no-op post-processing pass for the Cursor renderer.
// Path normalization is already applied per-resource during compilation.
func (r *Renderer) Finalize(files map[string]string, rootFiles map[string]string) (map[string]string, map[string]string, []renderer.FidelityNote, error) {
	return files, rootFiles, nil, nil
}

// renderProjectInstructions emits AGENTS.md at root and one AGENTS.md per scope.
// Cursor uses the closest-wins nesting class: each subdirectory's AGENTS.md is
// authoritative for that directory; parent files do not cascade automatically.
//
// Deviation handling:
//   - concat-tagged scopes are pre-flattened: child = root + scope content.
//     A INSTRUCTIONS_CLOSEST_WINS_FORCED_CONCAT warning is emitted per scope.
//   - InstructionsImports are inlined because Cursor has no native @-import support.
//     A single INSTRUCTIONS_IMPORT_INLINED info note is emitted when any imports exist.
func (r *Renderer) renderProjectInstructions(config *ast.XcaffoldConfig, baseDir string, files map[string]string) []renderer.FidelityNote {
	rootContent := renderer.ResolveContextBody(config, targetName)
	if rootContent == "" {
		return nil
	}

	files["AGENTS.md"] = rootContent
	return nil
}

// xcaffoldToCursorEvent maps xcaffold hook event names to their Cursor equivalents.
// Unmapped events fall back to toCamelCase with a fidelity note.
var xcaffoldToCursorEvent = map[string]string{
	// xcaffold native names (from Claude Code event names)
	"PreToolUse":       "preToolUse",
	"PostToolUse":      "postToolUse",
	"UserPromptSubmit": "beforeSubmitPrompt",
	"SessionStart":     "sessionStart",
	"SessionEnd":       "sessionEnd",
	"SubagentStart":    "subagentStart",
	"SubagentStop":     "subagentStop",
	"Stop":             "stop",
	// Gemini-style aliases (to handle xcaf files originally authored for Gemini)
	"PreToolExecution":  "preToolUse",
	"PostToolExecution": "postToolUse",
	"BeforeTool":        "preToolUse",
	"AfterTool":         "postToolUse",
	"BeforeAgent":       "beforeSubmitPrompt",
}

// toCamelCase lowercases the first character of a string (PreToolUse -> preToolUse)
func toCamelCase(s string) string {
	if s == "" {
		return ""
	}
	return strings.ToLower(s[:1]) + s[1:]
}

// compileCursorHooks flattens Claude's 3-level HookConfig to Cursor's 2-level format.
func compileCursorHooks(hooks ast.HookConfig) (string, []renderer.FidelityNote, error) {
	flatHooks := make(map[string][]map[string]interface{})
	var notes []renderer.FidelityNote

	for eventName, groups := range hooks {
		cursorEvent, mapped := xcaffoldToCursorEvent[eventName]
		if !mapped {
			cursorEvent = toCamelCase(eventName)
			notes = append(notes, renderer.NewNote(
				renderer.LevelWarning, targetName, "hooks", "hooks", eventName,
				renderer.CodeFieldUnsupported,
				fmt.Sprintf("hook event %q has no verified Cursor equivalent; emitted as %q (camelCase fallback)", eventName, cursorEvent),
				"Verify this event name against Cursor hooks documentation",
			))
		}

		var eventHandlers []map[string]interface{}

		for _, group := range groups {
			for _, handler := range group.Hooks {
				handler.Command = renderer.TranslateHookCommand(handler.Command, "$CURSOR_PROJECT_DIR", ".cursor/hooks/")

				b, err := json.Marshal(handler)
				if err != nil {
					return "", nil, err
				}
				var flatHandler map[string]interface{}
				if err := json.Unmarshal(b, &flatHandler); err != nil {
					return "", nil, err
				}

				if group.Matcher != "" {
					flatHandler["matcher"] = group.Matcher
				}

				if strings.Contains(string(b), "${") {
					notes = append(notes, renderer.NewNote(
						renderer.LevelWarning, targetName, "agent", eventName, "hooks",
						renderer.CodeHookInterpolationRequiresEnvSyntax,
						fmt.Sprintf("interpolation pattern ${...} in hook %q; Cursor requires ${env:NAME} syntax", eventName),
						"Rewrite ${VAR} as ${env:VAR} in hook configuration",
					))
				}

				eventHandlers = append(eventHandlers, flatHandler)
			}
		}

		if len(eventHandlers) > 0 {
			flatHooks[cursorEvent] = eventHandlers
		}
	}

	envelope := map[string]any{"version": 1}
	for k, v := range flatHooks {
		envelope[k] = v
	}

	data, err := json.MarshalIndent(envelope, "", "  ")
	if err != nil {
		return "", nil, fmt.Errorf("hook json marshal: %w", err)
	}
	return string(data), notes, nil
}

// compileCursorRule renders a single RuleConfig to a Cursor .mdc file.
// It returns the rendered content, any fidelity notes, and an error.
func compileCursorRule(id string, rule ast.RuleConfig, caps renderer.CapabilitySet, baseDir string) (string, []renderer.FidelityNote, error) {
	if strings.TrimSpace(id) == "" {
		return "", nil, fmt.Errorf("rule id must not be empty")
	}

	body := resolver.StripFrontmatter(rule.Body)

	var sb strings.Builder
	var notes []renderer.FidelityNote

	sb.WriteString("---\n")
	sb.WriteString(renderer.BuildRuleDescriptionFrontmatter(rule, caps))

	activation := renderer.ResolvedActivation(rule)
	if !renderer.ValidateRuleActivation(rule, caps) {
		sb.WriteString("always-apply: false\n")
		notes = append(notes, renderer.NewNote(
			renderer.LevelWarning, targetName, "rule", id, "activation",
			renderer.CodeActivationDegraded,
			fmt.Sprintf("rule %q activation %q has no Cursor equivalent; lowered to always-apply: false", id, activation),
			"Use a supported activation (always, path-glob) or add a targets.cursor.provider override",
		))
	} else {
		switch activation {
		case ast.RuleActivationAlways:
			sb.WriteString("always-apply: true\n")
		case ast.RuleActivationPathGlob:
			if len(rule.Paths.Values) > 0 {
				fmt.Fprintf(&sb, "globs: [%s]\n", strings.Join(rule.Paths.Values, ", "))
			}
		case ast.RuleActivationManualMention:
			sb.WriteString("always-apply: false\n")
		}
	}

	sb.WriteString("---\n")

	if body != "" {
		sb.WriteString("\n")
		sb.WriteString(strings.TrimRight(body, "\n"))
		sb.WriteString("\n")
	}

	return sb.String(), notes, nil
}

// compileCursorAgent renders a single AgentConfig to a Cursor agents/<id>.md file.
//
//nolint:gocyclo
func compileCursorAgent(id string, agent ast.AgentConfig, baseDir string, caps renderer.CapabilitySet) (string, []renderer.FidelityNote, error) {
	if strings.TrimSpace(id) == "" {
		return "", nil, fmt.Errorf("agent id must not be empty")
	}

	body := resolver.StripFrontmatter(agent.Body)

	var sb strings.Builder
	var notes []renderer.FidelityNote

	sb.WriteString("---\n")

	if agent.Name != "" {
		fmt.Fprintf(&sb, "name: %s\n", renderer.YAMLScalar(agent.Name))
	}
	if agent.Description != "" {
		fmt.Fprintf(&sb, "description: %s\n", renderer.YAMLScalar(agent.Description))
	}

	_, toolNotes := renderer.SanitizeAgentTools(agent.Tools.Values, caps, targetName, id)
	notes = append(notes, toolNotes...)

	resolvedModel, modelNotes := renderer.SanitizeAgentModel(agent.Model, caps, targetName, id)
	notes = append(notes, modelNotes...)
	// Cursor only accepts explicitly mapped models.
	if resolvedModel != "" && renderer.IsMappedModel(agent.Model, targetName) {
		fmt.Fprintf(&sb, "model: %s\n", renderer.YAMLScalar(resolvedModel))
	} else if agent.Model != "" && !renderer.IsMappedModel(agent.Model, targetName) {
		notes = append(notes, renderer.NewNote(
			renderer.LevelWarning, targetName, "agent", id, "model",
			renderer.CodeAgentModelUnmapped,
			fmt.Sprintf("literal model %q is unmapped for cursor \u2014 must be omitted", agent.Model),
			"Use a mapped alias defined in xcaffold model aliases",
		))
	}
	if agent.Background != nil && *agent.Background {
		sb.WriteString("is_background: true\n")
	}
	if agent.Readonly != nil && *agent.Readonly {
		sb.WriteString("readonly: true\n")
	}

	sb.WriteString("---\n")

	if body != "" {
		sb.WriteString("\n")
		sb.WriteString(strings.TrimRight(body, "\n"))
		sb.WriteString("\n")
	}

	return sb.String(), notes, nil
}

// compileCursorSkill renders a single SkillConfig to a Cursor skills/<id>/SKILL.md file.
func compileCursorSkill(id string, skill ast.SkillConfig, baseDir string) (string, error) {
	if strings.TrimSpace(id) == "" {
		return "", fmt.Errorf("skill id must not be empty")
	}

	body := resolver.StripFrontmatter(skill.Body)

	var sb strings.Builder

	sb.WriteString("---\n")

	if skill.Name != "" {
		fmt.Fprintf(&sb, "name: %s\n", renderer.YAMLScalar(skill.Name))
	}
	if skill.Description != "" {
		fmt.Fprintf(&sb, "description: %s\n", renderer.YAMLScalar(skill.Description))
	}

	sb.WriteString("---\n")

	if body != "" {
		sb.WriteString("\n")
		sb.WriteString(strings.TrimRight(body, "\n"))
		sb.WriteString("\n")
	}

	return sb.String(), nil
}

// cursorMCPEntry is the Cursor-compatible MCP server entry shape.
type cursorMCPEntry struct {
	Env     map[string]string `json:"env,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
	Command string            `json:"command,omitempty"`
	URL     string            `json:"url,omitempty"`
	Args    []string          `json:"args,omitempty"`
}

// compileCursorMCP renders all MCP server configs to a single mcp.json file.
func compileCursorMCP(servers map[string]ast.MCPConfig) (string, []renderer.FidelityNote, error) {
	entries := make(map[string]cursorMCPEntry, len(servers))
	var notes []renderer.FidelityNote

	for id, srv := range servers {
		entries[id] = cursorMCPEntry{
			Command: srv.Command,
			Args:    srv.Args,
			Env:     srv.Env,
			URL:     srv.URL,
			Headers: srv.Headers,
		}

		if strings.Contains(srv.Command, "${") {
			notes = append(notes, renderer.NewNote(
				renderer.LevelWarning, targetName, "settings", id, "mcp.command",
				renderer.CodeHookInterpolationRequiresEnvSyntax,
				fmt.Sprintf("interpolation pattern ${...} in MCP command for %q; Cursor requires ${env:NAME} syntax", id),
				"Rewrite ${VAR} as ${env:VAR} in MCP server command",
			))
		}
		for _, arg := range srv.Args {
			if strings.Contains(arg, "${") {
				notes = append(notes, renderer.NewNote(
					renderer.LevelWarning, targetName, "settings", id, "mcp.args",
					renderer.CodeHookInterpolationRequiresEnvSyntax,
					fmt.Sprintf("interpolation pattern ${...} in MCP args for %q; Cursor requires ${env:NAME} syntax", id),
					"Rewrite ${VAR} as ${env:VAR} in MCP server args",
				))
				break
			}
		}
		for k, v := range srv.Env {
			if strings.Contains(v, "${") {
				notes = append(notes, renderer.NewNote(
					renderer.LevelWarning, targetName, "settings", id, "mcp.env",
					renderer.CodeHookInterpolationRequiresEnvSyntax,
					fmt.Sprintf("interpolation pattern ${...} in MCP env %q for server %q; Cursor requires ${env:NAME} syntax", k, id),
					"Rewrite ${VAR} as ${env:VAR} in MCP server env",
				))
			}
		}
	}

	envelope := map[string]map[string]cursorMCPEntry{
		"mcpServers": entries,
	}

	data, err := json.MarshalIndent(envelope, "", "  ")
	if err != nil {
		return "", nil, fmt.Errorf("mcp json marshal: %w", err)
	}
	return string(data), notes, nil
}
