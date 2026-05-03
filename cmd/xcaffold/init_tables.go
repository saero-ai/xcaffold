package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/importer"
	"github.com/spf13/cobra"
)

// nativeKindSupport declares which resource kinds each provider natively
// stores on disk. Verified from provider ground truth (2026-05-03).
// This is distinct from renderer capabilities (which declare what the
// compiler can translate TO a provider, including cross-kind lowering).
var nativeKindSupport = map[string]map[importer.Kind]bool{
	"claude": {
		importer.KindAgent: true, importer.KindSkill: true,
		importer.KindRule: true, importer.KindMCP: true,
		importer.KindHookScript: true, importer.KindSettings: true,
		importer.KindMemory: true,
	},
	"gemini": {
		importer.KindAgent: true, importer.KindSkill: true,
		importer.KindRule: true, importer.KindMCP: true,
		importer.KindHookScript: true, importer.KindSettings: true,
	},
	"cursor": {
		importer.KindAgent: true, importer.KindSkill: true,
		importer.KindRule: true, importer.KindMCP: true,
		importer.KindHookScript: true,
	},
	"copilot": {
		importer.KindAgent: true, importer.KindSkill: true,
		importer.KindRule: true, importer.KindMCP: true,
		importer.KindHookScript: true,
	},
	"antigravity": {
		importer.KindSkill: true,
		importer.KindRule:  true, importer.KindWorkflow: true,
		importer.KindMCP: true,
	},
}

// kindDisplay defines the canonical display order and labels for resource kinds.
var kindDisplay = []struct {
	kind  importer.Kind
	label string
}{
	{importer.KindAgent, "Agents"},
	{importer.KindSkill, "Skills"},
	{importer.KindRule, "Rules"},
	{importer.KindWorkflow, "Workflows"},
	{importer.KindMCP, "MCP"},
	{importer.KindHookScript, "Hooks"},
	{importer.KindSettings, "Settings"},
	{importer.KindMemory, "Memory"},
}

// configKindCount returns the count of resources in an XcaffoldConfig
// for a given importer.Kind.
func configKindCount(
	config *ast.XcaffoldConfig,
	k importer.Kind,
) int {
	switch k {
	case importer.KindAgent:
		return len(config.Agents)
	case importer.KindSkill:
		return len(config.Skills)
	case importer.KindRule:
		return len(config.Rules)
	case importer.KindWorkflow:
		return len(config.Workflows)
	case importer.KindMCP:
		return len(config.MCP)
	case importer.KindHookScript:
		return len(config.Hooks)
	case importer.KindSettings:
		return len(config.Settings)
	case importer.KindMemory:
		return len(config.Memory)
	default:
		return 0
	}
}

// renderCurrentStateTable prints a summary of the current project.xcf state.
func renderCurrentStateTable(cmd *cobra.Command, config *ast.XcaffoldConfig) {
	if config == nil {
		return
	}
	_ = cmd

	// Collect kinds with non-zero counts.
	type entry struct {
		label string
		count int
	}
	var active []entry
	for _, kd := range kindDisplay {
		c := configKindCount(config, kd.kind)
		if c > 0 {
			active = append(active, entry{label: kd.label, count: c})
		}
	}
	if len(active) == 0 {
		return
	}

	// Build summary: "17 agents, 21 skills, 13 rules"
	parts := make([]string, len(active))
	for i, a := range active {
		parts[i] = fmt.Sprintf(
			"%d %s", a.count, strings.ToLower(a.label))
	}
	summary := strings.Join(parts, ", ")

	// Box width adapts to content (minimum 60 inner chars).
	inner := len(summary) + 4
	if inner < 60 {
		inner = 60
	}
	border := strings.Repeat("─", inner)

	fmt.Printf("  ┌─── CURRENT STATE ─%s┐\n",
		border[19:])
	srcLine := "Source: .xcaffold/project.xcf"
	fmt.Printf("  │ %-*s │\n", inner-2, srcLine)
	fmt.Printf("  ├─%s─┤\n", border)
	fmt.Printf("  │ %-*s │\n", inner-2, summary)
	fmt.Printf("  └─%s─┘\n", border)
}

