package main

import (
	"fmt"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/spf13/cobra"
)

// renderCurrentStateTable prints an ASCII table summarizing the current scaffold.xcf state.
func renderCurrentStateTable(cmd *cobra.Command, config *ast.XcaffoldConfig) {
	if config == nil {
		return
	}
	cmd.Println("  ┌───────────────────────────── CURRENT STATE ─────────────────────────────┐")
	cmd.Println("  │ Source: scaffold.xcf                                                    │")
	cmd.Println("  ├─────────────────────────────────────────────────────────────────────────┤")

	row := fmt.Sprintf("  │  %2d agent(s)  │  %2d skill(s)  │  %2d rule(s)  │  %2d workflow(s)       │",
		len(config.Agents), len(config.Skills), len(config.Rules), len(config.Workflows))
	cmd.Println(row)
	cmd.Println("  └─────────────────────────────────────────────────────────────────────────┘")
}

// renderCompiledOutputTable prints the compiled outputs depending on if there's 1 or many providers.
func renderCompiledOutputTable(cmd *cobra.Command, infos []platformDirInfo) {
	if len(infos) == 0 {
		return
	}

	cmd.Println("  ┌──────────────────────────── COMPILED OUTPUT ────────────────────────────┐")

	if len(infos) == 1 {
		info := infos[0]
		cmd.Printf("  │ Detected: %-61s │\n", info.dirName)
		cmd.Println("  ├─────────────────────────────────────────────────────────────────────────┤")
		row := fmt.Sprintf("  │  %2d agent(s)  │  %2d skill(s)  │  %2d rule(s)  │  %2d workflow(s)       │",
			info.agents, info.skills, info.rules, info.workflows)
		cmd.Println(row)
	} else {
		cmd.Println("  │ Provider             Agents       Skills       Rules        Workflows   │")
		cmd.Println("  ├─────────────────────────────────────────────────────────────────────────┤")
		for _, info := range infos {
			row := fmt.Sprintf("  │ %-18s    %3d          %3d          %3d          %3d        │",
				info.dirName, info.agents, info.skills, info.rules, info.workflows)
			cmd.Println(row)
		}
	}
	cmd.Println("  └─────────────────────────────────────────────────────────────────────────┘")
}
