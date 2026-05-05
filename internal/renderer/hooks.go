package renderer

import "strings"

// TranslateHookCommand rewrites hook commands to target-specific equivalents.
// It replaces xcf-native placeholders and provider path prefixes with the
// target's equivalents. targetEnvVar replaces "$XCF_PROJECT_DIR" (and the
// legacy "$CLAUDE_PROJECT_DIR" form preserved for backward compatibility).
// targetPathPrefix replaces ".xcf/hooks/" (and ".claude/hooks/" for backward
// compatibility with commands imported from Claude projects).
func TranslateHookCommand(command, targetEnvVar, targetPathPrefix string) string {
	// Rewrite xcf-native env var placeholder (shell and braced syntax)
	command = strings.ReplaceAll(command, "$XCF_PROJECT_DIR", targetEnvVar)
	command = strings.ReplaceAll(command, "${XCF_PROJECT_DIR}", targetEnvVar)

	// Backward-compat: rewrite Claude-specific env var imported from .claude/ projects
	command = strings.ReplaceAll(command, "$CLAUDE_PROJECT_DIR", targetEnvVar)
	command = strings.ReplaceAll(command, "${CLAUDE_PROJECT_DIR}", targetEnvVar)

	// Rewrite xcf-native hook path
	command = strings.ReplaceAll(command, ".xcf/hooks/", targetPathPrefix)

	// Backward-compat: rewrite Claude-specific hook path imported from .claude/ projects
	command = strings.ReplaceAll(command, ".claude/hooks/", targetPathPrefix)

	return command
}
