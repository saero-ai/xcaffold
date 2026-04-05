package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/saero-ai/xcaffold/internal/compiler"
	"github.com/saero-ai/xcaffold/internal/parser"
	"github.com/spf13/cobra"
)

var (
	exportFormat string
	exportOutput string
	exportTarget string
)

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Package compiled output as a distributable plugin",
	Long: `xcaffold export repackages your compiled output into a standard
plugin directory that can be shared and distributed.

  $ xcaffold export --format plugin --output ./my-plugin/`,
	Example: "  $ xcaffold export --format plugin --output ./my-plugin/",
	RunE:    runExport,
}

func init() {
	exportCmd.Flags().StringVar(&exportFormat, "format", "plugin", "Export format (currently only 'plugin')")
	exportCmd.Flags().StringVar(&exportOutput, "output", "", "Output directory (required)")
	exportCmd.Flags().StringVar(&exportTarget, "target", "", "Compilation target (claude, cursor, gemini; default: claude)")
	_ = exportCmd.MarkFlagRequired("output")
	rootCmd.AddCommand(exportCmd)
}

func runExport(cmd *cobra.Command, args []string) error {
	if exportFormat != "plugin" {
		return fmt.Errorf("unsupported export format %q; only 'plugin' is supported", exportFormat)
	}

	config, err := parser.ParseFile(xcfPath)
	if err != nil {
		return fmt.Errorf("parse error: %w", err)
	}

	baseDir := filepath.Dir(xcfPath)
	compiled, err := compiler.Compile(config, baseDir, exportTarget)
	if err != nil {
		return fmt.Errorf("compilation error: %w", err)
	}

	exported, err := compiler.ExportPlugin(config, compiled)
	if err != nil {
		return fmt.Errorf("export error: %w", err)
	}

	absOutput, err := filepath.Abs(exportOutput)
	if err != nil {
		return fmt.Errorf("could not resolve output path: %w", err)
	}

	for relPath, content := range exported.Files {
		absPath := filepath.Join(absOutput, relPath)
		if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
			return fmt.Errorf("failed to create directory for %q: %w", absPath, err)
		}
		if err := os.WriteFile(absPath, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write %q: %w", absPath, err)
		}
		fmt.Printf("  wrote %s\n", absPath)
	}

	fmt.Printf("\nPlugin exported to %s\n", absOutput)
	return nil
}
