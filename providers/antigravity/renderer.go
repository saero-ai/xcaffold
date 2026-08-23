// Package antigravity compiles an XcaffoldConfig AST into Antigravity output files.
// Rules are written as Markdown files under rules/ with YAML frontmatter.
// Skills are written to skills/<id>/SKILL.md with agentskills.io-standard frontmatter.
// Agents are written as Markdown files under agents/<id>.md with YAML frontmatter.
// Hooks are written to hooks.json in the provider output root.
// MCP is written to .agents/mcp_config.json (workspace-level).
//
// Key normalizations:
//   - Agents: native .agents/agents/<id>.md with YAML frontmatter + markdown system prompt body
//   - Rules: 4 activation modes (always, path-glob, model-decided, manual-mention)
//   - Hooks: 5-event system serialized to hooks.json
//   - MCP: workspace-level mcp_config.json with serverUrl and disabledTools
//   - Skills: agentskills.io standard with progressive disclosure fields
//   - Memory: unsupported (emits RENDERER_KIND_UNSUPPORTED fidelity note)
package antigravity

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/output"
	"github.com/saero-ai/xcaffold/internal/renderer"
	"github.com/saero-ai/xcaffold/internal/resolver"
)

const (
	ruleCharLimit = 12000
	targetName    = "antigravity"

	// ProjectContextFile is the output path for project-level instructions.
	// Antigravity reads GEMINI.md from the project root for project context.
	ProjectContextFile = "GEMINI.md"

	// HooksFile is the output path for the hooks configuration.
	HooksFile = "hooks.json"

	// MCPConfigFile is the output path for workspace-level MCP configuration.
	MCPConfigFile = "mcp_config.json"
)

// Renderer compiles an XcaffoldConfig AST into Antigravity output files.
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

// SupportsGlobalScope returns false; Antigravity global scope behavior is not
// documented in official sources.
func (r *Renderer) SupportsGlobalScope() bool {
	return false
}

// Capabilities declares which resource kinds this renderer supports.
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
		SkillArtifactDirs: map[string]string{
			"references": "examples",
			"scripts":    "scripts",
			"assets":     "resources",
			"examples":   "examples",
		},
		RuleActivations: []string{"always", "path-glob", "model-decided", "manual-mention"},
		RuleEncoding: renderer.RuleEncodingCapabilities{
			Description: "frontmatter",
			Activation:  "frontmatter",
		},
	}
}

// CompileAgents renders all agents to flat .agents/agents/<id>.md files with YAML
// frontmatter and Markdown system prompt body.
func (r *Renderer) CompileAgents(agents map[string]ast.AgentConfig, baseDir string) (map[string]string, []renderer.FidelityNote, error) {
	files := make(map[string]string)
	var notes []renderer.FidelityNote
	caps := r.Capabilities()

	for id, agent := range agents {
		md, agentNotes, err := compileAgentMarkdown(id, agent, caps)
		if err != nil {
			return nil, nil, fmt.Errorf("antigravity: agent %q: %w", id, err)
		}
		safePath := filepath.Clean(fmt.Sprintf("agents/%s.md", id))
		files[safePath] = md
		notes = append(notes, agentNotes...)
	}

	return files, notes, nil
}

// CompileSkills renders all skills to skills/<id>/SKILL.md files using the
// agentskills.io standard with progressive disclosure fields.
func (r *Renderer) CompileSkills(skills map[string]ast.SkillConfig, baseDir string) (map[string]string, []renderer.FidelityNote, error) {
	files := make(map[string]string)
	caps := r.Capabilities()

	for id, skill := range skills {
		md, err := compileSkill(id, skill)
		if err != nil {
			return nil, nil, fmt.Errorf("antigravity: skill %q: %w", id, err)
		}
		safePath := filepath.Clean(fmt.Sprintf("skills/%s/SKILL.md", id))
		files[safePath] = md

		out := &output.Output{Files: make(map[string]string)}
		skillSourceDir := filepath.Join("xcaf", "skills", id)
		if err := compileSkillArtifacts(renderer.SkillArtifactContext{
			ID: id, Skill: skill, Caps: caps, BaseDir: baseDir, SkillSourceDir: skillSourceDir,
		}, out); err != nil {
			return nil, nil, fmt.Errorf("antigravity: skill %q: %w", id, err)
		}
		for k, v := range out.Files {
			files[k] = v
		}
	}

	return files, nil, nil
}

