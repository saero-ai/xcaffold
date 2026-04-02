package proxy

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/saero-ai/xcaffold/internal/trace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestServer(t *testing.T) (*Server, *trace.Recorder) {
	t.Helper()
	var buf bytes.Buffer
	rec := trace.NewRecorder(&buf)
	srv, err := New(rec)
	require.NoError(t, err)
	t.Cleanup(func() { _ = srv.Close() })
	return srv, rec
}

func TestNew_BindsToLoopback(t *testing.T) {
	srv, _ := newTestServer(t)
	assert.True(t, strings.HasPrefix(srv.Addr(), "127.0.0.1:"),
		"proxy must bind exclusively to loopback, got: %s", srv.Addr())
}

func TestNew_ProxyURL_Format(t *testing.T) {
	srv, _ := newTestServer(t)
	assert.True(t, strings.HasPrefix(srv.ProxyURL(), "http://127.0.0.1:"))
}

func TestHandleRequest_ForbidsNonAnthropicHost(t *testing.T) {
	srv, _ := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "http://evil.example.com/steal", nil)
	req.Host = "evil.example.com"
	w := httptest.NewRecorder()

	srv.handleRequest(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "forbidden")
}

func TestIsToolUseRequest_DetectsToolUse(t *testing.T) {
	body := []byte(`{"content":[{"type":"tool_use","name":"Bash","input":{"command":"ls"}}]}`)
	assert.True(t, isToolUseRequest(body))
}

func TestIsToolUseRequest_PassesThroughNonTool(t *testing.T) {
	body := []byte(`{"role":"user","content":[{"type":"text","text":"Hello"}]}`)
	assert.False(t, isToolUseRequest(body))
}

func TestExtractToolName_ReturnsName(t *testing.T) {
	body := []byte(`{"content":[{"type":"tool_use","name":"Write","input":{}}]}`)
	assert.Equal(t, "Write", extractToolName(body))
}

func TestExtractToolName_ReturnsFallbackOnBadJSON(t *testing.T) {
	assert.Equal(t, "unknown", extractToolName([]byte(`not-json`)))
}

func TestExtractInputParams_ParsesParams(t *testing.T) {
	body := []byte(`{"content":[{"type":"tool_use","name":"Bash","input":{"command":"npm test"}}]}`)
	params := extractInputParams(body)
	assert.Equal(t, "npm test", params["command"])
}

func TestHandleToolUse_WritesTrace(t *testing.T) {
	srv, rec := newTestServer(t)

	body := []byte(`{"content":[{"type":"tool_use","name":"Bash","input":{"command":"ls"}}]}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/messages", io.NopCloser(bytes.NewReader(body)))
	req.Host = anthropicAPIHost
	w := httptest.NewRecorder()

	srv.handleToolUse(w, req, body)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "simulated-response")

	summary := rec.Summary()
	assert.Equal(t, 1, summary.TotalCalls)
	assert.Equal(t, "Bash", summary.Events[0].ToolName)
}

func TestHandleToolUse_ResponseContainsMock(t *testing.T) {
	srv, _ := newTestServer(t)

	body := []byte(`{"content":[{"type":"tool_use","name":"Write","input":{}}]}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
	req.Host = anthropicAPIHost
	w := httptest.NewRecorder()

	srv.handleToolUse(w, req, body)

	assert.Contains(t, w.Body.String(), mockToolResponse)
}
