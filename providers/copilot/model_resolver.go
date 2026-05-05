package copilot

import "strings"

// copilotModelResolver implements renderer.ModelResolver for GitHub Copilot.
type copilotModelResolver struct{}

// NewModelResolver creates a ModelResolver for GitHub Copilot.
func NewModelResolver() *copilotModelResolver {
	return &copilotModelResolver{}
}

// ResolveAlias maps short aliases to full GitHub Copilot model IDs.
// Copilot supports both Claude models and OpenAI models (gpt-4o, etc).
// Ground truth: models.json verified 2026-04-30.
func (r *copilotModelResolver) ResolveAlias(alias string) (modelID string, ok bool) {
	// Versioned aliases map to Claude models (Copilot's primary integration)
	aliasMap := map[string]string{
		"sonnet-4":  "claude-sonnet-4-6",
		"opus-4":    "claude-opus-4-7",
		"haiku-3.5": "claude-haiku-4-5",
	}

	if id, found := aliasMap[alias]; found {
		return id, true
	}

	// If it starts with "claude-" or "gpt-", it's a full model ID.
	// Pass it through for Copilot to validate at runtime.
	lowerAlias := strings.ToLower(alias)
	if strings.HasPrefix(lowerAlias, "claude-") || strings.HasPrefix(lowerAlias, "gpt-") {
		return alias, true
	}

	return "", false
}

// SupportsBareAliases reports that GitHub Copilot does not accept bare aliases
// in its schema; it requires full model IDs.
func (r *copilotModelResolver) SupportsBareAliases() bool {
	return false
}