// CompileRules renders all rules to rules/<id>.md files with YAML frontmatter
// supporting 4 activation modes.
func (r *Renderer) CompileRules(rules map[string]ast.RuleConfig, baseDir string) (map[string]string, []renderer.FidelityNote, error) {
	files := make(map[string]string)
	var notes []renderer.FidelityNote

	for id, rule := range rules {
		md, ruleNotes, err := compileRule(id, rule, r.Capabilities())
		if err != nil {
			return nil, nil, fmt.Errorf("antigravity: rule %q: %w", id, err)
		}
		safePath := filepath.Clean(fmt.Sprintf("rules/%s.md", id))
		files[safePath] = md
		notes = append(notes, ruleNotes...)
	}

	return files, notes, nil
}

// CompileWorkflows renders all workflows to workflows/<id>.md files.
func (r *Renderer) CompileWorkflows(workflows map[string]ast.WorkflowConfig, baseDir string) (map[string]string, []renderer.FidelityNote, error) {
	files := make(map[string]string)
	var notes []renderer.FidelityNote

	for id, wf := range workflows {
		if strings.TrimSpace(id) == "" {
			return nil, nil, fmt.Errorf("antigravity: workflow id must not be empty")
		}
		wfCopy := wf
		if wfCopy.Name == "" {
			wfCopy.Name = id
		}
		md := compileWorkflow(id, wfCopy)
		safePath := filepath.Clean(fmt.Sprintf("workflows/%s.md", id))
		files[safePath] = md
	}

	caps := r.Capabilities()
	for id, wf := range workflows {
		if len(wf.Artifacts) == 0 {
			continue
		}
		workflowSourceDir := filepath.Join("xcaf", "workflows", id)
		artifactNotes := renderer.CompileArtifactsDemoted(targetName, renderer.ArtifactJob{
			ID: id, BaseDir: baseDir, Caps: caps, Files: files, SourceDir: workflowSourceDir,
		}, wf.Artifacts)
		notes = append(notes, artifactNotes...)
	}
	return files, notes, nil
}

// CompileHooks renders the hooks configuration to hooks.json in the output root.
// Antigravity supports a 5-event system: PreToolUse, PostToolUse, PreInvocation,
// PostInvocation, Stop.
func (r *Renderer) CompileHooks(hooks ast.HookConfig, baseDir string) (map[string]string, []renderer.FidelityNote, error) {
	if len(hooks) == 0 {
		return nil, nil, nil
	}

	data, err := json.MarshalIndent(hooks, "", "  ")
	if err != nil {
		return nil, nil, fmt.Errorf("antigravity: hooks marshal: %w", err)
	}

	return map[string]string{
		HooksFile: string(data) + "\n",
	}, nil, nil
}

// CompileMCP renders workspace-level MCP configuration to mcp_config.json.
func (r *Renderer) CompileMCP(servers map[string]ast.MCPConfig) (map[string]string, []renderer.FidelityNote, error) {
	if len(servers) == 0 {
		return nil, nil, nil
	}

	type mcpServerJSON struct {
		Command       string            `json:"command,omitempty"`
		Args          []string          `json:"args,omitempty"`
		Env           map[string]string `json:"env,omitempty"`
		ServerURL     string            `json:"serverUrl,omitempty"`
		Headers       map[string]string `json:"headers,omitempty"`
		DisabledTools []string          `json:"disabledTools,omitempty"`
	}

	type mcpWrapper struct {
		MCPServers map[string]mcpServerJSON `json:"mcpServers"`
	}

	payload := mcpWrapper{
		MCPServers: make(map[string]mcpServerJSON, len(servers)),
	}

	for name, s := range servers {
		entry := mcpServerJSON{
			Command:       s.Command,
			Args:          s.Args,
			Env:           s.Env,
			Headers:       s.Headers,
			DisabledTools: s.DisabledTools,
		}
		if s.URL != "" {
			entry.ServerURL = s.URL
		}
		payload.MCPServers[name] = entry
	}

	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return nil, nil, fmt.Errorf("antigravity: mcp marshal: %w", err)
	}

	return map[string]string{
		MCPConfigFile: string(data) + "\n",
	}, nil, nil
}

// CompileSettings emits fidelity notes for unsupported settings fields.
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
			Reason:     "settings.permissions dropped; Antigravity has no permission enforcement model in settings",
			Mitigation: "Remove the permissions block for this target",
		})
	}
	return nil, notes, nil
}

// CompileProjectInstructions renders project context into GEMINI.md at root.
func (r *Renderer) CompileProjectInstructions(config *ast.XcaffoldConfig, baseDir string) (map[string]string, map[string]string, []renderer.FidelityNote, error) {
	rootFiles := make(map[string]string)
	bodies := renderer.ResolveContextBodies(config, targetName)
	for path, content := range bodies {
		if path == "" {
			rootFiles[ProjectContextFile] = content
		} else {
			rootFiles[filepath.Join(path, ProjectContextFile)] = content
		}
	}
	return nil, rootFiles, nil, nil
}

