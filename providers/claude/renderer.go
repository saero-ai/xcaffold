package claude

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/output"
	"github.com/saero-ai/xcaffold/internal/renderer"
	"github.com/saero-ai/xcaffold/internal/resolver"
	"github.com/saero-ai/xcaffold/internal/translator"
)

// Renderer compiles an XcaffoldConfig AST into Claude Code output files.
// It targets the ".claude/" directory structure understood by Claude Code.
type Renderer struct {
	memoryAgents map[string]bool
}

// New returns a new Renderer instance.
func New() *Renderer {
	return &Renderer{}
}

// SetMemoryRefs satisfies renderer.MemoryAwareRenderer. The orchestrator calls
// this before CompileAgents so that agent frontmatter can carry memory: user
// when the agent has associated memory entries.
func (r *Renderer) SetMemoryRefs(agentRefs map[string]bool) {
	r.memoryAgents = agentRefs
}

// Target returns the identifier for this renderer's target platform.
func (r *Renderer) Target() string {
	return "claude"
}

// OutputDir returns the base output directory for this renderer.
func (r *Renderer) OutputDir() string {
	return ".claude"
}

// Capabilities declares the full set of resource kinds this renderer supports.
func (r *Renderer) Capabilities() renderer.CapabilitySet {
	return renderer.CapabilitySet{
		Agents:              true,
		Skills:              true,
		Rules:               true,
		Workflows:           true,
		Hooks:               true,
		Settings:            true,
		MCP:                 true,
		Memory:              true,
		ProjectInstructions: true,
		SkillArtifactDirs: map[string]string{
			"references": "references",
			"scripts":    "scripts",
			"assets":     "assets",
			"examples":   "", // empty string = flatten to skill root alongside SKILL.md
		},
		AgentNativeToolsOnly: true,
		RuleActivations:      []string{"always", "path-glob"},
		RuleEncoding: renderer.RuleEncodingCapabilities{
			Description: "frontmatter",
			Activation:  "frontmatter",
		},
	}
}

// CompileAgents compiles all agent configs to agents/<id>.md files.
func (r *Renderer) CompileAgents(agents map[string]ast.AgentConfig, baseDir string) (map[string]string, []renderer.FidelityNote, error) {
	files := make(map[string]string)
	caps := r.Capabilities()
	var notes []renderer.FidelityNote
	for id, agent := range agents {
		md, agentNotes, err := compileAgentMarkdown(agentMarkdownInput{id: id, agent: agent, baseDir: baseDir, caps: caps, memoryAgents: r.memoryAgents})
		if err != nil {
			return nil, nil, fmt.Errorf("failed to compile agent %q: %w", id, err)
		}
		safePath := filepath.Clean(fmt.Sprintf("agents/%s.md", id))
		files[safePath] = md
		notes = append(notes, agentNotes...)
	}
	return files, notes, nil
}

// CompileSkills compiles all skill configs to skills/<id>/SKILL.md plus subdirs.
func (r *Renderer) CompileSkills(skills map[string]ast.SkillConfig, baseDir string) (map[string]string, []renderer.FidelityNote, error) {
	out := &output.Output{Files: make(map[string]string)}
	caps := r.Capabilities()
	for id, skill := range skills {
		md, err := compileSkillMarkdown(id, skill, baseDir)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to compile skill %q: %w", id, err)
		}
		safePath := filepath.Clean(fmt.Sprintf("skills/%s/SKILL.md", id))
		out.Files[safePath] = md

		skillSourceDir := filepath.Join("xcaf", "skills", id)
		if err := compileSkillArtifacts(renderer.SkillArtifactContext{ID: id, Skill: skill, Caps: caps, BaseDir: baseDir, SkillSourceDir: skillSourceDir}, out); err != nil {
			return nil, nil, err
		}
	}
	return out.Files, nil, nil
}

// CompileRules compiles all rule configs to rules/<id>.md files.
func (r *Renderer) CompileRules(rules map[string]ast.RuleConfig, baseDir string) (map[string]string, []renderer.FidelityNote, error) {
	files := make(map[string]string)
	var notes []renderer.FidelityNote
	for id, rule := range rules {
		md, ruleNotes, err := compileClaudeRule(id, rule, r.Capabilities(), baseDir)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to compile rule %q: %w", id, err)
		}
		safePath := filepath.Clean(fmt.Sprintf("rules/%s.md", id))
		files[safePath] = md
		notes = append(notes, ruleNotes...)
	}
	return files, notes, nil
}

