package analyzer

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
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
// and returns on-disk artifacts not declared in the XCF.
//
//nolint:gocyclo
func (a *Analyzer) ScanOutputDir(dir string, declared map[string]bool) ([]ArtifactEntry, error) {
	var entries []ArtifactEntry

	source := "disk"
	if strings.Contains(dir, ".claude") {
		source = "disk-claude"
	} else if strings.Contains(dir, ".agents") {
		source = "disk-agents"
	}

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

		// .agents/rules/foo.md -> kind="rule", id="foo"
		// .agents/skills/foo/SKILL.md -> kind="skill", id="foo"

		var kind string
		var id string

		//nolint:goconst
		switch parts[0] {
		case "agents":
			if len(parts) == 2 && strings.HasSuffix(parts[1], ".md") {
				kind, id = "agent", strings.TrimSuffix(parts[1], ".md")
			}
		case "rules":
			if len(parts) == 2 && strings.HasSuffix(parts[1], ".md") {
				kind, id = "rule", strings.TrimSuffix(parts[1], ".md")
			}
		case "skills":
			if len(parts) == 3 && parts[2] == "SKILL.md" {
				kind, id = "skill", parts[1]
			} else if len(parts) >= 3 {
				// Skip sub-files under skill directories — too granular.
				return nil
			}
		case "workflows":
			if len(parts) == 2 && strings.HasSuffix(parts[1], ".md") {
				kind, id = "workflow", strings.TrimSuffix(parts[1], ".md")
			}
		case "hooks":
			if len(parts) == 2 && strings.HasSuffix(parts[1], ".sh") {
				kind, id = "hook", parts[1] // ID is the script name
			}
		default:
			return nil
		}

		// Skip if this artifact is managed by XCF
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
