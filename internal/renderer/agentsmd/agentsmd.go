// Package agentsmd compiles an XcaffoldConfig AST into one or more AGENTS.md
// files — a root-level file and optionally directory-scoped files for rules
// with paths: constraints.
package agentsmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/output"
	"github.com/saero-ai/xcaffold/internal/renderer"
	"github.com/saero-ai/xcaffold/internal/resolver"
)

const targetName = "agentsmd"

// Renderer compiles an XcaffoldConfig AST into AGENTS.md output files.
type Renderer struct{}

// New returns a new Renderer instance.
func New() *Renderer { return &Renderer{} }

// Target returns the canonical target name for this renderer.
func (r *Renderer) Target() string { return targetName }

// OutputDir returns the base output directory for this renderer.
// AGENTS.md lives at the repository root, so this returns ".".
func (r *Renderer) OutputDir() string { return "." }

// Render wraps a files map in an output.Output. Identity operation, consistent
// with the Claude renderer.
func (r *Renderer) Render(files map[string]string) *output.Output {
	return &output.Output{Files: files}
}

// Compile translates an XcaffoldConfig AST into AGENTS.md files along with a
// slice of fidelity notes for fields that have no AGENTS.md equivalent.
// baseDir is the directory containing scaffold.xcf; used to resolve
// instructions_file: paths. Compile never panics — all errors are returned.
func (r *Renderer) Compile(config *ast.XcaffoldConfig, baseDir string) (*output.Output, []renderer.FidelityNote, error) {
	out := &output.Output{Files: make(map[string]string)}
	var notes []renderer.FidelityNote

	ruleGroups := groupRulesByDirectory(config.Rules)

	rootContent, err := buildRootFile(config, baseDir, ruleGroups["."])
	if err != nil {
		return nil, nil, err
	}
	out.Files["AGENTS.md"] = rootContent

	for dir, rules := range ruleGroups {
		if dir == "." {
			continue
		}
		dirContent, err := buildDirFile(rules, baseDir)
		if err != nil {
			return nil, nil, err
		}
		out.Files[filepath.Clean(dir+"/AGENTS.md")] = dirContent
	}

	if config.Project != nil {
		instrNotes := r.renderProjectInstructions(config, baseDir, out.Files)
		notes = append(notes, instrNotes...)
	}

	for id, agent := range config.Agents {
		notes = append(notes, collectNotesAgent(id, agent)...)
	}
	for id, skill := range config.Skills {
		notes = append(notes, collectNotesSkill(id, skill)...)
	}
	for id, rule := range config.Rules {
		notes = append(notes, collectNotesRule(id, rule)...)
	}

	return out, notes, nil
}

// renderProjectInstructions emits a root AGENTS.md and one AGENTS.md per scope.
// agentsmd uses the closest-wins nesting class: each subdirectory's AGENTS.md is
// authoritative for that directory; parent files do not cascade automatically.
//
// Deviation handling:
//   - concat-tagged scopes are pre-flattened: child = root + scope content.
//     An INSTRUCTIONS_CLOSEST_WINS_FORCED_CONCAT warning is emitted per scope.
//   - InstructionsImports are inlined because AGENTS.md has no native @-import support.
//     A single INSTRUCTIONS_IMPORT_INLINED info note is emitted when any imports exist.
func (r *Renderer) renderProjectInstructions(config *ast.XcaffoldConfig, baseDir string, files map[string]string) []renderer.FidelityNote {
	p := config.Project
	if p.Instructions == "" && p.InstructionsFile == "" {
		return nil
	}

	var notes []renderer.FidelityNote

	rootContent := agentsmdResolveInstructions(p.Instructions, p.InstructionsFile, baseDir)

	// Inline @-imports — AGENTS.md has no native @-import mechanism.
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
			"@-imports inlined; AGENTS.md lacks native @-import support",
			"Remove InstructionsImports or use a target that supports @-imports (e.g. claude)",
		))
	}

	// Merge project-instructions root content with any existing rule-aggregated content
	// from buildRootFile. Overwriting would silently discard rules and agents sections.
	existing := files[filepath.Clean("AGENTS.md")]
	if existing != "" {
		files[filepath.Clean("AGENTS.md")] = existing + "\n\n" + rootContent
	} else {
		files[filepath.Clean("AGENTS.md")] = rootContent
	}

	for _, scope := range p.InstructionsScopes {
		scopeContent := agentsmdResolveScopeContent(scope, targetName, baseDir)
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
				"Use merge-strategy: closest-wins or flat for agentsmd targets",
			))
		} else {
			// closest-wins or flat: child AGENTS.md = scope content only.
			files[safePath] = scopeContent
		}
	}

	return notes
}

