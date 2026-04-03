package proxy

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/saero-ai/xcaffold/internal/trace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Host validation edge cases
// ---------------------------------------------------------------------------

// TestHandleRequest_SSRFSuffixConfusion verifies that a host like
// "evil-api.anthropic.com" — which ends with ".anthropic.com" — is still
// rejected (403) and does NOT slip through the EqualFold check.
func TestHandleRequest_SSRFSuffixConfusion(t *testing.T) {
	srv, _ := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "http://evil-api.anthropic.com/v1/messages", nil)
	req.Host = "evil-api.anthropic.com"
	w := httptest.NewRecorder()

	srv.handleRequest(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code,
		"suffix-confused host must be rejected with 403")
	assert.Contains(t, w.Body.String(), "forbidden")
}

// TestHandleRequest_SSRFCaseVariation verifies that "API.ANTHROPIC.COM"
// (all-caps) is accepted — the check uses EqualFold so case should not matter.
// The request will be forwarded (not a tool_use POST), so we expect NOT 403.
// Because we are not starting a real upstream, the response may be a 502/error
// from the reverse proxy — that is fine; we only care it is not 403.
func TestHandleRequest_SSRFCaseVariation(t *testing.T) {
	srv, _ := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "http://API.ANTHROPIC.COM/v1/messages", nil)
	req.Host = "API.ANTHROPIC.COM"
	w := httptest.NewRecorder()

	srv.handleRequest(w, req)

	assert.NotEqual(t, http.StatusForbidden, w.Code,
		"uppercase host API.ANTHROPIC.COM should NOT be forbidden (EqualFold is case-insensitive)")
}

// TestHandleRequest_EmptyHost verifies that a request where both r.Host and
// r.URL.Host are empty strings is rejected with 403.
func TestHandleRequest_EmptyHost(t *testing.T) {
	srv, _ := newTestServer(t)

	// httptest.NewRequest with a path-only URL leaves URL.Host empty.
	req := httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
	req.Host = "" // explicitly clear Host
	// req.URL.Host is already "" because we used a path-only target.

	w := httptest.NewRecorder()
	srv.handleRequest(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code,
		"empty host should be rejected with 403")
}

// TestHandleRequest_GETToMessages verifies that a GET to /v1/messages is NOT
// intercepted as a tool call (only POST is intercepted). It should be
// forwarded, so the response must not be 403.
func TestHandleRequest_GETToMessages(t *testing.T) {
	srv, _ := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "http://api.anthropic.com/v1/messages", nil)
	req.Host = anthropicAPIHost
	w := httptest.NewRecorder()

	srv.handleRequest(w, req)

	assert.NotEqual(t, http.StatusForbidden, w.Code,
		"GET to /v1/messages from a valid host must not return 403")
}

// ---------------------------------------------------------------------------
// extractToolName edge cases
// ---------------------------------------------------------------------------

// TestExtractToolName_EmptyNameField checks that a tool_use block with an
// explicit empty "name" field falls through to return "unknown".
func TestExtractToolName_EmptyNameField(t *testing.T) {
	body := []byte(`{"content":[{"type":"tool_use","name":""}]}`)
	result := extractToolName(body)
	assert.Equal(t, "unknown", result,
		"empty name string should fall back to \"unknown\"")
}

// ---------------------------------------------------------------------------
// extractInputParams edge cases
// ---------------------------------------------------------------------------

// TestExtractInputParams_NullInput checks that a tool_use block whose "input"
// field is JSON null returns an empty map (not nil, no panic).
func TestExtractInputParams_NullInput(t *testing.T) {
	body := []byte(`{"content":[{"type":"tool_use","input":null}]}`)
	result := extractInputParams(body)
	assert.NotNil(t, result, "result should be a non-nil map even when input is null")
	assert.Empty(t, result, "result should be empty when input is null")
}

// ---------------------------------------------------------------------------
// extractAgentID
// ---------------------------------------------------------------------------

// TestExtractAgentID_WithHeader verifies that the X-Xcaffold-Agent header
// value is returned verbatim.
func TestExtractAgentID_WithHeader(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("X-Xcaffold-Agent", "agent-007")

	id := extractAgentID(req)
	assert.Equal(t, "agent-007", id)
}

// TestExtractAgentID_NoHeader verifies the fallback is "unknown" when the
// header is absent.
func TestExtractAgentID_NoHeader(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	// Do not set X-Xcaffold-Agent header.

	id := extractAgentID(req)
	assert.Equal(t, "unknown", id)
}

// ---------------------------------------------------------------------------
// Concurrency
// ---------------------------------------------------------------------------

// TestProxy_ConcurrentRequests fires 20 simultaneous POST tool_use requests at
// the proxy server and asserts that all 20 events are recorded in the summary.
func TestProxy_ConcurrentRequests(t *testing.T) {
	const n = 20

	var buf bytes.Buffer
	rec := trace.NewRecorder(&buf)
	srv, err := New(rec)
	require.NoError(t, err)
	t.Cleanup(func() { _ = srv.Close() })

	// Start the server in the background.
	go func() { _ = srv.Start() }()

	addr := srv.Addr()
	proxyURL := "http://" + addr

	// Build a tool_use body.
	toolBody := []byte(`{"content":[{"type":"tool_use","name":"Bash","input":{"command":"ls"}}]}`)

	var wg sync.WaitGroup
	wg.Add(n)

	for i := 0; i < n; i++ {
		go func(i int) {
			defer wg.Done()

			// Send the request directly to the proxy's listener address,
			// but set the Host header so handleRequest passes the host check.
			url := proxyURL + "/v1/messages"
			req, err2 := http.NewRequest(http.MethodPost, url, bytes.NewReader(toolBody))
			if err2 != nil {
				t.Errorf("worker %d: failed to build request: %v", i, err2)
				return
			}
			req.Host = anthropicAPIHost
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Xcaffold-Agent", fmt.Sprintf("agent-%02d", i))

			resp, err3 := http.DefaultClient.Do(req)
			if err3 != nil {
				t.Errorf("worker %d: request failed: %v", i, err3)
				return
			}
			defer resp.Body.Close()
			io.ReadAll(resp.Body) //nolint:errcheck
		}(i)
	}

	wg.Wait()

	summary := rec.Summary()
	assert.Equal(t, n, summary.TotalCalls,
		"all %d concurrent tool_use requests must be recorded", n)
}
