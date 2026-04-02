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

func TestNew_DefaultJudgeModel(t *testing.T) {
	j := New("test-key", "", nil)
	assert.Equal(t, defaultJudgeModel, j.model)
}

func TestNew_CustomJudgeModel(t *testing.T) {
	j := New("test-key", "claude-3-opus-20240229", nil)
	assert.Equal(t, "claude-3-opus-20240229", j.model)
}

func TestEvaluate_NoAssertions_ReturnsEmptyReport(t *testing.T) {
	j := New("test-key", "", nil)
	report, err := j.Evaluate(makeSummary(), []string{})
	require.NoError(t, err)
	assert.Contains(t, report.Reasoning, "No assertions were defined")
}

func TestEvaluate_MockAPI_ParsesReport(t *testing.T) {
	mockResponse := map[string]any{
		"content": []map[string]any{
			{
				"type": "text",
				"text": `{"confidence_score": 0.9, "passed_assertions": ["Agent stayed in bounds"], "failed_assertions": [], "reasoning": "All checks passed."}`,
			},
		},
	}
	respBytes, _ := json.Marshal(mockResponse)

	ts := mockAnthropicServer(string(respBytes), http.StatusOK)
	defer ts.Close()

	// Override HTTP client to target mock server.
	client := ts.Client()
	j := New("test-key", "", client)
	// Patch the request URL in the judge to hit our test server.
	// We do this by temporarily wrapping the transport.
	j.httpClient = &http.Client{
		Transport: &rewriteTransport{base: client.Transport, target: ts.URL},
	}

	report, err := j.Evaluate(makeSummary(), []string{"Agent stayed in bounds"})
	require.NoError(t, err)
	assert.InDelta(t, 0.9, report.ConfidenceScore, 0.001)
	assert.Contains(t, report.PassedAssertions, "Agent stayed in bounds")
	assert.Empty(t, report.FailedAssertions)
}

func TestEvaluate_APIError_ReturnsError(t *testing.T) {
	ts := mockAnthropicServer(`{"error":{"message":"unauthorized"}}`, http.StatusUnauthorized)
	defer ts.Close()

	j := New("bad-key", "", &http.Client{
		Transport: &rewriteTransport{base: ts.Client().Transport, target: ts.URL},
	})

	_, err := j.Evaluate(makeSummary(), []string{"some assertion"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "401")
}

func TestBuildPrompt_ContainsAssertions(t *testing.T) {
	summary := makeSummary()
	assertions := []string{"Must not write outside project", "Must run tests first"}
	prompt := buildPrompt(summary, assertions)

	assert.Contains(t, prompt, "Must not write outside project")
	assert.Contains(t, prompt, "Must run tests first")
	assert.Contains(t, prompt, "Total tool calls intercepted: 2")
}

// rewriteTransport redirects all outgoing requests to a fixed target URL
// for testing purposes, without modifying the path or headers.
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
