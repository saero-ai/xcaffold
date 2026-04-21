// Package antigravity compiles an XcaffoldConfig AST into Antigravity (Antigravity) output files.
// Rules are written as plain Markdown files under rules/ — no YAML frontmatter.
// Skills are written to skills/<id>/SKILL.md with minimal frontmatter (name + description only).
//
// Key normalizations applied during compilation:
//   - Rules: NO --- frontmatter; description becomes a # heading; 12K char limit enforced with warning
//   - Rules: paths/globs fields are dropped — AG handles activation via UI
//   - Skills: only name and description emitted in frontmatter; all other fields dropped
//   - Agents, hooks, and MCP are silently skipped — AG has no file format for them
package antigravity

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/output"
	"github.com/saero-ai/xcaffold/internal/renderer"
	"github.com/saero-ai/xcaffold/internal/resolver"
	"github.com/saero-ai/xcaffold/internal/translator"
)

const (
	ruleCharLimit = 12000
	targetName    = "antigravity"

	// RulesFile is the output path for the project-level flat-singleton rules file.
	// All project instructions and scopes are merged into this single file with
	// xcaffold:scope provenance markers preserving origin metadata.
	RulesFile = "rules/project-instructions.md"
)

// Renderer compiles an XcaffoldConfig AST into Antigravity (Antigravity) output files.
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
	return ".agents"
}

// Render wraps a files map in an output.Output. This is an identity
// operation — no additional path rewriting is needed at this layer.
func (r *Renderer) Render(files map[string]string) *output.Output {
	return &output.Output{Files: files}
}