// colDef pairs an importer.Kind with its display label for table columns.
type colDef struct {
	kind  importer.Kind
	label string
}

// activeColumns returns the subset of kindDisplay entries that have at least
// one non-zero count across all providers.
func activeColumns(
	allCounts []map[importer.Kind]int,
) []colDef {
	var cols []colDef
	for _, kd := range kindDisplay {
		for _, counts := range allCounts {
			if counts[kd.kind] > 0 {
				cols = append(cols, colDef{
					kind:  kd.kind,
					label: kd.label,
				})
				break
			}
		}
	}
	return cols
}

// renderCompiledOutputTable prints a summary of detected compiled output.
func renderCompiledOutputTable(
	cmd *cobra.Command,
	providers []importer.ProviderImporter,
) {
	if len(providers) == 0 {
		return
	}
	_ = cmd

	// Scan all providers and collect counts.
	allCounts := make([]map[importer.Kind]int, len(providers))
	allSupported := make([]map[importer.Kind]bool, len(providers))
	for i, imp := range providers {
		allCounts[i] = importer.ScanDir(imp, imp.InputDir())
		if s, ok := nativeKindSupport[imp.Provider()]; ok {
			allSupported[i] = s
		} else {
			allSupported[i] = make(map[importer.Kind]bool)
		}
	}

	cols := activeColumns(allCounts)
	if len(cols) == 0 {
		return
	}

	if len(providers) == 1 {
		renderSingleProvider(providers[0], allCounts[0], cols)
		return
	}
	renderMultiProvider(providers, allCounts, allSupported, cols)
}

// renderSingleProvider formats the table for exactly one provider.
func renderSingleProvider(
	imp importer.ProviderImporter,
	counts map[importer.Kind]int,
	cols []colDef,
) {
	// Build summary: "17 agents, 21 skills, 13 rules"
	parts := make([]string, len(cols))
	for i, c := range cols {
		parts[i] = fmt.Sprintf(
			"%d %s", counts[c.kind], strings.ToLower(c.label))
	}
	summary := strings.Join(parts, ", ")

	dir := imp.InputDir()
	detected := fmt.Sprintf("Detected: %s", dir)

	// Box width adapts to content.
	inner := len(detected) + 4
	if w := len(summary) + 4; w > inner {
		inner = w
	}
	if inner < 60 {
		inner = 60
	}
	border := strings.Repeat("─", inner)

	fmt.Printf("  ┌─── COMPILED OUTPUT ─%s┐\n",
		border[21:])
	fmt.Printf("  │ %-*s │\n", inner-2, detected)
	fmt.Printf("  ├─%s─┤\n", border)
	fmt.Printf("  │ %-*s │\n", inner-2, summary)
	fmt.Printf("  └─%s─┘\n", border)
}

// renderMultiProvider formats the table for two or more providers.
// Transposed layout: kinds as rows, providers as columns.
// Shows "-" for unsupported kinds, counts for supported kinds.
func renderMultiProvider(
	providers []importer.ProviderImporter,
	allCounts []map[importer.Kind]int,
	allSupported []map[importer.Kind]bool,
	cols []colDef,
) {
	const kindW = 20  // width for kind name column
	const countW = 10 // width for each provider count column

	// Header row: "Kind" followed by provider names.
	header := fmt.Sprintf(
		"  %-*s", kindW, "Kind")
	for _, imp := range providers {
		name := filepath.Base(imp.InputDir())
		header += fmt.Sprintf("%*s", countW, name)
	}

	totalW := kindW + len(providers)*countW
	sep := "  " + strings.Repeat("─", totalW)

	fmt.Println(header)
	fmt.Println(sep)

	// Rows: one per kind, with counts across all providers.
	// Show "-" for unsupported kinds, count for supported kinds.
	for _, c := range cols {
		row := fmt.Sprintf("  %-*s", kindW, c.label)
		for i := range providers {
			if !allSupported[i][c.kind] {
				row += fmt.Sprintf("%*s", countW, "-")
			} else {
				row += fmt.Sprintf("%*d", countW, allCounts[i][c.kind])
			}
		}
		fmt.Println(row)
	}
	fmt.Println()
}
