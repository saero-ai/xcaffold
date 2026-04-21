package renderer

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
