package analyzer

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/saero-ai/xcaffold/providers"
)

// ArtifactEntry represents a single artifact detected in the output directory.
type ArtifactEntry struct {
	ID     string `json:"id"`
	Kind   string `json:"kind"`
	Source string `json:"source"`
}

// Analyzer handles static analysis of the AST without invoking network calls.
type Analyzer struct{}

// New returns a new Analyzer instance.
func New() *Analyzer {
	return &Analyzer{}
}

// ScanOutputDir walks a compiled output directory (.claude/ or .agents/)
// and returns on-disk artifacts not declared in the XCAF.
func (a *Analyzer) ScanOutputDir(dir string, declared map[string]bool) ([]ArtifactEntry, error) {
	var entries []ArtifactEntry
	source := resolveSourceLabel(dir)

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return nil
		}

		parts := strings.Split(filepath.ToSlash(rel), "/")
		if len(parts) == 0 {
			return nil
		}

		kind, id, skip := classifyPath(parts)
		if skip {
			return nil
		}

		if kind != "" && id != "" && !declared[fmt.Sprintf("%s:%s", kind, id)] {
			entries = append(entries, ArtifactEntry{
				ID:     id,
				Kind:   kind,
				Source: source,
			})
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Kind != entries[j].Kind {
			return entries[i].Kind < entries[j].Kind
		}
		return entries[i].ID < entries[j].ID
	})

	return entries, nil
}

// resolveSourceLabel determines the source label for a given output directory.
func resolveSourceLabel(dir string) string {
	for _, m := range providers.Manifests() {
		if m.OutputDir != "" && strings.Contains(dir, m.OutputDir) {
			return "disk-" + m.Name
		}
	}
	if strings.Contains(dir, ".agents") {
		return "disk-agents"
	}
	return "disk"
}

// classifyPath maps a slash-separated path parts slice to a (kind, id) pair.
// It returns skip=true when the path should be silently ignored (e.g. skill
// sub-files beneath the top-level SKILL.md).
//
//nolint:goconst
func classifyPath(parts []string) (kind, id string, skip bool) {
	switch parts[0] {
	case "agents":
		if len(parts) == 2 && strings.HasSuffix(parts[1], ".md") {
			return "agent", strings.TrimSuffix(parts[1], ".md"), false
		}
	case "rules":
		if len(parts) == 2 && strings.HasSuffix(parts[1], ".md") {
			return "rule", strings.TrimSuffix(parts[1], ".md"), false
		}
	case "skills":
		if len(parts) == 3 && parts[2] == "SKILL.md" {
			return "skill", parts[1], false
		}
		if len(parts) >= 3 {
			// Skip sub-files under skill directories — too granular.
			return "", "", true
		}
	case "workflows":
		if len(parts) == 2 && strings.HasSuffix(parts[1], ".md") {
			return "workflow", strings.TrimSuffix(parts[1], ".md"), false
		}
	case "hooks":
		if len(parts) == 2 && strings.HasSuffix(parts[1], ".sh") {
			return "hook", parts[1], false // ID is the script name
		}
	default:
		return "", "", true
	}
	return "", "", false
}
