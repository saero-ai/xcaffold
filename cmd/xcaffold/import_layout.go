package main

import (
	"os"
	"path/filepath"
	"strings"
)

// layoutMode represents the organization structure of .xcaf files in a project.
type layoutMode int

const (
	// layoutNested means .xcaf files are organized in subdirectories
	// (e.g., xcaf/rules/security/rule.xcaf)
	layoutNested layoutMode = iota
	// layoutFlat means .xcaf files are directly in the kind directory
	// (e.g., xcaf/rules/security.xcaf)
	layoutFlat
)

// detectLayout analyzes the structure of a kind directory (e.g., "rules", "agents")
// and determines whether .xcaf files are stored flat or nested.
//
// Returns layoutFlat if >= 50% of entries in the kind directory are .xcaf files,
// otherwise returns layoutNested (including when the directory is empty or doesn't exist).
func detectLayout(xcafDir, kind string) layoutMode {
	kindDir := filepath.Join(xcafDir, kindToPlural(kind))

	entries, err := os.ReadDir(kindDir)
	if err != nil {
		// Directory doesn't exist or can't be read — default to nested
		return layoutNested
	}

	var flatCount, nestedCount int
	for _, entry := range entries {
		if entry.IsDir() {
			nestedCount++
		} else if strings.HasSuffix(entry.Name(), ".xcaf") {
			flatCount++
		}
	}

	// If no .xcaf files or directories found, default to nested
	if flatCount == 0 && nestedCount == 0 {
		return layoutNested
	}

	// Majority vote: flat wins if >= 50%
	total := flatCount + nestedCount
	if flatCount*2 >= total {
		return layoutFlat
	}
	return layoutNested
}

// kindToPlural converts a resource kind to its plural form.
// Used to construct the directory name where .xcaf files are stored.
func kindToPlural(kind string) string {
	switch kind {
	case "mcp":
		return "mcp"
	default:
		return kind + "s"
	}
}
