package antigravity

import "strings"

// antigravityModelResolver implements renderer.ModelResolver for Antigravity.
// Antigravity supports multi-model selection across Gemini, Claude, and
// GPT-OSS tiers via the --model CLI flag or the UI model picker.
type antigravityModelResolver struct{}

// NewModelResolver creates a ModelResolver for Antigravity.
func NewModelResolver() *antigravityModelResolver {
	return &antigravityModelResolver{}
}

// knownModels maps xcaffold short aliases to Antigravity model IDs.
// Ground truth: Antigravity model spec (2026-05-22 / 2026-08-23).
var knownModels = map[string]string{
	// Gemini tier
	"gemini-3.5-flash":    "gemini-3.5-flash",
	"gemini-3.1-pro-high": "gemini-3.1-pro-high",
	"gemini-3.1-pro-low":  "gemini-3.1-pro-low",
	"gemini-3.1-pro":      "gemini-3.1-pro-high",
	"gemini-3-flash":      "gemini-3-flash",
	"gemini-2.5-pro":      "gemini-2.5-pro",
	"gemini-2.5-flash":    "gemini-2.5-flash",
	// Standard xcaffold tier aliases
	"flagship": "gemini-3.1-pro-high",
	"balanced": "gemini-3.5-flash",
	"fast":     "gemini-2.5-flash",

	// Short aliases
	"flash":      "gemini-3.5-flash",
	"flash_lite": "gemini-2.5-flash",
	"pro":        "gemini-3.1-pro-high",
	"pro-low":    "gemini-3.1-pro-low",
	"inherit":    "inherit",

	// Claude reasoning tier
	"claude-sonnet-4-6-thinking": "claude-sonnet-4-6-thinking",
	"claude-opus-4-6-thinking":   "claude-opus-4-6-thinking",
	"sonnet-thinking":            "claude-sonnet-4-6-thinking",
	"opus-thinking":              "claude-opus-4-6-thinking",

	// GPT-OSS tier
	"gpt-oss-120b": "gpt-oss-120b",
	"gpt-oss":      "gpt-oss-120b",

	// Image generation tier
	"nano-banana-2": "nano-banana-2",
}

// ResolveAlias returns the full model ID for a short alias.
func (r *antigravityModelResolver) ResolveAlias(alias string) (modelID string, ok bool) {
	alias = strings.TrimSpace(alias)
	if alias == "" {
		return "", false
	}
	id, ok := knownModels[alias]
	if ok {
		return id, true
	}
	// Pass through unknown specific model IDs directly
	return alias, true
}
