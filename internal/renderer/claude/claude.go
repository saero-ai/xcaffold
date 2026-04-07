package claude

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/output"
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
// representation. baseDir is the directory that contains the scaffold.xcf file;
// it is used to resolve instructions_file: and references: paths.
// Compile returns an error if any resource fails to compile. It never panics.
func (r *Renderer) Compile(config *ast.XcaffoldConfig, baseDir string) (*output.Output, error) {
	out := &output.Output{
		Files: make(map[string]string),
	}

	// Compile all agent personas to .claude/agents/*.md
	for id, agent := range config.Agents {
		md, err := compileAgentMarkdown(id, agent, baseDir)
		if err != nil {
			return nil, fmt.Errorf("failed to compile agent %q: %w", id, err)
		}
		safePath := filepath.Clean(fmt.Sprintf("agents/%s.md", id))
		out.Files[safePath] = md
	}

	// Compile all skills to .claude/skills/<id>/SKILL.md
	for id, skill := range config.Skills {
		md, err := compileSkillMarkdown(id, skill, baseDir)
		if err != nil {
			return nil, fmt.Errorf("failed to compile skill %q: %w", id, err)
		}
		safePath := filepath.Clean(fmt.Sprintf("skills/%s/SKILL.md", id))
		out.Files[safePath] = md

		// Copy reference files into skills/<id>/references/
		if err := compileSkillReferences(id, skill, baseDir, out); err != nil {
			return nil, fmt.Errorf("failed to compile references for skill %q: %w", id, err)
		}
	}

	// Compile all rules to .claude/rules/*.md
	for id, rule := range config.Rules {
		md, err := compileRuleMarkdown(id, rule, baseDir)
		if err != nil {
			return nil, fmt.Errorf("failed to compile rule %q: %w", id, err)
		}
		safePath := filepath.Clean(fmt.Sprintf("rules/%s.md", id))
		out.Files[safePath] = md
	}

	// Hooks
	if len(config.Hooks) > 0 {
		hooksJSON, err := compileHooksJSON(config.Hooks)
		if err != nil {
			return nil, fmt.Errorf("failed to compile hooks: %w", err)
		}
		out.Files["hooks.json"] = hooksJSON
	}

	// settings.json: merge top-level mcp: block with the settings: block.
	settingsJSON, err := compileSettingsJSON(config.MCP, config.Settings)
	if err != nil {
		return nil, fmt.Errorf("failed to compile settings: %w", err)
	}
	if settingsJSON != "" {
		out.Files["settings.json"] = settingsJSON
	}

	// settings.local.json: compile the local: block (gitignored settings).
	localJSON, err := compileSettingsJSON(nil, config.Local)
	if err != nil {
		return nil, fmt.Errorf("failed to compile local settings: %w", err)
	}
	if localJSON != "" {
		out.Files["settings.local.json"] = localJSON
	}

	return out, nil
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
func resolveInstructions(inline, filePath, conventionPath, baseDir string) (string, error) {
	if inline != "" {
		return inline, nil
	}
	if filePath != "" {
		// Security: path must not traverse above baseDir.
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
	// Convention-over-configuration: try the standard auto-discovery path.
	// Missing convention file is not an error — resource compiles with empty body.
	if conventionPath != "" && baseDir != "" {
		abs := filepath.Join(baseDir, conventionPath)
		if data, err := os.ReadFile(abs); err == nil {
			return stripFrontmatter(string(data)), nil
		}
	}
	return "", nil
}

// stripFrontmatter removes YAML frontmatter delimited by "---" from the start
// of a markdown file, returning only the body content with leading whitespace trimmed.
func stripFrontmatter(content string) string {
	// Normalise line endings.
	content = strings.ReplaceAll(content, "\r\n", "\n")
	lines := strings.SplitN(content, "\n", -1)

	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return strings.TrimLeft(content, "\n")
	}

	// Find the closing "---"
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			body := strings.Join(lines[i+1:], "\n")
			return strings.TrimLeft(body, "\n")
		}
	}

	// No closing delimiter found — return as-is (no frontmatter detected).
	return strings.TrimLeft(content, "\n")
}

