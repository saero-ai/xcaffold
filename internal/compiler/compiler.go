package compiler

import (
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
		Files: make(map[string]string, len(config.Agents)),
	}

	// Compile all agent personas to .claude/agents/*.md
	for id, agent := range config.Agents {
		md, err := compileAgentMarkdown(id, agent)
		if err != nil {
			return nil, fmt.Errorf("failed to compile agent %q: %w", id, err)
		}
		// Use filepath.Clean to prevent any path traversal via agent IDs.
		safePath := filepath.Clean(fmt.Sprintf("agents/%s.md", id))
		out.Files[safePath] = md
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
