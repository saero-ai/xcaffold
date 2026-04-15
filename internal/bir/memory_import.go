package bir

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// MemoryImportEntry represents a single memory file discovered in a provider's
// memory directory, normalized into xcaffold's canonical form.
type MemoryImportEntry struct {
	Key         string
	Type        string
	Description string
	Body        string
	SourcePath  string
}

// ImportOpts controls the import flow.
type ImportOpts struct {
	PlanOnly   bool
	Force      bool
	SidecarDir string
}

// ImportSummary describes the outcome of an import run.
type ImportSummary struct {
	Imported    int
	Skipped     int
	WouldImport int
	Written     []string
}

// DeriveMemoryKey normalizes a filename to a canonical kebab-case key.
// It strips the file extension, lowercases, replaces spaces/underscores/dots
// with hyphens, collapses consecutive hyphens, and trims leading/trailing hyphens.
func DeriveMemoryKey(filename string) string {
	// Strip extension (everything after and including the last dot,
	// but only if the result is non-empty).
	base := filename
	if ext := filepath.Ext(filename); ext != "" {
		base = strings.TrimSuffix(filename, ext)
	}

	// Lowercase.
	base = strings.ToLower(base)

	// Replace spaces, underscores, and dots with hyphens.
	var b strings.Builder
	for _, r := range base {
		switch r {
		case ' ', '_', '.':
			b.WriteRune('-')
		default:
			b.WriteRune(r)
		}
	}
	result := b.String()

	// Collapse consecutive hyphens.
	for strings.Contains(result, "--") {
		result = strings.ReplaceAll(result, "--", "-")
	}

	// Trim leading and trailing hyphens.
	result = strings.Trim(result, "-")

	return result
}

// memoryFrontmatter holds the parsed YAML keys from a memory file's frontmatter.
type memoryFrontmatter struct {
	Type        string `yaml:"type"`
	Description string `yaml:"description"`
}

// DiscoverClaudeMemory reads all *.md files at the top level of memDir and
// returns a slice of MemoryImportEntry values sorted by Key for determinism.
func DiscoverClaudeMemory(memDir string) ([]MemoryImportEntry, error) {
	pattern := filepath.Join(memDir, "*.md")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return []MemoryImportEntry{}, err
	}

	entries := make([]MemoryImportEntry, 0, len(matches))

	for _, path := range matches {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", path, err)
		}

		content := strings.ReplaceAll(string(data), "\r\n", "\n")
		fm, body := parseFrontmatterAndBody(content)

		memType := fm.Type
		if memType == "" {
			memType = "project"
		}

		entries = append(entries, MemoryImportEntry{
			Key:         DeriveMemoryKey(filepath.Base(path)),
			Type:        memType,
			Description: fm.Description,
			Body:        body,
			SourcePath:  path,
		})
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Key < entries[j].Key
	})

	return entries, nil
}

// parseFrontmatterAndBody splits content into YAML frontmatter and body.
// If no frontmatter delimiter is found, the full content is returned as the body.
func parseFrontmatterAndBody(content string) (memoryFrontmatter, string) {
	lines := strings.Split(content, "\n")

	var fm memoryFrontmatter

	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return fm, strings.TrimLeft(content, "\n")
	}

	// Find closing ---
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			fmBlock := strings.Join(lines[1:i], "\n")
			_ = yaml.Unmarshal([]byte(fmBlock), &fm)

			body := strings.Join(lines[i+1:], "\n")
			body = strings.TrimLeft(body, "\n")
			return fm, body
		}
	}

	// No closing ---, treat whole content as body.
	return fm, strings.TrimLeft(content, "\n")
}

