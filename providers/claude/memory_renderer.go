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
// (internal/state -> internal/compiler -> providers/claude). Callers
// (e.g. runApply) translate these into state.MemorySeed by direct field copy
// before handing them to state.GenerateWithOpts.
//
// One MemorySeed is recorded per agent directory (keyed by AgentRef).
// Name holds the AgentRef value.
type MemorySeed struct {
	Name     string
	Target   string
	Path     string
	Hash     string
	SeededAt string
}

// MemoryRenderer writes memory entries into a Claude project memory directory.
// Entries are grouped by AgentRef, and for each agent the renderer produces:
//   - MEMORY.md: a link-list index of all entries
//   - Individual .md files: one per entry, containing the entry's Content
//
// There is no seed-once behavior; apply always overwrites.
type MemoryRenderer struct {
	targetDir string
	seeds     []MemorySeed
}

// NewMemoryRenderer constructs a MemoryRenderer rooted at targetDir (the
// resolved path to the Claude project memory directory).
func NewMemoryRenderer(targetDir string) *MemoryRenderer {
	return &MemoryRenderer{targetDir: targetDir}
}

// Seeds returns MemorySeed records produced by the last Compile call. Used by
// runApply to feed state.GenerateWithOpts.
func (r *MemoryRenderer) Seeds() []MemorySeed {
	out := make([]MemorySeed, len(r.seeds))
	copy(out, r.seeds)
	return out
}

// Compile writes each agent's MEMORY.md index and individual .md files,
// always overwriting existing content. Returns fidelity notes (currently none).
func (r *MemoryRenderer) Compile(config *ast.XcaffoldConfig, baseDir string) (*output.Output, []renderer.FidelityNote, error) {
	out := &output.Output{Files: make(map[string]string)}
	var notes []renderer.FidelityNote
	r.seeds = nil

	if config == nil || len(config.Memory) == 0 {
		return out, notes, nil
	}

	grouped, err := groupMemoryEntries(config.Memory)
	if err != nil {
		return nil, notes, err
	}

	agentRefs := make([]string, 0, len(grouped))
	for ref := range grouped {
		agentRefs = append(agentRefs, ref)
	}
	sort.Strings(agentRefs)

	for _, agentRef := range agentRefs {
		seed, err := r.writeAgentMemory(agentRef, grouped[agentRef])
		if err != nil {
			return nil, notes, err
		}
		r.seeds = append(r.seeds, seed)
	}

	return out, notes, nil
}

// memoryEntryData holds the resolved fields for a single memory entry.
type memoryEntryData struct {
	key     string
	fname   string
	name    string
	desc    string
	content string
}

// groupMemoryEntries sorts and validates memory entries, grouping them by agent-ref.
func groupMemoryEntries(memory map[string]ast.MemoryConfig) (map[string][]memoryEntryData, error) {
	names := make([]string, 0, len(memory))
	for name := range memory {
		names = append(names, name)
	}
	sort.Strings(names)

	grouped := make(map[string][]memoryEntryData)
	for _, key := range names {
		entry := memory[key]
		if strings.TrimSpace(entry.Content) == "" {
			continue
		}
		agentRef := entry.AgentRef
		if agentRef == "" {
			agentRef = "default"
		}
		if agentRef == ".." || strings.Contains(agentRef, "..") {
			return nil, fmt.Errorf("memory %q: agent-ref %q must not contain traversal sequences", key, agentRef)
		}
		if filepath.IsAbs(agentRef) {
			return nil, fmt.Errorf("memory %q: agent-ref %q must not be absolute", key, agentRef)
		}
		fname := key
		if parts := strings.SplitN(key, "/", 2); len(parts) == 2 {
			fname = parts[1]
		}
		grouped[agentRef] = append(grouped[agentRef], memoryEntryData{
			key: key, fname: fname + ".md",
			name: entry.Name, desc: entry.Description, content: entry.Content,
		})
	}
	return grouped, nil
}

// writeAgentMemory writes MEMORY.md and individual entry files for one agent-ref
// directory, then returns a MemorySeed for the written index.
func (r *MemoryRenderer) writeAgentMemory(agentRef string, entries []memoryEntryData) (MemorySeed, error) {
	agentDir := filepath.Join(r.targetDir, agentRef)
	if err := os.MkdirAll(agentDir, 0o700); err != nil {
		return MemorySeed{}, fmt.Errorf("memory agent-ref %q: create dir: %w", agentRef, err)
	}

	var indexBuf strings.Builder
	for _, e := range entries {
		fmt.Fprintf(&indexBuf, "- [%s](%s) — %s\n", e.name, e.fname, e.desc)
		indivPath := filepath.Join(agentDir, e.fname)
		if err := os.WriteFile(indivPath, []byte(e.content+"\n"), 0o600); err != nil {
			return MemorySeed{}, fmt.Errorf("memory %q: write file: %w", e.key, err)
		}
	}

	indexContent := indexBuf.String()
	indexPath := filepath.Join(agentDir, "MEMORY.md")
	if err := os.WriteFile(indexPath, []byte(indexContent), 0o600); err != nil {
		return MemorySeed{}, fmt.Errorf("memory agent-ref %q: write MEMORY.md: %w", agentRef, err)
	}

	return MemorySeed{
		Name:     agentRef,
		Target:   "claude",
		Path:     indexPath,
		Hash:     hashSHA256(indexContent),
		SeededAt: time.Now().UTC().Format(time.RFC3339),
	}, nil
}

// hashSHA256 returns the "sha256:<hex>" hash of the given content.
func hashSHA256(content string) string {
	sum := sha256.Sum256([]byte(content))
	return "sha256:" + hex.EncodeToString(sum[:])
}
