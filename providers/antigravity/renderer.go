// Package antigravity compiles an XcaffoldConfig AST into Antigravity output files.
// Rules are written as Markdown files under rules/ with optional YAML frontmatter.
// Skills are written to skills/<id>/SKILL.md with minimal frontmatter (name + description only).
//
// Key normalizations applied during compilation:
//   - Rules: optional --- frontmatter; description in frontmatter (not a # heading)
//   - Rules: PathGlob activation → trigger: glob + globs: <patterns> in frontmatter
//   - Rules: ModelDecided activation → trigger: model_decision in frontmatter
//   - Rules: AlwaysOn / absent → no trigger field (always-on by default)
//   - Rules: ManualMention / ExplicitInvoke → fidelity note; no frontmatter encoding
//   - Skills: only name and description emitted in frontmatter; all other fields dropped
//   - Agents are rendered as specialist profiles (Markdown notes) at agents/<id>.md
//   - Hooks are not supported by this renderer; the orchestrator emits RENDERER_KIND_UNSUPPORTED
//   - MCP is global-only; no project-local file is written; a MCP_GLOBAL_CONFIG_ONLY note is emitted
package antigravity

import (
	"fmt"
	"path/filepath"
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

	// ProjectContextFile is the output path for project-level instructions.
	// All project instructions and scopes are merged into this single file with
	// xcaffold:scope provenance markers preserving origin metadata.
	// Antigravity reads GEMINI.md from the project root for project context.
	ProjectContextFile = "GEMINI.md"
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
		Agents:               true,
		Skills:               true,
		Rules:                true,
		Workflows:            true,
		Hooks:                false,
		Settings:             true,
		MCP:                  true,
		Memory:               false,
		ProjectInstructions:  true,
		AgentNativeToolsOnly: false,
		SkillArtifactDirs: map[string]string{
			"references": "examples",
			"scripts":    "scripts",
			"assets":     "resources",
			"examples":   "examples",
		},
		RuleActivations: []string{"always", "path-glob", "model-decided"},
		RuleEncoding: renderer.RuleEncodingCapabilities{
			Description: "frontmatter",
			Activation:  "frontmatter",
		},
	}
}

// CompileAgents renders all agents to agents/<id>.md files as specialist
// profiles (notes). This is a downgrade path from the native agent model
// supported by Claude and Cursor.
func (r *Renderer) CompileAgents(agents map[string]ast.AgentConfig, baseDir string) (map[string]string, []renderer.FidelityNote, error) {
	files := make(map[string]string)
	var notes []renderer.FidelityNote

	for id, agent := range agents {
		md, err := compileAntigravityAgent(id, agent, baseDir)
		if err != nil {
			return nil, nil, fmt.Errorf("antigravity: failed to compile agent %q: %w", id, err)
		}
		safePath := filepath.Clean(fmt.Sprintf("agents/%s.md", id))
		files[safePath] = md

		notes = append(notes, renderer.FidelityNote{
			Level:      renderer.LevelInfo,
			Target:     targetName,
			Kind:       "agent",
			Resource:   id,
			Code:       renderer.CodeRendererKindDowngraded,
			Reason:     fmt.Sprintf("agent %q rendered as a specialist note; Antigravity does not support native agent definitions", id),
			Mitigation: "Use a target that supports native agents (claude, cursor) for full agentic behavior",
		})
	}

	return files, notes, nil
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
	caps := r.Capabilities()

	for id, skill := range skills {
		md, err := compileAntigravitySkill(id, skill, baseDir)
		if err != nil {
			return nil, nil, fmt.Errorf("antigravity: failed to compile skill %q: %w", id, err)
		}
		safePath := filepath.Clean(fmt.Sprintf("skills/%s/SKILL.md", id))
		files[safePath] = md

		out := &output.Output{Files: make(map[string]string)}

		skillSourceDir := filepath.Join("xcaf", "skills", id)
		if err := compileSkillArtifacts(renderer.SkillArtifactContext{ID: id, Skill: skill, Caps: caps, BaseDir: baseDir, SkillSourceDir: skillSourceDir}, out); err != nil {
			return nil, nil, fmt.Errorf("antigravity: skill %q: %w", id, err)
		}

		for k, v := range out.Files {
			files[k] = v
		}
	}

	return files, nil, nil
}

// compileSkillArtifacts iterates ctx.Skill.Artifacts and dispatches each artifact
// to the correct output subdirectory using the renderer's SkillArtifactDirs map.
// Files are discovered automatically from the artifact subdirectory on disk.
func compileSkillArtifacts(ctx renderer.SkillArtifactContext, out *output.Output) error {
	for _, artifactName := range ctx.Skill.Artifacts {
		outputSubdir, ok := ctx.Caps.SkillArtifactDirs[artifactName]
		if !ok {
			outputSubdir = artifactName
		}
		paths, err := renderer.DiscoverArtifactFiles(ctx.BaseDir, ctx.SkillSourceDir, artifactName)
		if err != nil {
			return fmt.Errorf("skill %s artifact %s: discover files: %w", ctx.ID, artifactName, err)
		}
		if len(paths) == 0 {
			continue
		}
		if err := renderer.CompileSkillSubdir(renderer.SkillSubdirOpts{
			ID:              ctx.ID,
			CanonicalSubdir: artifactName,
			OutputSubdir:    outputSubdir,
			Paths:           paths,
			BaseDir:         ctx.BaseDir,
			SkillSourceDir:  ctx.SkillSourceDir,
		}, out); err != nil {
			return fmt.Errorf("artifact %s: %w", artifactName, err)
		}
	}
	return nil
}

