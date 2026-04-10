package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/judge"
	"github.com/saero-ai/xcaffold/internal/parser"
	"github.com/saero-ai/xcaffold/internal/proxy"
	"github.com/saero-ai/xcaffold/internal/trace"
	"github.com/spf13/cobra"
)

var (
	testAgentFlag      string
	testJudgeFlag      bool
	testOutputFlag     string
	testCliPathFlag    string
	testJudgeModelFlag string
)

var testCmd = &cobra.Command{
	Use:   "test",
	Short: "Run a sandboxed local simulation of a Claude agent",
	Long: `xcaffold test runs a local mocked simulation of your agent boundaries.

┌───────────────────────────────────────────────────────────────────┐
│                          VALIDATION PHASE                         │
└───────────────────────────────────────────────────────────────────┘
 • 🛡️ Spawns an HTTP intercept proxy binding your agent to an isolated sandbox
 • 🔬 Translates assertions from your scaffold into LLM-as-a-judge directives
 • 📝 Outputs a trace.jsonl detailing every tool call the agent attempted

Prerequisites:
  - The target CLI binary (e.g. 'claude', 'cursor', 'gemini') must be available on $PATH
    (or set test.cli_path in scaffold.xcf).
  - Set ANTHROPIC_API_KEY or XCAFFOLD_LLM_API_KEY in your environment for the judge.

Usage:
  $ xcaffold test --agent backend-dev
  $ xcaffold test --agent backend-dev --judge`,
	Example: `  $ xcaffold test --agent backend-dev
  $ xcaffold test --agent data-analyst --judge
  $ xcaffold test -a frontend-dev --output custom_trace.jsonl`,
	RunE: runTest,
}

func init() {
	testCmd.Flags().StringVarP(&testAgentFlag, "agent", "a", "", "Agent ID to simulate (required)")
	testCmd.Flags().BoolVar(&testJudgeFlag, "judge", false, "Run LLM-as-a-Judge evaluation after simulation")
	testCmd.Flags().StringVarP(&testOutputFlag, "output", "o", "trace.jsonl", "Path to write the execution trace")
	testCmd.Flags().StringVar(&testCliPathFlag, "cli-path", "", "Path to underlying CLI binary (overrides scaffold.xcf test.cli_path)")
	testCmd.Flags().StringVar(&testJudgeModelFlag, "judge-model", "", "Anthropic model for the judge (overrides scaffold.xcf test.judge_model)")

	_ = testCmd.MarkFlagRequired("agent")
	rootCmd.AddCommand(testCmd)
}

func runTest(cmd *cobra.Command, args []string) error {
	// 1. Load and validate the project config.
	config, err := parser.ParseFile(xcfPath)
	if err != nil {
		return fmt.Errorf("parse error: %w", err)
	}

	agentConfig, ok := config.Agents[testAgentFlag]
	if !ok {
		return fmt.Errorf("agent %q not found in scaffold.xcf", testAgentFlag)
	}

	// 2. Resolve the CLI binary path (flag > xcf > PATH).
	var testCfg ast.TestConfig
	if config.Project != nil {
		testCfg = config.Project.Test
	}
	cliPath := resolveCliPath(testCfg.CliPath, testCfg.ClaudePath)

	// 3. Set up trace file.
	traceFile, err := os.Create(filepath.Clean(testOutputFlag))
	if err != nil {
		return fmt.Errorf("failed to create trace file %q: %w", testOutputFlag, err)
	}
	defer traceFile.Close()

	recorder := trace.NewRecorder(traceFile)

	// 4. Start the intercept proxy.
	proxyServer, err := proxy.New(recorder)
	if err != nil {
		return fmt.Errorf("failed to start proxy: %w", err)
	}
	defer proxyServer.Close()

	// Run proxy in background.
	proxyErrCh := make(chan error, 1)
	go func() { proxyErrCh <- proxyServer.Start() }()

	fmt.Printf("✓ Intercept proxy started at %s\n", proxyServer.Addr())
	fmt.Printf("  Trace output: %s\n\n", testOutputFlag)

	// 5. Spawn the target CLI subprocess with HTTPS_PROXY set.
	cliCmd := exec.Command(cliPath, "--agent", testAgentFlag) //nolint:gosec
	cliCmd.Stdout = os.Stdout
	cliCmd.Stderr = os.Stderr
	cliCmd.Env = append(os.Environ(),
		"HTTPS_PROXY="+proxyServer.ProxyURL(),
		"HTTP_PROXY="+proxyServer.ProxyURL(),
	)

	fmt.Printf("▶ Running: %s --agent %s\n\n", cliPath, testAgentFlag)

	if err := cliCmd.Run(); err != nil {
		// Non-zero exit from subprocess is surfaced as a warning, not a fatal error,
		// so we can still print the trace summary and run the judge.
		fmt.Fprintf(os.Stderr, "\nWarning: target CLI exited with error: %v\n", err)
	}

	proxyServer.Close()

	// 6. Print trace summary.
	summary := recorder.Summary()
	summary.Print(os.Stdout)
	fmt.Printf("  Trace written to: %s\n\n", testOutputFlag)

	// 7. Optional: Run LLM-as-a-Judge.
	if testJudgeFlag {
		if err := runJudge(summary, agentConfig.Assertions, testCfg.JudgeModel, cliPath); err != nil {
			return fmt.Errorf("judge evaluation failed: %w", err)
		}
	}

	return nil
}