// Compile translates an XcaffoldConfig AST into its Antigravity (Antigravity) output representation.
// baseDir is the directory that contains the project.xcf file; it is used to
// resolve instructions_file: paths. Compile returns an error if any resource
// fails to compile. It never panics.
//
// Only rules and skills are compiled. Agents, hooks, and MCP configs are silently
// skipped because Antigravity has no file format for those resource types.
//
//nolint:gocyclo
func (r *Renderer) Compile(config *ast.XcaffoldConfig, baseDir string) (*output.Output, []renderer.FidelityNote, error) {
	out := &output.Output{Files: make(map[string]string)}
	var notes []renderer.FidelityNote

	for id, rule := range config.Rules {
		md, err := compileAntigravityRule(id, rule, baseDir)
		if err != nil {
			return nil, nil, fmt.Errorf("antigravity: failed to compile rule %q: %w", id, err)
		}
		safePath := filepath.Clean(fmt.Sprintf("rules/%s.md", id))
		out.Files[safePath] = md
	}

	for id, skill := range config.Skills {
		md, err := compileAntigravitySkill(id, skill, baseDir)
		if err != nil {
			return nil, nil, fmt.Errorf("antigravity: failed to compile skill %q: %w", id, err)
		}
		safePath := filepath.Clean(fmt.Sprintf("skills/%s/SKILL.md", id))
		out.Files[safePath] = md

		if len(skill.Scripts) > 0 {
			notes = append(notes, renderer.NewNote(
				renderer.LevelWarning, targetName, "skill", id, "scripts",
				renderer.CodeSkillScriptsDropped,
				fmt.Sprintf("skill %q scripts dropped; Antigravity does not support skill scripts/ directories", id),
				"Move script logic into the skill instructions or use a target that supports scripts",
			))
		}
		if len(skill.Assets) > 0 {
			notes = append(notes, renderer.NewNote(
				renderer.LevelWarning, targetName, "skill", id, "assets",
				renderer.CodeSkillAssetsDropped,
				fmt.Sprintf("skill %q assets dropped; Antigravity does not support skill assets/ directories", id),
				"Inline asset references into the instructions body",
			))
		}
	}

	for id, wf := range config.Workflows {
		wfCopy := wf
		if wfCopy.Name == "" {
			wfCopy.Name = id
		}
		// When promote-rules-to-workflows is set, use TranslateWorkflow for
		// the native path (emits a fidelity note). Otherwise fall back to the
		// existing step-body compiler which handles single-body workflows too.
		if override, ok := wfCopy.Targets["antigravity"]; ok {
			if v, ok2 := override.Provider["promote-rules-to-workflows"]; ok2 {
				if promote, _ := v.(bool); promote {
					primitives, wfNotes := translator.TranslateWorkflow(&wfCopy, targetName)
					notes = append(notes, wfNotes...)
					for _, p := range primitives {
						content := p.Content
						if content == "" {
							content = p.Body
						}
						if p.Kind == "workflow" {
							safePath := filepath.Clean(fmt.Sprintf("workflows/%s.md", p.ID))
							out.Files[safePath] = content
						}
					}
					continue
				}
			}
		}
		md, err := compileAntigravityWorkflow(id, wfCopy, baseDir)
		if err != nil {
			return nil, nil, fmt.Errorf("antigravity: failed to compile workflow %q: %w", id, err)
		}
		safePath := filepath.Clean(fmt.Sprintf("workflows/%s.md", id))
		out.Files[safePath] = md
	}

	if len(config.MCP) > 0 {
		// Antigravity reads MCP configuration exclusively from the global user-level
		// path (~/.gemini/antigravity/mcp_config.json). A project-local mcp_config.json
		// is silently ignored by the tool, so writing one would produce a non-functional
		// file. Emit a note directing the user to configure MCP via the UI or by editing
		// the global config file directly.
		notes = append(notes, renderer.NewNote(
			renderer.LevelWarning, targetName, "settings", "global", "mcp",
			renderer.CodeMCPGlobalConfigOnly,
			fmt.Sprintf("%d MCP server(s) declared but not written; Antigravity reads MCP config from ~/.gemini/antigravity/mcp_config.json (global only, not project-local)", len(config.MCP)),
			"Configure MCP servers via the Antigravity MCP Store UI or edit ~/.gemini/antigravity/mcp_config.json directly",
		))
	}

	if config.Project != nil {
		instrNotes := r.renderProjectInstructions(config, baseDir, out.Files)
		notes = append(notes, instrNotes...)
	}

	settings := config.Settings["default"]
	if settings.Permissions != nil {
		notes = append(notes, renderer.NewNote(
			renderer.LevelWarning, targetName, "settings", "global", "permissions",
			renderer.CodeSettingsFieldUnsupported,
			"settings.permissions dropped; Antigravity has no permission enforcement model",
			"Remove the permissions block for this target or use a platform that enforces permissions",
		))
	}
	if settings.Sandbox != nil {
		notes = append(notes, renderer.NewNote(
			renderer.LevelWarning, targetName, "settings", "global", "sandbox",
			renderer.CodeSettingsFieldUnsupported,
			"settings.sandbox dropped; Antigravity has no sandbox model",
			"Remove the sandbox block for this target or use a platform that supports sandboxing",
		))
	}

	for id := range config.Agents {
		notes = append(notes, renderer.NewNote(
			renderer.LevelWarning, targetName, "agent", id, "",
			renderer.CodeRendererKindUnsupported,
			fmt.Sprintf("agent %q dropped; Antigravity does not support agent definitions", id),
			"Use a target that supports agents (claude, cursor, gemini, copilot)",
		))
	}

	for id, agent := range config.Agents {
		if agent.PermissionMode != "" {
			notes = append(notes, renderer.NewNote(
				renderer.LevelWarning, targetName, "agent", id, "permissionMode",
				renderer.CodeAgentSecurityFieldsDropped,
				fmt.Sprintf("agent %q permissionMode dropped; Antigravity has no permission mode equivalent", id),
				"Remove permissionMode from the antigravity target override",
			))
		}
		if len(agent.DisallowedTools) > 0 {
			notes = append(notes, renderer.NewNote(
				renderer.LevelWarning, targetName, "agent", id, "disallowedTools",
				renderer.CodeAgentSecurityFieldsDropped,
				fmt.Sprintf("agent %q disallowedTools dropped; tool restrictions will NOT be enforced by Antigravity", id),
				"Enforce tool restrictions via a different target or accept the loss",
			))
		}
		if agent.Isolation != "" {
			notes = append(notes, renderer.NewNote(
				renderer.LevelWarning, targetName, "agent", id, "isolation",
				renderer.CodeAgentSecurityFieldsDropped,
				fmt.Sprintf("agent %q isolation dropped; Antigravity has no process isolation model", id),
				"Remove isolation from the antigravity target override",
			))
		}
	}

	return out, notes, nil
}

