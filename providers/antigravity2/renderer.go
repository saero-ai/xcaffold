// Package antigravity2 compiles an XcaffoldConfig AST into Antigravity 2.0 output files.
// Rules are written as Markdown files under rules/ with YAML frontmatter.
// Skills are written to skills/<id>/SKILL.md with agentskills.io-standard frontmatter.
// Agents are written as agent.json definitions under agents/<id>/agent.json.
// Hooks are written to hooks.json in the provider output root.
// MCP is written to .agents/mcp_config.json (workspace-level).
//
// Key normalizations:
//   - Rules: 4 activation modes (always, path-glob, model-decided, manual-mention)
//   - Agents: native agent.json definitions (not downgraded specialist notes)
//   - Hooks: 5-event system serialized to hooks.json
//   - MCP: workspace-level mcp_config.json with serverUrl and disabledTools
//   - Skills: agentskills.io standard with progressive disclosure fields
package antigravity2

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
	targetName    = "antigravity2"

	// ProjectContextFile is the output path for project-level instructions.
	ProjectContextFile = "GEMINI.md"

	// HooksFile is the output path for the hooks configuration.
	HooksFile = "hooks.json"

	// MCPConfigFile is the output path for workspace-level MCP configuration.
	MCPConfigFile = "mcp_config.json"
)

// Renderer compiles an XcaffoldConfig AST into Antigravity 2.0 output files.
type Renderer struct{}

// New returns a new Renderer instance.
func New() *Renderer {
	return &Renderer{}
}

// Target returns the identifier for this renderer's target platform.
func (r *Renderer) Target() string {
	return targetName
}

// SupportsGlobalScope returns false; Antigravity 2.0 global-scope compilation
// is unverified against the current runtime.
func (r *Renderer) SupportsGlobalScope() bool {
	return false
}

// OutputDir returns the output directory prefix for this renderer.
func (r *Renderer) OutputDir() string {
	return ".agents"
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
		Memory:               true,
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

// CompileAgents renders all agents to agents/<id>/agent.json files using the
// Antigravity 2.0 native agent definition format.
func (r *Renderer) CompileAgents(agents map[string]ast.AgentConfig, baseDir string) (map[string]string, []renderer.FidelityNote, error) {
	files := make(map[string]string)
	var notes []renderer.FidelityNote

	for id, agent := range agents {
		content, agentNotes, err := compileAgentJSON(id, agent)
		if err != nil {
			return nil, nil, fmt.Errorf("antigravity2: agent %q: %w", id, err)
		}
		safePath := filepath.Clean(fmt.Sprintf("agents/%s/agent.json", id))
		files[safePath] = content
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
			return nil, nil, fmt.Errorf("antigravity2: skill %q: %w", id, err)
		}
		safePath := filepath.Clean(fmt.Sprintf("skills/%s/SKILL.md", id))
		files[safePath] = md

		out := &output.Output{Files: make(map[string]string)}
		skillSourceDir := filepath.Join("xcaf", "skills", id)
		if err := compileSkillArtifacts(renderer.SkillArtifactContext{
			ID: id, Skill: skill, Caps: caps, BaseDir: baseDir, SkillSourceDir: skillSourceDir,
		}, out); err != nil {
			return nil, nil, fmt.Errorf("antigravity2: skill %q: %w", id, err)
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
			return nil, nil, fmt.Errorf("antigravity2: rule %q: %w", id, err)
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
			return nil, nil, fmt.Errorf("antigravity2: workflow id must not be empty")
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
// Antigravity 2.0 supports a 5-event system: PreToolUse, PostToolUse,
// Notification, Stop, SessionStart.
func (r *Renderer) CompileHooks(hooks ast.HookConfig, baseDir string) (map[string]string, []renderer.FidelityNote, error) {
	if len(hooks) == 0 {
		return nil, nil, nil
	}

	data, err := json.MarshalIndent(hooks, "", "  ")
	if err != nil {
		return nil, nil, fmt.Errorf("antigravity2: marshal hooks.json: %w", err)
	}
	return map[string]string{HooksFile: string(data) + "\n"}, nil, nil
}

// CompileSettings emits fidelity notes for settings fields that Antigravity 2.0
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
			Reason:     "settings.permissions dropped; Antigravity 2.0 has no permission enforcement model",
			Mitigation: "Remove the permissions block for this target",
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
			Reason:     "settings.sandbox dropped; Antigravity 2.0 has no sandbox model",
			Mitigation: "Remove the sandbox block for this target",
		})
	}

	return nil, notes, nil
}

