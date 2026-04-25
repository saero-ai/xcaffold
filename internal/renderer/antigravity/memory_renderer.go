package antigravity

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/output"
	"github.com/saero-ai/xcaffold/internal/renderer"
)

// MemoryRenderer compiles memory entries from an XcaffoldConfig into Antigravity
// Knowledge Item files under knowledge/<name>.md. Each entry is rendered with
// YAML frontmatter (title, type, description, tags) followed by the body.
//
// Antigravity supports memory natively via Knowledge Items, so no FidelityNotes
// are emitted for a successful mapping.
type MemoryRenderer struct{}

// NewMemoryRenderer constructs a MemoryRenderer. Knowledge items are returned
// in output.Output.Files under knowledge/<name>.md and are not written to disk
// by this renderer — disk writes are the caller's responsibility.
func NewMemoryRenderer() *MemoryRenderer {
	return &MemoryRenderer{}
}

// Compile translates each memory entry into an Antigravity Knowledge Item file.
// Files are returned in output.Output.Files (relative to the renderer's output
// root) following the same in-memory pattern as the main Antigravity Renderer.
// No FidelityNotes are emitted — Antigravity supports memory natively.
func (r *MemoryRenderer) Compile(config *ast.XcaffoldConfig, baseDir string) (*output.Output, []renderer.FidelityNote, error) {
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

		// Path-safety: reject names that could escape the knowledge directory.
		if strings.Contains(name, "..") || filepath.IsAbs(name) {
			return nil, nil, fmt.Errorf("memory %q: invalid entry name", name)
		}

		body, err := resolveKnowledgeBody(name, entry, baseDir)
		if err != nil {
			return nil, nil, err
		}

		tags := deriveKITags(entry)
		content := renderKnowledgeItem(name, entry, tags, body)

		safeName := renderer.SlugifyFilename(name)
		relPath := filepath.Clean(fmt.Sprintf("knowledge/%s.md", safeName))
		out.Files[relPath] = content
	}

	return out, nil, nil
}

// resolveKnowledgeBody returns the effective body for a memory entry.
// Content is populated by the compiler's filesystem scan of xcf/agents/<id>/memory/
// .md files — the renderer simply returns it.
func resolveKnowledgeBody(_ string, entry ast.MemoryConfig, _ string) (string, error) {
	return entry.Content, nil
}

// deriveKITags returns the default tags for a Knowledge Item.
// Type and targets were removed from MemoryConfig in the agent-scoped memory
// refactor, so all entries receive the generic "memory" tag.
func deriveKITags(_ ast.MemoryConfig) []string {
	return []string{"memory"}
}

// renderKnowledgeItem composes the Knowledge Item markdown content:
// YAML frontmatter with title, type, description, and tags, followed by the body.
func renderKnowledgeItem(name string, entry ast.MemoryConfig, tags []string, body string) string {
	var sb strings.Builder

	sb.WriteString("---\n")

	// title: use Name field if populated, else fall back to the map key (name).
	title := entry.Name
	if title == "" {
		title = name
	}
	fmt.Fprintf(&sb, "title: %s\n", renderer.YAMLScalar(title))

	if entry.Description != "" {
		fmt.Fprintf(&sb, "description: %s\n", renderer.YAMLScalar(entry.Description))
	}

	if len(tags) > 0 {
		sb.WriteString("tags:\n")
		for _, tag := range tags {
			fmt.Fprintf(&sb, "  - %s\n", tag)
		}
	}

	sb.WriteString("---\n")

	if body != "" {
		sb.WriteString("\n")
		sb.WriteString(strings.TrimRight(body, "\n"))
		sb.WriteString("\n")
	}

	return sb.String()
}