// WriteSidecar writes a memory entry as a sidecar markdown file under sidecarDir.
// The file is named <entry.Key>.md and contains YAML frontmatter followed by the body.
func WriteSidecar(sidecarDir string, entry MemoryImportEntry) error {
	// Guard against path traversal via a crafted Key attribute.
	if entry.Key == "" || strings.ContainsAny(entry.Key, "/\\") || entry.Key == ".." || strings.Contains(entry.Key, "..") || filepath.IsAbs(entry.Key) {
		return fmt.Errorf("memory sidecar: unsafe key %q (contains path separator, traversal, or absolute)", entry.Key)
	}

	if err := os.MkdirAll(sidecarDir, 0o755); err != nil {
		return fmt.Errorf("creating sidecar directory: %w", err)
	}

	var sb strings.Builder
	sb.WriteString("---\n")
	fmt.Fprintf(&sb, "type: %s\n", entry.Type)
	fmt.Fprintf(&sb, "description: %q\n", entry.Description)
	sb.WriteString("---\n\n")
	sb.WriteString(entry.Body)
	if !strings.HasSuffix(entry.Body, "\n") {
		sb.WriteRune('\n')
	}

	dest := filepath.Join(sidecarDir, entry.Key+".md")
	// Defense-in-depth: confirm the resolved path stays inside sidecarDir.
	rel, err := filepath.Rel(sidecarDir, dest)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("memory sidecar: key %q escapes sidecar dir", entry.Key)
	}

	if err := os.WriteFile(dest, []byte(sb.String()), 0o600); err != nil {
		return fmt.Errorf("writing sidecar %s: %w", dest, err)
	}
	return nil
}

// HandleConflict determines whether an entry should be skipped based on whether
// a sidecar already exists and the force flag.
// Returns (skipped bool, warning string).
func HandleConflict(existing map[string]bool, entry MemoryImportEntry, force bool) (bool, string) {
	if !existing[entry.Key] {
		return false, ""
	}
	if force {
		return false, fmt.Sprintf("overwriting memory entry %q", entry.Key)
	}
	return true, fmt.Sprintf(
		"skipping memory entry %q — sidecar already exists (use --force to overwrite)",
		entry.Key,
	)
}

// ImportClaudeMemory runs a full import from a Claude memory directory into
// xcaffold-managed sidecars. It respects PlanOnly and Force from opts.
func ImportClaudeMemory(memDir string, opts ImportOpts) (*ImportSummary, error) {
	entries, err := DiscoverClaudeMemory(memDir)
	if err != nil {
		return nil, err
	}

	// Build existing-sidecar index.
	existing := make(map[string]bool)
	if opts.SidecarDir != "" {
		pattern := filepath.Join(opts.SidecarDir, "*.md")
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return nil, fmt.Errorf("scanning sidecar directory: %w", err)
		}
		for _, m := range matches {
			key := DeriveMemoryKey(filepath.Base(m))
			existing[key] = true
		}
	}

	summary := &ImportSummary{}

	for _, entry := range entries {
		skipped, _ := HandleConflict(existing, entry, opts.Force)
		if skipped {
			summary.Skipped++
			continue
		}
		if opts.PlanOnly {
			summary.WouldImport++
			continue
		}
		if err := WriteSidecar(opts.SidecarDir, entry); err != nil {
			return nil, err
		}
		summary.Imported++
		summary.Written = append(summary.Written, filepath.Join(opts.SidecarDir, entry.Key+".md"))
	}

	return summary, nil
}

