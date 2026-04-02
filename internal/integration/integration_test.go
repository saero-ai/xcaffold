package integration_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/saero-ai/xcaffold/internal/judge"
	"github.com/saero-ai/xcaffold/internal/proxy"
	"github.com/saero-ai/xcaffold/internal/trace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// simulateToolCall makes an HTTP POST to the proxy that mimics
// what the claude CLI sends when invoking a tool.
func simulateToolCall(t *testing.T, proxyAddr, toolName, command string) {
	t.Helper()
	body := map[string]any{
		"model": "claude-3-5-haiku-20241022",
		"content": []map[string]any{
			{
				"type":  "tool_use",
				"name":  toolName,
				"input": map[string]string{"command": command},
			},
		},
	}
	bodyBytes, _ := json.Marshal(body)

	req, err := http.NewRequest(
		http.MethodPost,
		"http://"+proxyAddr+"/v1/messages",
		bytes.NewReader(bodyBytes),
	)
	require.NoError(t, err)
	req.Host = "api.anthropic.com"
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// TestIntegration_ProxyRecordsToolCalls tests the full proxy → trace pipeline.
func TestIntegration_ProxyRecordsToolCalls(t *testing.T) {
	var buf bytes.Buffer
	recorder := trace.NewRecorder(&buf)

	srv, err := proxy.New(recorder)
	require.NoError(t, err)
	defer srv.Close()

	// Start proxy in background.
	go func() { _ = srv.Start() }()

	// Give the server a moment to start.
	time.Sleep(10 * time.Millisecond)

	// Simulate 3 tool calls as if the claude CLI made them.
	simulateToolCall(t, srv.Addr(), "Bash", "npm test")
	simulateToolCall(t, srv.Addr(), "Bash", "ls -la")
	simulateToolCall(t, srv.Addr(), "Write", "path=src/main.go")

	summary := recorder.Summary()
	assert.Equal(t, 3, summary.TotalCalls)
	assert.Equal(t, 2, summary.CallsByTool["Bash"])
	assert.Equal(t, 1, summary.CallsByTool["Write"])
}

// TestIntegration_TraceJSONL_IsValid ensures the JSONL output is parseable.
func TestIntegration_TraceJSONL_IsValid(t *testing.T) {
	var buf bytes.Buffer
	recorder := trace.NewRecorder(&buf)

	srv, err := proxy.New(recorder)
	require.NoError(t, err)
	defer srv.Close()

	go func() { _ = srv.Start() }()
	time.Sleep(10 * time.Millisecond)

	simulateToolCall(t, srv.Addr(), "Read", "path=README.md")
	simulateToolCall(t, srv.Addr(), "Bash", "go build ./...")

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	require.Len(t, lines, 2)

	for i, line := range lines {
		var event trace.ToolCallEvent
		err := json.Unmarshal([]byte(line), &event)
		require.NoError(t, err, "line %d must be valid JSON: %s", i, line)
		assert.NotEmpty(t, event.ToolName)
		assert.Equal(t, "[SIMULATED SUCCESS]", event.MockResponse)
	}
}

// TestIntegration_ProxySummaryPrint verifies human-readable output.
func TestIntegration_ProxySummaryPrint(t *testing.T) {
	var buf bytes.Buffer
	recorder := trace.NewRecorder(&buf)

	srv, err := proxy.New(recorder)
	require.NoError(t, err)
	defer srv.Close()

	go func() { _ = srv.Start() }()
	time.Sleep(10 * time.Millisecond)

	simulateToolCall(t, srv.Addr(), "Bash", "go test ./...")

	var out bytes.Buffer
	recorder.Summary().Print(&out)
	output := out.String()

	assert.Contains(t, output, "Total intercepted tool calls: 1")
	assert.Contains(t, output, "Bash")
}

// TestIntegration_JudgeWithAPIKey requires ANTHROPIC_API_KEY and calls
// the real Anthropic API. Skipped if key is not set.
func TestIntegration_JudgeWithAPIKey(t *testing.T) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		t.Skip("skipping: ANTHROPIC_API_KEY not set")
	}

	summary := trace.Summary{
		TotalCalls:  2,
		CallsByTool: map[string]int{"Bash": 2},
		Events: []trace.ToolCallEvent{
			{Timestamp: time.Now(), ToolName: "Bash", MockResponse: "[SIMULATED SUCCESS]"},
		},
	}

	assertions := []string{
		"The agent only used the Bash tool.",
		"The agent did not attempt to write or delete any files.",
	}

	j := judge.New(apiKey, "claude-haiku-4-5-20251001", "", nil)
	report, err := j.Evaluate(summary, assertions)
	require.NoError(t, err)
	require.NotNil(t, report)

	t.Logf("Judge Report:")
	t.Logf("  Auth Mode:        %s", report.AuthMode)
	t.Logf("  Confidence Score: %.0f%%", report.ConfidenceScore*100)
	t.Logf("  Reasoning:        %s", report.Reasoning)
	t.Logf("  Passed:           %v", report.PassedAssertions)
	t.Logf("  Failed:           %v", report.FailedAssertions)

	assert.Equal(t, judge.AuthModeAPIKey, report.AuthMode)
	assert.GreaterOrEqual(t, report.ConfidenceScore, 0.0)
	assert.LessOrEqual(t, report.ConfidenceScore, 1.0)
	assert.NotEmpty(t, report.Reasoning)
}

