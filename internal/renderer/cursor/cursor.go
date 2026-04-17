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
	"os"
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

// Render wraps a files map in an output.Output. This is an identity
// operation — no additional path rewriting is needed at this layer.
func (r *Renderer) Render(files map[string]string) *output.Output {
	return &output.Output{Files: files}
}

// Compile translates an XcaffoldConfig AST into its Cursor output representation.
// baseDir is the directory that contains the scaffold.xcf file; it is used to
// resolve instructions_file: paths. The second return is a slice of fidelity
// notes describing information loss relative to the native Claude target;
// suppression is applied at the command layer, not inside this renderer.
// Compile returns an error if any resource fails to compile. It never panics.
func (r *Renderer) Compile(config *ast.XcaffoldConfig, baseDir string) (*output.Output, []renderer.FidelityNote, error) {
	out := &output.Output{Files: make(map[string]string)}
	var notes []renderer.FidelityNote

	for id, rule := range config.Rules {
		mdc, ruleNotes, err := compileCursorRule(id, rule, baseDir)
		if err != nil {
			return nil, nil, fmt.Errorf("cursor: failed to compile rule %q: %w", id, err)
		}
		safePath := filepath.Clean(fmt.Sprintf("rules/%s.mdc", id))
		out.Files[safePath] = mdc
		notes = append(notes, ruleNotes...)
	}

	for id, agent := range config.Agents {
		md, agentNotes, err := compileCursorAgent(id, agent, baseDir)
		if err != nil {
			return nil, nil, fmt.Errorf("cursor: failed to compile agent %q: %w", id, err)
		}
		safePath := filepath.Clean(fmt.Sprintf("agents/%s.md", id))
		out.Files[safePath] = md
		notes = append(notes, agentNotes...)
	}

	for id, skill := range config.Skills {
		md, err := compileCursorSkill(id, skill, baseDir)
		if err != nil {
			return nil, nil, fmt.Errorf("cursor: failed to compile skill %q: %w", id, err)
		}
		safePath := filepath.Clean(fmt.Sprintf("skills/%s/SKILL.md", id))
		out.Files[safePath] = md

		if err := compileCursorSkillSubdir(id, "references", skill.References, baseDir, out); err != nil {
			return nil, nil, fmt.Errorf("cursor: skill %q references: %w", id, err)
		}
		if err := compileCursorSkillSubdir(id, "scripts", skill.Scripts, baseDir, out); err != nil {
			return nil, nil, fmt.Errorf("cursor: skill %q scripts: %w", id, err)
		}
		if err := compileCursorSkillSubdir(id, "assets", skill.Assets, baseDir, out); err != nil {
			return nil, nil, fmt.Errorf("cursor: skill %q assets: %w", id, err)
		}
	}

	// Lower workflows via TranslateWorkflow; cursor uses rule-plus-skill when
	// an explicit lowering-strategy is set, otherwise emits a no-native note.
	for id, wf := range config.Workflows {
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
				out.Files[safePath] = content
			case "skill":
				safePath := filepath.Clean(fmt.Sprintf("skills/%s/SKILL.md", p.ID))
				out.Files[safePath] = content
			}
		}
	}

	if len(config.MCP) > 0 {
		mcpJSON, mcpNotes, err := compileCursorMCP(config.MCP)
		if err != nil {
			return nil, nil, fmt.Errorf("cursor: failed to compile mcp servers: %w", err)
		}
		out.Files["mcp.json"] = mcpJSON
		notes = append(notes, mcpNotes...)
	}

	if len(config.Hooks) > 0 {
		hooksJSON, hookNotes, err := compileCursorHooks(config.Hooks)
		if err != nil {
			return nil, nil, fmt.Errorf("cursor: failed to compile hooks: %w", err)
		}
		out.Files["hooks.json"] = hooksJSON
		notes = append(notes, hookNotes...)
	}

	if config.Project != nil {
		instrNotes := r.renderProjectInstructions(config, baseDir, out.Files)
		notes = append(notes, instrNotes...)
	}

	if config.Settings.Permissions != nil {
		notes = append(notes, renderer.NewNote(
			renderer.LevelWarning, targetName, "settings", "global", "permissions",
			renderer.CodeSettingsFieldUnsupported,
			"settings.permissions dropped; Cursor has no permission enforcement. Declared allow/deny/ask rules will NOT apply",
			"Enforce permissions via repository tooling or remove the permissions block for this target",
		))
	}
	if config.Settings.Sandbox != nil {
		notes = append(notes, renderer.NewNote(
			renderer.LevelWarning, targetName, "settings", "global", "sandbox",
			renderer.CodeSettingsFieldUnsupported,
			"settings.sandbox dropped; Cursor has no sandbox model. Filesystem and network restrictions will NOT apply",
			"Remove the sandbox block for this target or use a platform that supports sandboxing",
		))
	}

	return out, notes, nil
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
	p := config.Project
	if p.Instructions == "" && p.InstructionsFile == "" {
		return nil
	}

	var notes []renderer.FidelityNote

	rootContent := cursorResolveInstructions(p.Instructions, p.InstructionsFile, baseDir)

	// Inline @-imports — Cursor has no native @-import mechanism.
	if len(p.InstructionsImports) > 0 {
		for _, imp := range p.InstructionsImports {
			data, err := os.ReadFile(filepath.Join(baseDir, imp))
			if err == nil {
				rootContent += "\n\n" + string(data)
			}
			// On read failure, skip silently; the note still fires below.
		}
		notes = append(notes, renderer.NewNote(
			renderer.LevelInfo,
			targetName,
			"instructions",
			"<root>",
			"instructions-imports",
			renderer.CodeInstructionsImportInlined,
			"@-imports inlined; cursor lacks native @-import support",
			"Remove InstructionsImports or use a target that supports @-imports (e.g. claude)",
		))
	}

	files["AGENTS.md"] = rootContent

	for _, scope := range p.InstructionsScopes {
		scopeContent := cursorResolveScopeContent(scope, targetName, baseDir)
		safePath := filepath.Clean(scope.Path + "/AGENTS.md")

		if scope.MergeStrategy == "concat" {
			// Pre-flatten: child AGENTS.md = root content + scope content.
			files[safePath] = rootContent + "\n\n" + scopeContent
			notes = append(notes, renderer.NewNote(
				renderer.LevelWarning,
				targetName,
				"instructions",
				scope.Path,
				"merge-strategy",
				renderer.CodeInstructionsClosestWinsForcedConcat,
				fmt.Sprintf("concat scope %q pre-flattened into closest-wins child file", scope.Path),
				"Use merge-strategy: closest-wins or flat for Cursor targets",
			))
		} else {
			// closest-wins or flat: child AGENTS.md = scope content only.
			files[safePath] = scopeContent
		}
	}

	return notes
}

