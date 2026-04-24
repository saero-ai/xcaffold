// Package gemini compiles memory entries from an XcaffoldConfig into GEMINI.md
// using provenance-marked blocks. Each entry is appended under a dedicated
// section header; re-compilation replaces the stale block in place.
package gemini

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/output"
	"github.com/saero-ai/xcaffold/internal/renderer"
)

const (
	// geminiMemoryFile is the single file Gemini uses for project memory context.
	geminiMemoryFile = "GEMINI.md"

	// geminiMemorySection is the heading under which xcaffold appends seeded
	// entries. Keeping the section scoped prevents interference with user-authored
	// content elsewhere in GEMINI.md.
	geminiMemorySection = "## Gemini Added Memories"

	// closeMarker terminates every xcaffold memory block.
	closeMarker = "<!-- xcaffold:/memory -->"
)

// MemoryRenderer appends memory entries from an XcaffoldConfig into GEMINI.md
// as provenance-marked blocks. Re-compilation replaces stale blocks in place.
// It does NOT enforce seed-once / tracked lifecycle — Gemini's single-file
// flattening makes per-entry drift detection less meaningful.
type MemoryRenderer struct {
	targetDir string
}

// NewMemoryRenderer constructs a MemoryRenderer rooted at targetDir (the
// resolved path to the Gemini project context directory).
func NewMemoryRenderer(targetDir string) *MemoryRenderer {
	return &MemoryRenderer{targetDir: targetDir}
}

// Compile appends each memory entry in config to GEMINI.md under the
// "## Gemini Added Memories" section, replacing any stale block for the same
// entry name. One LevelInfo FidelityNote with code MEMORY_PARTIAL_FIDELITY is
// emitted per entry to surface the loss of per-file granularity.
func (r *MemoryRenderer) Compile(config *ast.XcaffoldConfig, baseDir string) (*output.Output, []renderer.FidelityNote, error) {
	out := &output.Output{Files: make(map[string]string)}
	var notes []renderer.FidelityNote

	if config == nil || len(config.Memory) == 0 {
		return out, notes, nil
	}

	// Sort entry names for deterministic output.
	names := make([]string, 0, len(config.Memory))
	for name := range config.Memory {
		names = append(names, name)
	}
	sort.Strings(names)

	// Compute seeded-at once for the entire compile pass.
	seededAt := time.Now().UTC().Format(time.RFC3339)

	// Read or initialize GEMINI.md.
	geminiPath := filepath.Join(r.targetDir, geminiMemoryFile)
	existing, err := os.ReadFile(geminiPath)
	if err != nil && !os.IsNotExist(err) {
		return nil, nil, fmt.Errorf("gemini memory: read %s: %w", geminiPath, err)
	}
	content := string(existing)

	// Ensure the section header exists exactly once.
	if !strings.Contains(content, geminiMemorySection) {
		if content != "" && !strings.HasSuffix(content, "\n") {
			content += "\n"
		}
		if content != "" {
			content += "\n"
		}
		content += geminiMemorySection + "\n"
	}

	// Process each entry.
	for _, name := range names {
		entry := config.Memory[name]

		// Path-safety: reject names that could escape the target directory.
		if name == ".." || strings.Contains(name, "..") {
			return nil, nil, fmt.Errorf("gemini memory %q: entry name must not contain traversal sequences", name)
		}
		if filepath.IsAbs(name) {
			return nil, nil, fmt.Errorf("gemini memory %q: entry name must not be absolute", name)
		}

		body, err := resolveBody(name, entry, baseDir)
		if err != nil {
			return nil, nil, fmt.Errorf("gemini memory %q: %w", name, err)
		}
		if strings.TrimSpace(body) == "" {
			notes = append(notes, renderer.NewNote(
				renderer.LevelInfo,
				"gemini",
				"memory",
				name,
				"",
				renderer.CodeMemoryPartialFidelity,
				"Gemini has no native multi-file memory store; entry appended to GEMINI.md, losing per-file granularity",
				"Review GEMINI.md to confirm context ordering.",
			))
			notes = append(notes, renderer.NewNote(
				renderer.LevelWarning,
				"gemini",
				"memory",
				name,
				"instructions",
				renderer.CodeMemoryBodyEmpty,
				"memory entry has no instructions or instructions-file content; skipping",
				"Provide instructions or instructions-file in the .xcf memory entry.",
			))
			continue
		}

		// Remove any existing block for this entry (idempotency).
		content = removeMemoryBlock(content, name)

		// Build the new block.
		block := buildBlock(name, entry, body, seededAt)

		// Append block after the section header (at end of content).
		if !strings.HasSuffix(content, "\n") {
			content += "\n"
		}
		content += block

		// Emit partial-fidelity note: Gemini flattens all entries into one file.
		notes = append(notes, renderer.NewNote(
			renderer.LevelInfo,
			"gemini",
			"memory",
			name,
			"",
			renderer.CodeMemoryPartialFidelity,
			"Gemini has no native multi-file memory store; entry appended to GEMINI.md, losing per-file granularity",
			"Review GEMINI.md to confirm context ordering.",
		))
	}

	// Write GEMINI.md atomically.
	if err := os.MkdirAll(r.targetDir, 0o700); err != nil {
		return nil, nil, fmt.Errorf("gemini memory: create target dir: %w", err)
	}
	if err := os.WriteFile(geminiPath, []byte(content), 0o600); err != nil {
		return nil, nil, fmt.Errorf("gemini memory: write %s: %w", geminiPath, err)
	}

	return out, notes, nil
}

