package main

import (
	"fmt"
	"path/filepath"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/importer"
	"github.com/spf13/cobra"
)

// renderCurrentStateTable prints a summary of the current project.xcf state.
// Uses two-space indent and the standard glyph/output helpers.
func renderCurrentStateTable(cmd *cobra.Command, config *ast.XcaffoldConfig) {
	if config == nil {
		return
	}
	_ = cmd // output goes to stdout via fmt to match init output standard
	fmt.Println("  ┌───────────────────────────── CURRENT STATE ─────────────────────────────┐")
	fmt.Println("  │ Source: .xcaffold/project.xcf                                          │")
	fmt.Println("  ├─────────────────────────────────────────────────────────────────────────┤")

	row := fmt.Sprintf("  │  %2d agent(s)  │  %2d skill(s)  │  %2d rule(s)  │  %2d workflow(s)       │",
		len(config.Agents), len(config.Skills), len(config.Rules), len(config.Workflows))
	fmt.Println(row)
	fmt.Println("  └─────────────────────────────────────────────────────────────────────────┘")
}

// renderCompiledOutputTable prints a summary of detected compiled output directories.
func renderCompiledOutputTable(cmd *cobra.Command, providers []importer.ProviderImporter) {
	if len(providers) == 0 {
		return
	}
	_ = cmd

	fmt.Println("  ┌──────────────────────────── COMPILED OUTPUT ────────────────────────────┐")

	if len(providers) == 1 {
		imp := providers[0]
		dir := imp.InputDir()
		counts := importer.ScanDir(imp, dir)
		agents := counts[importer.KindAgent]
		skills := counts[importer.KindSkill]
		rules := counts[importer.KindRule]
		workflows := counts[importer.KindWorkflow]
		fmt.Printf("  │ Detected: %-61s │\n", dir)
		fmt.Println("  ├─────────────────────────────────────────────────────────────────────────┤")
		row := fmt.Sprintf("  │  %2d agent(s)  │  %2d skill(s)  │  %2d rule(s)  │  %2d workflow(s)       │",
			agents, skills, rules, workflows)
		fmt.Println(row)
	} else {
		fmt.Println("  │ Provider             Agents       Skills       Rules        Workflows   │")
		fmt.Println("  ├─────────────────────────────────────────────────────────────────────────┤")
		for _, imp := range providers {
			dir := imp.InputDir()
			counts := importer.ScanDir(imp, dir)
			agents := counts[importer.KindAgent]
			skills := counts[importer.KindSkill]
			rules := counts[importer.KindRule]
			workflows := counts[importer.KindWorkflow]
			row := fmt.Sprintf("  │ %-18s    %3d          %3d          %3d          %3d        │",
				filepath.Base(dir), agents, skills, rules, workflows)
			fmt.Println(row)
		}
	}
	fmt.Println("  └─────────────────────────────────────────────────────────────────────────┘")
}