// cursorResolveInstructions returns inline instructions or reads InstructionsFile
// relative to baseDir. Returns empty string on any read error.
func cursorResolveInstructions(inline, file, baseDir string) string {
	if inline != "" {
		return inline
	}
	if file == "" {
		return ""
	}
	data, err := os.ReadFile(filepath.Join(baseDir, file))
	if err != nil {
		return ""
	}
	return string(data)
}

// cursorResolveScopeContent returns the effective content for a scope, preferring
// a cursor-specific variant when one is declared.
func cursorResolveScopeContent(scope ast.InstructionsScope, provider, baseDir string) string {
	if v, ok := scope.Variants[provider]; ok {
		return cursorResolveInstructions("", v.InstructionsFile, baseDir)
	}
	return cursorResolveInstructions(scope.Instructions, scope.InstructionsFile, baseDir)
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
		camelEvent := toCamelCase(eventName)
		var eventHandlers []map[string]interface{}

		for _, group := range groups {
			for _, handler := range group.Hooks {
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
			flatHooks[camelEvent] = eventHandlers
		}
	}

	data, err := json.MarshalIndent(flatHooks, "", "  ")
	if err != nil {
		return "", nil, fmt.Errorf("hook json marshal: %w", err)
	}
	return string(data), notes, nil
}

