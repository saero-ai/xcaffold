package antigravity

// antigravityModelResolver implements renderer.ModelResolver for Antigravity.
type antigravityModelResolver struct{}

// NewModelResolver creates a ModelResolver for Antigravity.
func NewModelResolver() *antigravityModelResolver {
	return &antigravityModelResolver{}
}

// ResolveAlias returns false for all aliases because Antigravity does not
// support per-agent model selection in its schema.
// Ground truth: models.json verified 2026-04-30.
func (r *antigravityModelResolver) ResolveAlias(alias string) (modelID string, ok bool) {
	return "", false
}

// SupportsBareAliases reports that Antigravity does not support bare aliases.
func (r *antigravityModelResolver) SupportsBareAliases() bool {
	return false
}
