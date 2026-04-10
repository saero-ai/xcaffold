package policy

import (
	"path/filepath"
	"slices"
)

// MatchAgent checks all match conditions against a flattened agent property map.
// Returns true if all non-empty conditions match (AND semantics).
// Empty PolicyMatch matches everything.
func MatchAgent(m PolicyMatch, props map[string]any) bool {
	if m.HasTool != "" {
		tools, ok := props["tools"].([]string)
		if !ok || !slices.Contains(tools, m.HasTool) {
			return false
		}
	}
	if m.HasField != "" {
		v, ok := props[m.HasField]
		if !ok {
			return false
		}
		if s, ok := v.(string); ok && s == "" {
			return false
		}
	}
	return true
}

// MatchName checks the name_matches glob condition against a resource name.
// Returns true if m.NameMatches is empty (wildcard) or the glob matches.
func MatchName(m PolicyMatch, name string) bool {
	if m.NameMatches == "" {
		return true
	}
	matched, err := filepath.Match(m.NameMatches, name)
	if err != nil {
		return false // invalid glob pattern = no match
	}
	return matched
}
