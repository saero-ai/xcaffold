package judge

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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

func mockAnthropicServer(responseBody string, statusCode int) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		io.WriteString(w, responseBody)
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
	j := New("sk-test-key", "", "", nil)
	assert.Equal(t, AuthModeAPIKey, j.authMode)
}

func TestNew_SubscriptionMode_WhenNoKey(t *testing.T) {
	j := New("", "", "", nil)
	assert.Equal(t, AuthModeSubscription, j.authMode)
}

func TestNew_DefaultJudgeModel(t *testing.T) {
	j := New("", "", "", nil)
	assert.Equal(t, defaultJudgeModel, j.model)
}

func TestNew_CustomClaudePath(t *testing.T) {
	j := New("", "", "/usr/local/bin/claude", nil)
	assert.Equal(t, "/usr/local/bin/claude", j.claudePath)
}

// --- No-assertion fast path ---

func TestEvaluate_NoAssertions_ReturnsEmptyReport(t *testing.T) {
	j := New("test-key", "", "", nil)
	report, err := j.Evaluate(makeSummary(), []string{})
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
	ts := mockAnthropicServer(string(respBytes), http.StatusOK)
	defer ts.Close()

	j := New("test-key", "", "", &http.Client{
		Transport: &rewriteTransport{base: ts.Client().Transport, target: ts.URL},
	})

	report, err := j.Evaluate(makeSummary(), []string{"Agent stayed in bounds"})
	require.NoError(t, err)
	assert.Equal(t, AuthModeAPIKey, report.AuthMode)
	assert.Equal(t, "PASS", report.Verdict)
	assert.Contains(t, report.PassedAssertions, "Agent stayed in bounds")
}

func TestEvaluate_APIKey_ErrorOnNon200(t *testing.T) {
	ts := mockAnthropicServer(`{"error":{"message":"unauthorized"}}`, http.StatusUnauthorized)
	defer ts.Close()

	j := New("bad-key", "", "", &http.Client{
		Transport: &rewriteTransport{base: ts.Client().Transport, target: ts.URL},
	})

	_, err := j.Evaluate(makeSummary(), []string{"some assertion"})
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

func TestParseCLIReport_StrictJSON(t *testing.T) {
	text := "### Check: A\n**Result:** PASS\n```json\n{\"verdict\": \"PASS\", \"passed_assertions\": [\"A\"], \"failed_assertions\": []}\n```"
	report := parseCLIReport("test-model", text)
	assert.Equal(t, "PASS", report.Verdict)
	assert.Equal(t, "test-model", report.Model)
}

func TestParseCLIReport_JSONEmbeddedInText(t *testing.T) {
	// Simulates a model that adds preamble before the JSON.
	text := "Here is my evaluation:\n```json\n{\"verdict\": \"PASS\", \"passed_assertions\": [], \"failed_assertions\": []}\n```"
	report := parseCLIReport("test-model", text)
	assert.Equal(t, "PASS", report.Verdict)
}

func TestParseCLIReport_FallsBackToRawText(t *testing.T) {
	// If no valid JSON, reasoning should be the raw text and Verdict should fall back to FAIL.
	text := "I cannot evaluate this trace."
	report := parseCLIReport("test-model", text)
	assert.Equal(t, text, report.Reasoning)
	assert.Equal(t, "FAIL", report.Verdict)
}
