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
	"gopkg.in/yaml.v3"
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
		md, agentNotes, err := compileAgentMarkdown(id, agent, baseDir, caps, r.memoryAgents)
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

		if len(skill.Artifacts) > 0 {
			if err := compileSkillArtifacts(id, skill, caps, baseDir, out); err != nil {
				return nil, nil, err
			}
		} else {
			// Legacy path: individual fields for skills that predate the artifacts field.
			if err := renderer.CompileSkillSubdir(id, "references", "references", skill.References, baseDir, out); err != nil {
				return nil, nil, fmt.Errorf("failed to compile references for skill %q: %w", id, err)
			}
			if err := renderer.CompileSkillSubdir(id, "scripts", "scripts", skill.Scripts, baseDir, out); err != nil {
				return nil, nil, fmt.Errorf("failed to compile scripts for skill %q: %w", id, err)
			}
			if err := renderer.CompileSkillSubdir(id, "assets", "assets", skill.Assets, baseDir, out); err != nil {
				return nil, nil, fmt.Errorf("failed to compile assets for skill %q: %w", id, err)
			}
			// Claude flattens examples alongside SKILL.md (no subdirectory).
			if err := renderer.FlattenToSkillRoot(id, "examples", skill.Examples, baseDir, out); err != nil {
				return nil, nil, fmt.Errorf("failed to compile examples for skill %q: %w", id, err)
			}
		}
	}
	return out.Files, nil, nil
}

