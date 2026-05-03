package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/compiler"
	"github.com/saero-ai/xcaffold/internal/judge"
	"github.com/saero-ai/xcaffold/internal/llmclient"
	"github.com/saero-ai/xcaffold/internal/parser"
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
	Use:    "test",
	Hidden: true,
	Short:  "Run a sandboxed local simulation of a Claude agent",
	Long: `xcaffold test simulates your compiled agent by sending a task to the LLM
directly and recording every tool call the model declares.

Steps:
  1. Reads the compiled agent system prompt from .claude/agents/<agent>.md
  2. Sends the task to the LLM via the Anthropic API (or XCAFFOLD_LLM_API_KEY)
  3. Extracts declared tool calls from the response
  4. Writes a trace.jsonl detailing every tool call the agent attempted
  5. Optionally runs LLM-as-a-Judge evaluation against scaffold assertions

Prerequisites:
  - Run 'xcaffold apply' before testing — the agent must be compiled.
  - Set ANTHROPIC_API_KEY or XCAFFOLD_LLM_API_KEY in your environment.

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
	testCmd.Flags().StringVar(&testCliPathFlag, "cli-path", "", "Path to underlying CLI binary (overrides project.xcf test.cli-path)")
	testCmd.Flags().StringVar(&testJudgeModelFlag, "judge-model", "", "Anthropic model for the judge (overrides project.xcf test.judge-model)")

	_ = testCmd.MarkFlagRequired("agent")
	rootCmd.AddCommand(testCmd)
}

func runTest(cmd *cobra.Command, args []string) error {
	// 1. Load and validate the project config.
	config, err := parser.ParseDirectory(projectParseRoot())
	if err != nil {
		return fmt.Errorf("parse error: %w", err)
	}

	agentConfig, ok := config.Agents[testAgentFlag]
	if !ok {
		return fmt.Errorf("agent %q not found in project.xcf", testAgentFlag)
	}

	// 2. Resolve test config (CLI flag > xcf > defaults).
	var testCfg ast.TestConfig
	if config.Project != nil {
		testCfg = config.Project.Test
	}

	// 3. Set up trace file.
	traceFile, err := os.Create(filepath.Clean(testOutputFlag))
	if err != nil {
		return fmt.Errorf("failed to create trace file %q: %w", testOutputFlag, err)
	}
	defer traceFile.Close()

	recorder := trace.NewRecorder(traceFile)

	// 4. Read compiled agent system prompt from disk.
	agentMDPath := filepath.Join(projectRoot, ".claude", "agents", testAgentFlag+".md")
	systemPromptBytes, err := os.ReadFile(agentMDPath) //nolint:gosec
	if err != nil {
		return fmt.Errorf("agent %q not compiled — run 'xcaffold apply' first: %w", testAgentFlag, err)
	}

	// 5. Resolve the test task.
	task := testCfg.Task
	if task == "" {
		task = "Describe what tools you have available and what you would do first."
	}

	// 6. Build LLM client for the test run.
	anthropicKey := os.Getenv("ANTHROPIC_API_KEY")
	genericAPIKey := os.Getenv("XCAFFOLD_LLM_API_KEY")
	genericAPIBase := os.Getenv("XCAFFOLD_LLM_BASE_URL")

	model := agentConfig.Model
	if model == "" {
		model = "claude-sonnet-4-20250514"
	}

	client, err := llmclient.New(llmclient.Config{
		AnthropicKey:   anthropicKey,
		GenericAPIKey:  genericAPIKey,
		GenericAPIBase: genericAPIBase,
		Model:          model,
		CLIPath:        resolveCliPath(testCfg.CliPath, testCfg.ClaudePath),
		MaxTokens:      4096,
	})
	if err != nil {
		return fmt.Errorf("failed to create LLM client: %w", err)
	}

	// 7. Run single-turn simulation.
	fmt.Printf("Testing agent %q with task: %s\n", testAgentFlag, task)
	fmt.Printf("Using model: %s (auth: %s)\n\n", model, client.AuthMode())

	prompt := buildTestPrompt(string(systemPromptBytes), task)
	response, err := client.Call(cmd.Context(), prompt)
	if err != nil {
		return fmt.Errorf("LLM call failed: %w", err)
	}

	// 8. Extract tool calls from response and record trace.
	toolCalls := extractToolCallsFromResponse(response)
	for _, tc := range toolCalls {
		_ = recorder.Record(trace.ToolCallEvent{
			Timestamp:   time.Now(),
			ToolName:    tc.name,
			InputParams: tc.input,
			AgentID:     testAgentFlag,
		})
	}

	// 9. Print trace summary.
	summary := recorder.Summary()
	summary.Print(os.Stdout)
	fmt.Printf("  Trace written to: %s\n\n", testOutputFlag)

	// 10. Optional: Run LLM-as-a-Judge.
	if testJudgeFlag {
		cliPath := resolveCliPath(testCfg.CliPath, testCfg.ClaudePath)
		if err := runJudge(summary, agentConfig.Assertions, testCfg.JudgeModel, cliPath); err != nil {
			return fmt.Errorf("judge evaluation failed: %w", err)
		}
	}

	return nil
}

// buildTestPrompt wraps the compiled system prompt and task into a single user message
// that asks the model to describe the tool calls it would make.
func buildTestPrompt(systemPrompt, task string) string {
	return fmt.Sprintf(
		"You are an AI agent with the following system prompt:\n\n%s\n\nUser task: %s\n\nRespond with the tools you would use. Format each tool use as a tool_use JSON block with fields: type, name, input.",
		systemPrompt,
		task,
	)
}

// toolCall holds the parsed name and input of a single tool use block.
type toolCall struct {
	name  string
	input map[string]any
}

// extractToolCallsFromResponse parses tool_use blocks from the model's text response.
// It handles two formats:
//  1. A top-level JSON object with a "content" array of Anthropic-style message blocks.
//  2. Inline JSON objects with "type":"tool_use" embedded in free-form text.
func extractToolCallsFromResponse(response string) []toolCall {
	// Try structured Anthropic content array first.
	var payload struct {
		Content []struct {
			Type  string         `json:"type"`
			Name  string         `json:"name"`
			Input map[string]any `json:"input"`
		} `json:"content"`
	}
	if json.Unmarshal([]byte(response), &payload) == nil && len(payload.Content) > 0 {
		var result []toolCall
		for _, block := range payload.Content {
			if block.Type == "tool_use" && block.Name != "" {
				result = append(result, toolCall{name: block.Name, input: block.Input})
			}
		}
		return result
	}

	// Fall back: scan for individual JSON objects with "type":"tool_use".
	var result []toolCall
	dec := json.NewDecoder(bytes.NewReader([]byte(response)))
	for {
		var obj map[string]any
		if err := dec.Decode(&obj); err != nil {
			break
		}
		if typ, _ := obj["type"].(string); typ == "tool_use" {
			name, _ := obj["name"].(string)
			input, _ := obj["input"].(map[string]any)
			if name != "" {
				result = append(result, toolCall{name: name, input: input})
			}
		}
	}
	return result
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
		fmt.Fprintf(&out, "\n  Passed:\n")
		for _, a := range report.PassedAssertions {
			fmt.Fprintf(&out, "    - %s\n", a)
		}
	}
	if len(report.FailedAssertions) > 0 {
		fmt.Fprintf(&out, "\n  Failed:\n")
		for _, a := range report.FailedAssertions {
			fmt.Fprintf(&out, "    - %s\n", a)
		}
	}
	fmt.Fprintf(&out, "──────────────────────────────────────────────────────\n")

	_, _ = io.Copy(os.Stdout, &out)
	return nil
}

// resolveCliPath returns the effective path to the underlying CLI binary.
// Priority: CLI flag > project.xcf test.cli-path > default "claude".
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
	return compiler.TargetClaude
}

// resolveJudgeModel returns the effective judge model.
// Priority: CLI flag > project.xcf test.judge-model > default Haiku.
func resolveJudgeModel(xcfModel string) string {
	if testJudgeModelFlag != "" {
		return testJudgeModelFlag
	}
	if xcfModel != "" {
		return xcfModel
	}
	return "claude-haiku-4-5-20251001"
}