// CompileMCP writes workspace-level MCP configuration to .agents/mcp_config.json.
// Antigravity 2.0 supports serverUrl, disabledTools, command, args, and env fields.
func (r *Renderer) CompileMCP(servers map[string]ast.MCPConfig) (map[string]string, []renderer.FidelityNote, error) {
	if len(servers) == 0 {
		return nil, nil, nil
	}

	wrapper := map[string]interface{}{
		"mcpServers": buildMCPServerMap(servers),
	}
	data, err := json.MarshalIndent(wrapper, "", "  ")
	if err != nil {
		return nil, nil, fmt.Errorf("antigravity2: marshal mcp_config.json: %w", err)
	}
	return map[string]string{MCPConfigFile: string(data) + "\n"}, nil, nil
}

// CompileProjectInstructions renders the project-level instructions into
// GEMINI.md (root file).
func (r *Renderer) CompileProjectInstructions(config *ast.XcaffoldConfig, baseDir string) (map[string]string, map[string]string, []renderer.FidelityNote, error) {
	rootFiles := make(map[string]string)
	rootContent := renderer.ResolveContextBody(config, targetName)
	if rootContent != "" {
		rootFiles[ProjectContextFile] = rootContent
	}
	return nil, rootFiles, nil, nil
}

// CompileMemory delegates to MemoryRenderer, emitting Knowledge Item files under
// knowledge/<name>.md for each declared memory entry.
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

// Finalize is a no-op post-processing pass for the Antigravity 2.0 renderer.
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

// agentJSON is the Antigravity 2.0 native agent.json schema.
type agentJSON struct {
	Name          string   `json:"name,omitempty"`
	Description   string   `json:"description,omitempty"`
	Model         string   `json:"model,omitempty"`
	MaxTurns      *int     `json:"maxTurns,omitempty"`
	Tools         []string `json:"tools,omitempty"`
	DisabledTools []string `json:"disabledTools,omitempty"`
	Readonly      *bool    `json:"readonly,omitempty"`
	UserInvocable *bool    `json:"userInvocable,omitempty"`
	InitialPrompt string   `json:"initialPrompt,omitempty"`
	Skills        []string `json:"skills,omitempty"`
	Rules         []string `json:"rules,omitempty"`
	Instructions  string   `json:"instructions,omitempty"`
}

// compileAgentJSON serializes a single AgentConfig to an agent.json payload.
func compileAgentJSON(id string, agent ast.AgentConfig) (string, []renderer.FidelityNote, error) {
	if strings.TrimSpace(id) == "" {
		return "", nil, fmt.Errorf("agent id must not be empty")
	}

	body := strings.TrimSpace(renderer.StripAllFrontmatter(resolver.StripFrontmatter(agent.Body)))

	obj := agentJSON{
		Name:          agent.Name,
		Description:   agent.Description,
		Model:         agent.Model,
		MaxTurns:      agent.MaxTurns,
		Tools:         agent.Tools.Values,
		DisabledTools: agent.DisallowedTools.Values,
		Readonly:      agent.Readonly,
		UserInvocable: agent.UserInvocable,
		InitialPrompt: agent.InitialPrompt,
		Skills:        agent.Skills.Values,
		Rules:         agent.Rules.Values,
		Instructions:  body,
	}

	data, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return "", nil, fmt.Errorf("marshal agent.json: %w", err)
	}
	return string(data) + "\n", nil, nil
}

// compileSkill renders a single SkillConfig to a skills/<id>/SKILL.md file
// using the agentskills.io standard with progressive disclosure fields.
func compileSkill(id string, skill ast.SkillConfig) (string, error) {
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
	if skill.WhenToUse != "" {
		fmt.Fprintf(&sb, "when-to-use: %s\n", renderer.YAMLScalar(skill.WhenToUse))
	}
	if skill.ArgumentHint != "" {
		fmt.Fprintf(&sb, "argument-hint: %s\n", renderer.YAMLScalar(skill.ArgumentHint))
	}
	if len(skill.AllowedTools.Values) > 0 {
		fmt.Fprintf(&sb, "allowed-tools: [%s]\n", strings.Join(skill.AllowedTools.Values, ", "))
	}
	sb.WriteString("---\n")

	if body != "" {
		sb.WriteString("\n")
		sb.WriteString(strings.TrimRight(renderer.StripAllFrontmatter(body), "\n"))
		sb.WriteString("\n")
	}

	return sb.String(), nil
}

