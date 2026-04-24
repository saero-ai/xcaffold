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

// MemorySeed records a single seeded memory entry. Its shape mirrors
// state.MemorySeed, but the type is declared locally to avoid an import cycle
// (internal/state → internal/compiler → internal/renderer/claude). Callers
// (e.g. runApply) translate these into state.MemorySeed by direct field copy
// before handing them to state.GenerateWithOpts.
type MemorySeed struct {
	Name      string
	Target    string
	Path      string
	Hash      string
	SeededAt  string
	Lifecycle string
}

// Memory lifecycle values.
const (
	memoryLifecycleSeedOnce = "seed-once"
	memoryLifecycleTracked  = "tracked"
)

// memoryIndexSection is the MEMORY.md heading under which xcaffold-seeded
// entries are listed. Keeping the section scoped prevents us from interfering
// with any user-authored headings elsewhere in the file.
const memoryIndexSection = "## xcaffold seeds"

// MemoryRenderer writes memory entries into a Claude project memory directory.
// It enforces the seed-once / tracked lifecycle contract and detects drift
// between the last seeded hash and the current on-disk hash.
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
// memory files regardless of lifecycle or drift state. Used by
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

// Compile writes each memory entry according to its lifecycle, returning
// fidelity notes for no-op seeds.
func (r *MemoryRenderer) Compile(config *ast.XcaffoldConfig, baseDir string) (*output.Output, []renderer.FidelityNote, error) {
	return r.CompileWithPriorSeeds(config, baseDir, nil)
}

// CompileWithPriorSeeds is Compile + drift detection. priorHashes is a map from
// memory entry name to the SHA-256 hash recorded in the state file on the last
// apply. For tracked entries, the current on-disk hash is compared to
// priorHashes[name]; any mismatch produces a drift error unless WithReseed(true)
// is set.
func (r *MemoryRenderer) CompileWithPriorSeeds(config *ast.XcaffoldConfig, baseDir string, priorHashes map[string]string) (*output.Output, []renderer.FidelityNote, error) {
	out := &output.Output{Files: make(map[string]string)}
	var notes []renderer.FidelityNote
	r.seeds = nil

	if config == nil || len(config.Memory) == 0 {
		return out, notes, nil
	}

	names := make([]string, 0, len(config.Memory))
	for name := range config.Memory {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		entry := config.Memory[name]
		entryNotes, err := r.compileEntry(name, entry, baseDir, priorHashes)
		// Always collect notes even when err != nil: drift detection returns
		// both a structured FidelityNote and a hard error simultaneously so
		// callers (e.g. audit.json writers) see the structured event.
		notes = append(notes, entryNotes...)
		if err != nil {
			return nil, notes, err
		}
	}

	return out, notes, nil
}

// compileEntry handles a single memory entry end-to-end. It returns any
// fidelity notes produced and a fatal error if drift is detected.
func (r *MemoryRenderer) compileEntry(name string, entry ast.MemoryConfig, baseDir string, priorHashes map[string]string) ([]renderer.FidelityNote, error) {
	body, err := resolveMemoryBody(name, entry, baseDir)
	if err != nil {
		return nil, fmt.Errorf("memory %q: %w", name, err)
	}
	if strings.TrimSpace(body) == "" {
		return []renderer.FidelityNote{renderer.NewNote(
			renderer.LevelWarning,
			"claude",
			"memory",
			name,
			"instructions",
			renderer.CodeMemoryBodyEmpty,
			"memory entry has no instructions or instructions-file content; skipping",
			"Provide instructions or instructions-file in the .xcf memory entry.",
		)}, nil
	}

	content := renderMemoryMarkdown(entry, body)
	newHash := hashSHA256(content)

	lifecycle := entry.Lifecycle
	if lifecycle == "" {
		lifecycle = memoryLifecycleSeedOnce
	}

	if name == ".." || strings.Contains(name, "..") {
		return nil, fmt.Errorf("memory %q: entry name must not contain traversal sequences", name)
	}
	if filepath.IsAbs(name) {
		return nil, fmt.Errorf("memory %q: entry name must not be absolute", name)
	}

	safeName := renderer.SlugifyFilename(name)
	targetPath := filepath.Join(r.targetDir, safeName+".md")
	exists, err := fileExists(targetPath)
	if err != nil {
		return nil, fmt.Errorf("memory %q: stat target: %w", name, err)
	}

	// Issue 4: reject unknown lifecycle values rather than silently treating them as seed-once.
	switch lifecycle {
	case "", memoryLifecycleSeedOnce:
		lifecycle = memoryLifecycleSeedOnce
	case memoryLifecycleTracked:
		// ok
	default:
		return nil, fmt.Errorf("memory %q: unknown lifecycle %q (expected seed-once or tracked)", name, entry.Lifecycle)
	}

	switch lifecycle {
	case memoryLifecycleSeedOnce:
		return r.applySeedOnce(name, targetPath, content, newHash, lifecycle, exists)
	default: // memoryLifecycleTracked
		return r.applyTracked(name, targetPath, content, newHash, lifecycle, exists, priorHashes)
	}
}

