package main

import (
	"fmt"

	"github.com/saero-ai/xcaffold/internal/ast"
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
func renderCompiledOutputTable(cmd *cobra.Command, infos []platformDirInfo) {
	if len(infos) == 0 {
		return
	}
	_ = cmd

	fmt.Println("  ┌──────────────────────────── COMPILED OUTPUT ────────────────────────────┐")

	if len(infos) == 1 {
		info := infos[0]
		fmt.Printf("  │ Detected: %-61s │\n", info.dirName)
		fmt.Println("  ├─────────────────────────────────────────────────────────────────────────┤")
		row := fmt.Sprintf("  │  %2d agent(s)  │  %2d skill(s)  │  %2d rule(s)  │  %2d workflow(s)       │",
			info.agents, info.skills, info.rules, info.workflows)
		fmt.Println(row)
	} else {
		fmt.Println("  │ Provider             Agents       Skills       Rules        Workflows   │")
		fmt.Println("  ├─────────────────────────────────────────────────────────────────────────┤")
		for _, info := range infos {
			row := fmt.Sprintf("  │ %-18s    %3d          %3d          %3d          %3d        │",
				info.dirName, info.agents, info.skills, info.rules, info.workflows)
			fmt.Println(row)
		}
	}
	fmt.Println("  └─────────────────────────────────────────────────────────────────────────┘")
}