// writeWorkflowPrimitive writes a single translated primitive into files using
// Claude's standard paths. Primitives with a provider-native path (custom-command,
// prompt-file) are written directly after stripping the ".claude/" prefix so that
// apply.go does not double-prepend it.
func (r *Renderer) writeWorkflowPrimitive(p translator.TargetPrimitive, files map[string]string) {
	content := p.Content
	if content == "" {
		content = p.Body
	}
	switch p.Kind {
	case "rule":
		files[filepath.Clean(fmt.Sprintf("rules/%s.md", p.ID))] = content
	case "skill":
		files[filepath.Clean(fmt.Sprintf("skills/%s/SKILL.md", p.ID))] = content
	case "custom-command", "prompt-file":
		if p.Path != "" {
			files[strings.TrimPrefix(p.Path, r.OutputDir()+"/")] = content
		}
	}
}

// CompileWorkflows lowers workflow configs to provider-native primitives. Rule and
// skill primitives are written to their standard paths. Primitives with provider-
// native paths ("custom-command", "prompt-file") are written directly using the
// path set by the translator.
func (r *Renderer) CompileWorkflows(workflows map[string]ast.WorkflowConfig, baseDir string) (map[string]string, []renderer.FidelityNote, error) {
	files := make(map[string]string)
	var notes []renderer.FidelityNote
	for id, wf := range workflows {
		wfCopy := wf
		if wfCopy.Name == "" {
			wfCopy.Name = id
		}
		primitives, wfNotes := translator.TranslateWorkflow(&wfCopy, "claude")
		notes = append(notes, wfNotes...)
		for _, p := range primitives {
			r.writeWorkflowPrimitive(p, files)
		}
	}
	artifactNotes := renderer.AppendWorkflowArtifacts(renderer.WorkflowArtifactArgs{
		Target: "claude", Workflows: workflows, BaseDir: baseDir, Caps: r.Capabilities(), Files: files,
	})
	notes = append(notes, artifactNotes...)
	return files, notes, nil
}

// claudeHooksKey and claudeMCPSettingsKey are private staging keys used during
// Finalize to merge hooks and settings.MCPServers into their target output files.
// They are never written to disk; Finalize removes them before returning.
const (
	claudeHooksKey       = "__claude_hooks__"
	claudeMCPSettingsKey = "__claude_mcp_settings__"
)

// CompileHooks stages the hook config so Finalize can merge it into settings.json.
// Claude embeds hooks inside settings.json, not as a standalone file. Storing
// hooks under a private staging key avoids a last-writer-wins collision with
// CompileSettings (which also writes settings.json).
//
// Hook commands are translated before marshaling: $XCAF_PROJECT_DIR is rewritten
// to $CLAUDE_PROJECT_DIR and .xcaf/hooks/ paths become .claude/hooks/. A deep
// copy of the config is made so the shared input is never mutated.
func (r *Renderer) CompileHooks(hooks ast.HookConfig, baseDir string) (map[string]string, []renderer.FidelityNote, error) {
	if len(hooks) == 0 {
		return make(map[string]string), nil, nil
	}
	translated := make(ast.HookConfig, len(hooks))
	for event, groups := range hooks {
		translatedGroups := make([]ast.HookMatcherGroup, len(groups))
		for i, group := range groups {
			translatedHandlers := make([]ast.HookHandler, len(group.Hooks))
			for j, h := range group.Hooks {
				h.Command = renderer.TranslateHookCommand(h.Command, "$CLAUDE_PROJECT_DIR", ".claude/hooks/")
				translatedHandlers[j] = h
			}
			translatedGroups[i] = ast.HookMatcherGroup{
				Matcher: group.Matcher,
				Hooks:   translatedHandlers,
			}
		}
		translated[event] = translatedGroups
	}
	b, err := json.Marshal(translated)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal hooks: %w", err)
	}
	return map[string]string{claudeHooksKey: string(b)}, nil, nil
}

// CompileSettings compiles the settings block to settings.json. When
// settings.MCPServers is non-empty, it is staged under a private key so
// Finalize can merge those servers into mcp.json alongside config.MCP entries.
func (r *Renderer) CompileSettings(settings ast.SettingsConfig) (map[string]string, []renderer.FidelityNote, error) {
	files := make(map[string]string)

	settingsJSON, err := compileSettingsJSON(settings, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to compile settings: %w", err)
	}
	if settingsJSON != "" {
		files["settings.json"] = settingsJSON
	}

	// Stage settings.MCPServers for merging in Finalize.
	if len(settings.MCPServers) > 0 {
		b, err := json.Marshal(settings.MCPServers)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to marshal settings.MCPServers: %w", err)
		}
		files[claudeMCPSettingsKey] = string(b)
	}

	return files, nil, nil
}