// Render wraps a files map in an output.Output. This is an identity operation —
// the Gemini memory renderer writes directly to disk and does not use the
// in-memory file map for GEMINI.md.
func (r *MemoryRenderer) Render(files map[string]string) (*output.Output, error) {
	return &output.Output{Files: files}, nil
}

// buildBlock constructs the provenance-marked memory block for a single entry.
//
// Format:
//
//	<!-- xcaffold:memory name="<name>" type="<type>" seeded-at="<RFC3339>" -->
//	**<name>** (<type>): <description>
//
//	<instructions body>
//	<!-- xcaffold:/memory -->
func buildBlock(name string, entry ast.MemoryConfig, body, seededAt string) string {
	var sb strings.Builder

	// Open marker.
	fmt.Fprintf(&sb, "<!-- xcaffold:memory name=%q type=%q seeded-at=%q -->\n",
		name, entry.Type, seededAt)

	// Header line.
	if entry.Description != "" {
		fmt.Fprintf(&sb, "**%s** (%s): %s\n", name, entry.Type, entry.Description)
	} else {
		fmt.Fprintf(&sb, "**%s** (%s):\n", name, entry.Type)
	}

	// Body.
	sb.WriteString("\n")
	sb.WriteString(strings.TrimRight(body, "\n"))
	sb.WriteString("\n")

	// Close marker.
	sb.WriteString(closeMarker)
	sb.WriteString("\n")

	return sb.String()
}

// removeMemoryBlock removes all occurrences of the xcaffold memory block for
// the given entry name from content using plain string scanning (no regex).
func removeMemoryBlock(content, name string) string {
	openMarker := fmt.Sprintf(`<!-- xcaffold:memory name="%s"`, name)

	for {
		start := strings.Index(content, openMarker)
		if start < 0 {
			return content
		}
		end := strings.Index(content[start:], closeMarker)
		if end < 0 {
			// Malformed block — bail to avoid data loss.
			return content
		}
		end = start + end + len(closeMarker)

		// Strip the block plus any immediately trailing newlines.
		afterBlock := strings.TrimPrefix(content[end:], "\n")
		content = content[:start] + afterBlock
	}
}

// resolveBody returns the effective body content for a memory entry.
// It mirrors the pattern used in the Claude memory renderer.
func resolveBody(name string, entry ast.MemoryConfig, baseDir string) (string, error) {
	if entry.Instructions != "" {
		return entry.Instructions, nil
	}
	if entry.InstructionsFile == "" {
		return "", nil
	}

	if filepath.IsAbs(entry.InstructionsFile) {
		return "", fmt.Errorf("instructions-file %q must be relative", entry.InstructionsFile)
	}
	cleaned := filepath.Clean(entry.InstructionsFile)
	abs := filepath.Join(baseDir, cleaned)
	rel, relErr := filepath.Rel(baseDir, abs)
	if relErr != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("memory %q: instructions-file %q escapes base dir", name, entry.InstructionsFile)
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		return "", fmt.Errorf("read instructions-file: %w", err)
	}
	return string(data), nil
}
