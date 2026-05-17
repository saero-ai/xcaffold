package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/saero-ai/xcaffold/internal/ast"
)

// ConflictReport describes a resource where multiple providers have diverged
// from the existing .xcaf source in different ways.
type ConflictReport struct {
	ResourceName string
	Kind         string
	Providers    []ProviderVersion
}

// ProviderVersion represents a provider's version of a resource body.
type ProviderVersion struct {
	Provider string
	Body     string
	BodyLen  int
}

// detectConflicts identifies resources present in 2+ providers where the providers
// have diverged DIFFERENTLY from the existing .xcaf source.
// If only 1 provider diverged, or all diverged identically, no conflict is reported.
func detectConflicts(providerConfigs map[string]*ast.XcaffoldConfig, existing *ast.XcaffoldConfig) []ConflictReport {
	var conflicts []ConflictReport

	conflicts = append(conflicts, detectKindConflicts(kindRule, providerConfigs, existing)...)
	conflicts = append(conflicts, detectKindConflicts(kindAgent, providerConfigs, existing)...)
	conflicts = append(conflicts, detectKindConflicts(kindSkill, providerConfigs, existing)...)

	return conflicts
}

// resourceVersion groups a provider and its body for a resource.
type resourceVersion struct {
	provider string
	body     string
}

// resourceCheck bundles data needed for conflict checking.
type resourceCheck struct {
	kind         string
	name         string
	versions     []resourceVersion
	existingBody string
}

// detectKindConflicts detects conflicts for a specific kind (rule, agent, skill).
func detectKindConflicts(kind string, providerConfigs map[string]*ast.XcaffoldConfig, existing *ast.XcaffoldConfig) []ConflictReport {
	byName := make(map[string][]resourceVersion)

	// Collect versions from all providers
	for provider, cfg := range providerConfigs {
		resources := getResourceBodies(cfg, kind)
		for name, body := range resources {
			byName[name] = append(byName[name], resourceVersion{provider: provider, body: body})
		}
	}

	var conflicts []ConflictReport
	names := make([]string, 0, len(byName))
	for name := range byName {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		versions := byName[name]
		if len(versions) < 2 {
			continue
		}

		existingBody := getExistingBody(existing, kind, name)
		check := resourceCheck{kind: kind, name: name, versions: versions, existingBody: existingBody}
		if conflict := buildConflict(check); conflict != nil {
			conflicts = append(conflicts, *conflict)
		}
	}

	return conflicts
}

// buildConflict determines if a resource has a conflict and returns it.
// Returns nil if 0-1 providers diverged, or if all diverged identically.
func buildConflict(check resourceCheck) *ConflictReport {
	// Count providers that diverged from existing source
	diverged := make([]resourceVersion, 0)
	for _, v := range check.versions {
		if v.body != check.existingBody {
			diverged = append(diverged, v)
		}
	}

	// No conflict if fewer than 2 providers diverged
	if len(diverged) < 2 {
		return nil
	}

	// Check if all diverged versions are the same (not a conflict)
	for i := 1; i < len(diverged); i++ {
		if diverged[i].body != diverged[0].body {
			// Real conflict: providers diverged differently
			var pvs []ProviderVersion
			for _, v := range diverged {
				pvs = append(pvs, ProviderVersion{
					Provider: v.provider,
					Body:     v.body,
					BodyLen:  len(v.body),
				})
			}
			return &ConflictReport{
				ResourceName: check.name,
				Kind:         check.kind,
				Providers:    pvs,
			}
		}
	}
	return nil
}

// getResourceBodies extracts all resource bodies of a given kind from a config.
func getResourceBodies(cfg *ast.XcaffoldConfig, kind string) map[string]string {
	bodies := make(map[string]string)
	switch kind {
	case kindRule:
		for name, r := range cfg.Rules {
			bodies[name] = r.Body
		}
	case kindAgent:
		for name, a := range cfg.Agents {
			bodies[name] = a.Body
		}
	case kindSkill:
		for name, s := range cfg.Skills {
			bodies[name] = s.Body
		}
	}
	return bodies
}

// getExistingBody retrieves a resource's body from the existing config, or empty string if not found.
func getExistingBody(cfg *ast.XcaffoldConfig, kind, name string) string {
	switch kind {
	case kindRule:
		if r, ok := cfg.Rules[name]; ok {
			return r.Body
		}
	case kindAgent:
		if a, ok := cfg.Agents[name]; ok {
			return a.Body
		}
	case kindSkill:
		if s, ok := cfg.Skills[name]; ok {
			return s.Body
		}
	}
	return ""
}

// resolveConflict applies body-priority scoring: the longest body becomes base,
// all others become overrides.
func resolveConflict(conflict ConflictReport) (base ProviderVersion, overrides []ProviderVersion) {
	// Body-priority: longest body becomes base
	best := 0
	for i, pv := range conflict.Providers {
		if pv.BodyLen > conflict.Providers[best].BodyLen {
			best = i
		}
	}
	base = conflict.Providers[best]
	for i, pv := range conflict.Providers {
		if i != best {
			overrides = append(overrides, pv)
		}
	}
	return base, overrides
}

// formatConflictSummary produces a human-readable display of conflicts.
func formatConflictSummary(conflicts []ConflictReport) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("\n  %s  %d conflict(s) detected:\n\n", colorYellow("!!"), len(conflicts)))
	for _, c := range conflicts {
		sb.WriteString(fmt.Sprintf("    %s/%s — diverged in: ", c.Kind, c.ResourceName))
		for i, pv := range c.Providers {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(fmt.Sprintf("%s (%d chars)", pv.Provider, pv.BodyLen))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

// handleConflicts applies conflict resolution: non-interactive uses body-priority,
// interactive prompts the user for each conflict.
func handleConflicts(conflicts []ConflictReport, interactive bool) map[string]ProviderVersion {
	resolved := make(map[string]ProviderVersion)
	if !interactive {
		for _, c := range conflicts {
			base, _ := resolveConflict(c)
			resolved[c.Kind+"/"+c.ResourceName] = base
		}
		return resolved
	}
	// Interactive: prompt user for each conflict
	fmt.Print(formatConflictSummary(conflicts))
	for _, c := range conflicts {
		choice := promptConflictChoice(c)
		resolved[c.Kind+"/"+c.ResourceName] = c.Providers[choice]
	}
	return resolved
}

// promptConflictChoice prompts the user to select which provider's version to use for a conflict.
// Returns the index into conflict.Providers of the selected version.
func promptConflictChoice(conflict ConflictReport) int {
	fmt.Printf("    Choose version for %s/%s:\n", conflict.Kind, conflict.ResourceName)
	for i, pv := range conflict.Providers {
		fmt.Printf("      [%d] %s (%d chars)\n", i+1, pv.Provider, pv.BodyLen)
	}
	fmt.Print("    Selection (1, 2, ...): ")

	var input string
	if _, err := fmt.Scanln(&input); err != nil {
		input = ""
	}

	// Simple fallback: if invalid, use longest body
	choice := 0
	var n int
	if _, err := fmt.Sscanf(input, "%d", &n); err != nil || n < 1 || n > len(conflict.Providers) {
		// Default to longest body
		for i, pv := range conflict.Providers {
			if pv.BodyLen > conflict.Providers[choice].BodyLen {
				choice = i
			}
		}
	} else {
		choice = n - 1
	}
	return choice
}
