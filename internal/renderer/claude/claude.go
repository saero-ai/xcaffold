package claude

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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
type Renderer struct{}

// New returns a new Renderer instance.
func New() *Renderer {
	return &Renderer{}
}

// Target returns the identifier for this renderer's target platform.
func (r *Renderer) Target() string {
	return "claude"
}

// OutputDir returns the base output directory for this renderer.
func (r *Renderer) OutputDir() string {
	return ".claude"
}

// Render wraps a files map in a output.Output. This is an identity
// operation for Claude — no additional path prefix is needed.
func (r *Renderer) Render(files map[string]string) *output.Output {
	return &output.Output{Files: files}
}

// Compile translates an XcaffoldConfig AST into its Claude Code output
// representation. baseDir is the directory that contains the project.xcf file;
// it is used to resolve instructions_file: and references: paths. The second
// return is a slice of fidelity notes; Claude is the native target and has no
// fidelity gaps, so Compile always returns a nil notes slice.
// Compile returns an error if any resource fails to compile. It never panics.
func (r *Renderer) Compile(config *ast.XcaffoldConfig, baseDir string) (*output.Output, []renderer.FidelityNote, error) {
	out := &output.Output{
		Files: make(map[string]string),
	}

	// Compile all agent personas to .claude/agents/*.md
	for id, agent := range config.Agents {
		md, err := compileAgentMarkdown(id, agent, baseDir)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to compile agent %q: %w", id, err)
		}
		safePath := filepath.Clean(fmt.Sprintf("agents/%s.md", id))
		out.Files[safePath] = md
	}

	// Compile all skills to .claude/skills/<id>/SKILL.md
	for id, skill := range config.Skills {
		md, err := compileSkillMarkdown(id, skill, baseDir)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to compile skill %q: %w", id, err)
		}
		safePath := filepath.Clean(fmt.Sprintf("skills/%s/SKILL.md", id))
		out.Files[safePath] = md

		if err := compileSkillSubdir(id, "references", skill.References, baseDir, out); err != nil {
			return nil, nil, fmt.Errorf("failed to compile references for skill %q: %w", id, err)
		}
		if err := compileSkillSubdir(id, "scripts", skill.Scripts, baseDir, out); err != nil {
			return nil, nil, fmt.Errorf("failed to compile scripts for skill %q: %w", id, err)
		}
		if err := compileSkillSubdir(id, "assets", skill.Assets, baseDir, out); err != nil {
			return nil, nil, fmt.Errorf("failed to compile assets for skill %q: %w", id, err)
		}
	}

	// Compile all rules to .claude/rules/*.md
	var notes []renderer.FidelityNote
	for id, rule := range config.Rules {
		md, ruleNotes, err := compileRuleMarkdown(id, rule, baseDir)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to compile rule %q: %w", id, err)
		}
		safePath := filepath.Clean(fmt.Sprintf("rules/%s.md", id))
		out.Files[safePath] = md
		notes = append(notes, ruleNotes...)
	}

	// Lower workflows to rule + per-step skills.
	for id, wf := range config.Workflows {
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
				out.Files[safePath] = content
			case "skill":
				safePath := filepath.Clean(fmt.Sprintf("skills/%s/SKILL.md", p.ID))
				out.Files[safePath] = content
			}
		}
	}

	mcpJSON, err := compileClaudeMCP(config.MCP, config.Settings.MCPServers)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to compile MCP servers: %w", err)
	}
	if mcpJSON != "" {
		out.Files["mcp.json"] = mcpJSON
	}

	settingsJSON, err := compileSettingsJSON(config.Settings, config.Hooks)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to compile settings: %w", err)
	}
	if settingsJSON != "" {
		out.Files["settings.json"] = settingsJSON
	}

	var localSettings ast.SettingsConfig
	if config.Project != nil {
		localSettings = config.Project.Local
	}
	localJSON, err := compileSettingsJSON(localSettings, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to compile local settings: %w", err)
	}
	if localJSON != "" {
		out.Files["settings.local.json"] = localJSON
	}

	if config.Project != nil {
		instrNotes := r.renderProjectInstructions(config, baseDir, out.Files)
		notes = append(notes, instrNotes...)
	}

	// Memory rendering is handled separately by MemoryRenderer (called via
	// runMemoryPass in apply.go) which supports lifecycle tracking, drift
	// detection, and seed-once semantics. The compiler Compile() path
	// intentionally excludes memory from its output map.

	return out, notes, nil
}

