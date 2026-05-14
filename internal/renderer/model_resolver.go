package renderer

// ModelResolver maps short model aliases to full model IDs for a provider.
// Each provider registers its own resolver so the renderer can emit correct
// model references without hardcoding provider-specific alias tables.
type ModelResolver interface {
	// ResolveAlias returns the full model ID for a short alias.
	// Returns ok=false if the alias is not recognized.
	ResolveAlias(alias string) (modelID string, ok bool)
}
