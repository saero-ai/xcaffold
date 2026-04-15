package main

import (
	"fmt"
	"io"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/renderer"
)

// printFidelityNotes writes fidelity notes to w in the human-readable format
// previously used by individual renderers. It returns the count of notes that
// were printed (i.e. not suppressed).
//
// suppressedResources is a set of resource names for which
// suppress_fidelity_warnings was set to true in the source config.
//
// strict controls whether LevelWarning notes are promoted to errors in the
// output message. It does not change the return count.
func printFidelityNotes(w io.Writer, notes []renderer.FidelityNote, suppressedResources map[string]bool, strict bool) int {
	printed := 0
	for _, n := range notes {
		if suppressedResources[n.Resource] {
			continue
		}
		prefix := "WARNING"
		switch {
		case n.Level == renderer.LevelError:
			prefix = "ERROR"
		case strict && n.Level == renderer.LevelWarning:
			prefix = "ERROR"
		case n.Level == renderer.LevelInfo:
			prefix = "INFO"
		}
		fmt.Fprintf(w, "%s (%s): %s\n", prefix, n.Target, n.Reason)
		printed++
	}
	return printed
}

// buildSuppressedResourcesMap returns a set of resource names whose
// suppress_fidelity_warnings override is set to true for the given target.
// Currently only agent-level overrides are honoured, matching the behaviour
// encoded in the renderer packages prior to the fidelity-note migration.
func buildSuppressedResourcesMap(config *ast.XcaffoldConfig, target string) map[string]bool {
	result := make(map[string]bool)
	if config == nil {
		return result
	}
	for name, agent := range config.Agents {
		override, ok := agent.Targets[target]
		if !ok {
			continue
		}
		if override.SuppressFidelityWarnings != nil && *override.SuppressFidelityWarnings {
			result[name] = true
		}
	}
	return result
}