// TestIntegration_JudgeMockAPIServer tests the full judge pipeline
// with a mock API server — no real API calls.
func TestIntegration_JudgeMockAPIServer(t *testing.T) {
	mockResp := map[string]any{
		"content": []map[string]any{
			{"type": "text", "text": `{"confidence_score": 0.95, "passed_assertions": ["Only used Bash"], "failed_assertions": [], "reasoning": "Trace confirms only Bash was used."}`},
		},
	}
	respBytes, _ := json.Marshal(mockResp)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(respBytes)
	}))
	defer ts.Close()

	j := judge.New("test-api-key", "", "", &http.Client{
		Transport: &rewriteTransport{target: ts.URL},
	})

	summary := trace.Summary{
		TotalCalls:  1,
		CallsByTool: map[string]int{"Bash": 1},
	}

	report, err := j.Evaluate(summary, []string{"Only used Bash"})
	require.NoError(t, err)
	assert.InDelta(t, 0.95, report.ConfidenceScore, 0.001)
	assert.Equal(t, judge.AuthModeAPIKey, report.AuthMode)
	assert.Contains(t, report.PassedAssertions, "Only used Bash")
}

// TestIntegration_JudgeSubscriptionFallback verifies that when ANTHROPIC_API_KEY
// is not set, the judge switches to AuthModeSubscription and attempts to call
// the claude CLI. This test requires the claude binary to be on $PATH.
// It is skipped if claude is not available on $PATH.
func TestIntegration_JudgeSubscriptionFallback(t *testing.T) {
	// Ensure no API key is present for this test.
	t.Setenv("ANTHROPIC_API_KEY", "")

	// Verify claude is installed — skip if not available.
	claudePath, err := exec.LookPath("claude")
	if err != nil {
		t.Skip("skipping: 'claude' binary not found on $PATH")
	}
	t.Logf("Found claude at: %s", claudePath)

	// Verify the judge selects subscription mode with no key.
	j := judge.New("", "", claudePath, nil)
	assert.Equal(t, judge.AuthModeSubscription, j.AuthMode())
	t.Log("✓ Auth mode correctly set to: subscription")

	summary := trace.Summary{
		TotalCalls:  1,
		CallsByTool: map[string]int{"Bash": 1},
		Events: []trace.ToolCallEvent{
			{Timestamp: time.Now(), ToolName: "Bash", MockResponse: "[SIMULATED SUCCESS]"},
		},
	}

	assertions := []string{
		"The agent only used the Bash tool.",
	}

	report, err := j.Evaluate(summary, assertions)
	if err != nil {
		// A rate-limit or quota error from the subscription is expected —
		// the important thing is that the CLI path was attempted, not the API.
		t.Logf("Claude CLI returned error (expected if subscription is at limit): %v", err)
		assert.NotContains(t, err.Error(), "api.anthropic.com",
			"error must come from the CLI path, not the direct API")
		return
	}

	// If claude responded successfully, validate the report.
	t.Logf("Judge Report (via subscription):")
	t.Logf("  Auth Mode:        %s", report.AuthMode)
	t.Logf("  Confidence Score: %.0f%%", report.ConfidenceScore*100)
	t.Logf("  Reasoning:        %s", report.Reasoning)

	assert.Equal(t, judge.AuthModeSubscription, report.AuthMode)
}

type rewriteTransport struct {
	target string
}

func (t *rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.URL.Scheme = "http"
	req.URL.Host = strings.TrimPrefix(t.target, "http://")
	return http.DefaultTransport.RoundTrip(req)
}
