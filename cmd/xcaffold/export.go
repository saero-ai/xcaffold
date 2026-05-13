package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/saero-ai/xcaffold/internal/compiler"
	"github.com/saero-ai/xcaffold/internal/optimizer"
	"github.com/saero-ai/xcaffold/internal/parser"
	"github.com/saero-ai/xcaffold/internal/renderer"
	"github.com/saero-ai/xcaffold/providers"
	"github.com/spf13/cobra"
)

var (
	exportFormat  string
	exportOutput  string
	exportTarget  string
	exportVarFile string
)

var exportCmd = &cobra.Command{
	Use:    "export",
	Hidden: true,
	Short:  "Package compiled output as a distributable plugin",
	Long: `xcaffold export repackages your compiled output into a standard
plugin directory that can be shared and distributed.

  $ xcaffold export --format plugin --output ./my-plugin/`,
	Example: "  $ xcaffold export --format plugin --output ./my-plugin/",
	RunE:    runExport,
}

func init() {
	exportCmd.Flags().StringVar(&exportFormat, "format", "plugin", "Export format (currently only 'plugin')")
	exportCmd.Flags().StringVar(&exportOutput, "output", "", "Output directory for exported plugin")
	exportCmd.Flags().StringVar(&exportTarget, "target", "", fmt.Sprintf("compilation target (required: %s)", strings.Join(providers.PrimaryNames(), ", ")))
	exportCmd.Flags().StringVar(&exportVarFile, "var-file", "", "Load variables from a custom file")
	_ = exportCmd.MarkFlagRequired("output")
	rootCmd.AddCommand(exportCmd)
}

func runExport(cmd *cobra.Command, args []string) error {
	if exportFormat != "plugin" {
		return fmt.Errorf("unsupported export format %q; only 'plugin' is supported", exportFormat)
	}

	config, err := parser.ParseDirectory(projectParseRoot(), parser.WithVarFile(exportVarFile))
	if err != nil {
		return fmt.Errorf("parse error: %w", err)
	}

	baseDir := projectParseRoot()
	compiled, notes, err := compiler.Compile(config, baseDir, compiler.CompileOpts{
		Target:    exportTarget,
		Blueprint: "",
		VarFile:   exportVarFile,
	})
	if err != nil {
		return fmt.Errorf("compilation error: %w", err)
	}

	opt := optimizer.New(exportTarget)
	optimized, optNotes, optErr := opt.Run(compiled.Files)
	if optErr != nil {
		return fmt.Errorf("optimizer error: %w", optErr)
	}
	compiled.Files = optimized
	notes = append(notes, optNotes...)

	printFidelityNotes(os.Stderr, renderer.FilterNotes(notes, buildSuppressedResourcesMap(config, exportTarget)), verboseFlag)

	exported, err := compiler.ExportPlugin(config, compiled, exportTarget)
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
		if err := os.WriteFile(absPath, []byte(content), 0600); err != nil {
			return fmt.Errorf("failed to write %q: %w", absPath, err)
		}
		fmt.Printf("  wrote %s\n", absPath)
	}

	fmt.Printf("\nPlugin exported to %s\n", absOutput)
	return nil
}
