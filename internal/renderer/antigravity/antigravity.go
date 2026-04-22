// Package antigravity compiles an XcaffoldConfig AST into Antigravity output files.
// Rules are written as plain Markdown files under rules/ — no YAML frontmatter.
// Skills are written to skills/<id>/SKILL.md with minimal frontmatter (name + description only).
//
// Key normalizations applied during compilation:
//   - Rules: NO --- frontmatter; description becomes a # heading; 12K char limit enforced with warning
//   - Rules: paths/globs fields are dropped — AG handles activation via UI
//   - Skills: only name and description emitted in frontmatter; all other fields dropped
//   - Agents are not supported; RENDERER_KIND_UNSUPPORTED notes plus per-field security notes are emitted
//   - Hooks are not supported by this renderer; the orchestrator emits RENDERER_KIND_UNSUPPORTED
//   - MCP is global-only; no project-local file is written; a MCP_GLOBAL_CONFIG_ONLY note is emitted
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

// Capabilities declares which resource kinds this renderer supports.
// Agents and Hooks are handled via per-resource methods but produce no output
// files — they emit fidelity notes only. MCP and Settings are similarly supported
// via their per-resource methods (notes only, no project-local files).
func (r *Renderer) Capabilities() renderer.CapabilitySet {
	return renderer.CapabilitySet{
		Agents:              true,
		Skills:              true,
		Rules:               true,
		Workflows:           true,
		Hooks:               false,
		Settings:            true,
		MCP:                 true,
		Memory:              true,
		ProjectInstructions: true,
		SkillSubdirs:        []string{"references", "scripts", "assets", "examples"},
		RuleActivations:     []string{"always", "path-glob", "manual"},
		SecurityFields: renderer.SecurityFieldSupport{
			Effort: true,
		},
	}
}

