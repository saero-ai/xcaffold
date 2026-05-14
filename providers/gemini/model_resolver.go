package gemini

import "strings"

// geminiModelResolver implements renderer.ModelResolver for Gemini CLI.
type geminiModelResolver struct{}

// NewModelResolver creates a ModelResolver for Gemini CLI.
func NewModelResolver() *geminiModelResolver {
	return &geminiModelResolver{}
}

// ResolveAlias maps short aliases to full Gemini model IDs.
// Ground truth: models.json verified 2026-04-30.
func (r *geminiModelResolver) ResolveAlias(alias string) (modelID string, ok bool) {
	// Tier aliases
	aliasMap := map[string]string{
		"balanced": "gemini-2.5-flash",
		"flagship": "gemini-2.5-pro",
		"fast":     "gemini-2.5-flash",
	}

	if id, found := aliasMap[alias]; found {
		return id, true
	}

	// If it already starts with "gemini-", it's likely a full model ID.
	// Pass it through for Gemini CLI to validate at runtime.
	if strings.HasPrefix(strings.ToLower(alias), "gemini-") {
		return alias, true
	}

	return "", false
}