// CompileMemory reports memory as unsupported in Antigravity per ground truth (db/memory.json).
func (r *Renderer) CompileMemory(config *ast.XcaffoldConfig, baseDir string, opts renderer.MemoryOptions) (map[string]string, []renderer.FidelityNote, error) {
	if len(config.Memory) == 0 {
		return map[string]string{}, nil, nil
	}
	var notes []renderer.FidelityNote
	for name := range config.Memory {
		notes = append(notes, renderer.FidelityNote{
			Level:      renderer.LevelInfo,
			Target:     targetName,
			Kind:       "memory",
			Resource:   name,
			Code:       renderer.CodeRendererKindUnsupported,
			Reason:     fmt.Sprintf("memory %q dropped; Antigravity has no native persistent memory or MEMORY.md equivalent", name),
			Mitigation: "Antigravity uses runtime session transcripts and Knowledge Items; memory kind is not rendered",
		})
	}
	return map[string]string{}, notes, nil
}

// Finalize is a no-op post-processing pass for the Antigravity renderer.
func (r *Renderer) Finalize(files map[string]string, rootFiles map[string]string) (map[string]string, map[string]string, []renderer.FidelityNote, error) {
	return files, rootFiles, nil, nil
}

// --- helpers ---

// compileSkillArtifacts iterates ctx.Skill.Artifacts and dispatches each artifact
// to the correct output subdirectory using the renderer's SkillArtifactDirs map.
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

// compileAgentMarkdown serializes a single AgentConfig to .agents/agents/<id>.md
// with YAML frontmatter and Markdown body.
func compileAgentMarkdown(id string, agent ast.AgentConfig, caps renderer.CapabilitySet) (string, []renderer.FidelityNote, error) {
	if strings.TrimSpace(id) == "" {
		return "", nil, fmt.Errorf("agent id must not be empty")
	}

	var notes []renderer.FidelityNote

	body := strings.TrimSpace(renderer.StripAllFrontmatter(resolver.StripFrontmatter(agent.Body)))

	name := agent.Name
	if name == "" {
		name = id
	}

	resolvedModel, mn := renderer.SanitizeAgentModel(agent.Model, caps, targetName, id)
	if resolvedModel == "" {
		resolvedModel = "inherit"
	}
	notes = append(notes, mn...)

	sanitizedTools, toolNotes := renderer.SanitizeAgentTools(agent.Tools.Values, caps, targetName, id)
	notes = append(notes, toolNotes...)

	var sb strings.Builder
	sb.WriteString("---\n")
	fmt.Fprintf(&sb, "name: %s\n", renderer.YAMLScalar(name))
	if agent.Description != "" {
		fmt.Fprintf(&sb, "description: %s\n", renderer.YAMLScalar(agent.Description))
	}
	fmt.Fprintf(&sb, "model: %s\n", renderer.YAMLScalar(resolvedModel))

	if len(sanitizedTools) > 0 {
		sb.WriteString("tools:\n")
		for _, t := range sanitizedTools {
			fmt.Fprintf(&sb, "  - %s\n", t)
		}
	}

	mainAgent := true
	subagent := true
	if agent.UserInvocable != nil {
		mainAgent = *agent.UserInvocable
	}
	fmt.Fprintf(&sb, "mainAgent: %t\n", mainAgent)
	fmt.Fprintf(&sb, "subagent: %t\n", subagent)

	commandExecutionPolicy := "sandbox"
	if agent.PermissionMode == "bypass" || agent.PermissionMode == "allow" {
		commandExecutionPolicy = "auto"
	}
	fmt.Fprintf(&sb, "commandExecutionPolicy: %s\n", commandExecutionPolicy)

	if len(agent.Skills.Values) > 0 {
		sb.WriteString("skills:\n")
		for _, s := range agent.Skills.Values {
			fmt.Fprintf(&sb, "  - %s\n", s)
		}
	}

	sb.WriteString("---\n")

	if body != "" {
		sb.WriteString("\n")
		sb.WriteString(body)
		sb.WriteString("\n")
	}

	return sb.String(), notes, nil
}

