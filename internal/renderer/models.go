package renderer

// modelAliases translates a user-friendly alias from an .xcf file into
// a provider-specific actual model identifier.
// If a target doesn't support model definition per agent, it will return an empty lookup.
var modelAliases = map[string]map[string]string{
	"sonnet-4":  {"claude": "claude-3-7-sonnet-20250219"},
	"opus-4":    {"claude": "claude-3-5-sonnet-20241022"}, // Placeholder till opus 4 comes out
	"haiku-3.5": {"claude": "claude-3-5-haiku-20241022"},
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
