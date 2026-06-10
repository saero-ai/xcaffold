package antigravity2

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/output"
	"github.com/saero-ai/xcaffold/internal/renderer"
)

// MemoryRenderer compiles memory entries from an XcaffoldConfig into Antigravity 2.0
// Knowledge Item files under knowledge/<name>.md. Each entry is rendered with
// YAML frontmatter (title, description, tags) followed by the body.
type MemoryRenderer struct{}

// NewMemoryRenderer constructs a MemoryRenderer.
func NewMemoryRenderer() *MemoryRenderer {
	return &MemoryRenderer{}
}

// Compile translates each memory entry into an Antigravity 2.0 Knowledge Item file.
// Files are returned in output.Output.Files (relative to the renderer's output root).
func (r *MemoryRenderer) Compile(config *ast.XcaffoldConfig, _ string) (*output.Output, []renderer.FidelityNote, error) {
	out := &output.Output{Files: make(map[string]string)}

	if len(config.Memory) == 0 {
		return out, nil, nil
	}

	names := make([]string, 0, len(config.Memory))
	for k := range config.Memory {
		names = append(names, k)
	}
	sort.Strings(names)

	for _, name := range names {
		entry := config.Memory[name]

		if strings.Contains(name, "..") || filepath.IsAbs(name) {
			return nil, nil, fmt.Errorf("memory %q: invalid entry name", name)
		}

		content := renderKnowledgeItem2(name, entry)
		safeName := renderer.SlugifyFilename(name)
		relPath := filepath.Clean(fmt.Sprintf("knowledge/%s.md", safeName))
		out.Files[relPath] = content
	}

	return out, nil, nil
}

// renderKnowledgeItem2 composes the Knowledge Item markdown content with
// YAML frontmatter (title, description, tags) followed by the body.
func renderKnowledgeItem2(name string, entry ast.MemoryConfig) string {
	var sb strings.Builder

	sb.WriteString("---\n")

	title := entry.Name
	if title == "" {
		title = name
	}
	fmt.Fprintf(&sb, "title: %s\n", renderer.YAMLScalar(title))

	if entry.Description != "" {
		fmt.Fprintf(&sb, "description: %s\n", renderer.YAMLScalar(entry.Description))
	}

	sb.WriteString("tags:\n  - memory\n")
	sb.WriteString("---\n")

	if entry.Content != "" {
		sb.WriteString("\n")
		sb.WriteString(strings.TrimRight(entry.Content, "\n"))
		sb.WriteString("\n")
	}

	return sb.String()
}
