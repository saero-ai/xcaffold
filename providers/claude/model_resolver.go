package claude

import "strings"

// claudeModelResolver implements renderer.ModelResolver for Claude Code.
type claudeModelResolver struct{}

// NewModelResolver creates a ModelResolver for Claude Code.
func NewModelResolver() *claudeModelResolver {
	return &claudeModelResolver{}
}

// ResolveAlias maps short aliases to full Claude model IDs.
// Also recognizes and passes through full model IDs that start with "claude-".
// Ground truth: models.json verified 2026-04-30.
func (r *claudeModelResolver) ResolveAlias(alias string) (modelID string, ok bool) {
	// Tier aliases
	aliasMap := map[string]string{
		"balanced": "claude-sonnet-4-5",
		"flagship": "claude-opus-4-7",
		"fast":     "claude-haiku-4-5",
	}

	if id, found := aliasMap[alias]; found {
		return id, true
	}

	// Bare tier aliases (sonnet, opus, haiku) are supported by Claude Code
	// and resolved at runtime to the current recommended version.
	bare := strings.ToLower(alias)
	switch bare {
	case "sonnet", "opus", "haiku":
		return alias, true
	}

	// If it already starts with "claude-", it's likely a full model ID.
	// Pass it through for Claude Code to validate at runtime.
	if strings.HasPrefix(strings.ToLower(alias), "claude-") {
		return alias, true
	}

	return "", false
}
