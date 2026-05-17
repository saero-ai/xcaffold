package main

import (
	"reflect"

	"github.com/saero-ai/xcaffold/internal/ast"
)

// DiffEntry represents a single resource in a diff result.
type DiffEntry struct {
	Name string
	Kind string
}

// ResourceDiff categorizes resources into new, changed, unchanged, and xcaf-only.
type ResourceDiff struct {
	New       map[string][]DiffEntry
	Changed   map[string][]DiffEntry
	Unchanged map[string][]DiffEntry
	XcafOnly  map[string][]DiffEntry
}

// newResourceDiff creates a ResourceDiff with initialized maps.
func newResourceDiff() ResourceDiff {
	return ResourceDiff{
		New:       make(map[string][]DiffEntry),
		Changed:   make(map[string][]DiffEntry),
		Unchanged: make(map[string][]DiffEntry),
		XcafOnly:  make(map[string][]DiffEntry),
	}
}

// stripRuntimeFields removes fields that are set at runtime but not persisted to YAML.
// These fields (SourceProvider, Inherited, Body, SourceFile, Targets) should not cause a resource to be
// categorized as "Changed" during diff comparison.
func stripRuntimeFields(config *ast.XcaffoldConfig) {
	for name, a := range config.Agents {
		a.SourceProvider = ""
		a.SourceFile = ""
		a.Inherited = false
		a.Body = ""
		a.Targets = nil
		config.Agents[name] = a
	}
	for name, s := range config.Skills {
		s.SourceProvider = ""
		s.SourceFile = ""
		s.Inherited = false
		s.Body = ""
		s.Targets = nil
		config.Skills[name] = s
	}
	for name, r := range config.Rules {
		r.SourceProvider = ""
		r.SourceFile = ""
		r.Inherited = false
		r.Body = ""
		r.Targets = nil
		config.Rules[name] = r
	}
	for name, w := range config.Workflows {
		w.SourceProvider = ""
		w.SourceFile = ""
		w.Inherited = false
		w.Targets = nil
		config.Workflows[name] = w
	}
	for name, m := range config.MCP {
		m.SourceProvider = ""
		m.SourceFile = ""
		m.Targets = nil
		config.MCP[name] = m
	}
}

// diffResources compares scanned provider resources against existing xcaf/ resources
// and returns a ResourceDiff categorizing each resource.
// NOTE: This function modifies scanned and existing IN-PLACE by stripping runtime fields.
// Callers must preserve these objects before calling diffResources if they need the
// original values after the diff.
func diffResources(scanned, existing *ast.XcaffoldConfig) ResourceDiff {
	// Strip runtime-only fields before comparison to prevent false "Changed" categorization.
	// Both scanned and existing are working copies, so mutation is acceptable.
	stripRuntimeFields(scanned)
	stripRuntimeFields(existing)

	diff := newResourceDiff()
	diffKind(diff, "agent", scanned.Agents, existing.Agents)
	diffKind(diff, "skill", scanned.Skills, existing.Skills)
	diffKind(diff, "rule", scanned.Rules, existing.Rules)
	diffKind(diff, "workflow", scanned.Workflows, existing.Workflows)
	diffKind(diff, "mcp", scanned.MCP, existing.MCP)
	return diff
}

// diffKind compares scanned resources of a given kind against existing ones,
// populating the diff result with categorized entries.
func diffKind[T any](diff ResourceDiff, kind string, scanned, existing map[string]T) {
	// Process scanned resources
	for name, scannedVal := range scanned {
		entry := DiffEntry{Name: name, Kind: kind}
		existingVal, exists := existing[name]
		if !exists {
			diff.New[kind] = append(diff.New[kind], entry)
		} else if reflect.DeepEqual(scannedVal, existingVal) {
			diff.Unchanged[kind] = append(diff.Unchanged[kind], entry)
		} else {
			diff.Changed[kind] = append(diff.Changed[kind], entry)
		}
	}

	// Find xcaf-only resources (in existing but not in scanned)
	for name := range existing {
		if _, inScanned := scanned[name]; !inScanned {
			diff.XcafOnly[kind] = append(diff.XcafOnly[kind], DiffEntry{Name: name, Kind: kind})
		}
	}
}

// TotalNew returns the total count of new resources across all kinds.
func (d ResourceDiff) TotalNew() int {
	total := 0
	for _, entries := range d.New {
		total += len(entries)
	}
	return total
}

// TotalChanged returns the total count of changed resources across all kinds.
func (d ResourceDiff) TotalChanged() int {
	total := 0
	for _, entries := range d.Changed {
		total += len(entries)
	}
	return total
}

// TotalUnchanged returns the total count of unchanged resources across all kinds.
func (d ResourceDiff) TotalUnchanged() int {
	total := 0
	for _, entries := range d.Unchanged {
		total += len(entries)
	}
	return total
}

// TotalXcafOnly returns the total count of xcaf-only resources across all kinds.
func (d ResourceDiff) TotalXcafOnly() int {
	total := 0
	for _, entries := range d.XcafOnly {
		total += len(entries)
	}
	return total
}

// copyResource copies a single resource from src to dst by kind and name.
func copyResource(dst, src *ast.XcaffoldConfig, kind, name string) {
	switch kind {
	case "agent":
		copyAgentResource(dst, src, name)
	case "skill":
		copySkillResource(dst, src, name)
	case "rule":
		copyRuleResource(dst, src, name)
	case "workflow":
		copyWorkflowResource(dst, src, name)
	case "mcp":
		copyMCPResource(dst, src, name)
	}
}

func copyAgentResource(dst, src *ast.XcaffoldConfig, name string) {
	if src.Agents != nil {
		if dst.Agents == nil {
			dst.Agents = make(map[string]ast.AgentConfig)
		}
		dst.Agents[name] = src.Agents[name]
	}
}

func copySkillResource(dst, src *ast.XcaffoldConfig, name string) {
	if src.Skills != nil {
		if dst.Skills == nil {
			dst.Skills = make(map[string]ast.SkillConfig)
		}
		dst.Skills[name] = src.Skills[name]
	}
}

func copyRuleResource(dst, src *ast.XcaffoldConfig, name string) {
	if src.Rules != nil {
		if dst.Rules == nil {
			dst.Rules = make(map[string]ast.RuleConfig)
		}
		dst.Rules[name] = src.Rules[name]
	}
}

func copyWorkflowResource(dst, src *ast.XcaffoldConfig, name string) {
	if src.Workflows != nil {
		if dst.Workflows == nil {
			dst.Workflows = make(map[string]ast.WorkflowConfig)
		}
		dst.Workflows[name] = src.Workflows[name]
	}
}

func copyMCPResource(dst, src *ast.XcaffoldConfig, name string) {
	if src.MCP != nil {
		if dst.MCP == nil {
			dst.MCP = make(map[string]ast.MCPConfig)
		}
		dst.MCP[name] = src.MCP[name]
	}
}
