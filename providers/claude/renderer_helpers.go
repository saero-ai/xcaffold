package claude

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/output"
	"github.com/saero-ai/xcaffold/internal/renderer"
	"gopkg.in/yaml.v3"
)

// compileSkillArtifacts iterates ctx.Skill.Artifacts and dispatches each artifact
// to the correct output subdirectory using the renderer's SkillArtifactDirs map.
// An empty outputSubdir means the files are flattened to the skill root.
// Files are discovered automatically from the artifact subdirectory on disk.
func compileSkillArtifacts(ctx renderer.SkillArtifactContext, out *output.Output) error {
	for _, artifactName := range ctx.Skill.Artifacts {
		outputSubdir, ok := ctx.Caps.SkillArtifactDirs[artifactName]
		if !ok {
			outputSubdir = artifactName // default: use same name as canonical
		}
		paths, err := renderer.DiscoverArtifactFiles(ctx.BaseDir, ctx.SkillSourceDir, artifactName)
		if err != nil {
			return fmt.Errorf("skill %s artifact %s: discover files: %w", ctx.ID, artifactName, err)
		}
		if len(paths) == 0 {
			continue
		}
		if outputSubdir == "" {
			// FlattenToSkillRoot resolves paths from baseDir, so prefix with skillSourceDir.
			prefixed := make([]string, len(paths))
			for i, p := range paths {
				prefixed[i] = filepath.Join(ctx.SkillSourceDir, p)
			}
			if err := renderer.FlattenToSkillRoot(renderer.FlattenOpts{
				ID:            ctx.ID,
				CanonicalName: artifactName,
				Paths:         prefixed,
				BaseDir:       ctx.BaseDir,
			}, out); err != nil {
				return fmt.Errorf("skill %s artifact %s: %w", ctx.ID, artifactName, err)
			}
		} else {
			if err := renderer.CompileSkillSubdir(renderer.SkillSubdirOpts{
				ID:              ctx.ID,
				CanonicalSubdir: artifactName,
				OutputSubdir:    outputSubdir,
				Paths:           paths,
				BaseDir:         ctx.BaseDir,
				SkillSourceDir:  ctx.SkillSourceDir,
			}, out); err != nil {
				return fmt.Errorf("skill %s artifact %s: %w", ctx.ID, artifactName, err)
			}
		}
	}
	return nil
}

// appendAgentCoreMeta writes core identity fields to the agent frontmatter builder.
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
	if agent.MaxTurns != nil && *agent.MaxTurns > 0 {
		fmt.Fprintf(sb, "max-turns: %d\n", *agent.MaxTurns)
	}
}

// appendAgentConfigMeta writes permission and runtime config fields to the agent frontmatter builder.
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

// appendAgentYAMLMeta writes YAML-encoded hooks and MCP servers to the agent frontmatter builder.
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

// appendSkillMeta writes all skill frontmatter fields to the builder.
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
	if len(skill.AllowedTools.Values) > 0 {
		fmt.Fprintf(sb, "allowed-tools: %s\n", strings.Join(skill.AllowedTools.Values, " "))
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

// compileSettingsJSON produces a fully-populated settings.json.
// Note: mcpServers are now emitted to mcp.json.
//
// Merge rules:
//   - Output is suppressed (empty string) when the resulting object has no
//     meaningful content, to avoid writing a useless "{}".
//   - The $schema key is always emitted first when there is content.
func compileSettingsJSON(settings ast.SettingsConfig, hooks ast.HookConfig) (string, error) {
	out := map[string]any{
		"$schema": "https://json.schemastore.org/claude-code-settings.json",
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
	if len(settings.MdExcludes) > 0 {
		out["mdExcludes"] = settings.MdExcludes
	}
}