// renderProjectInstructions emits a single flat-singleton rules file containing
// the project root instructions followed by each InstructionsScope entry, each
// wrapped in xcaffold:scope HTML provenance markers (path, merge-strategy, origin).
//
// Antigravity has no multi-file nesting model, so all scopes are merged into one
// file. Structural distinction is preserved via provenance markers only. One
// INSTRUCTIONS_FLATTENED info note is emitted per scope.
func (r *Renderer) renderProjectInstructions(config *ast.XcaffoldConfig, baseDir string, files map[string]string) []renderer.FidelityNote {
	p := config.Project
	if p.Instructions == "" && p.InstructionsFile == "" {
		return nil
	}

	var notes []renderer.FidelityNote

	rootContent := agResolveInstructions(p.Instructions, p.InstructionsFile, baseDir)

	// Inline @-imports — Antigravity has no native @-import mechanism.
	for _, imp := range p.InstructionsImports {
		data, err := os.ReadFile(filepath.Join(baseDir, imp))
		if err == nil {
			rootContent += "\n\n" + string(data)
		}
	}

	var sb strings.Builder
	sb.WriteString(rootContent)

	// Sort scopes: depth ascending (fewer path separators first), then alphabetical.
	scopes := make([]ast.InstructionsScope, len(p.InstructionsScopes))
	copy(scopes, p.InstructionsScopes)
	sort.SliceStable(scopes, func(i, j int) bool {
		di := strings.Count(scopes[i].Path, "/")
		dj := strings.Count(scopes[j].Path, "/")
		if di != dj {
			return di < dj
		}
		return scopes[i].Path < scopes[j].Path
	})

	for _, scope := range scopes {
		scopeContent := agResolveScopeContent(scope, targetName, baseDir)

		// Build provenance marker attributes — the A-6 parser uses
		// `(\w[\w-]*)="([^"]*)"`, so any double quote inside an attribute value
		// would terminate the match early. Replace all double quotes with single
		// quotes before embedding — identical treatment across path, source
		// provider, and source filename keeps round-trip re-import consistent.
		safePath := strings.ReplaceAll(scope.Path, `"`, `'`)
		mergeStrategy := scope.MergeStrategy
		if mergeStrategy == "" {
			mergeStrategy = "concat"
		}

		origin := ""
		if scope.SourceProvider != "" || scope.SourceFilename != "" {
			safeProvider := strings.ReplaceAll(scope.SourceProvider, `"`, `'`)
			safeFilename := strings.ReplaceAll(scope.SourceFilename, `"`, `'`)
			origin = fmt.Sprintf(` origin="%s:%s"`, safeProvider, safeFilename)
		}

		fmt.Fprintf(&sb, "\n\n<!-- xcaffold:scope path=\"%s\" merge=\"%s\"%s -->\n",
			safePath, mergeStrategy, origin)
		sb.WriteString(scopeContent)
		sb.WriteString("\n<!-- xcaffold:/scope -->\n")

		notes = append(notes, renderer.NewNote(
			renderer.LevelInfo,
			targetName,
			"instructions",
			scope.Path,
			"merge-strategy",
			renderer.CodeInstructionsFlattened,
			fmt.Sprintf("scope %q flattened into single rules file with provenance marker", scope.Path),
			"Use a target that supports nested instruction files (e.g. claude) if scope isolation is required",
		))
	}

	safePath := filepath.Clean(RulesFile)
	files[safePath] = sb.String()
	return notes
}

