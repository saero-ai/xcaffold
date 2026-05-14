package codex

import "strings"

type codexModelResolver struct{}

func NewModelResolver() *codexModelResolver {
	return &codexModelResolver{}
}

// ResolveAlias maps xcaffold aliases to Codex model IDs.
// Also passes through any string starting with "gpt-" as a native model ID.
func (r *codexModelResolver) ResolveAlias(alias string) (modelID string, ok bool) {
	aliasMap := map[string]string{
		"sonnet-4":  "gpt-5.4",
		"opus-4":    "gpt-5.5",
		"haiku-3.5": "gpt-5.4-mini",
	}

	if id, found := aliasMap[alias]; found {
		return id, true
	}

	if strings.HasPrefix(alias, "gpt-") {
		return alias, true
	}

	return "", false
}

// SupportsBareAliases returns false because Codex requires explicit model IDs.
func (r *codexModelResolver) SupportsBareAliases() bool {
	return false
}
