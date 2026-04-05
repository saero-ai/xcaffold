package bir

import (
	"os"
	"path/filepath"
	"strings"
)

// ImportWorkflow reads a workflow markdown file, strips YAML frontmatter,
// and returns a populated SemanticUnit with intents detected.
// platform identifies the source platform (e.g. "gemini", "claude", "cursor").
func ImportWorkflow(path string, platform string) (*SemanticUnit, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	body := stripFrontmatter(string(data))
	id := deriveID(path)

	unit := &SemanticUnit{
		ID:             id,
		SourceKind:     SourceWorkflow,
		SourcePlatform: platform,
		SourcePath:     path,
		ResolvedBody:   body,
		Intents:        DetectIntents(body),
	}

	return unit, nil
}

// stripFrontmatter removes YAML frontmatter delimited by --- lines.
// If no frontmatter is present, the content is returned unchanged (leading newlines trimmed).
func stripFrontmatter(content string) string {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	lines := strings.Split(content, "\n")

	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return strings.TrimLeft(content, "\n")
	}

	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			body := strings.Join(lines[i+1:], "\n")
			return strings.TrimLeft(body, "\n")
		}
	}

	return strings.TrimLeft(content, "\n")
}

// deriveID returns the base filename without the .md extension.
func deriveID(path string) string {
	base := filepath.Base(path)
	return strings.TrimSuffix(base, filepath.Ext(base))
}
