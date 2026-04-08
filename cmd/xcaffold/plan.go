package main

import (
	"github.com/spf13/cobra"
)

var planCmd = &cobra.Command{
	Use:   "plan",
	Short: "Preview compilation output without writing files",
	RunE: func(cmd *cobra.Command, args []string) error {
		applyDryRun = true
		return runApply(cmd, args)
	},
}

func init() {
	planCmd.Flags().StringVar(&scopeFlag, "scope", scopeProject, "scope of plan (project or global)")
	planCmd.Flags().StringVarP(&targetFlag, "target", "t", targetClaude, "preview output for target platform")
	rootCmd.AddCommand(planCmd)
}
