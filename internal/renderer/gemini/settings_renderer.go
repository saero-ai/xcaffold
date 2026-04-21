package gemini

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/renderer"
)

// geminiHookEvent is the shape of a single hook entry in Gemini's settings.json hooks section.
type geminiHookEvent struct {
	Matcher string            `json:"matcher,omitempty"`
	Hooks   []geminiHookEntry `json:"hooks"`
}

// geminiHookEntry is a single Gemini hook handler (only "command" type is supported).
type geminiHookEntry struct {
	Name    string `json:"name,omitempty"`
	Type    string `json:"type"`
	Command string `json:"command"`
	Timeout int    `json:"timeout,omitempty"`
}

// geminiMCPEntry is the shape of a single MCP server entry in Gemini's settings.json.
type geminiMCPEntry struct {
	Env     map[string]string `json:"env,omitempty"`
	Args    []string          `json:"args,omitempty"`
	Command string            `json:"command,omitempty"`
	URL     string            `json:"url,omitempty"`
}

// xcaffoldToGeminiEvent maps xcaffold hook event names to their Gemini equivalents.
// Events not in this map pass through unchanged.
var xcaffoldToGeminiEvent = map[string]string{
	"PreToolExecution":  "BeforeTool",
	"PostToolExecution": "AfterTool",
}

// geminiNativeEvents is the set of Gemini-native event names that pass through unchanged.
var geminiNativeEvents = map[string]bool{
	"SessionStart":        true,
	"SessionEnd":          true,
	"BeforeAgent":         true,
	"AfterAgent":          true,
	"BeforeModel":         true,
	"AfterModel":          true,
	"BeforeToolSelection": true,
	"PreCompress":         true,
	"Notification":        true,
	"BeforeTool":          true,
	"AfterTool":           true,
}

// compileGeminiSettings produces a .gemini/settings.json JSON string from hook
// and MCP configuration. Returns an empty string when there is nothing to emit
// (no hooks and no MCP servers). Fidelity notes are returned for Claude-specific
// SettingsConfig fields that Gemini does not support.
func compileGeminiSettings(hooks ast.HookConfig, mcp map[string]ast.MCPConfig, settings ast.SettingsConfig) (string, []renderer.FidelityNote, error) {
	var notes []renderer.FidelityNote

	// Emit fidelity notes for Claude-specific settings fields.
	notes = append(notes, detectUnsupportedSettingsFields(settings)...)

	out := map[string]any{}

	if len(hooks) > 0 {
		hooksSection, hookNotes := compileGeminiHooks(hooks)
		notes = append(notes, hookNotes...)
		if len(hooksSection) > 0 {
			out["hooks"] = hooksSection
		}
	}

	if len(mcp) > 0 {
		mcpSection := compileGeminiMCP(mcp)
		if len(mcpSection) > 0 {
			out["mcpServers"] = mcpSection
		}
	}

	if len(out) == 0 {
		return "", notes, nil
	}

	b, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return "", notes, fmt.Errorf("marshal gemini settings: %w", err)
	}

	return string(b), notes, nil
}

// compileGeminiHooks translates xcaffold HookConfig into Gemini's hooks section.
// Each xcaffold event name is mapped to its Gemini equivalent. Events that have
// no Gemini equivalent are skipped and a CodeFieldUnsupported fidelity note is
// returned for each. Only "command" type handlers are supported in Gemini.
func compileGeminiHooks(hooks ast.HookConfig) (map[string][]geminiHookEvent, []renderer.FidelityNote) {
	section := map[string][]geminiHookEvent{}
	var notes []renderer.FidelityNote

	for _, eventName := range renderer.SortedKeys(hooks) {
		groups := hooks[eventName]
		geminiEvent, ok := mapEventName(eventName)
		if !ok {
			notes = append(notes, renderer.NewNote(
				renderer.LevelWarning, targetName, "hooks", "hooks", eventName,
				renderer.CodeFieldUnsupported,
				fmt.Sprintf("hook event %q has no Gemini CLI equivalent and was dropped", eventName),
				"Remove this hook or replace it with a Gemini-native event (BeforeTool, AfterTool, SessionStart, etc.)",
			))
			continue
		}

		var entries []geminiHookEvent
		for _, group := range groups {
			var handlerEntries []geminiHookEntry
			for _, h := range group.Hooks {
				entry := geminiHookEntry{
					Type:    "command",
					Command: h.Command,
				}
				if h.Timeout != nil {
					entry.Timeout = *h.Timeout
				}
				handlerEntries = append(handlerEntries, entry)
			}
			if len(handlerEntries) > 0 {
				entries = append(entries, geminiHookEvent{
					Matcher: group.Matcher,
					Hooks:   handlerEntries,
				})
			}
		}

		if len(entries) > 0 {
			section[geminiEvent] = append(section[geminiEvent], entries...)
		}
	}

	return section, notes
}

// compileGeminiMCP converts xcaffold MCP config into Gemini's mcpServers section.
func compileGeminiMCP(mcp map[string]ast.MCPConfig) map[string]geminiMCPEntry {
	section := make(map[string]geminiMCPEntry, len(mcp))

	for _, id := range sortedStringKeys(mcp) {
		srv := mcp[id]
		entry := geminiMCPEntry{
			Command: srv.Command,
			URL:     srv.URL,
		}
		if len(srv.Args) > 0 {
			entry.Args = srv.Args
		}
		if len(srv.Env) > 0 {
			entry.Env = srv.Env
		}
		section[id] = entry
	}

	return section
}

// detectUnsupportedSettingsFields emits fidelity notes for SettingsConfig fields
// that are Claude-specific and have no Gemini equivalent.
func detectUnsupportedSettingsFields(settings ast.SettingsConfig) []renderer.FidelityNote {
	var notes []renderer.FidelityNote

	if settings.Permissions != nil {
		notes = append(notes, renderer.NewNote(
			renderer.LevelWarning, targetName, "settings", "settings", "permissions",
			renderer.CodeSettingsFieldUnsupported,
			"settings.permissions has no Gemini CLI equivalent and was dropped",
			"Remove permissions or enforce access control via hooks",
		))
	}
	if settings.Sandbox != nil {
		notes = append(notes, renderer.NewNote(
			renderer.LevelWarning, targetName, "settings", "settings", "sandbox",
			renderer.CodeSettingsFieldUnsupported,
			"settings.sandbox has no Gemini CLI equivalent and was dropped",
			"Remove sandbox configuration for Gemini targets",
		))
	}
	if settings.StatusLine != nil {
		notes = append(notes, renderer.NewNote(
			renderer.LevelWarning, targetName, "settings", "settings", "statusLine",
			renderer.CodeSettingsFieldUnsupported,
			"settings.statusLine has no Gemini CLI equivalent and was dropped",
			"Remove statusLine or use targets.gemini.provider pass-through",
		))
	}

	return notes
}

// mapEventName translates an xcaffold hook event name to the Gemini equivalent.
// Known mappings: PreToolExecution → BeforeTool, PostToolExecution → AfterTool.
// Native Gemini event names pass through unchanged. Unknown events return ("", false)
// so callers can emit a fidelity note and skip writing the event.
func mapEventName(event string) (string, bool) {
	if gemini, ok := xcaffoldToGeminiEvent[event]; ok {
		return gemini, true
	}
	if geminiNativeEvents[event] {
		return event, true
	}
	return "", false
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
