// Package cursor compiles an XcaffoldConfig AST into Cursor rule files.
// Rules are written as .mdc files under rules/ with Cursor-compatible frontmatter.
//
// Key normalizations applied during compilation:
//   - Output extension: .md → .mdc (Rule 5)
//   - Frontmatter key: paths: → globs: (Normalization Rule 4)
//   - Rules without paths receive alwaysApply: true instead of a globs: field
//   - Agent field: background → is_background (Normalization Rule 6)
//   - Skills emitted to skills/<id>/SKILL.md (shared skills/ path)
//   - MCP field: url → serverUrl; type field omitted (Cursor infers transport)
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
)

// Renderer compiles an XcaffoldConfig AST into Cursor output files.
// It targets the ".cursor/rules/" directory structure understood by Cursor.
type Renderer struct{}

// New returns a new Renderer instance.
func New() *Renderer {
	return &Renderer{}
}

// Target returns the identifier for this renderer's target platform.
func (r *Renderer) Target() string {
	return "cursor"
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
// resolve instructions_file: paths. Compile returns an error if any resource
// fails to compile. It never panics.
func (r *Renderer) Compile(config *ast.XcaffoldConfig, baseDir string) (*output.Output, error) {
	out := &output.Output{
		Files: make(map[string]string),
	}

	// Compile all rules to rules/<id>.mdc
	for id, rule := range config.Rules {
		mdc, err := compileCursorRule(id, rule, baseDir)
		if err != nil {
			return nil, fmt.Errorf("cursor: failed to compile rule %q: %w", id, err)
		}
		safePath := filepath.Clean(fmt.Sprintf("rules/%s.mdc", id))
		out.Files[safePath] = mdc
	}

	// Compile all agents to agents/<id>.md
	for id, agent := range config.Agents {
		md, err := compileCursorAgent(id, agent, baseDir)
		if err != nil {
			return nil, fmt.Errorf("cursor: failed to compile agent %q: %w", id, err)
		}
		safePath := filepath.Clean(fmt.Sprintf("agents/%s.md", id))
		out.Files[safePath] = md
	}

	// Compile all skills to skills/<id>/SKILL.md
	for id, skill := range config.Skills {
		md, err := compileCursorSkill(id, skill, baseDir)
		if err != nil {
			return nil, fmt.Errorf("cursor: failed to compile skill %q: %w", id, err)
		}
		safePath := filepath.Clean(fmt.Sprintf("skills/%s/SKILL.md", id))
		out.Files[safePath] = md
	}

	// Compile MCP servers to mcp.json (only if any servers are defined)
	if len(config.MCP) > 0 {
		mcpJSON, err := compileCursorMCP(config.MCP)
		if err != nil {
			return nil, fmt.Errorf("cursor: failed to compile mcp servers: %w", err)
		}
		out.Files["mcp.json"] = mcpJSON
	}

	// Compile Hooks to hooks.json
	if len(config.Hooks) > 0 {
		hooksJSON, err := compileCursorHooks(config.Hooks)
		if err != nil {
			return nil, fmt.Errorf("cursor: failed to compile hooks: %w", err)
		}
		out.Files["hooks.json"] = hooksJSON
	}

	// Emit security fidelity warnings for dropped settings fields.
	if config.Settings.Permissions != nil {
		fmt.Fprintf(os.Stderr, "WARNING (cursor): settings.permissions dropped — Cursor has no permission enforcement. Declared allow/deny/ask rules will NOT apply.\n")
	}
	if config.Settings.Sandbox != nil {
		fmt.Fprintf(os.Stderr, "WARNING (cursor): settings.sandbox dropped — Cursor has no sandbox model. Filesystem and network restrictions will NOT apply.\n")
	}

	return out, nil
}

// toCamelCase lowercases the first character of a string (PreToolUse -> preToolUse)
func toCamelCase(s string) string {
	if s == "" {
		return ""
	}
	return strings.ToLower(s[:1]) + s[1:]
}

// compileCursorHooks flattens Claude's 3-level HookConfig to Cursor's 2-level format.
func compileCursorHooks(hooks ast.HookConfig) (string, error) {
	// target map is: eventName -> array of flat hook handlers (with matcher added inline)
	flatHooks := make(map[string][]map[string]interface{})

	for eventName, groups := range hooks {
		camelEvent := toCamelCase(eventName)
		var eventHandlers []map[string]interface{}

		for _, group := range groups {
			for _, handler := range group.Hooks {
				// Convert to generic map to inject the matcher field safely
				b, err := json.Marshal(handler)
				if err != nil {
					return "", err
				}
				var flatHandler map[string]interface{}
				if err := json.Unmarshal(b, &flatHandler); err != nil {
					return "", err
				}

				if group.Matcher != "" {
					flatHandler["matcher"] = group.Matcher
				}

				// Warn on interpolation syntax differences
				if strings.Contains(string(b), "${") {
					fmt.Fprintf(os.Stderr, "WARNING (cursor): interpolation pattern ${...} found in hook %q. Cursor requires ${env:NAME} syntax.\n", eventName)
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
		return "", fmt.Errorf("hook json marshal: %w", err)
	}
	return string(data), nil
}

// compileCursorRule renders a single RuleConfig to a Cursor .mdc file.
//
// Normalizations:
//   - paths: values are emitted as globs: in frontmatter
//   - absent paths → alwaysApply: true
//   - body content is preserved verbatim after the closing frontmatter delimiter
func compileCursorRule(id string, rule ast.RuleConfig, baseDir string) (string, error) {
	if strings.TrimSpace(id) == "" {
		return "", fmt.Errorf("rule id must not be empty")
	}

	body, err := resolver.ResolveInstructions(rule.Instructions, rule.InstructionsFile, "", baseDir)
	if err != nil {
		return "", err
	}

	var sb strings.Builder

	sb.WriteString("---\n")

	if rule.Description != "" {
		fmt.Fprintf(&sb, "description: %s\n", rule.Description)
	}

	// Normalization: paths: → globs: (Normalization Rule 4)
	if len(rule.Paths) > 0 {
		fmt.Fprintf(&sb, "globs: [%s]\n", strings.Join(rule.Paths, ", "))
		if rule.AlwaysApply != nil && *rule.AlwaysApply {
			sb.WriteString("alwaysApply: true\n")
		}
	} else {
		// No paths = always active → alwaysApply: true, unless explicitly false
		if rule.AlwaysApply != nil && !*rule.AlwaysApply {
			sb.WriteString("alwaysApply: false\n")
		} else {
			sb.WriteString("alwaysApply: true\n")
		}
	}

	sb.WriteString("---\n")

	if body != "" {
		sb.WriteString("\n")
		sb.WriteString(strings.TrimRight(body, "\n"))
		sb.WriteString("\n")
	}

	return sb.String(), nil
}

// compileCursorAgent renders a single AgentConfig to a Cursor agents/<id>.md file.
//
// Normalizations:
//   - background: true → is_background: true (Normalization Rule 6)
//   - CC-only fields are dropped: effort, permissionMode, isolation, color,
//     memory, maxTurns, tools, disallowedTools, skills, initialPrompt
//
//nolint:gocyclo
func compileCursorAgent(id string, agent ast.AgentConfig, baseDir string) (string, error) {
	if strings.TrimSpace(id) == "" {
		return "", fmt.Errorf("agent id must not be empty")
	}

	body, err := resolver.ResolveInstructions(agent.Instructions, agent.InstructionsFile, "", baseDir)
	if err != nil {
		return "", err
	}

	var sb strings.Builder

	sb.WriteString("---\n")

	if agent.Name != "" {
		fmt.Fprintf(&sb, "name: %s\n", yamlScalar(agent.Name))
	}
	if agent.Description != "" {
		fmt.Fprintf(&sb, "description: %s\n", yamlScalar(agent.Description))
	}
	// Determine if fidelity warnings are suppressed
	suppress := false
	if override, ok := agent.Targets["cursor"]; ok && override.SuppressFidelityWarnings != nil && *override.SuppressFidelityWarnings {
		suppress = true
	}

	if agent.Model != "" {
		if resolved, ok := renderer.ResolveModel(agent.Model, "cursor"); ok && resolved != "" {
			// Cursor doesn't natively support full Claude/OpenAI model strings here normally.
			// Only emit the model if it's explicitly safely mapped. Literal fallbacks are omitted.
			if renderer.IsMappedModel(agent.Model, "cursor") {
				fmt.Fprintf(&sb, "model: %s\n", yamlScalar(resolved))
			} else {
				if !suppress {
					fmt.Fprintf(os.Stderr, "WARNING (cursor): unmapped model %q (resolved to %q) omitted for agent %q. Cursor requires specific model strings.\n", agent.Model, resolved, id)
				}
			}
		}
	}
	// Normalization Rule 6: background → is_background
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

	// Emit remaining per-agent security fidelity warnings unless suppressed.
	if !suppress {
		if agent.PermissionMode != "" {
			fmt.Fprintf(os.Stderr, "WARNING (cursor): agent %q permissionMode %q dropped — Cursor has no permission mode equivalent.\n", id, agent.PermissionMode)
		}
		if len(agent.DisallowedTools) > 0 {
			fmt.Fprintf(os.Stderr, "WARNING (cursor): agent %q disallowedTools dropped — tool restrictions will NOT be enforced by Cursor.\n", id)
		}
		if agent.Isolation != "" {
			fmt.Fprintf(os.Stderr, "WARNING (cursor): agent %q isolation %q dropped — Cursor has no process isolation model.\n", id, agent.Isolation)
		}
	}

	return sb.String(), nil
}

// compileCursorSkill renders a single SkillConfig to a Cursor skills/<id>/SKILL.md file.
//
// Normalizations:
//   - disable-model-invocation is supported by Cursor — emitted when true
//   - CC-only fields are dropped: tools, allowed-tools, context, agent, model,
//     effort, shell, argument-hint, user-invocable, hooks, paths
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
	if skill.DisableModelInvocation != nil && *skill.DisableModelInvocation {
		sb.WriteString("disable-model-invocation: true\n")
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
// The type field is intentionally omitted — Cursor infers the transport
// from the presence of command (stdio) or serverUrl (http/sse).
type cursorMCPEntry struct {
	Env       map[string]string `json:"env,omitempty"`
	Headers   map[string]string `json:"headers,omitempty"`
	Command   string            `json:"command,omitempty"`
	ServerURL string            `json:"serverUrl,omitempty"`
	Args      []string          `json:"args,omitempty"`
}

// compileCursorMCP renders all MCP server configs to a single mcp.json file.
//
// Normalizations:
//   - url → serverUrl (Normalization Rule 2)
//   - type field omitted — Cursor infers transport
func compileCursorMCP(servers map[string]ast.MCPConfig) (string, error) {
	entries := make(map[string]cursorMCPEntry, len(servers))
	for id, srv := range servers {
		entries[id] = cursorMCPEntry{
			Command:   srv.Command,
			Args:      srv.Args,
			Env:       srv.Env,
			ServerURL: srv.URL,
			Headers:   srv.Headers,
		}

		if strings.Contains(srv.Command, "${") {
			fmt.Fprintf(os.Stderr, "WARNING (cursor): interpolation pattern ${...} found in MCP command for %q. Cursor requires ${env:NAME} syntax.\n", id)
		}
		for _, arg := range srv.Args {
			if strings.Contains(arg, "${") {
				fmt.Fprintf(os.Stderr, "WARNING (cursor): interpolation pattern ${...} found in MCP args for %q. Cursor requires ${env:NAME} syntax.\n", id)
				break // warn once per args array
			}
		}
		for k, v := range srv.Env {
			if strings.Contains(v, "${") {
				fmt.Fprintf(os.Stderr, "WARNING (cursor): interpolation pattern ${...} found in MCP env %q. Cursor requires ${env:NAME} syntax.\n", k)
			}
		}
	}

	envelope := map[string]map[string]cursorMCPEntry{
		"mcpServers": entries,
	}

	data, err := json.MarshalIndent(envelope, "", "  ")
	if err != nil {
		return "", fmt.Errorf("mcp json marshal: %w", err)
	}
	return string(data), nil
}

// resolveFile returns the effective body content for a rule.
//
// Priority (highest to lowest):
//  1. inline    — the "instructions:" YAML field
//  2. filePath  — the "instructions_file:" YAML field (read from disk, frontmatter stripped)

// stripFrontmatter removes YAML frontmatter delimited by "---" from the start
// of a markdown file, returning only the body content with leading newlines trimmed.

// yamlScalar quotes a string value for safe inclusion in YAML if it contains
// characters that would otherwise need quoting. For simple values it returns
// the string as-is.
func yamlScalar(s string) string {
	needsQuote := strings.ContainsAny(s, ":#{}[]|>&*!,'\"\\%@`")
	if needsQuote {
		return fmt.Sprintf("%q", s)
	}
	return s
}
