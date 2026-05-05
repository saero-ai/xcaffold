package renderer

import (
	"fmt"
	"strings"
	"sync"

	"github.com/saero-ai/xcaffold/pkg/schema"
)

var (
	modelResolverMu sync.RWMutex
	modelResolvers  = make(map[string]ModelResolver)
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

// ResolveModel takes an alias from the Xcaffold configuration and a target name.
// It returns the target-specific model string and a boolean indicating if the target expects one.
// If the target doesn't support models or the alias cannot be resolved, it returns ("", false).
// Delegates to the provider's ModelResolver for the actual translation.
func ResolveModel(alias, target string) (string, bool) {
	resolver := LookupModelResolver(target)
	if resolver == nil {
		// Provider does not support model selection (e.g., antigravity)
		return "", false
	}

	modelID, ok := resolver.ResolveAlias(alias)
	if !ok {
		// The alias could not be resolved by this provider's resolver
		return "", false
	}

	return modelID, true
}

// IsMappedModel returns true if the input alias is one of xcaffold's canonical
// versioned aliases (sonnet-4, opus-4, haiku-3.5) that is explicitly mapped for
// the given target. This is used to distinguish xcaffold-normalized aliases from
// native provider literals or bare aliases.
//
// Note: This is different from whether the provider's resolver accepts the model.
// A model might be accepted by the provider (e.g., bare "sonnet" on Claude) but
// still not be a xcaffold-mapped alias.
func IsMappedModel(alias, target string) bool {
	// The xcaffold-canonical aliases that have explicit mappings per target
	xcaffoldAliases := map[string]bool{
		"sonnet-4":  true,
		"opus-4":    true,
		"haiku-3.5": true,
	}

	// Only the canonical xcaffold aliases are "mapped"
	if !xcaffoldAliases[alias] {
		return false
	}

	resolver := LookupModelResolver(target)
	if resolver == nil {
		return false
	}

	_, ok := resolver.ResolveAlias(alias)
	return ok
}

// IsKnownClaudeAlias returns true if the literal string is a naked tier name
// typical of raw Claude Code usage (sonnet, opus, haiku).
func IsKnownClaudeAlias(alias string) bool {
	bare := strings.ToLower(alias)
	switch bare {
	case "sonnet", "opus", "haiku":
		return true
	default:
		return false
	}
}

// SanitizeAgentModel maps a model alias to a provider-specific literal.
// It returns the sanitized model string and a slice of FidelityNotes.
func SanitizeAgentModel(model string, caps CapabilitySet, targetName, agentID string) (string, []FidelityNote) {
	if model == "" {
		return "", nil // Nothing to do
	}

	var notes []FidelityNote

	// If the provider does not support the model field, drop it.
	modelSupport := schema.FieldSupportForTarget("agent", "model", targetName)
	if modelSupport == "unsupported" || modelSupport == "" {
		return "", nil
	}

	// Check if this is a known bare alias and whether the target supports them
	isClaudeAlias := IsKnownClaudeAlias(model)
	isMapped := IsMappedModel(model, targetName)
	resolver := LookupModelResolver(targetName)
	supportsBare := resolver != nil && resolver.SupportsBareAliases()

	resolved, ok := ResolveModel(model, targetName)
	if !ok || resolved == "" {
		// Resolution failed. If it's a bare alias on a target that doesn't support them, warn.
		if isClaudeAlias && !supportsBare {
			notes = append(notes, NewNote(
				LevelWarning, targetName, "agent", agentID, "model",
				CodeAgentModelUnmapped,
				fmt.Sprintf("bare alias %q passed through for agent %q unmapped; this may fail on %s", model, agentID, targetName),
				fmt.Sprintf("Use a mapped alias (e.g. sonnet-4) or a native literal for %s", targetName),
			))
		}
		return "", notes
	}

	// We kept the field. Check if the model was NOT from the xcaffold alias list.
	if !isMapped {
		// Is it a bare alias (e.g. "sonnet", "opus", "haiku")?
		if isClaudeAlias {
			if supportsBare {
				// Claude Code resolves bare tier aliases at runtime to the current
				// recommended version. Pass through as-is and emit an info note.
				// Ground truth: models.json verified 2026-04-30 — "sonnet", "opus",
				// and "haiku" are documented Claude Code aliases.
				notes = append(notes, NewNote(
					LevelInfo, targetName, "agent", agentID, "model",
					CodeFieldTransformed,
					fmt.Sprintf("bare alias %q passed through for agent %q on claude target; resolved at runtime", model, agentID),
					"Use a versioned alias (e.g. sonnet-4) for deterministic resolution across targets",
				))
				return model, notes
			}
			// Bare Claude alias on a non-Claude target — meaningless outside Claude Code.
			notes = append(notes, NewNote(
				LevelWarning, targetName, "agent", agentID, "model",
				CodeAgentModelUnmapped,
				fmt.Sprintf("bare alias %q passed through for agent %q unmapped; this may fail on %s", model, agentID, targetName),
				fmt.Sprintf("Use a mapped alias (e.g. sonnet-4) or a native literal for %s", targetName),
			))
			return "", notes
		}
		// It's not a known alias, meaning it's a native literal. Pass it through safely.
		notes = append(notes, NewNote(
			LevelInfo, targetName, "agent", agentID, "model",
			CodeFieldTransformed,
			fmt.Sprintf("native literal %q passed through for agent %q", model, agentID),
			"",
		))
	}

	return resolved, notes
}