// compileSkill renders a single SkillConfig to SKILL.md.
func compileSkill(id string, skill ast.SkillConfig) (string, error) {
	if strings.TrimSpace(id) == "" {
		return "", fmt.Errorf("skill id must not be empty")
	}

	body := resolver.StripFrontmatter(skill.Body)

	name := skill.Name
	if name == "" {
		name = id
	}

	var sb strings.Builder
	sb.WriteString("---\n")
	fmt.Fprintf(&sb, "name: %s\n", renderer.YAMLScalar(name))
	if skill.Description != "" {
		fmt.Fprintf(&sb, "description: %s\n", renderer.YAMLScalar(skill.Description))
	}
	if skill.WhenToUse != "" {
		fmt.Fprintf(&sb, "when-to-use: %s\n", renderer.YAMLScalar(skill.WhenToUse))
	}
	if skill.License != "" {
		fmt.Fprintf(&sb, "license: %s\n", renderer.YAMLScalar(skill.License))
	}
	if len(skill.AllowedTools.Values) > 0 {
		sb.WriteString("allowed-tools:\n")
		for _, t := range skill.AllowedTools.Values {
			fmt.Fprintf(&sb, "  - %s\n", t)
		}
	}
	if skill.DisableModelInvocation != nil {
		fmt.Fprintf(&sb, "disable-model-invocation: %t\n", *skill.DisableModelInvocation)
	}
	if skill.UserInvocable != nil {
		fmt.Fprintf(&sb, "user-invocable: %t\n", *skill.UserInvocable)
	}
	if skill.ArgumentHint != "" {
		fmt.Fprintf(&sb, "argument-hint: %s\n", renderer.YAMLScalar(skill.ArgumentHint))
	}
	sb.WriteString("---\n")

	if body != "" {
		sb.WriteString("\n")
		sb.WriteString(strings.TrimRight(renderer.StripAllFrontmatter(body), "\n"))
		sb.WriteString("\n")
	}

	return sb.String(), nil
}

// compileRule renders a single RuleConfig to Markdown with YAML frontmatter.
func compileRule(id string, rule ast.RuleConfig, caps renderer.CapabilitySet) (string, []renderer.FidelityNote, error) {
	if strings.TrimSpace(id) == "" {
		return "", nil, fmt.Errorf("rule id must not be empty")
	}

	body := renderer.StripAllFrontmatter(resolver.StripFrontmatter(rule.Body))
	activation := renderer.ResolvedActivation(rule)

	var sb strings.Builder
	var notes []renderer.FidelityNote

	needsFrontmatter := rule.Description != "" ||
		activation == ast.RuleActivationPathGlob ||
		activation == ast.RuleActivationModelDecided ||
		activation == ast.RuleActivationManualMention

	if needsFrontmatter {
		sb.WriteString("---\n")
		if rule.Description != "" {
			fmt.Fprintf(&sb, "description: %s\n", renderer.YAMLScalar(rule.Description))
		}
		switch activation {
		case ast.RuleActivationModelDecided:
			sb.WriteString("trigger: model_decision\n")
		case ast.RuleActivationPathGlob:
			sb.WriteString("trigger: glob\n")
			if len(rule.Paths.Values) > 0 {
				sb.WriteString("globs:\n")
				for _, p := range rule.Paths.Values {
					fmt.Fprintf(&sb, "  - %s\n", p)
				}
			}
		case ast.RuleActivationManualMention:
			notes = append(notes, renderer.FidelityNote{
				Level:      renderer.LevelInfo,
				Target:     targetName,
				Kind:       "rule",
				Resource:   id,
				Field:      "activation",
				Code:       renderer.CodeRuleActivationUnsupported,
				Reason:     fmt.Sprintf("rule %q activation %q configured via mention in Antigravity", id, activation),
				Mitigation: "Mention rule by name in agent prompts",
			})
		}
		sb.WriteString("---\n\n")
	}

	if len(body) > ruleCharLimit {
		fmt.Fprintf(&sb, "<!-- WARNING: rule body exceeds %d characters recommended for Antigravity rules. -->\n\n", ruleCharLimit)
	}

	if body != "" {
		sb.WriteString(strings.TrimRight(body, "\n"))
		sb.WriteString("\n")
	}

	return sb.String(), notes, nil
}

// compileWorkflow renders a single WorkflowConfig to Markdown.
func compileWorkflow(id string, wf ast.WorkflowConfig) string {
	var sb strings.Builder
	sb.WriteString("---\n")
	desc := wf.Description
	if desc == "" {
		desc = wf.Name
	}
	if desc != "" {
		fmt.Fprintf(&sb, "description: %s\n", renderer.YAMLScalar(desc))
	}
	sb.WriteString("---\n\n")

	for _, step := range wf.Steps {
		fmt.Fprintf(&sb, "## %s\n\n", step.Name)
		if step.Skill != "" {
			fmt.Fprintf(&sb, "Invoke `/%s`.\n\n", step.Skill)
		}
		if step.Instructions != "" {
			sb.WriteString(step.Instructions)
			sb.WriteString("\n\n")
		}
	}

	return sb.String()
}
