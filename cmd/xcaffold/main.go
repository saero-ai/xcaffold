package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "xcaffold",
	Short: "xcaffold — deterministic agent-as-code orchestration",
	Long: `xcaffold is an open-source, deterministic agent configuration compiler engine for Claude Code.

┌───────────────────────────────────────────────────────────────────┐
│                 THE 6-PHASE ORCHESTRATION ENGINE                  │
└───────────────────────────────────────────────────────────────────┘
 • Bootstrap   [xcaffold init]    Creates base project scaffolding
 • Audit       [xcaffold analyze] Inspects repo & builds XCF config
 • Token Cost  [xcaffold plan]    Statically estimates token budget
 • Compilation [xcaffold apply]   Compiles XCF to .claude/ prompts
 • Drift Check [xcaffold diff]    Detects manual config tampering
 • Validation  [xcaffold test]    Runs an LLM-in-the-loop proxy

┌───────────────────────────────────────────────────────────────────┐
│                      DIAGNOSTICS & TELEMETRY                      │
└───────────────────────────────────────────────────────────────────┘
 • Review      [xcaffold review]  Universally parses state files
   ↳ Supports: scaffold.xcf, audit.json, plan.json, trace.jsonl
   ↳ Try: 'xcaffold review all'

Use 'xcaffold --help' for more information on available commands.`,
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