// CompileMCP compiles MCP server definitions from the top-level mcp: block to
// mcp.json. settings.MCPServers is merged in Finalize.
func (r *Renderer) CompileMCP(servers map[string]ast.MCPConfig) (map[string]string, []renderer.FidelityNote, error) {
	files := make(map[string]string)
	mcpJSON, err := compileClaudeMCP(servers, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to compile MCP servers: %w", err)
	}
	if mcpJSON != "" {
		files["mcp.json"] = mcpJSON
	}
	return files, nil, nil
}

// CompileProjectInstructions emits CLAUDE.md at root and one CLAUDE.md per scope.
func (r *Renderer) CompileProjectInstructions(config *ast.XcaffoldConfig, baseDir string) (map[string]string, map[string]string, []renderer.FidelityNote, error) {
	files := make(map[string]string)
	rootFiles := make(map[string]string)

	// Synthesize a minimal config so renderProjectInstructions can read it.
	notes := r.renderProjectInstructions(config, baseDir, rootFiles)

	return files, rootFiles, notes, nil
}

// CompileMemory delegates to MemoryRenderer for disk writes when opts.OutputDir
// is set, or falls back to map-based compilation with no disk writes.
func (r *Renderer) CompileMemory(config *ast.XcaffoldConfig, baseDir string, opts renderer.MemoryOptions) (map[string]string, []renderer.FidelityNote, error) {
	if len(config.Memory) == 0 {
		return map[string]string{}, nil, nil
	}
	memDir := opts.OutputDir
	if memDir == "" {
		// No output directory — caller is in compile-only (orchestrator) mode.
		// Produce link-list index + individual files without disk writes.
		return r.compileMemoryToMap(config)
	}
	mr := NewMemoryRenderer(memDir)
	out, notes, err := mr.Compile(config, baseDir)
	if err != nil {
		return nil, notes, err
	}
	return out.Files, notes, nil
}

// compileMemoryToMap produces a link-list index (MEMORY.md) and individual
// content files per memory entry, grouped by AgentRef. No disk writes occur.
// Keys are split on "/" to derive filenames: "dev/orm-decision" → "orm-decision.md";
// keys without "/" use the full key as the filename stem.
func (r *Renderer) compileMemoryToMap(config *ast.XcaffoldConfig) (map[string]string, []renderer.FidelityNote, error) {
	type entry struct {
		key, fname, name, desc, body string
	}
	grouped := make(map[string][]entry)

	keys := make([]string, 0, len(config.Memory))
	for k := range config.Memory {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		e := config.Memory[key]
		if strings.TrimSpace(e.Content) == "" {
			continue
		}
		agentRef := e.AgentRef
		if agentRef == "" {
			agentRef = "default"
		}
		parts := strings.SplitN(key, "/", 2)
		fname := key + ".md"
		if len(parts) == 2 {
			fname = parts[1] + ".md"
		}
		grouped[agentRef] = append(grouped[agentRef], entry{
			key: key, fname: fname, name: e.Name, desc: e.Description, body: e.Content,
		})
	}

	files := make(map[string]string)
	for agentRef, entries := range grouped {
		var indexBuf strings.Builder
		for _, e := range entries {
			fmt.Fprintf(&indexBuf, "- [%s](%s) — %s\n", e.name, e.fname, e.desc)
			files[filepath.Join("agent-memory", agentRef, e.fname)] = e.body
		}
		files[filepath.Join("agent-memory", agentRef, "MEMORY.md")] = indexBuf.String()
	}
	return files, nil, nil
}

// Finalize merges staged hooks and settings.MCPServers into their target output
// files (settings.json and mcp.json), then removes the private staging keys.
//
// Hooks (claudeHooksKey): staged by CompileHooks and merged into the "hooks"
// key of settings.json. If settings.json is absent, a minimal one is created.
//
// MCPServers (claudeMCPSettingsKey): staged by CompileSettings and merged into
// mcp.json alongside any top-level config.MCP entries compiled by CompileMCP.
func (r *Renderer) Finalize(files map[string]string, rootFiles map[string]string) (map[string]string, map[string]string, []renderer.FidelityNote, error) {
	if err := mergeHooksIntoSettings(files); err != nil {
		return nil, nil, nil, err
	}
	if err := mergeMCPSettings(files); err != nil {
		return nil, nil, nil, err
	}
	return files, rootFiles, nil, nil
}

