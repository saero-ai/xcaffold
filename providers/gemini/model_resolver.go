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
	// Versioned aliases
	aliasMap := map[string]string{
		"sonnet-4":  "gemini-2.5-flash",
		"opus-4":    "gemini-2.5-pro",
		"haiku-3.5": "gemini-2.5-flash",
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

// SupportsBareAliases reports that Gemini CLI does not accept bare aliases
// like "sonnet" or "opus"; it requires full model IDs.
func (r *geminiModelResolver) SupportsBareAliases() bool {
	return false
}
