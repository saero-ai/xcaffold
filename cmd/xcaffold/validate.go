package main

import (
	"fmt"
	"os"
	"path/filepath"

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
	xcfPath := filepath.Clean("scaffold.xcf")

	f, err := os.Open(xcfPath)
	if err != nil {
		return fmt.Errorf("could not open %s: %w", xcfPath, err)
	}
	defer f.Close()

	if _, err := parser.Parse(f); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	fmt.Println("✓ scaffold.xcf is valid and perfectly structured.")
	return nil
}
