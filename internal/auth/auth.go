package auth

// AuthMode describes how a component authenticates with the Anthropic API.
// It is shared across the judge and generator packages to avoid duplication.
type AuthMode string

const (
	// AuthModeAPIKey uses a direct Anthropic API key via the ANTHROPIC_API_KEY env var.
	AuthModeAPIKey AuthMode = "api_key"
	// AuthModeSubscription uses the local claude CLI subprocess (Claude Code subscription).
	AuthModeSubscription AuthMode = "subscription"
)
