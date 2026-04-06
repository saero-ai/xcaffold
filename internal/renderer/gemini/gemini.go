// Package gemini compiles an XcaffoldConfig AST into Gemini (Antigravity) output files.
// Rules are written as plain Markdown files under rules/ — no YAML frontmatter.
// Skills are written to skills/<id>/SKILL.md with minimal frontmatter (name + description only).
//
// Key normalizations applied during compilation:
//   - Rules: NO --- frontmatter; description becomes a # heading; 12K char limit enforced with warning
//   - Rules: paths/globs fields are dropped — AG handles activation via UI
//   - Skills: only name and description emitted in frontmatter; all other fields dropped
//   - Agents, hooks, and MCP are silently skipped — AG has no file format for them
package gemini

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/output"
)

const ruleCharLimit = 12000

// Renderer compiles an XcaffoldConfig AST into Gemini (Antigravity) output files.
type Renderer struct{}

// New returns a new Renderer instance.
func New() *Renderer {
	return &Renderer{}
}

// Target returns the identifier for this renderer's target platform.
func (r *Renderer) Target() string {
	return "gemini"
}

// OutputDir returns the output directory prefix for this renderer.
func (r *Renderer) OutputDir() string {
	return ".agents"
}

// Render wraps a files map in an output.Output. This is an identity
// operation — no additional path rewriting is needed at this layer.
func (r *Renderer) Render(files map[string]string) *output.Output {
	return &output.Output{Files: files}
}

// Compile translates an XcaffoldConfig AST into its Gemini (Antigravity) output representation.
// baseDir is the directory that contains the scaffold.xcf file; it is used to
// resolve instructions_file: paths. Compile returns an error if any resource
// fails to compile. It never panics.
//
// Only rules and skills are compiled. Agents, hooks, and MCP configs are silently
// skipped because Antigravity has no file format for those resource types.
func (r *Renderer) Compile(config *ast.XcaffoldConfig, baseDir string) (*output.Output, error) {
	out := &output.Output{
		Files: make(map[string]string),
	}

	// Compile all rules to rules/<id>.md
	for id, rule := range config.Rules {
		md, err := compileGeminiRule(id, rule, baseDir)
		if err != nil {
			return nil, fmt.Errorf("gemini: failed to compile rule %q: %w", id, err)
		}
		safePath := filepath.Clean(fmt.Sprintf("rules/%s.md", id))
		out.Files[safePath] = md
	}

	// Compile all skills to skills/<id>/SKILL.md
	for id, skill := range config.Skills {
		md, err := compileGeminiSkill(id, skill, baseDir)
		if err != nil {
			return nil, fmt.Errorf("gemini: failed to compile skill %q: %w", id, err)
		}
		safePath := filepath.Clean(fmt.Sprintf("skills/%s/SKILL.md", id))
		out.Files[safePath] = md
	}

	// Compile all workflows to workflows/<id>.md
	for id, wf := range config.Workflows {
		md, err := compileGeminiWorkflow(id, wf, baseDir)
		if err != nil {
			return nil, fmt.Errorf("gemini: failed to compile workflow %q: %w", id, err)
		}
		safePath := filepath.Clean(fmt.Sprintf("workflows/%s.md", id))
		out.Files[safePath] = md
	}

	// Agents, hooks, and MCP are silently skipped structurally,
	// but we surface validation warnings on MCP usage.
	for _, srv := range config.MCP {
		for k, v := range srv.Env {
			if strings.Contains(v, "${") {
				fmt.Fprintf(os.Stderr, "WARNING (gemini): interpolation pattern ${...} found in MCP env %q. Antigravity requires literal strings.\n", k)
			}
		}
	}

	return out, nil
}

// compileGeminiRule renders a single RuleConfig to a plain Markdown file.
//
// Antigravity rules have NO YAML frontmatter. Key normalizations:
//   - Description becomes a # heading at the top of the file (not frontmatter)
//   - paths/globs are dropped — AG handles activation via UI
//   - Bodies exceeding 12,000 characters receive a leading warning HTML comment
func compileGeminiRule(id string, rule ast.RuleConfig, baseDir string) (string, error) {
	if strings.TrimSpace(id) == "" {
		return "", fmt.Errorf("rule id must not be empty")
	}

	body, err := resolveFile(rule.Instructions, rule.InstructionsFile, baseDir)
	if err != nil {
		return "", err
	}

	var sb strings.Builder

	body = stripFrontmatter(body)

	// Prepend 12K warning comment before any other content if body is too long.
	if len(body) > ruleCharLimit {
		fmt.Fprintf(&sb, "<!-- WARNING: rule body exceeds the %d-character limit recommended for Antigravity rules. Consider splitting this rule into smaller, focused rules. -->\n\n", ruleCharLimit)
	}

	// Description becomes a markdown heading — no --- frontmatter delimiters.
	if rule.Description != "" {
		fmt.Fprintf(&sb, "# %s\n\n", rule.Description)
	}

	if body != "" {
		sb.WriteString(strings.TrimRight(body, "\n"))
		sb.WriteString("\n")
	}

	return sb.String(), nil
}

// compileGeminiSkill renders a single SkillConfig to a skills/<id>/SKILL.md file.
//
// Antigravity skills support only name and description in frontmatter.
// All other fields (tools, context, model, effort, disable-model-invocation, etc.)
// are dropped — they have no AG equivalent.
func compileGeminiSkill(id string, skill ast.SkillConfig, baseDir string) (string, error) {
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
	// All other fields dropped — AG skills only support name + description.
	sb.WriteString("---\n")

	if body != "" {
		sb.WriteString("\n")
		// Strip any inner frontmatter the user might have accidentally provided inline
		sb.WriteString(strings.TrimRight(stripFrontmatter(body), "\n"))
		sb.WriteString("\n")
	}

	return sb.String(), nil
}

// compileGeminiWorkflow renders a single WorkflowConfig to a workflows/<id>.md file.
func compileGeminiWorkflow(id string, wf ast.WorkflowConfig, baseDir string) (string, error) {
	if strings.TrimSpace(id) == "" {
		return "", fmt.Errorf("workflow id must not be empty")
	}

	body, err := resolveFile(wf.Instructions, wf.InstructionsFile, baseDir)
	if err != nil {
		return "", err
	}

	var sb strings.Builder

	sb.WriteString("---\n")
	if wf.Description != "" {
		fmt.Fprintf(&sb, "description: %s\n", yamlScalar(wf.Description))
	} else if wf.Name != "" {
		fmt.Fprintf(&sb, "description: %s\n", yamlScalar(wf.Name))
	}
	sb.WriteString("---\n")

	if body != "" {
		sb.WriteString("\n")
		// Strip any inner frontmatter
		sb.WriteString(strings.TrimRight(stripFrontmatter(body), "\n"))
		sb.WriteString("\n")
	}

	return sb.String(), nil
}

// resolveFile returns the effective body content for a rule or skill.
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
		return fmt.Sprintf("%q", s)
	}
	return s
}
