package antigravity2

// antigravity2ModelResolver implements renderer.ModelResolver for Antigravity 2.0.
// Antigravity 2.0 supports multi-model selection across Gemini, Claude, and
// GPT-OSS tiers via the --model CLI flag or the UI model picker.
type antigravity2ModelResolver struct{}

// NewModelResolver creates a ModelResolver for Antigravity 2.0.
func NewModelResolver() *antigravity2ModelResolver {
	return &antigravity2ModelResolver{}
}

// knownModels maps xcaffold short aliases to Antigravity 2.0 model IDs.
// Ground truth: Antigravity 2.0 model spec (2026-05-22).
var knownModels = map[string]string{
	// Gemini tier
	"gemini-3.5-flash":    "gemini-3.5-flash",
	"gemini-3.1-pro-high": "gemini-3.1-pro-high",
	"gemini-3.1-pro-low":  "gemini-3.1-pro-low",
	"gemini-3-flash":      "gemini-3-flash",
	// Short aliases
	"flash":   "gemini-3.5-flash",
	"pro":     "gemini-3.1-pro-high",
	"pro-low": "gemini-3.1-pro-low",

	// Claude reasoning tier
	"claude-sonnet-4-6-thinking": "claude-sonnet-4-6-thinking",
	"claude-opus-4-6-thinking":   "claude-opus-4-6-thinking",
	"sonnet-thinking":            "claude-sonnet-4-6-thinking",
	"opus-thinking":              "claude-opus-4-6-thinking",

	// GPT-OSS tier
	"gpt-oss-120b": "gpt-oss-120b",
	"gpt-oss":      "gpt-oss-120b",

	// Image generation tier (non-selectable via --model flag; UI-only)
	"nano-banana-2": "nano-banana-2",
}

// ResolveAlias returns the full model ID for a short alias.
func (r *antigravity2ModelResolver) ResolveAlias(alias string) (modelID string, ok bool) {
	id, ok := knownModels[alias]
	return id, ok
}
