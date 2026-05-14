package cursor

// cursorModelResolver implements renderer.ModelResolver for Cursor.
type cursorModelResolver struct{}

// NewModelResolver creates a ModelResolver for Cursor.
func NewModelResolver() *cursorModelResolver {
	return &cursorModelResolver{}
}

// ResolveAlias maps short aliases to full Cursor model IDs.
// Cursor does not expose a model selection field for agents in its schema,
// so only explicitly mapped versioned aliases are accepted.
// Ground truth: models.json verified 2026-04-30.
func (r *cursorModelResolver) ResolveAlias(alias string) (modelID string, ok bool) {
	// Tier aliases that map to Cursor's typical default
	aliasMap := map[string]string{
		"balanced": "claude-sonnet-4-5",
		"flagship": "claude-sonnet-4-5",
		"fast":     "claude-sonnet-4-5",
	}

	if id, found := aliasMap[alias]; found {
		return id, true
	}

	// Cursor does not accept bare aliases or literal model IDs.
	// Only the mapped versioned aliases above are recognized.
	return "", false
}
