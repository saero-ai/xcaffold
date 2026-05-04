package copilot

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/renderer"
)

// xcaffoldToCopilotEvent maps xcaffold hook event names to their Copilot equivalents.
// Events not in this map are unsupported and emit a fidelity note.
var xcaffoldToCopilotEvent = map[string]string{
	"PreToolUse":       "preToolUse",
	"PostToolUse":      "postToolUse",
	"SessionStart":     "sessionStart",
	"SessionEnd":       "sessionEnd",
	"UserPromptSubmit": "userPromptSubmitted",
	"Stop":             "agentStop",
	"SubagentStop":     "subagentStop",
	// Aliases for Gemini-style event names
	"PreToolExecution":  "preToolUse",
	"PostToolExecution": "postToolUse",
}

// copilotHookEntry is the shape of a single hook entry in Copilot's hooks JSON.
type copilotHookEntry struct {
	Type       string            `json:"type"`
	Bash       string            `json:"bash"`
	Env        map[string]string `json:"env,omitempty"`
	TimeoutSec int               `json:"timeoutSec,omitempty"`
}

// compileCopilotSettings produces separate files for hooks and MCP configuration.
// Hooks go to hooks/xcaffold-hooks.json (relative to OutputDir ".github").
// MCP requires manual placement at .vscode/mcp.json (outside .github/); a
// FidelityNote is emitted explaining the required manual step.
// Returns the files map, fidelity notes, and any marshaling error.
func compileCopilotSettings(hooks ast.HookConfig, mcp map[string]ast.MCPConfig, settings *ast.SettingsConfig) (map[string]string, []renderer.FidelityNote, error) {
	files := make(map[string]string)
	var notes []renderer.FidelityNote

	if settings != nil {
		notes = append(notes, detectUnsupportedCopilotSettings(settings)...)
	}

	if len(hooks) > 0 {
		hooksJSON, hookNotes := compileCopilotHooks(hooks)
		notes = append(notes, hookNotes...)
		if hooksJSON != "" {
			files["hooks/xcaffold-hooks.json"] = hooksJSON
		}
	}

	if len(mcp) > 0 {
		mcpJSON, mcpNotes, err := compileCopilotMCP(mcp)
		notes = append(notes, mcpNotes...)
		if err != nil {
			return nil, notes, err
		}
		if mcpJSON != "" {
			// .vscode/mcp.json is at the project root, not inside .github/.
			// The orchestrator writes file map keys relative to the project root.
			files[".vscode/mcp.json"] = mcpJSON
		}
	}

	return files, notes, nil
}

// compileCopilotHooks translates xcaffold HookConfig into Copilot's hooks JSON
// at .github/hooks/xcaffold-hooks.json. The schema is:
//
//	{"version": 1, "hooks": {"<event>": [<entry>, ...]}}
//
// Each handler becomes a {type, bash, env, timeoutSec} entry. The xcaffold
// timeout field is in milliseconds; Copilot expects seconds (integer division).
// Unknown events emit CodeFieldUnsupported and are skipped.
func compileCopilotHooks(hookConfig ast.HookConfig) (string, []renderer.FidelityNote) {
	hooksSection := make(map[string][]copilotHookEntry)
	var notes []renderer.FidelityNote

	for _, eventName := range renderer.SortedKeys(hookConfig) {
		groups := hookConfig[eventName]
		copilotEvent, ok := mapCopilotEvent(eventName)
		if !ok {
			notes = append(notes, renderer.NewNote(
				renderer.LevelWarning, targetName, "hooks", "hooks", eventName,
				renderer.CodeFieldUnsupported,
				fmt.Sprintf("hook event %q has no Copilot equivalent and was dropped", eventName),
				"Remove this hook or replace it with a supported event (PreToolUse, PostToolUse, SessionStart, etc.)",
			))
			continue
		}

		for _, group := range groups {
			for _, h := range group.Hooks {
				entry := copilotHookEntry{
					Type: "command",
					Bash: renderer.TranslateHookCommand(h.Command, "$GITHUB_WORKSPACE", ".github/hooks/scripts/"),
				}
				if h.Timeout != nil && *h.Timeout > 0 {
					entry.TimeoutSec = *h.Timeout / 1000
				}
				hooksSection[copilotEvent] = append(hooksSection[copilotEvent], entry)
			}
		}
	}

	if len(hooksSection) == 0 {
		return "", notes
	}

	out := map[string]any{
		"version": 1,
		"hooks":   hooksSection,
	}

	b, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		// json.MarshalIndent cannot fail for this structure; return empty on error.
		return "", notes
	}

	return string(b), notes
}

