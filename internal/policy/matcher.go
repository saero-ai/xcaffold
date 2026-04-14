package policy

import (
	"path/filepath"

	"github.com/saero-ai/xcaffold/internal/ast"
)

// matchResource checks if a named resource satisfies all match conditions.
// All conditions are AND-ed. Nil match matches everything.
func matchResource(match *ast.PolicyMatch, name string, acc fieldAccessor) bool {
	if match == nil {
		return true
	}

	if match.HasTool != "" {
		if !containsString(acc.tools(), match.HasTool) {
			return false
		}
	}

	if match.HasField != "" {
		if acc.fieldValue(match.HasField) == "" {
			return false
		}
	}

	if match.NameMatches != "" {
		matched, _ := filepath.Match(match.NameMatches, name)
		if !matched {
			return false
		}
	}

	if match.TargetIncludes != "" {
		if !containsString(acc.targets(), match.TargetIncludes) {
			return false
		}
	}

	return true
}
