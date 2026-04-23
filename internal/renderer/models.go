package renderer

import (
	"fmt"
	"strings"
)

// modelAliases translates a user-friendly alias from an .xcf file into
// a provider-specific actual model identifier.
// If a target doesn't support model definition per agent, it will return an empty lookup.
var modelAliases = map[string]map[string]string{
	"sonnet-4": {
		"claude":  "claude-sonnet-4-5",
		"gemini":  "gemini-2.5-flash",
		"copilot": "claude-sonnet-4-6",
		"cursor":  "claude-sonnet-4-5",
	},
	"opus-4": {
		"claude":  "claude-opus-4-7",
		"gemini":  "gemini-2.5-pro",
		"copilot": "claude-opus-4-7",
		"cursor":  "claude-sonnet-4-5",
	},
	"haiku-3.5": {
		"claude":  "claude-haiku-4-5",
		"gemini":  "gemini-2.5-flash",
		"copilot": "claude-haiku-4-5",
		"cursor":  "claude-sonnet-4-5",
	},
}

var knownClaudeAliases = map[string]bool{
	"sonnet": true,
	"opus":   true,
	"haiku":  true,
}

// IsKnownClaudeAlias returns true if the literal string is a naked tier name
// typical of raw Claude Code usage.
func IsKnownClaudeAlias(alias string) bool {
	return knownClaudeAliases[strings.ToLower(alias)]
}

// ResolveModel takes an alias from the Xcaffold configuration and a target name (e.g. "claude", "cursor").
// It returns the target-specific model string and a boolean indicating if the target expects one.
// If the target doesn't support models (like antigravity), it returns ("", false).
func ResolveModel(alias, target string) (string, bool) {
	if target == "antigravity" {
		// Antigravity does not compile agents/skills that include the model field
		return "", false
	}

	// If it's a known alias, return the translation
	if mappings, ok := modelAliases[alias]; ok {
		if val, exists := mappings[target]; exists {
			return val, true
		}
	}

	// If it wasn't an alias, assume the user provided a full literal string
	return alias, true
}

// IsMappedModel returns true if the input alias was explicitly mapped for the given target.
// This is used by renderers to determine if a model parameter was safely translated
// or passed through as an unverified literal.
func IsMappedModel(alias, target string) bool {
	if mappings, ok := modelAliases[alias]; ok {
		_, exists := mappings[target]
		return exists
	}
	return false
}

// SanitizeAgentModel maps a model alias to a provider-specific literal.
// It returns the sanitized model string and a slice of FidelityNotes.
func SanitizeAgentModel(model string, caps CapabilitySet, targetName, agentID string) (string, []FidelityNote) {
	if model == "" {
		return "", nil // Nothing to do
	}

	var notes []FidelityNote

	// If the provider does not support the model field via configuration, drop it.
	if !caps.ModelField {
		return "", nil
	}

	resolved, ok := ResolveModel(model, targetName)
	if !ok || resolved == "" {
		return "", nil // Target does not expect one based on ResolveModel
	}

	// We kept the field. But if it was NOT properly mapped...
	if !IsMappedModel(model, targetName) {
		// Is it a bare Claude alias?
		if IsKnownClaudeAlias(model) {
			// Bare Claude alias with no explicit version map (e.g. "sonnet" instead of "sonnet-4").
			notes = append(notes, NewNote(
				LevelWarning, targetName, "agent", agentID, "model",
				CodeAgentModelUnmapped,
				fmt.Sprintf("bare alias %q passed through for agent %q unmapped; this may fail on %s", model, agentID, targetName),
				fmt.Sprintf("Use a mapped alias (e.g. sonnet-4) or a native literal for %s", targetName),
			))
			return "", notes
		} else {
			// It's not a known alias, meaning it's a native literal. Pass it through safely.
			notes = append(notes, NewNote(
				LevelInfo, targetName, "agent", agentID, "model",
				CodeFieldTransformed,
				fmt.Sprintf("native literal %q passed through for agent %q", model, agentID),
				"",
			))
		}
	}

	return resolved, notes
}