// renderProjectInstructions emits CLAUDE.md at root and one CLAUDE.md per scope.
// This is the concat-nested class — the reference implementation with zero fidelity loss.
func (r *Renderer) renderProjectInstructions(config *ast.XcaffoldConfig, baseDir string, files map[string]string) []renderer.FidelityNote {
	p := config.Project
	if p.Instructions == "" && p.InstructionsFile == "" {
		return nil
	}

	rootContent := renderer.ResolveInstructionsContent(p.Instructions, p.InstructionsFile, baseDir)

	// Append @-import lines for each import entry.
	for _, imp := range p.InstructionsImports {
		rootContent += "\n@" + imp
	}
	files["CLAUDE.md"] = rootContent

	// Emit one file per scope.
	for _, scope := range p.InstructionsScopes {
		content := renderer.ResolveScopeContent(scope, "claude", baseDir)
		files[filepath.Clean(scope.Path+"/CLAUDE.md")] = content
	}
	return nil // concat-nested: zero fidelity notes
}

// resolveInstructions returns the effective body content for an agent/skill/rule.
//
// Priority (highest to lowest):
//  1. inline          — the "instructions:" YAML field
//  2. filePath        — the "instructions_file:" YAML field (read from disk)
//  3. conventionPath  — auto-discovered by convention (agents/<id>.md etc.); silent no-op if missing
//
// The file is read relative to baseDir. Its frontmatter (--- blocks) is stripped
// so that referencing an existing .md file with frontmatter works transparently.

// stripFrontmatter removes YAML frontmatter delimited by "---" from the start
// of a markdown file, returning only the body content with leading whitespace trimmed.

// compileAgentMarkdown renders a single AgentConfig to Claude Code markdown.
func compileAgentMarkdown(id string, agent ast.AgentConfig, baseDir string) (string, error) {
	if strings.TrimSpace(id) == "" {
		return "", fmt.Errorf("agent id must not be empty")
	}

	body, err := resolver.ResolveInstructions(agent.Instructions, agent.InstructionsFile, fmt.Sprintf("agents/%s.md", id), baseDir)
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	sb.WriteString("---\n")

	appendAgentCoreMeta(&sb, agent)
	appendAgentToolsMeta(&sb, agent)
	appendAgentConfigMeta(&sb, agent)

	if err := appendAgentYAMLMeta(&sb, agent); err != nil {
		return "", err
	}

	sb.WriteString("---\n")

	if body != "" {
		sb.WriteString("\n")
		sb.WriteString(strings.TrimRight(body, "\n"))
		sb.WriteString("\n")
	}

	return sb.String(), nil
}

func appendAgentCoreMeta(sb *strings.Builder, agent ast.AgentConfig) {
	if agent.Name != "" {
		fmt.Fprintf(sb, "name: %s\n", agent.Name)
	}
	if agent.Description != "" {
		fmt.Fprintf(sb, "description: %s\n", agent.Description)
	}
	if agent.Model != "" {
		if resolved, ok := renderer.ResolveModel(agent.Model, "claude"); ok && resolved != "" {
			fmt.Fprintf(sb, "model: %s\n", resolved)
		} else {
			fmt.Fprintf(sb, "model: %s\n", agent.Model) // fallback if something fails
		}
	}
	if agent.Effort != "" {
		fmt.Fprintf(sb, "effort: %s\n", agent.Effort)
	}
	if agent.MaxTurns > 0 {
		fmt.Fprintf(sb, "max-turns: %d\n", agent.MaxTurns)
	}
}

func appendAgentToolsMeta(sb *strings.Builder, agent ast.AgentConfig) {
	if agent.Readonly != nil && *agent.Readonly && len(agent.Tools) == 0 {
		sb.WriteString("tools: [Read, Grep, Glob]\n")
	} else if len(agent.Tools) > 0 {
		fmt.Fprintf(sb, "tools: [%s]\n", strings.Join(agent.Tools, ", "))
	}
	if len(agent.DisallowedTools) > 0 {
		fmt.Fprintf(sb, "disallowed-tools: [%s]\n", strings.Join(agent.DisallowedTools, ", "))
	}
	if len(agent.Skills) > 0 {
		fmt.Fprintf(sb, "skills: [%s]\n", strings.Join(agent.Skills, ", "))
	}
	if len(agent.Rules) > 0 {
		fmt.Fprintf(sb, "rules: [%s]\n", strings.Join(agent.Rules, ", "))
	}
}

