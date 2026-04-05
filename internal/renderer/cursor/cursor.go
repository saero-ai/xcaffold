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

	return out, nil
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

	body, err := resolveFile(rule.Instructions, rule.InstructionsFile, baseDir)
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
	} else {
		// No paths = always active → alwaysApply: true
		sb.WriteString("alwaysApply: true\n")
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
func compileCursorAgent(id string, agent ast.AgentConfig, baseDir string) (string, error) {
	if strings.TrimSpace(id) == "" {
		return "", fmt.Errorf("agent id must not be empty")
	}

	body, err := resolveFile(agent.Instructions, agent.InstructionsFile, baseDir)
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
	if agent.Model != "" {
		fmt.Fprintf(&sb, "model: %s\n", yamlScalar(agent.Model))
	}
	// Normalization Rule 6: background → is_background
	if agent.Background != nil && *agent.Background {
		sb.WriteString("is_background: true\n")
	}

	sb.WriteString("---\n")

	if body != "" {
		sb.WriteString("\n")
		sb.WriteString(strings.TrimRight(body, "\n"))
		sb.WriteString("\n")
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

	body, err := resolveFile(skill.Instructions, skill.InstructionsFile, baseDir)
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
	Command   string            `json:"command,omitempty"`
	Args      []string          `json:"args,omitempty"`
	Env       map[string]string `json:"env,omitempty"`
	ServerURL string            `json:"serverUrl,omitempty"` // Normalization: url → serverUrl
	Headers   map[string]string `json:"headers,omitempty"`
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
func resolveFile(inline, filePath, baseDir string) (string, error) {
	if inline != "" {
		return inline, nil
	}
	if filePath != "" {
		cleaned := filepath.Clean(filePath)
		if strings.HasPrefix(cleaned, "..") {
			return "", fmt.Errorf("instructions_file must be a relative path inside the project: %q traverses above the project root", filePath)
		}
		abs := filepath.Join(baseDir, cleaned)
		data, err := os.ReadFile(abs)
		if err != nil {
			return "", fmt.Errorf("instructions_file %q: %w", filePath, err)
		}
		return stripFrontmatter(string(data)), nil
	}
	return "", nil
}

// stripFrontmatter removes YAML frontmatter delimited by "---" from the start
// of a markdown file, returning only the body content with leading newlines trimmed.
func stripFrontmatter(content string) string {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	lines := strings.SplitN(content, "\n", -1)

	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return strings.TrimLeft(content, "\n")
	}

	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			body := strings.Join(lines[i+1:], "\n")
			return strings.TrimLeft(body, "\n")
		}
	}

	return strings.TrimLeft(content, "\n")
}

// yamlScalar quotes a string value for safe inclusion in YAML if it contains
// characters that would otherwise need quoting. For simple values it returns
// the string as-is.
func yamlScalar(s string) string {
	needsQuote := strings.ContainsAny(s, ":#{}[]|>&*!,'\"\\%@`")
	if needsQuote {
		escaped := strings.ReplaceAll(s, `"`, `\"`)
		return fmt.Sprintf("%q", escaped)
	}
	return s
}
