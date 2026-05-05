package judge

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/saero-ai/xcaffold/internal/auth"
	"github.com/saero-ai/xcaffold/internal/trace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeSummary() trace.Summary {
	return trace.Summary{
		TotalCalls:  2,
		CallsByTool: map[string]int{"Bash": 2},
		Events: []trace.ToolCallEvent{
			{Timestamp: time.Now(), ToolName: "Bash", MockResponse: "[SIMULATED SUCCESS]"},
		},
	}
}

func mockLLMServer(responseBody string, statusCode int) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		_, _ = io.WriteString(w, responseBody)
	}))
}

// rewriteTransport redirects requests to a fixed target URL for testing.
type rewriteTransport struct {
	base   http.RoundTripper
	target string
}

func (t *rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.URL.Scheme = "http"
	req.URL.Host = strings.TrimPrefix(t.target, "http://")
	if t.base != nil {
		return t.base.RoundTrip(req)
	}
	return http.DefaultTransport.RoundTrip(req)
}

// --- Mode selection tests ---

func TestNew_APIKeyMode_WhenKeyProvided(t *testing.T) {
	j, err := New("sk-test-key", "", "", "claude-haiku-4-5", "", nil)
	require.NoError(t, err)
	assert.Equal(t, auth.AuthModeAPIKey, j.AuthMode())
}

func TestNew_SubscriptionMode_WhenNoKey(t *testing.T) {
	j, err := New("", "", "", "claude-haiku-4-5", "", nil)
	require.NoError(t, err)
	assert.Equal(t, auth.AuthModeSubscription, j.AuthMode())
}

func TestNew_EmptyModel_ReturnsError(t *testing.T) {
	j, err := New("sk-test-key", "", "", "", "", nil)
	require.Error(t, err)
	assert.Nil(t, j)
	assert.Contains(t, err.Error(), "model must be specified")
}

func TestNew_CustomClaudePath(t *testing.T) {
	j, err := New("", "", "", "claude-haiku-4-5", "/usr/local/bin/claude", nil)
	require.NoError(t, err)
	assert.NotNil(t, j)
}

// --- No-assertion fast path ---

func TestEvaluate_NoAssertions_ReturnsEmptyReport(t *testing.T) {
	j, err := New("test-key", "", "", "claude-haiku-4-5", "", nil)
	require.NoError(t, err)
	report, err := j.Evaluate(context.Background(), makeSummary(), []string{})
	require.NoError(t, err)
	assert.Contains(t, report.Reasoning, "No assertions were defined")
}

// --- API key path tests ---

func TestEvaluate_APIKey_ParsesReport(t *testing.T) {
	mockResponse := map[string]any{
		"content": []map[string]any{
			{
				"type": "text",
				"text": "### Check: Agent stayed in bounds\n**Result:** PASS\n```json\n{\"verdict\": \"PASS\", \"passed_assertions\": [\"Agent stayed in bounds\"], \"failed_assertions\": []}\n```",
			},
		},
	}
	respBytes, _ := json.Marshal(mockResponse)
	ts := mockLLMServer(string(respBytes), http.StatusOK)
	defer ts.Close()

	j, err := New("test-key", "", "", "claude-haiku-4-5", "", &http.Client{
		Transport: &rewriteTransport{base: ts.Client().Transport, target: ts.URL},
	})
	require.NoError(t, err)

	report, err := j.Evaluate(context.Background(), makeSummary(), []string{"Agent stayed in bounds"})
	require.NoError(t, err)
	assert.Equal(t, auth.AuthModeAPIKey, report.AuthMode)
	assert.Equal(t, "PASS", report.Verdict)
	assert.Contains(t, report.PassedAssertions, "Agent stayed in bounds")
}