// compileCursorRule renders a single RuleConfig to a Cursor .mdc file.
// It returns the rendered content, any fidelity notes, and an error.
func compileCursorRule(id string, rule ast.RuleConfig, baseDir string) (string, []renderer.FidelityNote, error) {
	if strings.TrimSpace(id) == "" {
		return "", nil, fmt.Errorf("rule id must not be empty")
	}

	body, err := resolver.ResolveInstructions(rule.Instructions, rule.InstructionsFile, "", baseDir)
	if err != nil {
		return "", nil, err
	}

	var sb strings.Builder
	var notes []renderer.FidelityNote

	sb.WriteString("---\n")

	if rule.Description != "" {
		fmt.Fprintf(&sb, "description: %s\n", rule.Description)
	}

	activation := renderer.ResolvedActivation(rule)

	switch activation {
	case ast.RuleActivationAlways:
		sb.WriteString("always-apply: true\n")
	case ast.RuleActivationPathGlob:
		if len(rule.Paths) > 0 {
			fmt.Fprintf(&sb, "globs: [%s]\n", strings.Join(rule.Paths, ", "))
		}
	case ast.RuleActivationManualMention:
		sb.WriteString("always-apply: false\n")
	case ast.RuleActivationModelDecided:
		sb.WriteString("always-apply: false\n")
		notes = append(notes, renderer.NewNote(
			renderer.LevelWarning, targetName, "rule", id, "activation",
			renderer.CodeActivationDegraded,
			fmt.Sprintf("rule %q activation \"model-decided\" has no Cursor equivalent; lowered to always-apply: false", id),
			"Use a supported activation (always, path-glob, manual-mention) or add a targets.cursor.provider override",
		))
	case ast.RuleActivationExplicitInvoke:
		sb.WriteString("always-apply: false\n")
		notes = append(notes, renderer.NewNote(
			renderer.LevelWarning, targetName, "rule", id, "activation",
			renderer.CodeActivationDegraded,
			fmt.Sprintf("rule %q activation \"explicit-invoke\" has no Cursor equivalent; lowered to always-apply: false", id),
			"Use a supported activation (always, path-glob, manual-mention) or add a targets.cursor.provider override",
		))
	default:
		sb.WriteString("always-apply: true\n")
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
func compileCursorAgent(id string, agent ast.AgentConfig, baseDir string) (string, []renderer.FidelityNote, error) {
	if strings.TrimSpace(id) == "" {
		return "", nil, fmt.Errorf("agent id must not be empty")
	}

	body, err := resolver.ResolveInstructions(agent.Instructions, agent.InstructionsFile, "", baseDir)
	if err != nil {
		return "", nil, err
	}

	var sb strings.Builder
	var notes []renderer.FidelityNote

	sb.WriteString("---\n")

	if agent.Name != "" {
		fmt.Fprintf(&sb, "name: %s\n", yamlScalar(agent.Name))
	}
	if agent.Description != "" {
		fmt.Fprintf(&sb, "description: %s\n", yamlScalar(agent.Description))
	}

	if agent.Model != "" {
		if resolved, ok := renderer.ResolveModel(agent.Model, targetName); ok && resolved != "" {
			if renderer.IsMappedModel(agent.Model, targetName) {
				fmt.Fprintf(&sb, "model: %s\n", yamlScalar(resolved))
			} else {
				notes = append(notes, renderer.NewNote(
					renderer.LevelWarning, targetName, "agent", id, "model",
					renderer.CodeAgentModelUnmapped,
					fmt.Sprintf("unmapped model %q (resolved to %q) omitted for agent %q; Cursor requires specific model strings", agent.Model, resolved, id),
					"Use a Cursor-supported model identifier or add a targets.cursor.provider.model override",
				))
			}
		}
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

	if agent.PermissionMode != "" {
		notes = append(notes, renderer.NewNote(
			renderer.LevelWarning, targetName, "agent", id, "permissionMode",
			renderer.CodeAgentSecurityFieldsDropped,
			fmt.Sprintf("agent %q permissionMode %q dropped; Cursor has no permission mode equivalent", id, agent.PermissionMode),
			"Remove permissionMode from the cursor target override",
		))
	}
	if len(agent.DisallowedTools) > 0 {
		notes = append(notes, renderer.NewNote(
			renderer.LevelWarning, targetName, "agent", id, "disallowedTools",
			renderer.CodeAgentSecurityFieldsDropped,
			fmt.Sprintf("agent %q disallowedTools dropped; tool restrictions will NOT be enforced by Cursor", id),
			"Enforce tool restrictions via a different target or accept the loss",
		))
	}
	if agent.Isolation != "" {
		notes = append(notes, renderer.NewNote(
			renderer.LevelWarning, targetName, "agent", id, "isolation",
			renderer.CodeAgentSecurityFieldsDropped,
			fmt.Sprintf("agent %q isolation %q dropped; Cursor has no process isolation model", id, agent.Isolation),
			"Remove isolation from the cursor target override",
		))
	}

	return sb.String(), notes, nil
}

// compileCursorSkill renders a single SkillConfig to a Cursor skills/<id>/SKILL.md file.
func compileCursorSkill(id string, skill ast.SkillConfig, baseDir string) (string, error) {
	if strings.TrimSpace(id) == "" {
		return "", fmt.Errorf("skill id must not be empty")
	}

	body, err := resolver.ResolveInstructions(skill.Instructions, skill.InstructionsFile, "", baseDir)
	if err != nil {
		return "", err
	}

	var sb strings.Builder

	sb.WriteString("---\n")

	if skill.Name != "" {
		fmt.Fprintf(&sb, "name: %s\n", yamlScalar(skill.Name))
	}
	if skill.Description != "" {
		fmt.Fprintf(&sb, "description: %s\n", yamlScalar(skill.Description))
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

// compileCursorSkillSubdir copies skill subdirectory files (references/, scripts/,
// assets/) into the output map under skills/<id>/<subdir>/<filename>.
// Path traversal above baseDir is rejected. Glob patterns are expanded;
// literal paths are read directly.
func compileCursorSkillSubdir(id, subdir string, paths []string, baseDir string, out *output.Output) error {
	if len(paths) == 0 {
		return nil
	}

	for _, pattern := range paths {
		cleanedPattern := filepath.Clean(pattern)
		if strings.HasPrefix(cleanedPattern, "..") {
			return fmt.Errorf("%s path %q traverses above the project root", subdir, pattern)
		}

		absPattern := filepath.Join(baseDir, cleanedPattern)

		matches, err := filepath.Glob(absPattern)
		if err != nil {
			return fmt.Errorf("invalid glob pattern %q: %w", pattern, err)
		}
		if len(matches) == 0 {
			data, readErr := os.ReadFile(absPattern)
			if readErr != nil {
				return fmt.Errorf("%s file %q: %w", subdir, pattern, readErr)
			}
			baseName := filepath.Base(absPattern)
			outPath := filepath.Clean(fmt.Sprintf("skills/%s/%s/%s", id, subdir, baseName))
			out.Files[outPath] = string(data)
			continue
		}

		for _, match := range matches {
			data, err := os.ReadFile(match)
			if err != nil {
				return fmt.Errorf("%s file %q: %w", subdir, match, err)
			}
			baseName := filepath.Base(match)
			outPath := filepath.Clean(fmt.Sprintf("skills/%s/%s/%s", id, subdir, baseName))
			out.Files[outPath] = string(data)
		}
	}
	return nil
}

// yamlScalar quotes a string value for safe inclusion in YAML if it contains
// characters that would otherwise need quoting.
func yamlScalar(s string) string {
	needsQuote := strings.ContainsAny(s, ":#{}[]|>&*!,'\"\\%@`")
	if needsQuote {
		return fmt.Sprintf("%q", s)
	}
	return s
}
