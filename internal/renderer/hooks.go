package renderer

import "strings"

// TranslateHookCommand rewrites hook commands to target-specific equivalents.
// It replaces the default Claude-centric paths and environment variables with target-specific ones.
// E.g., it replaces "$CLAUDE_PROJECT_DIR" with targetEnvVar and ".claude/hooks/" with targetPathPrefix.
// It also rewrites ".xcf/hooks/" (the abstract representation) to targetPathPrefix.
func TranslateHookCommand(command, targetEnvVar, targetPathPrefix string) string {
	// Standardize environment variable (for shell and standard env syntax)
	command = strings.ReplaceAll(command, "$CLAUDE_PROJECT_DIR", targetEnvVar)
	command = strings.ReplaceAll(command, "${CLAUDE_PROJECT_DIR}", targetEnvVar)

	// Standardize paths
	command = strings.ReplaceAll(command, ".claude/hooks/", targetPathPrefix)
	command = strings.ReplaceAll(command, ".xcf/hooks/", targetPathPrefix)

	return command
}
