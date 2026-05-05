package renderer

import (
	"fmt"
	"strings"

	"github.com/saero-ai/xcaffold/pkg/schema"
)

// nativeTools contains the xcaffold-standard tool names that correspond to
// Claude Code's built-in toolkit. These are stripped when compiling for
// providers that do not support them natively.
var nativeTools = map[string]bool{
	"Read": true, "Write": true, "Edit": true, "MultiEdit": true,
	"Bash": true, "Glob": true, "Grep": true,
	"LS": true, "TodoRead": true, "TodoWrite": true,
	"WebSearch": true, "WebFetch": true,
	"NotebookRead": true, "NotebookEdit": true,
	"Task": true, "exit_plan_mode": true,
}

// ContainsNativeTools reports whether a tool slice contains any native tools.
func ContainsNativeTools(tools []string) bool {
	for _, t := range tools {
		if IsNativeTool(t) {
			return true
		}
	}
	return false
}

// IsNativeTool returns true if the tool name exactly matches a known native tool.
// This is case-sensitive as tool names must be exact to match provider schemas.
func IsNativeTool(name string) bool {
	return nativeTools[name]
}

// SanitizeAgentTools filters an agent's tool list according to the provider's CapabilitySet.
// Native tools unsupported by the target are dropped and reported as FidelityNotes.
// Returns the sanitized tool slice and a slice of FidelityNotes detailing dropped tool warnings.
func SanitizeAgentTools(tools []string, caps CapabilitySet, targetName, agentID string) ([]string, []FidelityNote) {
	if len(tools) == 0 {
		return nil, nil // Nothing to do
	}

	var notes []FidelityNote

	// Consult the schema registry to determine whether this provider supports
	// the agent tools field. "unsupported" or a missing entry means silently drop.
	toolsSupport := schema.FieldSupportForTarget("agent", "tools", targetName)
	if toolsSupport == "unsupported" || toolsSupport == "" {
		return nil, nil
	}

	// For Claude (AgentNativeToolsOnly = true), all tools are passed through as-is.
	if caps.AgentNativeToolsOnly {
		return append([]string{}, tools...), nil
	}

	// For other providers (AgentNativeToolsOnly: false),
	// native tools are unsupported and must be stripped.
	var sanitized []string
	var dropped []string

	for _, t := range tools {
		// MCP tools (mcp_*) are allowed
		if strings.HasPrefix(t, "mcp_") {
			sanitized = append(sanitized, t)
			continue
		}

		// Explicit provider-native specific tools (e.g., custom integrations)
		// we keep, but native tools unsupported on the target we drop
		if IsNativeTool(t) {
			dropped = append(dropped, t)
		} else {
			sanitized = append(sanitized, t)
		}
	}

	if len(dropped) > 0 {
		notes = append(notes, NewNote(
			LevelWarning, targetName, "agent", agentID, "tools",
			CodeAgentToolsDropped,
			fmt.Sprintf("agent %q drops native tools %v; only MCP/custom tools supported on %s", agentID, dropped, targetName),
			fmt.Sprintf("Remove native tools from %s tools list or accept dropped capability", targetName),
		))
	}

	return sanitized, notes
}
