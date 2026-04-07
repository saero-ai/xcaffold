package main

import (
	"fmt"

	"github.com/saero-ai/xcaffold/internal/parser"
	"github.com/spf13/cobra"
)

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Check schema compliance of scaffold.xcf",
	Long: `xcaffold validate parses and validates the target syntax tree.

┌───────────────────────────────────────────────────────────────────┐
│                       SCHEMA VALIDATION PHASE                     │
└───────────────────────────────────────────────────────────────────┘
 • Parses your scaffold.xcf AST map
 • Enforces strict typing against XCL (xcaffold Configuration Language)
 • Validates schema compliance and semantic structure
 • Exits cleanly on success, ideal for pre-commit CI/CD hooks
 
 Any invalid blocks, types, or missing overrides will produce exit code 1.`,
	Example: "  $ xcaffold validate",
	RunE:    runValidate,
}

func init() {
	rootCmd.AddCommand(validateCmd)
}

func runValidate(cmd *cobra.Command, args []string) error {
	if scopeFlag == scopeGlobal || scopeFlag == scopeAll {
		if err := validateScope(globalXcfPath, "global"); err != nil {
			return err
		}
	}
	if scopeFlag == scopeProject || scopeFlag == scopeAll {
		if err := validateScope(xcfPath, "project"); err != nil {
			return err
		}
	}
	return nil
}

func validateScope(configPath, scopeName string) error {
	if _, err := parser.ParseFile(configPath); err != nil {
		return fmt.Errorf("[%s] validation failed: %w", scopeName, err)
	}
	fmt.Printf("[%s] ✓ %s is valid and perfectly structured.\n", scopeName, configPath)
	return nil
}