// CompileAgents returns no output files — Antigravity does not support agent
// definitions. It emits a RENDERER_KIND_UNSUPPORTED fidelity note for each
// agent, and additional per-field notes for any security fields (permissionMode,
// disallowedTools, isolation) that would be silently lost.
func (r *Renderer) CompileAgents(agents map[string]ast.AgentConfig, baseDir string) (map[string]string, []renderer.FidelityNote, error) {
	var notes []renderer.FidelityNote

	for _, id := range renderer.SortedKeys(agents) {
		agent := agents[id]
		notes = append(notes, renderer.NewNote(
			renderer.LevelWarning, targetName, "agent", id, "",
			renderer.CodeRendererKindUnsupported,
			fmt.Sprintf("agent %q dropped; Antigravity does not support agent definitions", id),
			"Use a target that supports agents (claude, cursor, gemini, copilot)",
		))
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

	return nil, notes, nil
}

// CompileSkills renders all skills to skills/<id>/SKILL.md files with minimal
// frontmatter (name and description only). Subdirectories are compiled with
// Antigravity-specific name translation:
//   - references/ → examples/
//   - scripts/    → scripts/
//   - assets/     → resources/
//   - examples/   → examples/
func (r *Renderer) CompileSkills(skills map[string]ast.SkillConfig, baseDir string) (map[string]string, []renderer.FidelityNote, error) {
	files := make(map[string]string)

	for id, skill := range skills {
		md, err := compileAntigravitySkill(id, skill, baseDir)
		if err != nil {
			return nil, nil, fmt.Errorf("antigravity: failed to compile skill %q: %w", id, err)
		}
		safePath := filepath.Clean(fmt.Sprintf("skills/%s/SKILL.md", id))
		files[safePath] = md

		out := &output.Output{Files: make(map[string]string)}

		if err := renderer.CompileSkillSubdir(id, "references", "examples", skill.References, baseDir, out); err != nil {
			return nil, nil, fmt.Errorf("antigravity: references for skill %q: %w", id, err)
		}
		if err := renderer.CompileSkillSubdir(id, "scripts", "scripts", skill.Scripts, baseDir, out); err != nil {
			return nil, nil, fmt.Errorf("antigravity: scripts for skill %q: %w", id, err)
		}
		if err := renderer.CompileSkillSubdir(id, "assets", "resources", skill.Assets, baseDir, out); err != nil {
			return nil, nil, fmt.Errorf("antigravity: assets for skill %q: %w", id, err)
		}
		if err := renderer.CompileSkillSubdir(id, "examples", "examples", skill.Examples, baseDir, out); err != nil {
			return nil, nil, fmt.Errorf("antigravity: examples for skill %q: %w", id, err)
		}

		for k, v := range out.Files {
			files[k] = v
		}
	}

	return files, nil, nil
}

// CompileRules renders all rules to rules/<id>.md files. Rules use no YAML
// frontmatter; description becomes a # heading. Bodies exceeding 12,000
// characters receive a leading warning HTML comment.
func (r *Renderer) CompileRules(rules map[string]ast.RuleConfig, baseDir string) (map[string]string, []renderer.FidelityNote, error) {
	files := make(map[string]string)

	for id, rule := range rules {
		md, err := compileAntigravityRule(id, rule, baseDir)
		if err != nil {
			return nil, nil, fmt.Errorf("antigravity: failed to compile rule %q: %w", id, err)
		}
		safePath := filepath.Clean(fmt.Sprintf("rules/%s.md", id))
		files[safePath] = md
	}

	return files, nil, nil
}

// CompileWorkflows renders all workflows to workflows/<id>.md files. When
// promote-rules-to-workflows is set on the antigravity target override,
// TranslateWorkflow is used for the native lowering path.
func (r *Renderer) CompileWorkflows(workflows map[string]ast.WorkflowConfig, baseDir string) (map[string]string, []renderer.FidelityNote, error) {
	files := make(map[string]string)
	var notes []renderer.FidelityNote

	for id, wf := range workflows {
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
							files[safePath] = content
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
		files[safePath] = md
	}

	return files, notes, nil
}

// CompileHooks is a stub — Antigravity does not support hooks. This method
// exists to satisfy the ResourceRenderer interface; the orchestrator will not
// call it because Capabilities().Hooks is false.
func (r *Renderer) CompileHooks(_ ast.HookConfig, _ string) (map[string]string, []renderer.FidelityNote, error) {
	return nil, nil, nil
}

// CompileSettings emits fidelity notes for settings fields that Antigravity
// does not support (permissions, sandbox). No output files are produced.
func (r *Renderer) CompileSettings(settings ast.SettingsConfig) (map[string]string, []renderer.FidelityNote, error) {
	var notes []renderer.FidelityNote

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

	return nil, notes, nil
}

// CompileMCP emits a MCP_GLOBAL_CONFIG_ONLY fidelity note and produces no
// project-local files. Antigravity reads MCP configuration exclusively from
// ~/.gemini/antigravity/mcp_config.json; a project-local file is silently
// ignored by the tool.
func (r *Renderer) CompileMCP(servers map[string]ast.MCPConfig) (map[string]string, []renderer.FidelityNote, error) {
	if len(servers) == 0 {
		return nil, nil, nil
	}

	note := renderer.NewNote(
		renderer.LevelWarning, targetName, "settings", "global", "mcp",
		renderer.CodeMCPGlobalConfigOnly,
		fmt.Sprintf("%d MCP server(s) declared but not written; Antigravity reads MCP config from ~/.gemini/antigravity/mcp_config.json (global only, not project-local)", len(servers)),
		"Configure MCP servers via the Antigravity MCP Store UI or edit ~/.gemini/antigravity/mcp_config.json directly",
	)

	return nil, []renderer.FidelityNote{note}, nil
}

// CompileProjectInstructions renders the project-level instructions into a
// flat singleton rules file (rules/project-instructions.md) with xcaffold:scope
// provenance markers for each InstructionsScope entry.
func (r *Renderer) CompileProjectInstructions(project *ast.ProjectConfig, baseDir string) (map[string]string, []renderer.FidelityNote, error) {
	cfg := &ast.XcaffoldConfig{}
	cfg.Project = project
	files := make(map[string]string)
	notes := r.renderProjectInstructions(cfg, baseDir, files)
	return files, notes, nil
}

// CompileMemory delegates to MemoryRenderer, emitting Antigravity Knowledge
// Item files under knowledge/<name>.md for each declared memory entry.
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

// Finalize is a no-op post-processing pass for the Antigravity renderer.
// Path normalization is already applied per-resource during compilation.
func (r *Renderer) Finalize(files map[string]string) (map[string]string, []renderer.FidelityNote, error) {
	return files, nil, nil
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

	body = renderer.StripAllFrontmatter(body)

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
		fmt.Fprintf(&sb, "name: %s\n", renderer.YAMLScalar(skill.Name))
	}
	if skill.Description != "" {
		fmt.Fprintf(&sb, "description: %s\n", renderer.YAMLScalar(skill.Description))
	}
	// All other fields dropped — AG skills only support name + description.
	sb.WriteString("---\n")

	if body != "" {
		sb.WriteString("\n")
		// Strip any inner frontmatter the user might have accidentally provided inline
		sb.WriteString(strings.TrimRight(renderer.StripAllFrontmatter(body), "\n"))
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
		fmt.Fprintf(&sb, "description: %s\n", renderer.YAMLScalar(wf.Description))
	} else if wf.Name != "" {
		fmt.Fprintf(&sb, "description: %s\n", renderer.YAMLScalar(wf.Name))
	}
	sb.WriteString("---\n")

	if body != "" {
		sb.WriteString("\n")
		// Strip any inner frontmatter
		sb.WriteString(strings.TrimRight(renderer.StripAllFrontmatter(body), "\n"))
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