// runJudge runs the LLM-as-a-Judge evaluation against the summary.
// If ANTHROPIC_API_KEY is set it uses the direct API; otherwise it falls back
// to the underlying CLI subprocess using the user's subscription.
func runJudge(summary trace.Summary, assertions []string, configModel, cliPath string) error {
	anthropicKey := os.Getenv("ANTHROPIC_API_KEY")
	genericAPIKey := os.Getenv("XCAFFOLD_LLM_API_KEY")
	genericAPIBase := os.Getenv("XCAFFOLD_LLM_BASE_URL")

	model := resolveJudgeModel(configModel)

	fmt.Printf("── Judge Evaluation ──────────────────────────────────\n")
	fmt.Printf("  Model: %s\n", model)
	if genericAPIKey != "" {
		fmt.Printf("  Auth:  Platform-Agnostic LLM API\n")
	} else if anthropicKey != "" {
		fmt.Printf("  Auth:  Target Provider API Key\n")
	} else {
		fmt.Printf("  Auth:  Platform Subscription (fallback via local CLI config)\n")
	}
	fmt.Printf("  Assertions: %d\n\n", len(assertions))

	j, err := judge.New(anthropicKey, genericAPIKey, genericAPIBase, model, cliPath, nil)
	if err != nil {
		return err
	}
	report, err := j.Evaluate(context.Background(), summary, assertions)
	if err != nil {
		return err
	}

	var out bytes.Buffer
	fmt.Fprintf(&out, "  Verdict: %s\n", report.Verdict)
	fmt.Fprintf(&out, "  Reasoning: %s\n", report.Reasoning)

	if len(report.PassedAssertions) > 0 {
		fmt.Fprintf(&out, "\n  ✓ Passed:\n")
		for _, a := range report.PassedAssertions {
			fmt.Fprintf(&out, "    - %s\n", a)
		}
	}
	if len(report.FailedAssertions) > 0 {
		fmt.Fprintf(&out, "\n  ✗ Failed:\n")
		for _, a := range report.FailedAssertions {
			fmt.Fprintf(&out, "    - %s\n", a)
		}
	}
	fmt.Fprintf(&out, "──────────────────────────────────────────────────────\n")

	_, _ = io.Copy(os.Stdout, &out)
	return nil
}

// resolveCliPath returns the effective path to the underlying CLI binary.
// Priority: CLI flag > scaffold.xcf test.cli_path > default "claude".
func resolveCliPath(cliPath, claudePath string) string {
	if testCliPathFlag != "" {
		return testCliPathFlag
	}
	if cliPath != "" {
		return cliPath
	}
	if claudePath != "" {
		return claudePath
	}
	return targetClaude
}

// resolveJudgeModel returns the effective judge model.
// Priority: CLI flag > scaffold.xcf test.judge_model > default Haiku.
func resolveJudgeModel(xcfModel string) string {
	if testJudgeModelFlag != "" {
		return testJudgeModelFlag
	}
	if xcfModel != "" {
		return xcfModel
	}
	return "claude-haiku-4-5-20251001"
}
