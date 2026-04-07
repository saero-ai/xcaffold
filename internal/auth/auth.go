package auth

// AuthMode describes how a component authenticates with the Anthropic API.
// It is shared across the judge and generator packages to avoid duplication.
type AuthMode string

const (
	// AuthModeAPIKey uses a direct Anthropic API key via the ANTHROPIC_API_KEY env var.
	AuthModeAPIKey AuthMode = "api_key"
	// AuthModeGenericAPI uses an OpenAI-compatible endpoint via XCAFFOLD_LLM_API_KEY.
	AuthModeGenericAPI AuthMode = "generic_api"
	// AuthModeSubscription uses the local external CLI subprocess.
	AuthModeSubscription AuthMode = "subscription"
)