// CompileRules renders all rules to rules/<id>.md files. Rules use optional YAML
// frontmatter for description and activation. Bodies exceeding 12,000
// characters receive a leading warning HTML comment.
func (r *Renderer) CompileRules(rules map[string]ast.RuleConfig, baseDir string) (map[string]string, []renderer.FidelityNote, error) {
	files := make(map[string]string)
	var notes []renderer.FidelityNote

	for id, rule := range rules {
		md, ruleNotes, err := compileAntigravityRule(id, rule, r.Capabilities(), baseDir)
		if err != nil {
			return nil, nil, fmt.Errorf("antigravity: failed to compile rule %q: %w", id, err)
		}
		safePath := filepath.Clean(fmt.Sprintf("rules/%s.md", id))
		files[safePath] = md
		notes = append(notes, ruleNotes...)
	}

	return files, notes, nil
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
		notes = append(notes, renderer.FidelityNote{
			Level:      renderer.LevelWarning,
			Target:     targetName,
			Kind:       "settings",
			Resource:   "global",
			Field:      "permissions",
			Code:       renderer.CodeSettingsFieldUnsupported,
			Reason:     "settings.permissions dropped; Antigravity has no permission enforcement model",
			Mitigation: "Remove the permissions block for this target or use a platform that enforces permissions",
		})
	}
	if settings.Sandbox != nil {
		notes = append(notes, renderer.FidelityNote{
			Level:      renderer.LevelWarning,
			Target:     targetName,
			Kind:       "settings",
			Resource:   "global",
			Field:      "sandbox",
			Code:       renderer.CodeSettingsFieldUnsupported,
			Reason:     "settings.sandbox dropped; Antigravity has no sandbox model",
			Mitigation: "Remove the sandbox block for this target or use a platform that supports sandboxing",
		})
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

	note := renderer.FidelityNote{
		Level:      renderer.LevelWarning,
		Target:     targetName,
		Kind:       "settings",
		Resource:   "global",
		Field:      "mcp",
		Code:       renderer.CodeMCPGlobalConfigOnly,
		Reason:     fmt.Sprintf("%d MCP server(s) declared but not written; Antigravity reads MCP config from ~/.gemini/antigravity/mcp_config.json (global only, not project-local)", len(servers)),
		Mitigation: "Configure MCP servers via the Antigravity MCP Store UI or edit ~/.gemini/antigravity/mcp_config.json directly",
	}

	return nil, []renderer.FidelityNote{note}, nil
}

