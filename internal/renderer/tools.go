package renderer

import (
	"fmt"
	"strings"
)

// claudeNativeTools contains the exact string literals used by Claude Code
// for its built-in toolkit.
var claudeNativeTools = map[string]bool{
	"Read": true, "Write": true, "Edit": true, "MultiEdit": true,
	"Bash": true, "Glob": true, "Grep": true,
	"LS": true, "TodoRead": true, "TodoWrite": true,
	"WebSearch": true, "WebFetch": true,
	"NotebookRead": true, "NotebookEdit": true,
	"Task": true, "exit_plan_mode": true,
}

// init ensures all map keys are explicitly known to prevent accidental regressions.
func init() {
	if len(claudeNativeTools) != 16 {
		panic("claudeNativeTools length mismatch - update implementation")
	}
}

// containsClaudeNativeTools checks if a tool slice contains any Claude-native tools.
func containsClaudeNativeTools(tools []string) bool {
	for _, t := range tools {
		if isClaudeNativeTool(t) {
			return true
		}
	}
	return false
}

// isClaudeNativeTool returns true if the tool name exactly matches a known Claude-native tool.
// This is case-sensitive as tool names must be exact to match provider schemas.
func isClaudeNativeTool(name string) bool {
	return claudeNativeTools[name]
}

// SanitizeAgentTools filters an agent's tool list according to the provider's CapabilitySet.
// Returns the sanitized tool slice and a slice of FidelityNotes detailing dropped tool warnings.
func SanitizeAgentTools(tools []string, caps CapabilitySet, targetName, agentID string) ([]string, []FidelityNote) {
	if len(tools) == 0 {
		return nil, nil // Nothing to do
	}

	var notes []FidelityNote

	// If the provider does not support the tools field at all, silently drop.
	// This matches the baseline design logic.
	if !caps.AgentToolsField {
		return nil, nil
	}

	// For Claude (AgentNativeToolsOnly = true), all tools are passed through as-is.
	if caps.AgentNativeToolsOnly {
		return append([]string{}, tools...), nil
	}

	// For other providers (AgentToolsField: true, AgentNativeToolsOnly: false),
	// Claude-native tools are unsupported and must be stripped.
	var sanitized []string
	var dropped []string

	for _, t := range tools {
		// MCP tools (mcp_*) are allowed
		if strings.HasPrefix(t, "mcp_") {
			sanitized = append(sanitized, t)
			continue
		}

		// Explicit provider-native specific tools (e.g., custom integrations)
		// we keep, but Claude native tools we drop
		if isClaudeNativeTool(t) {
			dropped = append(dropped, t)
		} else {
			sanitized = append(sanitized, t)
		}
	}

	if len(dropped) > 0 {
		notes = append(notes, NewNote(
			LevelWarning, targetName, "agent", agentID, "tools",
			CodeAgentToolsDropped,
			fmt.Sprintf("agent %q drops Claude-native tools %v; only MCP/custom tools supported on %s", agentID, dropped, targetName),
			fmt.Sprintf("Remove Claude tools from %s tools list or accept dropped capability", targetName),
		))
	}

	return sanitized, notes
}
