package shared

import (
	"strings"
)

// TranslateHookCommand formats a local hook command string into an execution
// path relative to the provider's canonical root directory.
//
// It only modifies paths that explicitly start with "xcf/hooks/".
// For absolute paths or unstructured commands, it returns the string unaltered.
func TranslateHookCommand(cmd string, providerDirDepth int) string {
	if !strings.HasPrefix(cmd, "xcf/hooks/") {
		return cmd
	}
	prefix := ""
	for i := 0; i < providerDirDepth; i++ {
		prefix += "../"
	}
	return prefix + cmd
}
