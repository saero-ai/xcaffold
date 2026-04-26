package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var diffTargetFlag string
var diffBlueprintFlag string

var diffCmd = &cobra.Command{
	Use:          "diff",
	Hidden:       true,
	Short:        "Deprecated: use 'xcaffold status' instead",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Fprintln(os.Stderr, "Note: 'xcaffold diff' is deprecated — use 'xcaffold status' instead.")
		statusTargetFlag = diffTargetFlag
		statusBlueprintFlag = diffBlueprintFlag
		return runStatus(cmd, args)
	},
}

func init() {
	diffCmd.Flags().StringVar(&diffTargetFlag, "target", "", "compilation target platform")
	diffCmd.Flags().StringVar(&diffBlueprintFlag, "blueprint", "", "filter by blueprint")
	_ = diffCmd.Flags().MarkHidden("blueprint")
	rootCmd.AddCommand(diffCmd)
}