// compileCopilotMCP produces the content for .vscode/mcp.json and emits
// a fidelity note describing where xcaffold wrote it.
// VS Code MCP format uses root key "servers" (not "mcpServers").
func compileCopilotMCP(mcpServers map[string]ast.MCPConfig) (string, []renderer.FidelityNote, error) {
	var notes []renderer.FidelityNote

	if len(mcpServers) == 0 {
		return "", notes, nil
	}

	type vscodeEntry struct {
		Command string            `json:"command,omitempty"`
		Args    []string          `json:"args,omitempty"`
		Env     map[string]string `json:"env,omitempty"`
		URL     string            `json:"url,omitempty"`
		Type    string            `json:"type,omitempty"`
	}

	servers := make(map[string]vscodeEntry)
	for _, id := range sortedStringKeys(mcpServers) {
		srv := mcpServers[id]
		if srv.Command == "" && srv.URL == "" {
			notes = append(notes, renderer.NewNote(
				renderer.LevelWarning, targetName, "mcp", id, "command",
				renderer.CodeFieldUnsupported,
				fmt.Sprintf("MCP server %q has neither command nor url and was omitted from .vscode/mcp.json", id),
				"Set command or url on the MCP server config",
			))
			continue
		}
		servers[id] = vscodeEntry{
			Command: srv.Command,
			Args:    srv.Args,
			Env:     srv.Env,
			URL:     srv.URL,
			Type:    srv.Type,
		}
	}

	if len(servers) > 0 {
		notes = append(notes, renderer.NewNote(
			renderer.LevelWarning, targetName, "mcp", "mcp", "activation",
			renderer.CodeMCPGlobalConfigOnly,
			"Copilot requires MCP servers to be configured globally in VS Code settings.json",
			"Manually add the MCP server command to github.copilot.chat.mcp in settings.json",
		))
	}

	out := map[string]any{
		"servers": servers,
	}

	b, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return "", notes, fmt.Errorf("copilot mcp marshal: %w", err)
	}

	return string(b), notes, nil
}

// mapCopilotEvent translates an xcaffold hook event name to its Copilot equivalent.
// Returns (copilotEvent, true) when a mapping exists, ("", false) otherwise.
func mapCopilotEvent(event string) (string, bool) {
	if copilotEvent, ok := xcaffoldToCopilotEvent[event]; ok {
		return copilotEvent, true
	}
	return "", false
}

// detectUnsupportedCopilotSettings emits fidelity notes for SettingsConfig fields
// that are Claude-specific and have no Copilot equivalent.
func detectUnsupportedCopilotSettings(settings *ast.SettingsConfig) []renderer.FidelityNote {
	var notes []renderer.FidelityNote

	if settings.Permissions != nil {
		notes = append(notes, renderer.NewNote(
			renderer.LevelWarning, targetName, "settings", "settings", "permissions",
			renderer.CodeSettingsFieldUnsupported,
			"settings.permissions has no Copilot equivalent and was dropped",
			"Remove permissions or enforce access control via hooks",
		))
	}
	if settings.Sandbox != nil {
		notes = append(notes, renderer.NewNote(
			renderer.LevelWarning, targetName, "settings", "settings", "sandbox",
			renderer.CodeSettingsFieldUnsupported,
			"settings.sandbox has no Copilot equivalent and was dropped",
			"Remove sandbox configuration for Copilot targets",
		))
	}
	if settings.StatusLine != nil {
		notes = append(notes, renderer.NewNote(
			renderer.LevelWarning, targetName, "settings", "settings", "statusLine",
			renderer.CodeSettingsFieldUnsupported,
			"settings.statusLine has no Copilot equivalent and was dropped",
			"Remove statusLine or use targets.copilot.provider pass-through",
		))
	}

	return notes
}

// sortedStringKeys returns a sorted slice of keys from map[string]T.
func sortedStringKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