// mergeHooksIntoSettings moves the staged hook payload (claudeHooksKey) into
// the "hooks" key of settings.json, creating a minimal settings.json if absent.
func mergeHooksIntoSettings(files map[string]string) error {
	hooksRaw, ok := files[claudeHooksKey]
	if !ok {
		return nil
	}
	delete(files, claudeHooksKey)

	var hooks ast.HookConfig
	if err := json.Unmarshal([]byte(hooksRaw), &hooks); err != nil {
		return fmt.Errorf("Finalize: failed to unmarshal staged hooks: %w", err)
	}
	if len(hooks) == 0 {
		return nil
	}

	existing := make(map[string]any)
	if s, has := files["settings.json"]; has {
		if err := json.Unmarshal([]byte(s), &existing); err != nil {
			return fmt.Errorf("Finalize: failed to parse settings.json for hook merge: %w", err)
		}
	} else {
		existing["$schema"] = "https://json.schemastore.org/claude-code-settings.json"
	}
	existing["hooks"] = hooks
	b, err := json.MarshalIndent(existing, "", "  ")
	if err != nil {
		return fmt.Errorf("Finalize: failed to re-serialize settings.json with hooks: %w", err)
	}
	files["settings.json"] = string(b)
	return nil
}

// mergeMCPSettings moves the staged MCPServers payload (claudeMCPSettingsKey)
// into mcp.json, merging with any existing entries (settings payload wins on conflict).
func mergeMCPSettings(files map[string]string) error {
	mcpRaw, ok := files[claudeMCPSettingsKey]
	if !ok {
		return nil
	}
	delete(files, claudeMCPSettingsKey)

	var settingsMCPServers map[string]ast.MCPConfig
	if err := json.Unmarshal([]byte(mcpRaw), &settingsMCPServers); err != nil {
		return fmt.Errorf("Finalize: failed to unmarshal staged MCPServers: %w", err)
	}
	if len(settingsMCPServers) == 0 {
		return nil
	}

	mcpServers := make(map[string]ast.MCPConfig)
	if m, has := files["mcp.json"]; has {
		var envelope struct {
			MCPServers map[string]ast.MCPConfig `json:"mcpServers"`
		}
		if err := json.Unmarshal([]byte(m), &envelope); err != nil {
			return fmt.Errorf("Finalize: failed to parse mcp.json for merge: %w", err)
		}
		for k, v := range envelope.MCPServers {
			mcpServers[k] = v
		}
	}
	// settings.MCPServers wins on conflict (original compile() semantics).
	for k, v := range settingsMCPServers {
		mcpServers[k] = v
	}
	b, err := json.MarshalIndent(map[string]any{"mcpServers": mcpServers}, "", "  ")
	if err != nil {
		return fmt.Errorf("Finalize: failed to re-serialize mcp.json: %w", err)
	}
	files["mcp.json"] = string(b)
	return nil
}

// renderProjectInstructions emits CLAUDE.md at root and one CLAUDE.md per scope.
// This is the concat-nested class — the reference implementation with zero fidelity loss.
func (r *Renderer) renderProjectInstructions(config *ast.XcaffoldConfig, baseDir string, files map[string]string) []renderer.FidelityNote {
	rootContent := renderer.ResolveContextBody(config, "claude")
	if rootContent == "" {
		return nil
	}

	files["CLAUDE.md"] = rootContent
	return nil // concat-nested: zero fidelity notes
}

// agentMarkdownInput holds parameters for compileAgentMarkdown.
type agentMarkdownInput struct {
	id           string
	agent        ast.AgentConfig
	baseDir      string
	caps         renderer.CapabilitySet
	memoryAgents map[string]bool
}