// agentsmdResolveInstructions returns inline instructions or reads InstructionsFile
// relative to baseDir. Returns empty string on any read error.
func agentsmdResolveInstructions(inline, file, baseDir string) string {
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

// agentsmdResolveScopeContent returns the effective content for a scope, preferring
// an agentsmd-specific variant when one is declared.
func agentsmdResolveScopeContent(scope ast.InstructionsScope, provider, baseDir string) string {
	if v, ok := scope.Variants[provider]; ok {
		return agentsmdResolveInstructions("", v.InstructionsFile, baseDir)
	}
	return agentsmdResolveInstructions(scope.Instructions, scope.InstructionsFile, baseDir)
}

// buildRootFile constructs the root AGENTS.md content.
func buildRootFile(config *ast.XcaffoldConfig, baseDir string, globalRules []namedRule) (string, error) {
	var sb strings.Builder
	sb.WriteString("<!-- Generated by xcaffold. Do not edit manually. -->\n")

	var proj ast.ProjectConfig
	if config.Project != nil {
		proj = *config.Project
	}
	if err := renderProjectSection(&sb, proj); err != nil {
		return "", err
	}
	if err := renderAgentsSection(&sb, config.Agents, baseDir); err != nil {
		return "", err
	}
	if err := renderSkillsSection(&sb, config.Skills, baseDir); err != nil {
		return "", err
	}
	if err := renderRulesSection(&sb, globalRules, baseDir); err != nil {
		return "", err
	}
	if err := renderWorkflowsSection(&sb, config.Workflows, baseDir); err != nil {
		return "", err
	}

	return sb.String(), nil
}

// buildDirFile constructs a directory-scoped AGENTS.md containing only rules.
func buildDirFile(rules []namedRule, baseDir string) (string, error) {
	var sb strings.Builder
	sb.WriteString("<!-- Generated by xcaffold. Do not edit manually. -->\n")

	if err := renderRulesSection(&sb, rules, baseDir); err != nil {
		return "", err
	}

	return sb.String(), nil
}

func renderProjectSection(sb *strings.Builder, p ast.ProjectConfig) error {
	if p.Name == "" && p.Description == "" {
		return nil
	}
	sb.WriteString("\n## Project\n\n")
	sb.WriteString(p.Name)
	if p.Description != "" {
		sb.WriteString(" — ")
		sb.WriteString(p.Description)
	}
	sb.WriteString("\n")
	return nil
}

func renderAgentsSection(sb *strings.Builder, agents map[string]ast.AgentConfig, baseDir string) error {
	if len(agents) == 0 {
		return nil
	}
	sb.WriteString("\n## Agents\n")
	for id, agent := range agents {
		fmt.Fprintf(sb, "\n### %s\n", id)
		if agent.Model != "" {
			fmt.Fprintf(sb, "**Model**: %s\n", agent.Model)
		}
		if agent.Description != "" {
			fmt.Fprintf(sb, "**Description**: %s\n", agent.Description)
		}

		body, err := resolver.ResolveInstructions(agent.Instructions, agent.InstructionsFile, "", baseDir)
		if err != nil {
			return fmt.Errorf("failed to resolve instructions for agent %q: %w", id, err)
		}
		if body != "" {
			sb.WriteString("\n")
			sb.WriteString(strings.TrimRight(body, "\n"))
			sb.WriteString("\n")
		}
	}
	return nil
}

func renderSkillsSection(sb *strings.Builder, skills map[string]ast.SkillConfig, baseDir string) error {
	if len(skills) == 0 {
		return nil
	}
	sb.WriteString("\n## Skills\n")
	for id, skill := range skills {
		fmt.Fprintf(sb, "\n### %s\n", id)
		if skill.Description != "" {
			fmt.Fprintf(sb, "**Description**: %s\n", skill.Description)
		}

		body, err := resolver.ResolveInstructions(skill.Instructions, skill.InstructionsFile, "", baseDir)
		if err != nil {
			return fmt.Errorf("failed to resolve instructions for skill %q: %w", id, err)
		}
		if body != "" {
			sb.WriteString("\n")
			sb.WriteString(strings.TrimRight(body, "\n"))
			sb.WriteString("\n")
		}
	}
	return nil
}

func renderRulesSection(sb *strings.Builder, rules []namedRule, baseDir string) error {
	if len(rules) == 0 {
		return nil
	}
	sb.WriteString("\n## Rules\n")
	for _, nr := range rules {
		fmt.Fprintf(sb, "\n### %s\n", nr.id)
		if len(nr.rule.Paths) > 0 {
			fmt.Fprintf(sb, "**Applies to**: %s\n", strings.Join(nr.rule.Paths, ", "))
		} else {
			sb.WriteString("**Applies to**: all files\n")
		}

		body, err := resolver.ResolveInstructions(nr.rule.Instructions, nr.rule.InstructionsFile, "", baseDir)
		if err != nil {
			return fmt.Errorf("failed to resolve instructions for rule %q: %w", nr.id, err)
		}
		if body != "" {
			sb.WriteString("\n")
			sb.WriteString(strings.TrimRight(body, "\n"))
			sb.WriteString("\n")
		}
	}
	return nil
}

func renderWorkflowsSection(sb *strings.Builder, workflows map[string]ast.WorkflowConfig, baseDir string) error {
	if len(workflows) == 0 {
		return nil
	}
	sb.WriteString("\n## Workflows\n")
	for id, wf := range workflows {
		fmt.Fprintf(sb, "\n### %s\n", id)
		if wf.Description != "" {
			fmt.Fprintf(sb, "**Description**: %s\n", wf.Description)
		}

		body, err := resolver.ResolveInstructions(wf.Instructions, wf.InstructionsFile, "", baseDir)
		if err != nil {
			return fmt.Errorf("failed to resolve instructions for workflow %q: %w", id, err)
		}
		if body != "" {
			sb.WriteString("\n")
			sb.WriteString(strings.TrimRight(body, "\n"))
			sb.WriteString("\n")
		}
	}
	return nil
}

type namedRule struct {
	id   string
	rule ast.RuleConfig
}

// groupRulesByDirectory splits rules into buckets keyed by their common directory
// prefix. Key "." means the root AGENTS.md.
func groupRulesByDirectory(rules map[string]ast.RuleConfig) map[string][]namedRule {
	groups := map[string][]namedRule{".": {}}

	for id, rule := range rules {
		if len(rule.Paths) == 0 {
			groups["."] = append(groups["."], namedRule{id: id, rule: rule})
			continue
		}

		prefix := commonDirPrefix(rule.Paths)
		if prefix == "" || prefix == "." {
			groups["."] = append(groups["."], namedRule{id: id, rule: rule})
		} else {
			groups[prefix] = append(groups[prefix], namedRule{id: id, rule: rule})
		}
	}

	return groups
}

func commonDirPrefix(patterns []string) string {
	if len(patterns) == 0 {
		return ""
	}

	dirParts := make([][]string, 0, len(patterns))
	for _, p := range patterns {
		parts := strings.Split(p, "/")
		dirs := parts[:len(parts)-1]
		cleaned := make([]string, 0, len(dirs))
		for _, d := range dirs {
			if strings.ContainsAny(d, "*?[") {
				break
			}
			cleaned = append(cleaned, d)
		}
		dirParts = append(dirParts, cleaned)
	}

	if len(dirParts) == 0 || len(dirParts[0]) == 0 {
		return ""
	}

	prefix := dirParts[0]
	for _, parts := range dirParts[1:] {
		prefix = commonPrefix(prefix, parts)
	}

	if len(prefix) == 0 {
		return ""
	}

	return strings.Join(prefix, "/")
}

func commonPrefix(a, b []string) []string {
	max := len(a)
	if len(b) < max {
		max = len(b)
	}
	for i := 0; i < max; i++ {
		if a[i] != b[i] {
			return a[:i]
		}
	}
	return a[:max]
}

func fieldNote(kind, id, field string) renderer.FidelityNote {
	return renderer.NewNote(
		renderer.LevelWarning,
		targetName,
		kind,
		id,
		field,
		renderer.CodeFieldUnsupported,
		fmt.Sprintf("field %q on %s %q has no AGENTS.md equivalent and was dropped", field, kind, id),
		"",
	)
}

//nolint:gocyclo
func collectNotesAgent(id string, a ast.AgentConfig) []renderer.FidelityNote {
	var notes []renderer.FidelityNote
	if len(a.Tools) > 0 {
		notes = append(notes, fieldNote("agent", id, "tools"))
	}
	if len(a.DisallowedTools) > 0 {
		notes = append(notes, fieldNote("agent", id, "disallowedTools"))
	}
	if len(a.Skills) > 0 {
		notes = append(notes, fieldNote("agent", id, "skills"))
	}
	if len(a.Rules) > 0 {
		notes = append(notes, fieldNote("agent", id, "rules"))
	}
	if a.Effort != "" {
		notes = append(notes, fieldNote("agent", id, "effort"))
	}
	if a.PermissionMode != "" {
		notes = append(notes, fieldNote("agent", id, "permissionMode"))
	}
	if a.Isolation != "" {
		notes = append(notes, fieldNote("agent", id, "isolation"))
	}
	if a.Color != "" {
		notes = append(notes, fieldNote("agent", id, "color"))
	}
	if a.MaxTurns > 0 {
		notes = append(notes, fieldNote("agent", id, "maxTurns"))
	}
	if a.Background != nil {
		notes = append(notes, fieldNote("agent", id, "background"))
	}
	if a.Readonly != nil {
		notes = append(notes, fieldNote("agent", id, "readonly"))
	}
	if a.Mode != "" {
		notes = append(notes, fieldNote("agent", id, "mode"))
	}
	if a.When != "" {
		notes = append(notes, fieldNote("agent", id, "when"))
	}
	if a.InitialPrompt != "" {
		notes = append(notes, fieldNote("agent", id, "initialPrompt"))
	}
	if a.Memory != "" {
		notes = append(notes, fieldNote("agent", id, "memory"))
	}
	if len(a.Hooks) > 0 {
		notes = append(notes, fieldNote("agent", id, "hooks"))
	}
	if len(a.MCPServers) > 0 {
		notes = append(notes, fieldNote("agent", id, "mcpServers"))
	}
	if len(a.Targets) > 0 {
		notes = append(notes, fieldNote("agent", id, "targets"))
	}
	if len(a.Assertions) > 0 {
		notes = append(notes, fieldNote("agent", id, "assertions"))
	}
	if len(a.MCP) > 0 {
		notes = append(notes, fieldNote("agent", id, "mcp"))
	}
	return notes
}

func collectNotesSkill(id string, s ast.SkillConfig) []renderer.FidelityNote {
	var notes []renderer.FidelityNote
	if len(s.AllowedTools) > 0 {
		notes = append(notes, fieldNote("skill", id, "tools"))
	}
	if len(s.References) > 0 {
		notes = append(notes, fieldNote("skill", id, "references"))
	}
	if len(s.Scripts) > 0 {
		notes = append(notes, fieldNote("skill", id, "scripts"))
	}
	if len(s.Assets) > 0 {
		notes = append(notes, fieldNote("skill", id, "assets"))
	}
	return notes
}

func collectNotesRule(id string, rule ast.RuleConfig) []renderer.FidelityNote {
	var notes []renderer.FidelityNote
	if rule.AlwaysApply != nil {
		notes = append(notes, fieldNote("rule", id, "alwaysApply"))
	}
	return notes
}
