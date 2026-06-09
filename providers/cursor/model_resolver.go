package cursor

import "strings"

// cursorModelResolver implements renderer.ModelResolver for Cursor.
type cursorModelResolver struct{}

// NewModelResolver creates a ModelResolver for Cursor.
func NewModelResolver() *cursorModelResolver {
	return &cursorModelResolver{}
}

// ResolveAlias maps short aliases to full Cursor model IDs.
// Also passes through native model slugs from known prefix families.
// Ground truth: models.json verified 2026-06-09.
func (r *cursorModelResolver) ResolveAlias(alias string) (modelID string, ok bool) {
	aliasMap := map[string]string{
		"balanced": "claude-sonnet-4-6",
		"flagship": "gpt-5.5",
		"fast":     "composer-2.5",
	}

	if id, found := aliasMap[alias]; found {
		return id, true
	}

	lowered := strings.ToLower(alias)
	for _, prefix := range []string{"claude-", "gpt-", "gemini-", "cursor-", "composer-", "o1-", "o3-", "grok-", "kimi-"} {
		if strings.HasPrefix(lowered, prefix) {
			return lowered, true
		}
	}

	return "", false
}