func appendAgentConfigMeta(sb *strings.Builder, agent ast.AgentConfig) {
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
	if agent.Memory != "" {
		fmt.Fprintf(sb, "memory: %s\n", agent.Memory)
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

	body, err := resolver.ResolveInstructions(skill.Instructions, skill.InstructionsFile, fmt.Sprintf("skills/%s/SKILL.md", id), baseDir)
	if err != nil {
		return "", err
	}

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

// compileSkillSubdir copies a set of files (resolved via glob from baseDir)
// into a named subdirectory of the skill's output directory:
//
//	skills/<id>/<subdir>/<filename>
//
// Supported subdirs: references, scripts, assets.
// Each pattern in paths is resolved relative to baseDir. Path traversal above
// baseDir is rejected. Glob patterns are expanded; literal paths are read directly.
func compileSkillSubdir(id, subdir string, paths []string, baseDir string, out *output.Output) error {
	if len(paths) == 0 {
		return nil
	}

	for _, pattern := range paths {
		// Security: pattern must not traverse above baseDir.
		cleanedPattern := filepath.Clean(pattern)
		if strings.HasPrefix(cleanedPattern, "..") {
			return fmt.Errorf("%s path %q traverses above the project root", subdir, pattern)
		}

		absPattern := filepath.Join(baseDir, cleanedPattern)

		// Expand glob patterns (e.g. "docs/schema/*.sql")
		matches, err := filepath.Glob(absPattern)
		if err != nil {
			return fmt.Errorf("invalid glob pattern %q: %w", pattern, err)
		}
		if len(matches) == 0 {
			// Treat as a literal path — if missing, it's an error.
			data, readErr := os.ReadFile(absPattern)
			if readErr != nil {
				return fmt.Errorf("%s file %q: %w", subdir, pattern, readErr)
			}
			baseName := filepath.Base(absPattern)
			outPath := filepath.Clean(fmt.Sprintf("skills/%s/%s/%s", id, subdir, baseName))
			out.Files[outPath] = string(data)
			continue
		}

		for _, match := range matches {
			data, err := os.ReadFile(match)
			if err != nil {
				return fmt.Errorf("%s file %q: %w", subdir, match, err)
			}
			baseName := filepath.Base(match)
			outPath := filepath.Clean(fmt.Sprintf("skills/%s/%s/%s", id, subdir, baseName))
			out.Files[outPath] = string(data)
		}
	}
	return nil
}

// compileRuleMarkdown renders a single RuleConfig to Claude Code markdown.
// It returns the rendered content, zero or more fidelity notes, and any error.
//
// Activation handling:
//   - path-glob: emits paths: frontmatter from rule.Paths.
//   - always:    no paths: frontmatter.
//   - model-decided, manual-mention, explicit-invoke: Claude has no native
//     equivalent; rule is emitted as always-loaded with a warning FidelityNote.
//
// exclude-agents: Claude has no native equivalent; the field is silently dropped
// and an info FidelityNote is emitted.
func compileRuleMarkdown(id string, rule ast.RuleConfig, baseDir string) (string, []renderer.FidelityNote, error) {
	if strings.TrimSpace(id) == "" {
		return "", nil, fmt.Errorf("rule id must not be empty")
	}

	body, err := resolver.ResolveInstructions(
		rule.Instructions,
		rule.InstructionsFile,
		fmt.Sprintf("rules/%s.md", id), // convention: rules/<id>.md
		baseDir,
	)
	if err != nil {
		return "", nil, err
	}

	activation := renderer.ResolvedActivation(rule)

	var notes []renderer.FidelityNote

	// Claude natively supports always and path-glob. Everything else is
	// emitted as always-loaded with a warning note.
	switch activation {
	case ast.RuleActivationAlways, ast.RuleActivationPathGlob:
		// supported — no note needed
	default:
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
	if rule.Description != "" {
		fmt.Fprintf(&sb, "description: %s\n", rule.Description)
	}
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