// compileAgentMarkdown renders a single AgentConfig to Claude Code markdown.
func compileAgentMarkdown(id string, agent ast.AgentConfig, baseDir string) (string, error) {
	if strings.TrimSpace(id) == "" {
		return "", fmt.Errorf("agent id must not be empty")
	}

	body, err := resolveInstructions(agent.Instructions, agent.InstructionsFile, fmt.Sprintf("agents/%s.md", id), baseDir)
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
		fmt.Fprintf(sb, "model: %s\n", agent.Model)
	}
	if agent.Effort != "" {
		fmt.Fprintf(sb, "effort: %s\n", agent.Effort)
	}
	if agent.Memory != "" {
		fmt.Fprintf(sb, "memory: %s\n", agent.Memory)
	}
	if agent.MaxTurns > 0 {
		fmt.Fprintf(sb, "maxTurns: %d\n", agent.MaxTurns)
	}
}

func appendAgentToolsMeta(sb *strings.Builder, agent ast.AgentConfig) {
	if agent.Readonly != nil && *agent.Readonly && len(agent.Tools) == 0 {
		sb.WriteString("tools: [Read, Grep, Glob]\n")
	} else if len(agent.Tools) > 0 {
		fmt.Fprintf(sb, "tools: [%s]\n", strings.Join(agent.Tools, ", "))
	}
	if len(agent.DisallowedTools) > 0 {
		fmt.Fprintf(sb, "disallowedTools: [%s]\n", strings.Join(agent.DisallowedTools, ", "))
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
		fmt.Fprintf(sb, "permissionMode: %s\n", agent.PermissionMode)
	}
	if agent.Background != nil {
		fmt.Fprintf(sb, "background: %t\n", *agent.Background)
	}
	if agent.Isolation != "" {
		fmt.Fprintf(sb, "isolation: %s\n", agent.Isolation)
	}
	if agent.Color != "" {
		fmt.Fprintf(sb, "color: %s\n", agent.Color)
	}
	if agent.InitialPrompt != "" {
		fmt.Fprintf(sb, "initialPrompt: %s\n", agent.InitialPrompt)
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

	body, err := resolveInstructions(skill.Instructions, skill.InstructionsFile, fmt.Sprintf("skills/%s/SKILL.md", id), baseDir)
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	sb.WriteString("---\n")

	appendSkillCoreMeta(&sb, skill)
	appendSkillConfigMeta(&sb, skill)
	appendSkillAgentMeta(&sb, skill)

	if len(skill.Hooks) > 0 {
		hooksYAML, err := yaml.Marshal(map[string]ast.HookConfig{"hooks": skill.Hooks})
		if err != nil {
			return "", fmt.Errorf("failed to marshal skill hooks: %w", err)
		}
		sb.WriteString(string(hooksYAML))
	}
	sb.WriteString("---\n")

	if body != "" {
		sb.WriteString("\n")
		sb.WriteString(strings.TrimRight(body, "\n"))
		sb.WriteString("\n")
	}

	return sb.String(), nil
}

func appendSkillCoreMeta(sb *strings.Builder, skill ast.SkillConfig) {
	if skill.Name != "" {
		fmt.Fprintf(sb, "name: %s\n", skill.Name)
	}
	if skill.Type != "" {
		fmt.Fprintf(sb, "type: %s\n", skill.Type)
	}
	if skill.Description != "" {
		fmt.Fprintf(sb, "description: %s\n", skill.Description)
	}
	if len(skill.Tools) > 0 {
		fmt.Fprintf(sb, "tools: [%s]\n", strings.Join(skill.Tools, ", "))
	}
	if len(skill.AllowedTools) > 0 {
		fmt.Fprintf(sb, "allowed-tools: [%s]\n", strings.Join(skill.AllowedTools, ", "))
	}
	if len(skill.Paths) > 0 {
		fmt.Fprintf(sb, "paths: [%s]\n", strings.Join(skill.Paths, ", "))
	}
}

func appendSkillConfigMeta(sb *strings.Builder, skill ast.SkillConfig) {
	if skill.DisableModelInvocation != nil {
		fmt.Fprintf(sb, "disable-model-invocation: %t\n", *skill.DisableModelInvocation)
	}
	if skill.UserInvocable != nil {
		fmt.Fprintf(sb, "user-invocable: %t\n", *skill.UserInvocable)
	}
	if skill.Context != "" {
		fmt.Fprintf(sb, "context: %s\n", skill.Context)
	}
	if skill.Shell != "" {
		fmt.Fprintf(sb, "shell: %s\n", skill.Shell)
	}
	if skill.ArgumentHint != "" {
		fmt.Fprintf(sb, "argument-hint: %s\n", skill.ArgumentHint)
	}
}

func appendSkillAgentMeta(sb *strings.Builder, skill ast.SkillConfig) {
	if skill.Agent != "" {
		fmt.Fprintf(sb, "agent: %s\n", skill.Agent)
	}
	if skill.Model != "" {
		fmt.Fprintf(sb, "model: %s\n", skill.Model)
	}
	if skill.Effort != "" {
		fmt.Fprintf(sb, "effort: %s\n", skill.Effort)
	}
}

// compileSkillReferences copies reference files into the skill's output directory.
// Reference paths are resolved relative to baseDir and placed under skills/<id>/references/.
func compileSkillReferences(id string, skill ast.SkillConfig, baseDir string, out *output.Output) error {
	if len(skill.References) == 0 {
		return nil
	}

	for _, pattern := range skill.References {
		// Security: pattern must not traverse above baseDir.
		cleanedPattern := filepath.Clean(pattern)
		if strings.HasPrefix(cleanedPattern, "..") {
			return fmt.Errorf("references path %q traverses above the project root", pattern)
		}

		absPattern := filepath.Join(baseDir, cleanedPattern)

		// Expand glob patterns (e.g. "skills/my/references/*.md")
		matches, err := filepath.Glob(absPattern)
		if err != nil {
			return fmt.Errorf("invalid glob pattern %q: %w", pattern, err)
		}
		if len(matches) == 0 {
			// Treat as a literal path — if missing, it's an error.
			data, readErr := os.ReadFile(absPattern)
			if readErr != nil {
				return fmt.Errorf("reference file %q: %w", pattern, readErr)
			}
			baseName := filepath.Base(absPattern)
			outPath := filepath.Clean(fmt.Sprintf("skills/%s/references/%s", id, baseName))
			out.Files[outPath] = string(data)
			continue
		}

		for _, match := range matches {
			data, err := os.ReadFile(match)
			if err != nil {
				return fmt.Errorf("reference file %q: %w", match, err)
			}
			baseName := filepath.Base(match)
			outPath := filepath.Clean(fmt.Sprintf("skills/%s/references/%s", id, baseName))
			out.Files[outPath] = string(data)
		}
	}
	return nil
}

// compileRuleMarkdown renders a single RuleConfig to Claude Code markdown.
func compileRuleMarkdown(id string, rule ast.RuleConfig, baseDir string) (string, error) {
	if strings.TrimSpace(id) == "" {
		return "", fmt.Errorf("rule id must not be empty")
	}

	body, err := resolveInstructions(
		rule.Instructions,
		rule.InstructionsFile,
		fmt.Sprintf("rules/%s.md", id), // convention: rules/<id>.md
		baseDir,
	)
	if err != nil {
		return "", err
	}

	var sb strings.Builder

	sb.WriteString("---\n")
	if rule.Description != "" {
		fmt.Fprintf(&sb, "description: %s\n", rule.Description)
	}
	if len(rule.Paths) > 0 {
		fmt.Fprintf(&sb, "paths: [%s]\n", strings.Join(rule.Paths, ", "))
	}
	sb.WriteString("---\n")

	if body != "" {
		sb.WriteString("\n")
		sb.WriteString(strings.TrimRight(body, "\n"))
		sb.WriteString("\n")
	}

	return sb.String(), nil
}

func compileHooksJSON(hooks ast.HookConfig) (string, error) {
	wrapper := map[string]ast.HookConfig{
		"hooks": hooks,
	}
	b, err := json.MarshalIndent(wrapper, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// compileSettingsJSON produces a fully-populated settings.json by merging the
// top-level mcp: convenience block with the settings: block.
//
// Merge rules:
//   - settings.mcpServers takes precedence over mcp: on key conflicts.
//   - Output is suppressed (empty string) when the resulting object has no
//     meaningful content, to avoid writing a useless "{}".
//   - The $schema key is always emitted first when there is content.
func compileSettingsJSON(mcpShorthand map[string]ast.MCPConfig, settings ast.SettingsConfig) (string, error) {
	out := map[string]any{
		"$schema": "https://cdn.jsdelivr.net/npm/@anthropic-ai/claude-code@latest/config-schema.json",
	}

	populateSettingsCore(out, settings)
	populateSettingsFeatures(out, settings)
	populateSettingsDev(out, settings)
	populateSettingsAgent(out, settings)
	populateSettingsSystem(out, settings)

	mcpServers := make(map[string]ast.MCPConfig)
	for k, v := range mcpShorthand {
		mcpServers[k] = v
	}
	for k, v := range settings.MCPServers {
		mcpServers[k] = v
	}
	if len(mcpServers) > 0 {
		out["mcpServers"] = mcpServers
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
		out["model"] = settings.Model
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
