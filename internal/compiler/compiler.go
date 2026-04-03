package compiler

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/saero-ai/xcaffold/internal/ast"
)

// Output holds the in-memory result of a compilation pass.
type Output struct {
	// Files maps a clean, relative output path to its rendered content.
	// Keys are guaranteed to be cleaned with filepath.Clean before insertion.
	Files map[string]string
}

// Compile translates an XcaffoldConfig AST into its Claude Code output
// representation. It returns an error if any agent fails to compile.
// Compile never panics.
func Compile(config *ast.XcaffoldConfig) (*Output, error) {
	out := &Output{
		Files: make(map[string]string),
	}

	// Compile all agent personas to .claude/agents/*.md
	for id, agent := range config.Agents {
		md, err := compileAgentMarkdown(id, agent)
		if err != nil {
			return nil, fmt.Errorf("failed to compile agent %q: %w", id, err)
		}
		safePath := filepath.Clean(fmt.Sprintf("agents/%s.md", id))
		out.Files[safePath] = md
	}

	// Compile all skills to .claude/skills/*.md
	for id, skill := range config.Skills {
		md, err := compileSkillMarkdown(id, skill)
		if err != nil {
			return nil, fmt.Errorf("failed to compile skill %q: %w", id, err)
		}
		safePath := filepath.Clean(fmt.Sprintf("skills/%s.md", id))
		out.Files[safePath] = md
	}

	// Compile all rules to .claude/rules/*.md
	for id, rule := range config.Rules {
		md, err := compileRuleMarkdown(id, rule)
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

	// MCP / Settings (if we have MCP configs we merge them to settings.json)
	if len(config.MCP) > 0 {
		settingsJSON, err := compileSettingsJSON(config.MCP)
		if err != nil {
			return nil, fmt.Errorf("failed to compile settings: %w", err)
		}
		out.Files["settings.json"] = settingsJSON
	}

	return out, nil
}

// compileAgentMarkdown renders a single AgentConfig to Claude Code markdown.
func compileAgentMarkdown(id string, agent ast.AgentConfig) (string, error) {
	if strings.TrimSpace(id) == "" {
		return "", fmt.Errorf("agent id must not be empty")
	}

	var sb strings.Builder

	// --- Frontmatter ---
	sb.WriteString("---\n")

	if agent.Description != "" {
		fmt.Fprintf(&sb, "description: %s\n", agent.Description)
	}
	if agent.Model != "" {
		fmt.Fprintf(&sb, "model: %s\n", agent.Model)
	}
	if agent.Effort != "" {
		fmt.Fprintf(&sb, "model_setting_effort: %s\n", agent.Effort)
	}
	if len(agent.Tools) > 0 {
		fmt.Fprintf(&sb, "tools: [%s]\n", strings.Join(agent.Tools, ", "))
	}
	if len(agent.BlockedTools) > 0 {
		fmt.Fprintf(&sb, "tools_blocked: [%s]\n", strings.Join(agent.BlockedTools, ", "))
	}

	sb.WriteString("---\n")

	// --- Body (Instructions) ---
	if agent.Instructions != "" {
		sb.WriteString("\n")
		sb.WriteString(strings.TrimRight(agent.Instructions, "\n"))
		sb.WriteString("\n")
	}

	return sb.String(), nil
}

func compileSkillMarkdown(id string, skill ast.SkillConfig) (string, error) {
	if strings.TrimSpace(id) == "" {
		return "", fmt.Errorf("skill id must not be empty")
	}

	var sb strings.Builder

	sb.WriteString("---\n")
	if skill.Description != "" {
		fmt.Fprintf(&sb, "description: %s\n", skill.Description)
	}
	if len(skill.Tools) > 0 {
		fmt.Fprintf(&sb, "tools: [%s]\n", strings.Join(skill.Tools, ", "))
	}
	if len(skill.Paths) > 0 {
		fmt.Fprintf(&sb, "paths: [%s]\n", strings.Join(skill.Paths, ", "))
	}
	sb.WriteString("---\n")

	if skill.Instructions != "" {
		sb.WriteString("\n")
		sb.WriteString(strings.TrimRight(skill.Instructions, "\n"))
		sb.WriteString("\n")
	}

	return sb.String(), nil
}

func compileRuleMarkdown(id string, rule ast.RuleConfig) (string, error) {
	if strings.TrimSpace(id) == "" {
		return "", fmt.Errorf("rule id must not be empty")
	}

	var sb strings.Builder

	sb.WriteString("---\n")
	if len(rule.Paths) > 0 {
		fmt.Fprintf(&sb, "paths: [%s]\n", strings.Join(rule.Paths, ", "))
	}
	sb.WriteString("---\n")

	if rule.Instructions != "" {
		sb.WriteString("\n")
		sb.WriteString(strings.TrimRight(rule.Instructions, "\n"))
		sb.WriteString("\n")
	}

	return sb.String(), nil
}

func compileHooksJSON(hooks map[string]ast.HookConfig) (string, error) {
	b, err := json.MarshalIndent(hooks, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func compileSettingsJSON(mcpConfigs map[string]ast.MCPConfig) (string, error) {
	settings := map[string]any{
		"mcp": mcpConfigs,
	}
	b, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}