func TestEvaluate_APIKey_ErrorOnNon200(t *testing.T) {
	ts := mockLLMServer(`{"error":{"message":"unauthorized"}}`, http.StatusUnauthorized)
	defer ts.Close()

	j, err := New("bad-key", "", "", "claude-haiku-4-5", "", &http.Client{
		Transport: &rewriteTransport{base: ts.Client().Transport, target: ts.URL},
	})
	require.NoError(t, err)

	_, err = j.Evaluate(context.Background(), makeSummary(), []string{"some assertion"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "401")
}

// --- Prompt construction tests ---

func TestBuildPrompt_ContainsAssertions(t *testing.T) {
	summary := makeSummary()
	assertions := []string{"Must not write outside project", "Must run tests first"}
	prompt := buildPrompt(summary, assertions)

	assert.Contains(t, prompt, "Must not write outside project")
	assert.Contains(t, prompt, "Must run tests first")
	assert.Contains(t, prompt, "Total tool calls intercepted: 2")
	assert.Contains(t, prompt, "Bash: 2 call(s)")
}

// --- JSON parsing robustness tests ---

func TestParseTextReport_StrictJSON(t *testing.T) {
	text := "### Check: A\n**Result:** PASS\n```json\n{\"verdict\": \"PASS\", \"passed_assertions\": [\"A\"], \"failed_assertions\": []}\n```"
	report := parseTextReport("test-model", text)
	assert.Equal(t, "PASS", report.Verdict)
	assert.Equal(t, "test-model", report.Model)
}

func TestParseTextReport_JSONEmbeddedInText(t *testing.T) {
	// Simulates a model that adds preamble before the JSON.
	text := "Here is my evaluation:\n```json\n{\"verdict\": \"PASS\", \"passed_assertions\": [], \"failed_assertions\": []}\n```"
	report := parseTextReport("test-model", text)
	assert.Equal(t, "PASS", report.Verdict)
}

func TestParseTextReport_FallsBackToRawText(t *testing.T) {
	// If no valid JSON, reasoning should be the raw text and Verdict should fall back to FAIL.
	text := "I cannot evaluate this trace."
	report := parseTextReport("test-model", text)
	assert.Equal(t, text, report.Reasoning)
	assert.Equal(t, "FAIL", report.Verdict)
}

func TestParseTextReport_NormalizesVerdict(t *testing.T) {
	text := "```json\n{\"verdict\": \"pass\", \"passed_assertions\": [\"A\"], \"failed_assertions\": []}\n```"
	report := parseTextReport("test-model", text)
	assert.Equal(t, "PASS", report.Verdict)
}

// --- GenericAPI (OpenAI-compatible) path tests ---

func TestEvaluate_GenericAPI_ParsesReport(t *testing.T) {
	mockResponse := map[string]any{
		"choices": []map[string]any{
			{"message": map[string]any{
				"content": "### Check: Agent was sandboxed\n**Result:** PASS\n```json\n{\"verdict\": \"PASS\", \"passed_assertions\": [\"Agent was sandboxed\"], \"failed_assertions\": []}\n```",
			}},
		},
	}
	respBytes, _ := json.Marshal(mockResponse)
	ts := mockLLMServer(string(respBytes), http.StatusOK)
	defer ts.Close()

	j, err := New("", "generic-test-key", ts.URL, "gpt-4o", "", &http.Client{
		Transport: &rewriteTransport{base: ts.Client().Transport, target: ts.URL},
	})
	require.NoError(t, err)

	report, err := j.Evaluate(context.Background(), makeSummary(), []string{"Agent was sandboxed"})
	require.NoError(t, err)
	assert.Equal(t, auth.AuthModeGenericAPI, report.AuthMode)
	assert.Equal(t, "PASS", report.Verdict)
}

func TestNew_GenericAPIKeyTakesPrecedence(t *testing.T) {
	j, err := New("target-key", "generic-key", "", "claude-haiku-4-5", "", nil)
	require.NoError(t, err)
	assert.Equal(t, auth.AuthModeGenericAPI, j.AuthMode())
}

func TestNew_RejectsInvalidBaseURL(t *testing.T) {
	_, err := New("", "key", "http://169.254.169.254", "gpt-4o", "", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "prohibited")
}

func TestBuildPrompt_IncludesEventDetails(t *testing.T) {
	summary := trace.Summary{
		TotalCalls:  1,
		CallsByTool: map[string]int{"Bash": 1},
		Events: []trace.ToolCallEvent{
			{
				Timestamp:    time.Now(),
				ToolName:     "Bash",
				AgentID:      "backend-dev",
				InputParams:  map[string]any{"command": "ls -la"},
				MockResponse: "[SIMULATED SUCCESS]",
			},
		},
	}
	prompt := buildPrompt(summary, []string{"Must use Bash"})
	assert.Contains(t, prompt, "Detailed Tool Call Log")
	assert.Contains(t, prompt, "ls -la")
	assert.Contains(t, prompt, "backend-dev")
}

func TestBuildPrompt_TruncatesLongAssertions(t *testing.T) {
	longAssertion := strings.Repeat("A", 600)
	prompt := buildPrompt(makeSummary(), []string{longAssertion})
	assert.Contains(t, prompt, "... (truncated)")
	assert.NotContains(t, prompt, strings.Repeat("A", 600))
}