// agResolveInstructions returns inline instructions or reads InstructionsFile
// relative to baseDir. Returns empty string on any read error.
func agResolveInstructions(inline, file, baseDir string) string {
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

// agResolveScopeContent returns the effective content for a scope, preferring
// an antigravity-specific variant when one is declared.
func agResolveScopeContent(scope ast.InstructionsScope, provider, baseDir string) string {
	if v, ok := scope.Variants[provider]; ok {
		return agResolveInstructions("", v.InstructionsFile, baseDir)
	}
	return agResolveInstructions(scope.Instructions, scope.InstructionsFile, baseDir)
}

// compileAntigravityRule renders a single RuleConfig to a plain Markdown file.
//
// Antigravity rules have NO YAML frontmatter. Key normalizations:
//   - Description becomes a # heading at the top of the file (not frontmatter)
//   - paths/globs are dropped — AG handles activation via UI
//   - Bodies exceeding 12,000 characters receive a leading warning HTML comment
func compileAntigravityRule(id string, rule ast.RuleConfig, baseDir string) (string, error) {
	if strings.TrimSpace(id) == "" {
		return "", fmt.Errorf("rule id must not be empty")
	}

	body, err := resolver.ResolveInstructions(rule.Instructions, rule.InstructionsFile, "", baseDir)
	if err != nil {
		return "", err
	}

	var sb strings.Builder

	body = resolver.StripFrontmatter(body)

	// Prepend 12K warning comment before any other content if body is too long.
	if len(body) > ruleCharLimit {
		fmt.Fprintf(&sb, "<!-- WARNING: rule body exceeds the %d-character limit recommended for Antigravity rules. Consider splitting this rule into smaller, focused rules. -->\n\n", ruleCharLimit)
	}

	// Description becomes a markdown heading — no --- frontmatter delimiters.
	if rule.Description != "" {
		fmt.Fprintf(&sb, "# %s\n", rule.Description)
	}

	// Emit activation provenance comment(s) immediately after the heading (or at
	// top of file when no description is set). These allow the importer to recover
	// activation semantics on re-import.
	activation := renderer.ResolvedActivation(rule)
	switch activation {
	case ast.RuleActivationAlways:
		sb.WriteString("<!-- xcaffold:activation AlwaysOn -->\n")
	case ast.RuleActivationPathGlob:
		sb.WriteString("<!-- xcaffold:activation Glob -->\n")
		pathsJSON, err := json.Marshal(rule.Paths)
		if err != nil {
			return "", fmt.Errorf("failed to marshal rule paths: %w", err)
		}
		fmt.Fprintf(&sb, "<!-- xcaffold:paths %s -->\n", string(pathsJSON))
	default:
		// ManualMention, ModelDecided, ExplicitInvoke all map to Manual in
		// Antigravity. ModelDecided could map to AG's native "Model Decision"
		// mode, but Manual is the conservative choice until AG's model-decision
		// semantics are verified to match xcaffold's definition.
		sb.WriteString("<!-- xcaffold:activation Manual -->\n")
	}

	if body != "" {
		sb.WriteString("\n")
		sb.WriteString(strings.TrimRight(body, "\n"))
		sb.WriteString("\n")
	}

	return sb.String(), nil
}

// compileAntigravitySkill renders a single SkillConfig to a skills/<id>/SKILL.md file.
//
// Antigravity skills support only name and description in frontmatter.
func compileAntigravitySkill(id string, skill ast.SkillConfig, baseDir string) (string, error) {
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
	// All other fields dropped — AG skills only support name + description.
	sb.WriteString("---\n")

	if body != "" {
		sb.WriteString("\n")
		// Strip any inner frontmatter the user might have accidentally provided inline
		sb.WriteString(strings.TrimRight(resolver.StripFrontmatter(body), "\n"))
		sb.WriteString("\n")
	}

	return sb.String(), nil
}

// compileAntigravityWorkflow renders a single WorkflowConfig to a workflows/<id>.md file.
func compileAntigravityWorkflow(id string, wf ast.WorkflowConfig, baseDir string) (string, error) {
	if strings.TrimSpace(id) == "" {
		return "", fmt.Errorf("workflow id must not be empty")
	}

	body, err := resolver.ResolveInstructions(wf.Instructions, wf.InstructionsFile, "", baseDir)
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
		sb.WriteString(strings.TrimRight(resolver.StripFrontmatter(body), "\n"))
		sb.WriteString("\n")
	}

	return sb.String(), nil
}

// resolveFile returns the effective body content for a rule or skill.
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
