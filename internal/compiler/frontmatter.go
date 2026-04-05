package compiler

import "strings"

// stripFrontmatter removes YAML frontmatter delimited by "---" from the start
// of a markdown file, returning only the body content with leading whitespace trimmed.
// This function is used by tests and is the canonical implementation; the renderer
// package carries an identical copy to avoid an import cycle.
func stripFrontmatter(content string) string {
	// Normalise line endings.
	content = strings.ReplaceAll(content, "\r\n", "\n")
	lines := strings.SplitN(content, "\n", -1)

	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return strings.TrimLeft(content, "\n")
	}

	// Find the closing "---"
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			body := strings.Join(lines[i+1:], "\n")
			return strings.TrimLeft(body, "\n")
		}
	}

	// No closing delimiter found — return as-is (no frontmatter detected).
	return strings.TrimLeft(content, "\n")
}