// CompileProjectInstructions renders the project-level instructions into
// GEMINI.md (root file), with xcaffold:scope provenance markers for each
// InstructionsScope entry. Antigravity reads GEMINI.md for project context.
func (r *Renderer) CompileProjectInstructions(config *ast.XcaffoldConfig, baseDir string) (map[string]string, map[string]string, []renderer.FidelityNote, error) {
	rootFiles := make(map[string]string)
	notes := r.renderProjectInstructions(config, baseDir, rootFiles)
	return nil, rootFiles, notes, nil
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
func (r *Renderer) Finalize(files map[string]string, rootFiles map[string]string) (map[string]string, map[string]string, []renderer.FidelityNote, error) {
	return files, rootFiles, nil, nil
}

// renderProjectInstructions emits a single flat-singleton context file (GEMINI.md)
// containing the project root instructions followed by each InstructionsScope entry,
// each wrapped in xcaffold:scope HTML provenance markers (path, merge-strategy, origin).
//
// Antigravity has no multi-file nesting model, so all scopes are merged into one
// file. Structural distinction is preserved via provenance markers only. One
// INSTRUCTIONS_FLATTENED info note is emitted per scope.
func (r *Renderer) renderProjectInstructions(config *ast.XcaffoldConfig, baseDir string, files map[string]string) []renderer.FidelityNote {
	rootContent := renderer.ResolveContextBody(config, targetName)
	if rootContent == "" {
		return nil
	}

	files[ProjectContextFile] = rootContent
	return nil
}

// compileAntigravityAgent renders a single AgentConfig to a Markdown file
// as a specialist profile.
func compileAntigravityAgent(id string, agent ast.AgentConfig, baseDir string) (string, error) {
	if strings.TrimSpace(id) == "" {
		return "", fmt.Errorf("agent id must not be empty")
	}

	body := resolver.StripFrontmatter(agent.Body)

	var sb strings.Builder

	sb.WriteString("---\n")
	if agent.Name != "" {
		fmt.Fprintf(&sb, "name: %s\n", renderer.YAMLScalar(agent.Name))
	}
	if agent.Description != "" {
		fmt.Fprintf(&sb, "description: %s\n", renderer.YAMLScalar(agent.Description))
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

// compileAntigravityRule renders a single RuleConfig to a Markdown file with
// optional YAML frontmatter.
//
// Antigravity rules support YAML frontmatter with the following fields:
//   - description: human-readable trigger description shown in the Customizations panel UI
//   - trigger: model_decision | glob (absent = always-on default)
//   - globs: comma-separated glob patterns (only when trigger: glob)
//
// Key normalizations:
//   - description emitted in YAML frontmatter (not as a # heading)
//   - PathGlob activation → trigger: glob + globs: <comma-joined paths>
//   - ModelDecided activation → trigger: model_decision
//   - AlwaysOn / absent → no trigger field (always-on by default)
//   - ManualMention / ExplicitInvoke → no frontmatter encoding; fidelity note returned
//   - frontmatter block only emitted when needed (empty description + AlwaysOn → no --- block)
//   - Bodies exceeding 12,000 characters receive a warning HTML comment (after frontmatter)
func compileAntigravityRule(id string, rule ast.RuleConfig, caps renderer.CapabilitySet, baseDir string) (string, []renderer.FidelityNote, error) {
	if strings.TrimSpace(id) == "" {
		return "", nil, fmt.Errorf("rule id must not be empty")
	}

	body := renderer.StripAllFrontmatter(resolver.StripFrontmatter(rule.Body))
	activation := renderer.ResolvedActivation(rule)

	var sb strings.Builder
	fmNotes := buildAntigravityFrontmatter(&sb, id, rule, caps)

	// Emit 12K warning comment AFTER frontmatter (so frontmatter stays at byte 0).
	if len(body) > ruleCharLimit {
		fmt.Fprintf(&sb, "<!-- WARNING: rule body exceeds the %d-character limit recommended for Antigravity rules. Consider splitting this rule into smaller, focused rules. -->\n\n", ruleCharLimit)
	}

	var notes []renderer.FidelityNote
	notes = append(notes, fmNotes...)

	// ManualMention and ExplicitInvoke have no frontmatter encoding in Antigravity.
	// Emit a fidelity note directing the user to the Customizations panel.
	if activation == ast.RuleActivationManualMention || activation == ast.RuleActivationExplicitInvoke {
		notes = append(notes, renderer.FidelityNote{
			Level:      renderer.LevelWarning,
			Target:     targetName,
			Kind:       "rule",
			Resource:   id,
			Field:      "activation",
			Code:       renderer.CodeRuleActivationUnsupported,
			Reason:     fmt.Sprintf("rule %q activation %q has no native frontmatter encoding for Antigravity; configure via the Customizations panel", id, activation),
			Mitigation: "Set activation via the Customizations panel in Antigravity",
		})
	}

	if body != "" {
		sb.WriteString(strings.TrimRight(body, "\n"))
		sb.WriteString("\n")
	}

	return sb.String(), notes, nil
}

// buildAntigravityFrontmatter writes the YAML frontmatter block into sb when
// the rule requires one, and returns any activation-related fidelity notes.
// Frontmatter is omitted when description is empty and activation is AlwaysOn,
// to avoid emitting noise (---\n---\n). Frontmatter MUST appear at byte 0.
func buildAntigravityFrontmatter(sb *strings.Builder, id string, rule ast.RuleConfig, caps renderer.CapabilitySet) []renderer.FidelityNote {
	activation := renderer.ResolvedActivation(rule)
	needsFrontmatter := rule.Description != "" ||
		activation == ast.RuleActivationPathGlob ||
		activation == ast.RuleActivationModelDecided
	if !needsFrontmatter {
		return nil
	}

	sb.WriteString("---\n")
	sb.WriteString(renderer.BuildRuleDescriptionFrontmatter(rule, caps))

	if !renderer.ValidateRuleActivation(rule, caps) {
		sb.WriteString("---\n\n")
		return []renderer.FidelityNote{{
			Level:    renderer.LevelWarning,
			Target:   targetName,
			Kind:     "rule",
			Resource: id,
			Field:    "activation",
			Code:     renderer.CodeActivationDegraded,
			Reason:   fmt.Sprintf("activation %q lowers to standard rule injection for antigravity", activation),
		}}
	}

	switch activation {
	case ast.RuleActivationModelDecided:
		sb.WriteString("trigger: model_decision\n")
	case ast.RuleActivationPathGlob:
		sb.WriteString("trigger: glob\n")
		if len(rule.Paths.Values) > 0 {
			fmt.Fprintf(sb, "globs: %s\n", strings.Join(rule.Paths.Values, ","))
		}
	}
	sb.WriteString("---\n\n")
	return nil
}

// compileAntigravitySkill renders a single SkillConfig to a skills/<id>/SKILL.md file.
//
// Antigravity skills support only name and description in frontmatter.
func compileAntigravitySkill(id string, skill ast.SkillConfig, baseDir string) (string, error) {
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

	body := resolver.StripFrontmatter(wf.Body)

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