// applySeedOnce writes the entry only when the target file is absent, unless
// reseed is enabled. A FidelityNote is emitted for the no-op case.
func (r *MemoryRenderer) applySeedOnce(name, targetPath, content, newHash, lifecycle string, exists bool) ([]renderer.FidelityNote, error) {
	if !exists || r.reseed {
		notes, err := r.writeEntry(name, targetPath, content, newHash, lifecycle)
		if err != nil {
			return nil, err
		}
		return notes, nil
	}

	return []renderer.FidelityNote{renderer.NewNote(
		renderer.LevelInfo,
		"claude",
		"memory",
		name,
		"",
		renderer.CodeMemorySeedSkipped,
		"file exists; seed-once lifecycle preserves existing content",
		"use --reseed to overwrite",
	)}, nil
}

// applyTracked enforces drift detection against the prior seed hash. If the
// on-disk hash diverges from priorHashes[name] and reseed is off, a drift
// error is returned.
func (r *MemoryRenderer) applyTracked(name, targetPath, content, newHash, lifecycle string, exists bool, priorHashes map[string]string) ([]renderer.FidelityNote, error) {
	if !exists {
		notes, err := r.writeEntry(name, targetPath, content, newHash, lifecycle)
		if err != nil {
			return nil, err
		}
		return notes, nil
	}

	prior, hasPrior := priorHashes[name]
	if !hasPrior {
		// First tracked apply — adopt the existing file as the new seed baseline.
		notes, err := r.writeEntry(name, targetPath, content, newHash, lifecycle)
		if err != nil {
			return nil, err
		}
		return notes, nil
	}

	onDisk, err := hashFile(targetPath)
	if err != nil {
		return nil, fmt.Errorf("memory %q: hash existing file: %w", name, err)
	}

	if onDisk == prior {
		// No drift — xcf is authoritative, write the new content.
		notes, err := r.writeEntry(name, targetPath, content, newHash, lifecycle)
		if err != nil {
			return nil, err
		}
		return notes, nil
	}

	// Drift detected.
	if r.reseed {
		notes, err := r.writeEntry(name, targetPath, content, newHash, lifecycle)
		if err != nil {
			return nil, err
		}
		return notes, nil
	}

	driftNote := renderer.NewNote(
		renderer.LevelError,
		"claude",
		"memory",
		name,
		"",
		renderer.CodeMemoryDriftDetected,
		fmt.Sprintf("on-disk hash %s diverges from last-seed hash %s; entry was modified after last apply", onDisk, prior),
		"To capture agent changes: xcaffold import --with-memory\nTo discard agent changes and re-apply: xcaffold apply --include-memory --reseed",
	)
	return []renderer.FidelityNote{driftNote}, fmt.Errorf(
		"memory drift detected for entry %q\n  target: claude\n  path: %s\n  last-seed: %s\n  on-disk: %s (modified after last seed)\n  To capture agent changes: xcaffold import --with-memory\n  To discard agent changes and re-apply: xcaffold apply --include-memory --reseed",
		name, targetPath, prior, onDisk,
	)
}

