package renderer

import (
	"fmt"
	"sync"

	"github.com/saero-ai/xcaffold/pkg/schema"
)

var (
	modelResolverMu sync.RWMutex
	modelResolvers  = make(map[string]ModelResolver)

	tierOverrideMu sync.RWMutex
	tierOverrides  map[string]string
)

// RegisterModelResolver registers a ModelResolver for a provider.
// This must be called during provider initialization (typically in manifest.go init() functions).
func RegisterModelResolver(providerName string, resolver ModelResolver) {
	modelResolverMu.Lock()
	defer modelResolverMu.Unlock()
	modelResolvers[providerName] = resolver
}

// LookupModelResolver retrieves the registered ModelResolver for a provider.
// Returns nil if no resolver is registered for that provider.
func LookupModelResolver(providerName string) ModelResolver {
	modelResolverMu.RLock()
	defer modelResolverMu.RUnlock()
	return modelResolvers[providerName]
}

// SetTierOverrides installs user-defined tier mappings for the current
// compilation. The compiler calls this before rendering so that
// project.<target>.vars entries like model-tier-balanced override
// the built-in alias map. Pass nil to clear.
func SetTierOverrides(overrides map[string]string) {
	tierOverrideMu.Lock()
	defer tierOverrideMu.Unlock()
	tierOverrides = overrides
}

// ClearTierOverrides removes any active tier overrides.
func ClearTierOverrides() { SetTierOverrides(nil) }

// ResolveModel takes an alias from the Xcaffold configuration and a target name.
// It returns the target-specific model string and a boolean indicating if the target expects one.
// If the target doesn't support models or the alias cannot be resolved, it returns ("", false).
// User-defined tier overrides (via SetTierOverrides) take precedence over built-in mappings.
func ResolveModel(alias, target string) (string, bool) {
	tierOverrideMu.RLock()
	if override, ok := tierOverrides[alias]; ok {
		tierOverrideMu.RUnlock()
		return override, true
	}
	tierOverrideMu.RUnlock()

	resolver := LookupModelResolver(target)
	if resolver == nil {
		return "", false
	}

	modelID, ok := resolver.ResolveAlias(alias)
	if !ok {
		return "", false
	}

	return modelID, true
}

// IsMappedModel returns true if the input alias is one of xcaffold's canonical
// tier aliases (flagship, balanced, fast) that is explicitly mapped for the
// given target.
func IsMappedModel(alias, target string) bool {
	xcaffoldAliases := map[string]bool{
		"flagship": true,
		"balanced": true,
		"fast":     true,
	}

	if !xcaffoldAliases[alias] {
		return false
	}

	tierOverrideMu.RLock()
	_, hasOverride := tierOverrides[alias]
	tierOverrideMu.RUnlock()
	if hasOverride {
		return true
	}

	resolver := LookupModelResolver(target)
	if resolver == nil {
		return false
	}

	_, ok := resolver.ResolveAlias(alias)
	return ok
}

func SanitizeAgentModel(model string, caps CapabilitySet, targetName, agentID string) (string, []FidelityNote) {
	if model == "" {
		return "", nil
	}

	modelSupport := schema.FieldSupportForTarget("agent", "model", targetName)
	if modelSupport == "unsupported" || modelSupport == "" {
		return "", nil
	}

	isMapped := IsMappedModel(model, targetName)

	resolved, ok := ResolveModel(model, targetName)
	if !ok || resolved == "" {
		return "", []FidelityNote{{
			Level:      LevelWarning,
			Target:     targetName,
			Kind:       "agent",
			Resource:   agentID,
			Field:      "model",
			Code:       CodeAgentModelUnmapped,
			Reason:     fmt.Sprintf("model %q could not be resolved for agent %q on %s", model, agentID, targetName),
			Mitigation: fmt.Sprintf("Use a tier alias (e.g. balanced) or a native literal for %s", targetName),
		}}
	}

	if isMapped {
		return resolved, nil
	}

	return resolved, []FidelityNote{{
		Level:    LevelInfo,
		Target:   targetName,
		Kind:     "agent",
		Resource: agentID,
		Field:    "model",
		Code:     CodeFieldTransformed,
		Reason:   fmt.Sprintf("model %q passed through for agent %q as %q", model, agentID, resolved),
	}}
}