// compileAgentMarkdown renders a single AgentConfig to Claude Code markdown.
func compileAgentMarkdown(input agentMarkdownInput) (string, []renderer.FidelityNote, error) {
	id := input.id
	agent := input.agent
	caps := input.caps
	memoryAgents := input.memoryAgents
	if strings.TrimSpace(id) == "" {
		return "", nil, fmt.Errorf("agent id must not be empty")
	}

	body := resolver.StripFrontmatter(agent.Body)

	var sb strings.Builder
	var notes []renderer.FidelityNote

	sb.WriteString("---\n")

	appendAgentCoreMeta(&sb, agent)

	sanitizedTools, toolNotes := renderer.SanitizeAgentTools(agent.Tools.Values, caps, "claude", id)
	notes = append(notes, toolNotes...)

	if agent.Readonly != nil && *agent.Readonly && len(sanitizedTools) == 0 {
		sb.WriteString("tools: [Read, Grep, Glob]\n")
	} else if len(sanitizedTools) > 0 {
		fmt.Fprintf(&sb, "tools: [%s]\n", strings.Join(sanitizedTools, ", "))
	}
	if len(agent.DisallowedTools.Values) > 0 {
		fmt.Fprintf(&sb, "disallowed-tools: [%s]\n", strings.Join(agent.DisallowedTools.Values, ", "))
	}
	if len(agent.Skills.Values) > 0 {
		fmt.Fprintf(&sb, "skills: [%s]\n", strings.Join(agent.Skills.Values, ", "))
	}

	resolvedModel, modelNotes := renderer.SanitizeAgentModel(agent.Model, caps, "claude", id)
	notes = append(notes, modelNotes...)
	if resolvedModel != "" {
		fmt.Fprintf(&sb, "model: %s\n", resolvedModel)
	}

	appendAgentConfigMeta(&sb, id, agent, memoryAgents)

	if err := appendAgentYAMLMeta(&sb, agent); err != nil {
		return "", nil, err
	}

	sb.WriteString("---\n")

	if body != "" {
		sb.WriteString("\n")
		sb.WriteString(strings.TrimRight(body, "\n"))
		sb.WriteString("\n")
	}

	return sb.String(), notes, nil
}

// compileSkillMarkdown renders a single SkillConfig to its SKILL.md content.
func compileSkillMarkdown(id string, skill ast.SkillConfig, baseDir string) (string, error) {
	if strings.TrimSpace(id) == "" {
		return "", fmt.Errorf("skill id must not be empty")
	}

	body := resolver.StripFrontmatter(skill.Body)

	var sb strings.Builder
	sb.WriteString("---\n")

	appendSkillMeta(&sb, skill)
	sb.WriteString("---\n")

	if body != "" {
		sb.WriteString("\n")
		sb.WriteString(strings.TrimRight(body, "\n"))
		sb.WriteString("\n")
	}

	return sb.String(), nil
}

// compileClaudeRule compiles a single rule to markdown.
func compileClaudeRule(id string, rule ast.RuleConfig, caps renderer.CapabilitySet, baseDir string) (string, []renderer.FidelityNote, error) {
	if strings.TrimSpace(id) == "" {
		return "", nil, fmt.Errorf("rule id must not be empty")
	}

	body := resolver.StripFrontmatter(rule.Body)

	activation := renderer.ResolvedActivation(rule)

	var notes []renderer.FidelityNote

	// Claude natively supports always and path-glob. Everything else is
	// emitted as always-loaded with a warning note.
	if !renderer.ValidateRuleActivation(rule, caps) {
		notes = append(notes, renderer.FidelityNote{
			Level:      renderer.LevelWarning,
			Target:     "claude",
			Kind:       "rule",
			Resource:   id,
			Field:      "activation",
			Code:       renderer.CodeRuleActivationUnsupported,
			Reason:     fmt.Sprintf("rule %q: activation %q has no Claude native equivalent; rule emitted as always-loaded", id, activation),
			Mitigation: "Use activation: always or activation: path-glob for Claude.",
		})
	}

	// exclude-agents has no Claude equivalent; drop it and emit an info note.
	if len(rule.ExcludeAgents.Values) > 0 {
		notes = append(notes, renderer.FidelityNote{
			Level:      renderer.LevelInfo,
			Target:     "claude",
			Kind:       "rule",
			Resource:   id,
			Field:      "exclude-agents",
			Code:       renderer.CodeRuleExcludeAgentsDropped,
			Reason:     fmt.Sprintf("rule %q: exclude-agents %v has no Claude native equivalent and was dropped", id, rule.ExcludeAgents.Values),
			Mitigation: "Remove exclude-agents or target a provider that supports it (e.g. copilot).",
		})
	}

	var sb strings.Builder

	sb.WriteString("---\n")
	sb.WriteString(renderer.BuildRuleDescriptionFrontmatter(rule, caps))
	// Emit paths: only when activation resolves to path-glob.
	if activation == ast.RuleActivationPathGlob && len(rule.Paths.Values) > 0 {
		fmt.Fprintf(&sb, "paths: [%s]\n", strings.Join(rule.Paths.Values, ", "))
	}
	sb.WriteString("---\n")

	if body != "" {
		sb.WriteString("\n")
		sb.WriteString(strings.TrimRight(body, "\n"))
		sb.WriteString("\n")
	}

	return sb.String(), notes, nil
}