// compileRule renders a single RuleConfig to a rules/<id>.md file with
// optional YAML frontmatter. Supports 4 activation modes.
func compileRule(id string, rule ast.RuleConfig, caps renderer.CapabilitySet) (string, []renderer.FidelityNote, error) {
	if strings.TrimSpace(id) == "" {
		return "", nil, fmt.Errorf("rule id must not be empty")
	}

	body := renderer.StripAllFrontmatter(resolver.StripFrontmatter(rule.Body))

	var sb strings.Builder
	fmNotes := buildRuleFrontmatter(&sb, id, rule, renderer.ResolvedActivation(rule), caps)

	if len(body) > ruleCharLimit {
		fmt.Fprintf(&sb, "<!-- WARNING: rule body exceeds %d characters. Consider splitting this rule. -->\n\n", ruleCharLimit)
	}

	var notes []renderer.FidelityNote
	notes = append(notes, fmNotes...)

	if body != "" {
		sb.WriteString(strings.TrimRight(body, "\n"))
		sb.WriteString("\n")
	}

	return sb.String(), notes, nil
}

// buildRuleFrontmatter writes YAML frontmatter into sb and returns fidelity notes.
// Supports always, path-glob, model-decided, and manual-mention activations.
func buildRuleFrontmatter(sb *strings.Builder, id string, rule ast.RuleConfig, activation string, caps renderer.CapabilitySet) []renderer.FidelityNote {
	needsFrontmatter := rule.Description != "" ||
		activation == ast.RuleActivationPathGlob ||
		activation == ast.RuleActivationModelDecided ||
		activation == ast.RuleActivationManualMention
	if !needsFrontmatter {
		return nil
	}

	sb.WriteString("---\n")
	sb.WriteString(renderer.BuildRuleDescriptionFrontmatter(rule, caps))

	switch activation {
	case ast.RuleActivationModelDecided:
		sb.WriteString("trigger: model_decision\n")
	case ast.RuleActivationPathGlob:
		sb.WriteString("trigger: glob\n")
		if len(rule.Paths.Values) > 0 {
			fmt.Fprintf(sb, "globs: %s\n", strings.Join(rule.Paths.Values, ","))
		}
	case ast.RuleActivationManualMention:
		sb.WriteString("trigger: manual_mention\n")
	}
	sb.WriteString("---\n\n")
	return nil
}

// compileWorkflow renders a WorkflowConfig to the workflows/<id>.md format.
func compileWorkflow(id string, wf ast.WorkflowConfig) string {
	var sb strings.Builder

	sb.WriteString("---\n")
	if wf.Description != "" {
		fmt.Fprintf(&sb, "description: %s\n", renderer.YAMLScalar(wf.Description))
	} else if wf.Name != "" {
		fmt.Fprintf(&sb, "description: %s\n", renderer.YAMLScalar(wf.Name))
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

// mcpServerEntry is the Antigravity 2.0 mcp_config.json server entry schema.
// Supports both serverUrl (canonical) and url (v1.0.5+ alias) fields.
type mcpServerEntry struct {
	Command       string            `json:"command,omitempty"`
	Args          []string          `json:"args,omitempty"`
	Env           map[string]string `json:"env,omitempty"`
	ServerURL     string            `json:"serverUrl,omitempty"`
	URL           string            `json:"url,omitempty"`
	DisabledTools []string          `json:"disabledTools,omitempty"`
}

// buildMCPServerMap converts the xcaffold MCPConfig map to the AGY 2.0 schema.
// AGY 2.0 uses serverUrl (from cfg.URL) and disabledTools (from cfg.DisabledTools).
func buildMCPServerMap(servers map[string]ast.MCPConfig) map[string]mcpServerEntry {
	out := make(map[string]mcpServerEntry, len(servers))
	for name, cfg := range servers {
		out[name] = mcpServerEntry{
			Command:       cfg.Command,
			Args:          cfg.Args,
			Env:           cfg.Env,
			ServerURL:     cfg.URL,
			DisabledTools: cfg.DisabledTools,
		}
	}
	return out
}
