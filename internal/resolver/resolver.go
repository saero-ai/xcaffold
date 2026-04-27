package resolver

import (
	"bytes"
)

// StripFrontmatter removes YAML frontmatter (--- blocks) from markdown content.
func StripFrontmatter(content string) string {
	b := []byte(content)
	if !bytes.HasPrefix(b, []byte("---\n")) && !bytes.HasPrefix(b, []byte("---\r\n")) {
		return content
	}

	// Find the end of the frontmatter block
	endIdx := bytes.Index(b[4:], []byte("\n---"))
	if endIdx == -1 {
		return content // Malformed or missing closing '---', return as-is
	}

	// b[4:] starts after the first '---'. Add 4 (offset) + 4 (length of '\n---')
	startOfContent := 4 + endIdx + 4

	// Trim remaining whitespace (e.g. newline following the closing ---)
	result := bytes.TrimSpace(b[startOfContent:])
	return string(result)
}