// writeEntry persists the memory file, records a MemorySeed for the lock
// manifest, then appends the entry to the MEMORY.md index. The seed is recorded
// before the index update so that drift detection is never compromised by an
// index-append failure. A failed index update is downgraded to a warning note.
func (r *MemoryRenderer) writeEntry(name, targetPath, content, newHash, lifecycle string) ([]renderer.FidelityNote, error) {
	if err := os.MkdirAll(r.targetDir, 0o700); err != nil {
		return nil, fmt.Errorf("memory %q: create target dir: %w", name, err)
	}
	if err := os.WriteFile(targetPath, []byte(content), 0o600); err != nil {
		return nil, fmt.Errorf("memory %q: write file: %w", name, err)
	}

	// Issue 2: record seed immediately after successful write so drift detection
	// is not affected by a subsequent index-append failure.
	r.seeds = append(r.seeds, MemorySeed{
		Name:      name,
		Target:    "claude",
		Path:      targetPath,
		Hash:      newHash,
		SeededAt:  time.Now().UTC().Format(time.RFC3339),
		Lifecycle: lifecycle,
	})

	var notes []renderer.FidelityNote
	if err := appendMemoryIndex(r.targetDir, name); err != nil {
		notes = append(notes, renderer.NewNote(
			renderer.LevelWarning,
			"claude",
			"memory",
			name,
			"",
			renderer.CodeMemoryIndexUpdateFailed,
			fmt.Sprintf("failed to update MEMORY.md index: %v", err),
			"MEMORY.md index is advisory; drift detection unaffected",
		))
	}
	return notes, nil
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

	// Issue 3: reject absolute paths or any traversal that escapes baseDir.
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

// renderMemoryMarkdown composes the final memory file content: YAML frontmatter
// followed by the body. Output always ends with a trailing newline.
func renderMemoryMarkdown(entry ast.MemoryConfig, body string) string {
	var sb strings.Builder
	sb.WriteString("---\n")
	if entry.Type != "" {
		fmt.Fprintf(&sb, "type: %s\n", entry.Type)
	}
	if entry.Description != "" {
		// Issue 5: use %q (Go double-quoted scalar) which is a valid YAML
		// double-quoted string, safely escaping colons, newlines, and other
		// YAML-special characters.
		fmt.Fprintf(&sb, "description: %q\n", entry.Description)
	}
	sb.WriteString("---\n\n")
	sb.WriteString(strings.TrimRight(body, "\n"))
	sb.WriteString("\n")
	return sb.String()
}

// appendMemoryIndex ensures MEMORY.md in targetDir contains a "## xcaffold
// seeds" section listing the given entry name. The operation is idempotent:
// running twice does not duplicate the entry.
func appendMemoryIndex(targetDir, name string) error {
	indexPath := filepath.Join(targetDir, "MEMORY.md")
	existing, err := os.ReadFile(indexPath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	content := string(existing)
	listItem := "- " + name

	if !strings.Contains(content, memoryIndexSection) {
		var sb strings.Builder
		sb.WriteString(content)
		if content != "" && !strings.HasSuffix(content, "\n") {
			sb.WriteString("\n")
		}
		if content != "" {
			sb.WriteString("\n")
		}
		sb.WriteString(memoryIndexSection)
		sb.WriteString("\n\n")
		sb.WriteString(listItem)
		sb.WriteString("\n")
		return os.WriteFile(indexPath, []byte(sb.String()), 0o600)
	}

	// Section exists — append list item if not already present within the
	// xcaffold seeds section (bounded by the next "## " heading or EOF).
	if memoryIndexContainsItem(content, name) {
		return nil
	}

	updated := insertIntoMemorySection(content, listItem)
	return os.WriteFile(indexPath, []byte(updated), 0o600)
}

// memoryIndexContainsItem reports whether the xcaffold seeds section of the
// index already lists `- <name>`.
func memoryIndexContainsItem(content, name string) bool {
	section := extractMemorySection(content)
	for _, line := range strings.Split(section, "\n") {
		if strings.TrimSpace(line) == "- "+name {
			return true
		}
	}
	return false
}

// extractMemorySection returns the body of the xcaffold seeds section (without
// the heading), bounded by the next "## " heading or EOF.
func extractMemorySection(content string) string {
	idx := strings.Index(content, memoryIndexSection)
	if idx < 0 {
		return ""
	}
	rest := content[idx+len(memoryIndexSection):]
	// Find the next top-level section heading.
	next := strings.Index(rest, "\n## ")
	if next < 0 {
		return rest
	}
	return rest[:next]
}

// insertIntoMemorySection appends listItem to the xcaffold seeds section,
// immediately before the next "## " heading (or at EOF).
func insertIntoMemorySection(content, listItem string) string {
	idx := strings.Index(content, memoryIndexSection)
	if idx < 0 {
		return content
	}
	headingEnd := idx + len(memoryIndexSection)
	rest := content[headingEnd:]
	next := strings.Index(rest, "\n## ")

	var insertAt int
	if next < 0 {
		insertAt = len(content)
	} else {
		insertAt = headingEnd + next
	}

	before := content[:insertAt]
	after := content[insertAt:]

	// Ensure separator newline before the list item.
	if !strings.HasSuffix(before, "\n") {
		before += "\n"
	}
	return before + listItem + "\n" + after
}

// hashSHA256 returns the "sha256:<hex>" hash of the given content.
func hashSHA256(content string) string {
	sum := sha256.Sum256([]byte(content))
	return "sha256:" + hex.EncodeToString(sum[:])
}

// hashFile returns the "sha256:<hex>" hash of the file at path.
func hashFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return hashSHA256(string(data)), nil
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