// compileSkillArtifacts iterates skill.Artifacts and dispatches each artifact
// to the correct output subdirectory using the renderer's SkillArtifactDirs map.
// An empty outputSubdir means the files are flattened to the skill root.
func compileSkillArtifacts(id string, skill ast.SkillConfig, caps renderer.CapabilitySet, baseDir string, out *output.Output) error {
	for _, artifactName := range skill.Artifacts {
		outputSubdir, ok := caps.SkillArtifactDirs[artifactName]
		if !ok {
			outputSubdir = artifactName // default: use same name as canonical
		}
		var paths []string
		switch artifactName {
		case "references":
			paths = skill.References
		case "scripts":
			paths = skill.Scripts
		case "assets":
			paths = skill.Assets
		case "examples":
			paths = skill.Examples
		}
		if len(paths) == 0 {
			continue
		}
		if outputSubdir == "" {
			if err := renderer.FlattenToSkillRoot(id, artifactName, paths, baseDir, out); err != nil {
				return fmt.Errorf("skill %s artifact %s: %w", id, artifactName, err)
			}
		} else {
			if err := renderer.CompileSkillSubdir(id, artifactName, outputSubdir, paths, baseDir, out); err != nil {
				return fmt.Errorf("skill %s artifact %s: %w", id, artifactName, err)
			}
		}
	}
	return nil
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

// CompileWorkflows lowers workflow configs to rules and skills.
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
			content := p.Content
			if content == "" {
				content = p.Body
			}
			switch p.Kind {
			case "rule":
				safePath := filepath.Clean(fmt.Sprintf("rules/%s.md", p.ID))
				files[safePath] = content
			case "skill":
				safePath := filepath.Clean(fmt.Sprintf("skills/%s/SKILL.md", p.ID))
				files[safePath] = content
			}
		}
	}
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
func (r *Renderer) CompileHooks(hooks ast.HookConfig, baseDir string) (map[string]string, []renderer.FidelityNote, error) {
	if len(hooks) == 0 {
		return make(map[string]string), nil, nil
	}
	b, err := json.Marshal(hooks)
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

// CompileProjectInstructions emits CLAUDE.md at root and one CLAUDE.md per
// scope, plus settings.local.json when a project.local block is present.
func (r *Renderer) CompileProjectInstructions(config *ast.XcaffoldConfig, baseDir string) (map[string]string, map[string]string, []renderer.FidelityNote, error) {
	files := make(map[string]string)
	rootFiles := make(map[string]string)

	// Synthesize a minimal config so renderProjectInstructions can read it.
	notes := r.renderProjectInstructions(config, baseDir, rootFiles)

	// Emit settings.local.json when a local block is present.
	var local ast.SettingsConfig
	if config.Project != nil {
		local = config.Project.Local
	}
	localJSON, err := compileSettingsJSON(local, nil)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to compile local settings: %w", err)
	}
	if localJSON != "" {
		files["settings.local.json"] = localJSON
	}

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
	// Merge staged hooks into settings.json.
	if hooksRaw, ok := files[claudeHooksKey]; ok {
		delete(files, claudeHooksKey)
		var hooks ast.HookConfig
		if err := json.Unmarshal([]byte(hooksRaw), &hooks); err != nil {
			return nil, nil, nil, fmt.Errorf("Finalize: failed to unmarshal staged hooks: %w", err)
		}
		if len(hooks) > 0 {
			// Parse existing settings.json (if any) and inject hooks.
			existing := make(map[string]any)
			if s, has := files["settings.json"]; has {
				if err := json.Unmarshal([]byte(s), &existing); err != nil {
					return nil, nil, nil, fmt.Errorf("Finalize: failed to parse settings.json for hook merge: %w", err)
				}
			} else {
				existing["$schema"] = "https://cdn.jsdelivr.net/npm/@anthropic-ai/claude-code@latest/config-schema.json"
			}
			existing["hooks"] = hooks
			b, err := json.MarshalIndent(existing, "", "  ")
			if err != nil {
				return nil, nil, nil, fmt.Errorf("Finalize: failed to re-serialize settings.json with hooks: %w", err)
			}
			files["settings.json"] = string(b)
		}
	}

	// Merge staged settings.MCPServers into mcp.json.
	if mcpRaw, ok := files[claudeMCPSettingsKey]; ok {
		delete(files, claudeMCPSettingsKey)
		var settingsMCPServers map[string]ast.MCPConfig
		if err := json.Unmarshal([]byte(mcpRaw), &settingsMCPServers); err != nil {
			return nil, nil, nil, fmt.Errorf("Finalize: failed to unmarshal staged MCPServers: %w", err)
		}
		if len(settingsMCPServers) > 0 {
			// Parse existing mcp.json (if any) and merge.
			mcpServers := make(map[string]ast.MCPConfig)
			if m, has := files["mcp.json"]; has {
				var envelope struct {
					MCPServers map[string]ast.MCPConfig `json:"mcpServers"`
				}
				if err := json.Unmarshal([]byte(m), &envelope); err != nil {
					return nil, nil, nil, fmt.Errorf("Finalize: failed to parse mcp.json for merge: %w", err)
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
				return nil, nil, nil, fmt.Errorf("Finalize: failed to re-serialize mcp.json: %w", err)
			}
			files["mcp.json"] = string(b)
		}
	}

	return files, rootFiles, nil, nil
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

// compileAgentMarkdown renders a single AgentConfig to Claude Code markdown.
func compileAgentMarkdown(id string, agent ast.AgentConfig, baseDir string, caps renderer.CapabilitySet, memoryAgents map[string]bool) (string, []renderer.FidelityNote, error) {
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

func appendAgentCoreMeta(sb *strings.Builder, agent ast.AgentConfig) {
	if agent.Name != "" {
		fmt.Fprintf(sb, "name: %s\n", agent.Name)
	}
	if agent.Description != "" {
		fmt.Fprintf(sb, "description: %s\n", agent.Description)
	}
	if agent.Effort != "" {
		fmt.Fprintf(sb, "effort: %s\n", agent.Effort)
	}
	if agent.MaxTurns > 0 {
		fmt.Fprintf(sb, "max-turns: %d\n", agent.MaxTurns)
	}
}

func appendAgentConfigMeta(sb *strings.Builder, agentID string, agent ast.AgentConfig, memoryAgents map[string]bool) {
	if agent.PermissionMode != "" {
		fmt.Fprintf(sb, "permission-mode: %s\n", agent.PermissionMode)
	}
	// disable-model-invocation and user-invocable are Copilot-only agent fields.
	// Claude Code does not support them for agents (they are valid for skills).
	// Drop silently — no fidelity note needed because these fields have no effect.
	if agent.Background != nil {
		fmt.Fprintf(sb, "background: %t\n", *agent.Background)
	}
	if agent.Isolation != "" {
		fmt.Fprintf(sb, "isolation: %s\n", agent.Isolation)
	}
	if len(agent.Memory) > 0 {
		fmt.Fprintf(sb, "memory: %s\n", strings.Join([]string(agent.Memory), ", "))
	} else if memoryAgents[agentID] {
		fmt.Fprintf(sb, "memory: user\n")
	}
	if agent.Color != "" {
		fmt.Fprintf(sb, "color: %s\n", agent.Color)
	}
	if agent.InitialPrompt != "" {
		fmt.Fprintf(sb, "initial-prompt: %s\n", agent.InitialPrompt)
	}
}

func appendAgentYAMLMeta(sb *strings.Builder, agent ast.AgentConfig) error {
	if len(agent.Hooks) > 0 {
		hooksYAML, err := yaml.Marshal(map[string]ast.HookConfig{"hooks": agent.Hooks})
		if err != nil {
			return fmt.Errorf("failed to marshal agent hooks: %w", err)
		}
		sb.WriteString(string(hooksYAML))
	}
	if len(agent.MCPServers) > 0 {
		mcpYAML, err := yaml.Marshal(map[string]map[string]ast.MCPConfig{"mcpServers": agent.MCPServers})
		if err != nil {
			return fmt.Errorf("failed to marshal agent mcpServers: %w", err)
		}
		sb.WriteString(string(mcpYAML))
	}
	return nil
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

func appendSkillMeta(sb *strings.Builder, skill ast.SkillConfig) {
	// Group 1 — Identity
	if skill.Name != "" {
		fmt.Fprintf(sb, "name: %s\n", skill.Name)
	}
	if skill.Description != "" {
		fmt.Fprintf(sb, "description: %s\n", skill.Description)
	}
	// when_to_use uses snake_case (not kebab-case) because Claude Code's native
	// SKILL.md schema requires this exact field name.
	if skill.WhenToUse != "" {
		fmt.Fprintf(sb, "when_to_use: %s\n", skill.WhenToUse)
	}
	if skill.License != "" {
		fmt.Fprintf(sb, "license: %s\n", skill.License)
	}

	// Group 3 — Tool Access (Claude convention: space-separated string)
	if len(skill.AllowedTools) > 0 {
		fmt.Fprintf(sb, "allowed-tools: %s\n", strings.Join(skill.AllowedTools, " "))
	}

	// Group 4 — Permissions & Invocation Control (hyphenated kebab-case for Claude)
	if skill.DisableModelInvocation != nil {
		fmt.Fprintf(sb, "disable-model-invocation: %t\n", *skill.DisableModelInvocation)
	}
	if skill.UserInvocable != nil {
		fmt.Fprintf(sb, "user-invocable: %t\n", *skill.UserInvocable)
	}
	if skill.ArgumentHint != "" {
		data, err := yaml.Marshal(map[string]any{"argument-hint": skill.ArgumentHint})
		if err == nil {
			sb.Write(data)
		}
	}

	// Claude-specific provider pass-through
	if claude, ok := skill.Targets["claude"]; ok {
		emitClaudeProviderKeys(sb, claude.Provider)
	}
}

// emitClaudeProviderKeys writes Claude-recognized provider keys in a stable order.
// All values are routed through yaml.Marshal to ensure correct escaping and quoting.
// Unknown keys are ignored (renderer-level warnings handled by caller).
func emitClaudeProviderKeys(sb *strings.Builder, provider map[string]any) {
	if len(provider) == 0 {
		return
	}
	orderedKeys := []string{"context", "agent", "model", "effort", "shell", "paths", "hooks"}
	for _, k := range orderedKeys {
		v, ok := provider[k]
		if !ok {
			continue
		}
		data, err := yaml.Marshal(map[string]any{k: v})
		if err != nil {
			continue
		}
		sb.Write(data)
	}
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
		notes = append(notes, renderer.NewNote(
			renderer.LevelWarning,
			"claude",
			"rule",
			id,
			"activation",
			renderer.CodeRuleActivationUnsupported,
			fmt.Sprintf("rule %q: activation %q has no Claude native equivalent; rule emitted as always-loaded", id, activation),
			"Use activation: always or activation: path-glob for Claude.",
		))
	}

	// exclude-agents has no Claude equivalent; drop it and emit an info note.
	if len(rule.ExcludeAgents) > 0 {
		notes = append(notes, renderer.NewNote(
			renderer.LevelInfo,
			"claude",
			"rule",
			id,
			"exclude-agents",
			renderer.CodeRuleExcludeAgentsDropped,
			fmt.Sprintf("rule %q: exclude-agents %v has no Claude native equivalent and was dropped", id, rule.ExcludeAgents),
			"Remove exclude-agents or target a provider that supports it (e.g. copilot).",
		))
	}

	var sb strings.Builder

	sb.WriteString("---\n")
	sb.WriteString(renderer.BuildRuleDescriptionFrontmatter(rule, caps))
	// Emit paths: only when activation resolves to path-glob.
	if activation == ast.RuleActivationPathGlob && len(rule.Paths) > 0 {
		fmt.Fprintf(&sb, "paths: [%s]\n", strings.Join(rule.Paths, ", "))
	}
	sb.WriteString("---\n")

	if body != "" {
		sb.WriteString("\n")
		sb.WriteString(strings.TrimRight(body, "\n"))
		sb.WriteString("\n")
	}

	return sb.String(), notes, nil
}

// compileSettingsJSON produces a fully-populated settings.json.
// Note: mcpServers are now emitted to mcp.json.
//
// Merge rules:
//   - Output is suppressed (empty string) when the resulting object has no
//     meaningful content, to avoid writing a useless "{}".
//   - The $schema key is always emitted first when there is content.
func compileSettingsJSON(settings ast.SettingsConfig, hooks ast.HookConfig) (string, error) {
	out := map[string]any{
		"$schema": "https://cdn.jsdelivr.net/npm/@anthropic-ai/claude-code@latest/config-schema.json",
	}

	populateSettingsCore(out, settings)
	populateSettingsFeatures(out, settings)
	populateSettingsDev(out, settings)
	populateSettingsAgent(out, settings)
	populateSettingsSystem(out, settings)

	if len(hooks) > 0 {
		out["hooks"] = hooks
	}

	// len(out) == 1 means only $schema is present
	if len(out) <= 1 {
		return "", nil
	}

	b, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// compileClaudeMCP renders MCP definitions into mcp.json
// It merges the top-level mcp: shorthand block with settings.mcpServers.
func compileClaudeMCP(mcpShorthand map[string]ast.MCPConfig, settingsMCPServers map[string]ast.MCPConfig) (string, error) {
	mcpServers := make(map[string]ast.MCPConfig)
	for k, v := range mcpShorthand {
		mcpServers[k] = v
	}
	for k, v := range settingsMCPServers {
		mcpServers[k] = v
	}

	if len(mcpServers) == 0 {
		return "", nil
	}

	envelope := map[string]any{
		"mcpServers": mcpServers,
	}

	b, err := json.MarshalIndent(envelope, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func populateSettingsCore(out map[string]any, settings ast.SettingsConfig) {
	if len(settings.Env) > 0 {
		out["env"] = settings.Env
	}
	if settings.StatusLine != nil {
		out["statusLine"] = settings.StatusLine
	}
	if len(settings.EnabledPlugins) > 0 {
		out["enabledPlugins"] = settings.EnabledPlugins
	}
	if settings.Permissions != nil {
		out["permissions"] = settings.Permissions
	}
}

func populateSettingsFeatures(out map[string]any, settings ast.SettingsConfig) {
	if settings.AlwaysThinkingEnabled != nil {
		out["alwaysThinkingEnabled"] = *settings.AlwaysThinkingEnabled
	}
	if settings.EffortLevel != "" {
		out["effortLevel"] = settings.EffortLevel
	}
	if settings.SkipDangerousModePermissionPrompt != nil {
		out["skipDangerousModePermissionPrompt"] = *settings.SkipDangerousModePermissionPrompt
	}
	if settings.Sandbox != nil {
		out["sandbox"] = settings.Sandbox
	}
	if len(settings.Hooks) > 0 {
		out["hooks"] = settings.Hooks
	}
}

func populateSettingsDev(out map[string]any, settings ast.SettingsConfig) {
	if settings.OtelHeadersHelper != "" {
		out["otelHeadersHelper"] = settings.OtelHeadersHelper
	}
	if settings.DisableAllHooks != nil {
		out["disableAllHooks"] = *settings.DisableAllHooks
	}
	if settings.Attribution != nil {
		out["attribution"] = *settings.Attribution
	}
	if settings.OutputStyle != "" {
		out["outputStyle"] = settings.OutputStyle
	}
	if settings.Language != "" {
		out["language"] = settings.Language
	}
}

func populateSettingsAgent(out map[string]any, settings ast.SettingsConfig) {
	if settings.Model != "" {
		if resolved, ok := renderer.ResolveModel(settings.Model, "claude"); ok && resolved != "" {
			out["model"] = resolved
		} else {
			out["model"] = settings.Model
		}
	}
	if settings.Agent != nil {
		out["agent"] = settings.Agent
	}
	if settings.AutoMode != nil {
		out["autoMode"] = settings.AutoMode
	}
	if len(settings.AvailableModels) > 0 {
		out["availableModels"] = settings.AvailableModels
	}
	if settings.AutoMemoryEnabled != nil {
		out["autoMemoryEnabled"] = *settings.AutoMemoryEnabled
	}
	if settings.AutoMemoryDirectory != "" {
		out["autoMemoryDirectory"] = settings.AutoMemoryDirectory
	}
}

func populateSettingsSystem(out map[string]any, settings ast.SettingsConfig) {
	if settings.IncludeGitInstructions != nil {
		out["includeGitInstructions"] = *settings.IncludeGitInstructions
	}
	if settings.DisableSkillShellExecution != nil {
		out["disableSkillShellExecution"] = *settings.DisableSkillShellExecution
	}
	if settings.DefaultShell != "" {
		out["defaultShell"] = settings.DefaultShell
	}
	if settings.CleanupPeriodDays != nil {
		out["cleanupPeriodDays"] = *settings.CleanupPeriodDays
	}
	if settings.RespectGitignore != nil {
		out["respectGitignore"] = *settings.RespectGitignore
	}
	if settings.PlansDirectory != "" {
		out["plansDirectory"] = settings.PlansDirectory
	}
	if settings.Worktree != nil {
		out["worktree"] = settings.Worktree
	}
	if len(settings.ClaudeMdExcludes) > 0 {
		out["claudeMdExcludes"] = settings.ClaudeMdExcludes
	}
}
