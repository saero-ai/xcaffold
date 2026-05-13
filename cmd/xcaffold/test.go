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
	testCmd.Flags().StringVar(&testCliPathFlag, "cli-path", "", "Path to underlying CLI binary (overrides project.xcaf test.cli-path)")
	testCmd.Flags().StringVar(&testJudgeModelFlag, "judge-model", "", "Anthropic model for the judge (overrides project.xcaf test.judge-model)")

	_ = testCmd.MarkFlagRequired("agent")
	rootCmd.AddCommand(testCmd)
}

func runTest(cmd *cobra.Command, args []string) error {
	config, err := parser.ParseDirectory(projectParseRoot())
	if err != nil {
		return fmt.Errorf("parse error: %w", err)
	}

	agentConfig, ok := config.Agents[testAgentFlag]
	if !ok {
		return fmt.Errorf("agent %q not found in project.xcaf", testAgentFlag)
	}

	testCfg := resolveTestConfig(config)
	traceFile, err := os.Create(filepath.Clean(testOutputFlag))
	if err != nil {
		return fmt.Errorf("failed to create trace file %q: %w", testOutputFlag, err)
	}
	defer traceFile.Close()

	recorder := trace.NewRecorder(traceFile)
	systemPromptBytes, err := readCompiledAgent(testAgentFlag)
	if err != nil {
		return err
	}

	task := resolveTestTask(testCfg)
	model, err := resolveTestModel(agentConfig, testCfg)
	if err != nil {
		return err
	}

	client, err := createTestLLMClient(agentConfig, testCfg, model)
	if err != nil {
		return fmt.Errorf("failed to create LLM client: %w", err)
	}

	fmt.Printf("Testing agent %q with task: %s\n", testAgentFlag, task)
	fmt.Printf("Using model: %s (auth: %s)\n\n", model, client.AuthMode())

	response, err := client.Call(cmd.Context(), buildTestPrompt(string(systemPromptBytes), task))
	if err != nil {
		return fmt.Errorf("LLM call failed: %w", err)
	}

	recordToolCalls(recorder, response, testAgentFlag)
	summary := recorder.Summary()
	summary.Print(os.Stdout)
	fmt.Printf("  Trace written to: %s\n\n", testOutputFlag)

	if testJudgeFlag {
		cliPath := resolveCliPath(testCfg.CliPath)
		if err := runJudge(summary, agentConfig.Assertions.Values, testCfg.JudgeModel, cliPath); err != nil {
			return fmt.Errorf("judge evaluation failed: %w", err)
		}
	}

	return nil
}

// resolveTestConfig extracts test config from project or provides defaults.
func resolveTestConfig(config *ast.XcaffoldConfig) ast.TestConfig {
	if config.Project != nil {
		return config.Project.Test
	}
	return ast.TestConfig{}
}

// readCompiledAgent loads the compiled agent system prompt from disk.
func readCompiledAgent(agentID string) ([]byte, error) {
	agentMDPath := filepath.Join(projectRoot, ".claude", "agents", agentID+".md")
	data, err := os.ReadFile(agentMDPath) //nolint:gosec
	if err != nil {
		return nil, fmt.Errorf("agent %q not compiled — run 'xcaffold apply' first: %w", agentID, err)
	}
	return data, nil
}

// resolveTestTask determines the task string from config or default.
func resolveTestTask(testCfg ast.TestConfig) string {
	if testCfg.Task != "" {
		return testCfg.Task
	}
	return "Describe what tools you have available and what you would do first."
}

// resolveTestModel determines the model to use for the test run.
func resolveTestModel(agentConfig ast.AgentConfig, testCfg ast.TestConfig) (string, error) {
	model := agentConfig.Model
	if model == "" {
		detected := detectDefaultTarget()
		if detected != "" {
			model, _ = resolveTargetMeta(detected)
		}
		if model == "" {
			return "", fmt.Errorf("agent does not specify a model in .xcaf, and no CLI was detected on PATH")
		}
	}
	return model, nil
}

// createTestLLMClient builds the LLM client with appropriate auth.
func createTestLLMClient(agentConfig ast.AgentConfig, testCfg ast.TestConfig, model string) (*llmclient.Client, error) {
	anthropicKey := os.Getenv("ANTHROPIC_API_KEY")
	genericAPIKey := os.Getenv("XCAFFOLD_LLM_API_KEY")
	genericAPIBase := os.Getenv("XCAFFOLD_LLM_BASE_URL")

	return llmclient.New(llmclient.Config{
		AnthropicKey:   anthropicKey,
		GenericAPIKey:  genericAPIKey,
		GenericAPIBase: genericAPIBase,
		Model:          model,
		CLIPath:        resolveCliPath(testCfg.CliPath),
		MaxTokens:      4096,
	})
}

// recordToolCalls extracts tool calls from response and records them.
func recordToolCalls(recorder *trace.Recorder, response string, agentID string) {
	toolCalls := extractToolCallsFromResponse(response)
	for _, tc := range toolCalls {
		_ = recorder.Record(trace.ToolCallEvent{
			Timestamp:   time.Now(),
			ToolName:    tc.name,
			InputParams: tc.input,
			AgentID:     agentID,
		})
	}
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

	j, err := judge.New(judge.JudgeConfig{
		AnthropicKey:  anthropicKey,
		GenericAPIKey: genericAPIKey,
		APIBaseURL:    genericAPIBase,
		Model:         model,
		CLIPath:       cliPath,
		HTTPClient:    nil,
	})
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
// Priority: CLI flag > project.xcaf test.cli-path > empty string.
// Callers must handle the empty string case when CLI mode is needed.
func resolveCliPath(cliPath string) string {
	if testCliPathFlag != "" {
		return testCliPathFlag
	}
	if cliPath != "" {
		return cliPath
	}
	return ""
}

// resolveJudgeModel returns the effective judge model for the LLM-as-a-Judge evaluation.
// Priority: CLI flag > project.xcaf test.judge-model > xcaffold's default judge model.
// The default judge model is xcaffold's internal choice for evaluation, not a provider assumption.
func resolveJudgeModel(xcafModel string) string {
	const defaultJudgeModel = "claude-haiku-4-5-20251001"
	if testJudgeModelFlag != "" {
		return testJudgeModelFlag
	}
	if xcafModel != "" {
		return xcafModel
	}
	return defaultJudgeModel
}
