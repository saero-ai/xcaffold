package main

import (
	"fmt"
	"io"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/renderer"
)

// printFidelityNotes writes fidelity notes to w in the human-readable format
// previously used by individual renderers. It returns the count of notes
// printed. Suppression must be applied by the caller before this function is
// invoked — pass renderer.FilterNotes(notes, suppressed) to pre-filter.
//
// strict controls whether LevelWarning notes are promoted to errors in the
// output message. It does not change the return count.
func printFidelityNotes(w io.Writer, notes []renderer.FidelityNote, strict bool) int {
	printed := 0
	for _, n := range notes {
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
// suppress-fidelity-warnings override is set to true for the given target.
func buildSuppressedResourcesMap(config *ast.XcaffoldConfig, target string) map[string]bool {
	result := make(map[string]bool)
	if config == nil {
		return result
	}
	for name, agent := range config.Agents {
		if isSuppressed(agent.Targets, target) {
			result[name] = true
		}
	}
	for name, skill := range config.Skills {
		if isSuppressed(skill.Targets, target) {
			result[name] = true
		}
	}
	for name, rule := range config.Rules {
		if isSuppressed(rule.Targets, target) {
			result[name] = true
		}
	}
	for name, wf := range config.Workflows {
		if isSuppressed(wf.Targets, target) {
			result[name] = true
		}
	}
	return result
}

func isSuppressed(targets map[string]ast.TargetOverride, target string) bool {
	override, ok := targets[target]
	if !ok {
		return false
	}
	return override.SuppressFidelityWarnings != nil && *override.SuppressFidelityWarnings
}