// ExtractGeminiMemoryBlocks scans a GEMINI.md-style string for xcaffold memory
// block markers and returns a MemoryImportEntry for each block found.
// Marker format:
//
//	<!-- xcaffold:memory name="..." type="..." seeded-at="..." -->
//	**name** (type): description text
//
//	body lines...
//	<!-- xcaffold:/memory -->
func ExtractGeminiMemoryBlocks(content string) ([]MemoryImportEntry, error) {
	const openPrefix = "<!-- xcaffold:memory "
	const openSuffix = " -->"
	const closeMarker = "<!-- xcaffold:/memory -->"

	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	var entries []MemoryImportEntry

	i := 0
	for i < len(lines) {
		line := strings.TrimSpace(lines[i])

		if !strings.HasPrefix(line, openPrefix) {
			i++
			continue
		}

		// Parse attributes from open marker.
		attrStr := strings.TrimPrefix(line, openPrefix)
		attrStr = strings.TrimSuffix(attrStr, openSuffix)

		name := extractAttr(attrStr, "name")
		memType := extractAttr(attrStr, "type")
		i++

		// Collect body lines until close marker.
		var bodyLines []string
		var description string

		for i < len(lines) {
			if strings.TrimSpace(lines[i]) == closeMarker {
				i++ // consume close marker
				break
			}
			bodyLines = append(bodyLines, lines[i])
			i++
		}

		// The first non-empty body line is the header: **name** (type): description
		// Strip it and extract description from it.
		startBody := 0
		for startBody < len(bodyLines) {
			trimmed := strings.TrimSpace(bodyLines[startBody])
			if trimmed == "" {
				startBody++
				continue
			}
			// Try to parse header line: **name** (type): description
			desc := extractHeaderDescription(trimmed)
			if desc != "" {
				description = desc
				startBody++
			}
			break
		}

		// Remaining lines form the body, trimmed of leading/trailing blank lines.
		remaining := bodyLines[startBody:]
		body := strings.TrimSpace(strings.Join(remaining, "\n"))

		entries = append(entries, MemoryImportEntry{
			Key:         name,
			Type:        memType,
			Description: description,
			Body:        body,
		})
	}

	return entries, nil
}

// extractAttr parses a simple key="value" attribute from an HTML-comment attribute string.
// It scans character by character — no regex.
func extractAttr(attrs, key string) string {
	search := key + `="`
	idx := strings.Index(attrs, search)
	if idx == -1 {
		return ""
	}
	start := idx + len(search)
	end := strings.Index(attrs[start:], `"`)
	if end == -1 {
		return ""
	}
	return attrs[start : start+end]
}

// ImportGeminiMemory reads GEMINI.md from geminiDir, extracts xcaffold-seeded
// memory blocks via ExtractGeminiMemoryBlocks, and writes a sidecar file per
// entry under opts.SidecarDir. It mirrors the ImportClaudeMemory flow.
// If GEMINI.md does not exist, the function returns an empty summary without error
// (Gemini memory is optional).
func ImportGeminiMemory(geminiDir string, opts ImportOpts) (*ImportSummary, error) {
	geminiPath := filepath.Join(geminiDir, "GEMINI.md")
	data, err := os.ReadFile(geminiPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &ImportSummary{}, nil
		}
		return nil, fmt.Errorf("reading GEMINI.md: %w", err)
	}

	entries, err := ExtractGeminiMemoryBlocks(string(data))
	if err != nil {
		return nil, fmt.Errorf("parsing GEMINI.md memory blocks: %w", err)
	}

	// Build existing-sidecar index.
	existing := make(map[string]bool)
	if opts.SidecarDir != "" {
		pattern := filepath.Join(opts.SidecarDir, "*.md")
		matches, globErr := filepath.Glob(pattern)
		if globErr != nil {
			return nil, fmt.Errorf("scanning sidecar directory: %w", globErr)
		}
		for _, m := range matches {
			key := DeriveMemoryKey(filepath.Base(m))
			existing[key] = true
		}
	}

	summary := &ImportSummary{}

	for _, entry := range entries {
		skipped, _ := HandleConflict(existing, entry, opts.Force)
		if skipped {
			summary.Skipped++
			continue
		}
		if opts.PlanOnly {
			summary.WouldImport++
			continue
		}
		if err := WriteSidecar(opts.SidecarDir, entry); err != nil {
			return nil, err
		}
		summary.Imported++
		summary.Written = append(summary.Written, filepath.Join(opts.SidecarDir, entry.Key+".md"))
	}

	return summary, nil
}

// extractHeaderDescription returns the description from a Gemini header line of the form:
//
//	**name** (type): description text
//
// Returns empty string if the line does not match the pattern.
func extractHeaderDescription(line string) string {
	// Find the last ": " separator after the "(type)" part.
	// Pattern: **<name>** (<type>): <description>
	colonIdx := strings.Index(line, "): ")
	if colonIdx == -1 {
		return ""
	}
	return strings.TrimSpace(line[colonIdx+3:])
}
