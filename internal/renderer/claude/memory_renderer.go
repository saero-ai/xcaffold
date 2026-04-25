package claude

import (
	"crypto/sha256"
	"encoding/hex"
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

// MemorySeed records a single seeded memory agent directory. Its shape mirrors
// state.MemorySeed, but the type is declared locally to avoid an import cycle
// (internal/state → internal/compiler → internal/renderer/claude). Callers
// (e.g. runApply) translate these into state.MemorySeed by direct field copy
// before handing them to state.GenerateWithOpts.
//
// In the concatenated model, one MemorySeed is recorded per agent directory
// (keyed by AgentRef). Name holds the AgentRef value.
type MemorySeed struct {
	Name     string
	Target   string
	Path     string
	Hash     string
	SeededAt string
}

// MemoryRenderer writes memory entries into a Claude project memory directory.
// Entries are grouped by AgentRef and concatenated into a single
// <targetDir>/<agentRef>/MEMORY.md file per agent using ## <name> headings.
// Seed-once semantics apply to the MEMORY.md file: it is written on first
// apply and skipped on subsequent applies unless --reseed is set.
type MemoryRenderer struct {
	targetDir string
	seeds     []MemorySeed
	reseed    bool
}

// NewMemoryRenderer constructs a MemoryRenderer rooted at targetDir (the
// resolved path to the Claude project memory directory).
func NewMemoryRenderer(targetDir string) *MemoryRenderer {
	return &MemoryRenderer{targetDir: targetDir}
}

// WithReseed returns a copy of the renderer configured to overwrite existing
// MEMORY.md files regardless of drift state. Used by
// `xcaffold apply --include-memory --reseed`.
func (r *MemoryRenderer) WithReseed(reseed bool) *MemoryRenderer {
	cp := *r
	cp.reseed = reseed
	return &cp
}

// Seeds returns MemorySeed records produced by the last Compile call. Used by
// runApply to feed state.GenerateWithOpts.
func (r *MemoryRenderer) Seeds() []MemorySeed {
	out := make([]MemorySeed, len(r.seeds))
	copy(out, r.seeds)
	return out
}

// Compile writes each agent's MEMORY.md using seed-once semantics, returning
// fidelity notes for no-op seeds.
func (r *MemoryRenderer) Compile(config *ast.XcaffoldConfig, baseDir string) (*output.Output, []renderer.FidelityNote, error) {
	return r.CompileWithPriorSeeds(config, baseDir, nil)
}

// CompileWithPriorSeeds is Compile + signature compatibility for callers that
// previously passed per-entry drift hashes. In the concatenated model, drift
// detection is handled at the file level (seed-once); priorHashes is accepted
// for API compatibility but is not consulted for per-entry hash comparison.
//
// Groups all non-empty memory entries by AgentRef (defaulting to "default"),
// concatenates them into a single MEMORY.md per agent directory with ## <name>
// headings (sorted for deterministic output), and writes that file with
// seed-once semantics.
func (r *MemoryRenderer) CompileWithPriorSeeds(config *ast.XcaffoldConfig, baseDir string, priorHashes map[string]string) (*output.Output, []renderer.FidelityNote, error) {
	out := &output.Output{Files: make(map[string]string)}
	var notes []renderer.FidelityNote
	r.seeds = nil

	if config == nil || len(config.Memory) == 0 {
		return out, notes, nil
	}

	// Collect and sort entry names for deterministic output.
	names := make([]string, 0, len(config.Memory))
	for name := range config.Memory {
		names = append(names, name)
	}
	sort.Strings(names)

	type entryBody struct {
		name string
		body string
	}
	grouped := make(map[string][]entryBody)

	for _, name := range names {
		entry := config.Memory[name]

		body, err := resolveMemoryBody(name, entry, baseDir)
		if err != nil {
			return nil, notes, fmt.Errorf("memory %q: %w", name, err)
		}
		if strings.TrimSpace(body) == "" {
			notes = append(notes, renderer.NewNote(
				renderer.LevelWarning,
				"claude",
				"memory",
				name,
				"instructions",
				renderer.CodeMemoryBodyEmpty,
				"memory entry has no instructions or instructions-file content; skipping",
				"Provide instructions or instructions-file in the .xcf memory entry.",
			))
			continue
		}

		agentRef := entry.AgentRef
		if agentRef == "" {
			agentRef = "default"
		}

		// Validate agentRef for path traversal.
		if agentRef == ".." || strings.Contains(agentRef, "..") {
			return nil, notes, fmt.Errorf("memory %q: agent-ref %q must not contain traversal sequences", name, agentRef)
		}
		if filepath.IsAbs(agentRef) {
			return nil, notes, fmt.Errorf("memory %q: agent-ref %q must not be absolute", name, agentRef)
		}

		grouped[agentRef] = append(grouped[agentRef], entryBody{name: name, body: body})
	}

	// Process each agent in sorted order for deterministic MemorySeed output.
	agentRefs := make([]string, 0, len(grouped))
	for agentRef := range grouped {
		agentRefs = append(agentRefs, agentRef)
	}
	sort.Strings(agentRefs)

	for _, agentRef := range agentRefs {
		entries := grouped[agentRef]

		// Concatenate entries with ## <name> headings (already sorted).
		var sb strings.Builder
		for i, e := range entries {
			if i > 0 {
				sb.WriteString("\n")
			}
			fmt.Fprintf(&sb, "## %s\n\n", e.name)
			sb.WriteString(strings.TrimRight(e.body, "\n"))
			sb.WriteString("\n")
		}
		content := sb.String()
		newHash := hashSHA256(content)

		agentDir := filepath.Join(r.targetDir, agentRef)
		memPath := filepath.Join(agentDir, "MEMORY.md")

		exists, err := fileExists(memPath)
		if err != nil {
			return nil, notes, fmt.Errorf("memory agent-ref %q: stat target: %w", agentRef, err)
		}

		if exists && !r.reseed {
			notes = append(notes, renderer.NewNote(
				renderer.LevelInfo,
				"claude",
				"memory",
				agentRef,
				"",
				renderer.CodeMemorySeedSkipped,
				"file exists; seed-once lifecycle preserves existing content",
				"use --reseed to overwrite",
			))
			continue
		}

		// Write MEMORY.md.
		if err := os.MkdirAll(agentDir, 0o700); err != nil {
			return nil, notes, fmt.Errorf("memory agent-ref %q: create dir: %w", agentRef, err)
		}
		if err := os.WriteFile(memPath, []byte(content), 0o600); err != nil {
			return nil, notes, fmt.Errorf("memory agent-ref %q: write MEMORY.md: %w", agentRef, err)
		}

		r.seeds = append(r.seeds, MemorySeed{
			Name:     agentRef,
			Target:   "claude",
			Path:     memPath,
			Hash:     newHash,
			SeededAt: time.Now().UTC().Format(time.RFC3339),
		})
	}

	return out, notes, nil
}

// resolveMemoryBody returns the effective body content for a memory entry.
// Parser enforces mutual exclusion between instructions and instructions-file;
// the renderer treats both missing as an empty body (caller decides behavior).
func resolveMemoryBody(name string, entry ast.MemoryConfig, baseDir string) (string, error) {
	if entry.Instructions != "" {
		return entry.Instructions, nil
	}
	if entry.InstructionsFile == "" {
		return "", nil
	}

	// Reject absolute paths or any traversal that escapes baseDir.
	// filepath.Clean + filepath.Rel is the only correct escape check because
	// HasPrefix("..") false-matches names like "..config" and misses nested
	// traversal like "sub/../../etc/passwd".
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

// renderMemoryMarkdown composes the final memory file content for a single
// entry. When description is set, YAML frontmatter is emitted before the body.
// Output always ends with a trailing newline.
//
// Note: in the concatenated MEMORY.md model this function is used by callers
// that need per-entry frontmatter rendering (e.g. tests of the render helper
// in isolation). The CompileWithPriorSeeds path emits ## <name> headings
// directly without calling renderMemoryMarkdown.
func renderMemoryMarkdown(entry ast.MemoryConfig, body string) string {
	var sb strings.Builder
	if entry.Description != "" {
		sb.WriteString("---\n")
		// Use %q (Go double-quoted scalar) which is a valid YAML double-quoted
		// string, safely escaping colons, newlines, and other YAML-special characters.
		fmt.Fprintf(&sb, "description: %q\n", entry.Description)
		sb.WriteString("---\n\n")
	}
	sb.WriteString(strings.TrimRight(body, "\n"))
	sb.WriteString("\n")
	return sb.String()
}

// hashSHA256 returns the "sha256:<hex>" hash of the given content.
func hashSHA256(content string) string {
	sum := sha256.Sum256([]byte(content))
	return "sha256:" + hex.EncodeToString(sum[:])
}

// fileExists reports whether a regular file exists at path. Errors other than
// NotExist are surfaced.
func fileExists(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return !info.IsDir(), nil
}
