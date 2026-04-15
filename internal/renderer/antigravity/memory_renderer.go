package antigravity

import (
	"fmt"
	"os"
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
type MemoryRenderer struct {
	targetDir string
}

// NewMemoryRenderer constructs a MemoryRenderer. targetDir is the resolved path
// to the Antigravity output root (e.g. baseDir/.agents); knowledge items are
// written under <targetDir>/knowledge/.
func NewMemoryRenderer(targetDir string) *MemoryRenderer {
	return &MemoryRenderer{targetDir: targetDir}
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
		if strings.ContainsAny(name, "/\\") || strings.Contains(name, "..") || filepath.IsAbs(name) {
			return nil, nil, fmt.Errorf("memory %q: invalid entry name", name)
		}

		body, err := resolveKnowledgeBody(name, entry, baseDir)
		if err != nil {
			return nil, nil, err
		}

		tags := deriveKITags(entry)
		content := renderKnowledgeItem(name, entry, tags, body)

		relPath := filepath.Clean(fmt.Sprintf("knowledge/%s.md", name))
		out.Files[relPath] = content
	}

	return out, nil, nil
}

// resolveKnowledgeBody returns the effective body for a memory entry.
// instructions takes precedence over instructions-file. An empty body is
// allowed (caller may choose to emit an empty Knowledge Item).
func resolveKnowledgeBody(name string, entry ast.MemoryConfig, baseDir string) (string, error) {
	if entry.Instructions != "" {
		return entry.Instructions, nil
	}
	if entry.InstructionsFile == "" {
		return "", nil
	}

	if filepath.IsAbs(entry.InstructionsFile) {
		return "", fmt.Errorf("memory %q: instructions-file %q must be relative", name, entry.InstructionsFile)
	}
	cleaned := filepath.Clean(entry.InstructionsFile)
	abs := filepath.Join(baseDir, cleaned)
	rel, relErr := filepath.Rel(baseDir, abs)
	if relErr != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("memory %q: instructions-file %q escapes base dir", name, entry.InstructionsFile)
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		return "", fmt.Errorf("memory %q: read instructions-file: %w", name, err)
	}
	return string(data), nil
}

// deriveKITags returns the tags for a Knowledge Item.
// If a targets["antigravity"].Provider["ki-tags"] slice is present, those tags
// are used verbatim. Otherwise tags are derived from the entry's type field.
func deriveKITags(entry ast.MemoryConfig) []string {
	if override, ok := entry.Targets[targetName]; ok {
		if raw, ok := override.Provider["ki-tags"]; ok {
			if slice, ok := raw.([]interface{}); ok {
				tags := make([]string, 0, len(slice))
				for _, v := range slice {
					if s, ok := v.(string); ok {
						tags = append(tags, s)
					}
				}
				if len(tags) > 0 {
					return tags
				}
			}
		}
	}

	switch entry.Type {
	case "user":
		return []string{"user", "preferences"}
	case "feedback":
		return []string{"feedback"}
	case "project":
		return []string{"project", "context"}
	case "reference":
		return []string{"reference", "docs"}
	default:
		return []string{"memory"}
	}
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
	fmt.Fprintf(&sb, "title: %s\n", yamlScalar(title))

	if entry.Type != "" {
		fmt.Fprintf(&sb, "type: %s\n", entry.Type)
	}

	if entry.Description != "" {
		fmt.Fprintf(&sb, "description: %s\n", yamlScalar(entry.Description))
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
