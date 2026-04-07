package proxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/saero-ai/xcaffold/internal/trace"
)

const (
	mockToolResponse  = "[SIMULATED SUCCESS]"
	proxyReadTimeout  = 30 * time.Second
	proxyWriteTimeout = 60 * time.Second
	unknownIdentity   = "unknown"
)

var allowedHosts = map[string]bool{
	"api.anthropic.com":                 true,
	"generativelanguage.googleapis.com": true,
	"api.cursor.sh":                     true,
}

// Server is the local intercept proxy. It listens on a random loopback port
// and intercepts AI provider tool-use calls, recording them without
// forwarding the tool execution to the host OS.
type Server struct {
	listener net.Listener
	server   *http.Server
	recorder *trace.Recorder
}

// New creates and binds a new proxy Server on a random loopback port.
// The caller must call Start() to begin serving, and Close() when done.
func New(recorder *trace.Recorder) (*Server, error) {
	// Bind exclusively to loopback — never expose to the network.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("proxy: failed to bind listener: %w", err)
	}

	s := &Server{
		listener: ln,
		recorder: recorder,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleRequest)

	s.server = &http.Server{
		Handler:      mux,
		ReadTimeout:  proxyReadTimeout,
		WriteTimeout: proxyWriteTimeout,
	}

	return s, nil
}

// Addr returns the local address the proxy is listening on (e.g., "127.0.0.1:54321").
func (s *Server) Addr() string {
	return s.listener.Addr().String()
}

// ProxyURL returns the HTTP proxy URL for injecting into a subprocess environment.
func (s *Server) ProxyURL() string {
	return "http://" + s.Addr()
}

// Start begins serving requests in the foreground. It blocks until the server
// is shut down. It returns a non-nil error only if the server failed to start
// (i.e., not the expected http.ErrServerClosed on normal shutdown).
func (s *Server) Start() error {
	if err := s.server.Serve(s.listener); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("proxy: server error: %w", err)
	}
	return nil
}

// Close gracefully shuts down the proxy server.
func (s *Server) Close() error {
	return s.server.Close()
}

// handleRequest is the central dispatcher. It validates the target host and
// decides whether to intercept (tool calls) or forward (everything else).
func (s *Server) handleRequest(w http.ResponseWriter, r *http.Request) {
	// Security: reject any request not targeting the allowed LLM APIs.
	// Use exact equality to prevent SSRF via suffix confusion (e.g. evil-api.anthropic.com).
	targetHost := r.Host
	if targetHost == "" {
		targetHost = r.URL.Host
	}

	targetHost = strings.ToLower(targetHost)
	if !allowedHosts[targetHost] {
		http.Error(w, "proxy: forbidden — host is not an allowed AI provider", http.StatusForbidden)
		return
	}

	// Only inspect known messaging endpoints for tool interception.
	if r.Method == http.MethodPost && (strings.HasSuffix(r.URL.Path, "/v1/messages") || strings.Contains(r.URL.Path, "generateContent")) {
		// Limit to 10MB to prevent OOM DOS from massive requests
		r.Body = http.MaxBytesReader(w, r.Body, 10*1024*1024)
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "proxy: payload too large", http.StatusRequestEntityTooLarge)
			return
		}
		r.Body = io.NopCloser(bytes.NewReader(body))

		if isToolUseRequest(body) {
			s.handleToolUse(w, r, body)
			return
		}

		// Restore body for pass-through.
		r.Body = io.NopCloser(bytes.NewReader(body))
	}

	s.forward(w, r)
}

// isToolUseRequest returns true if the request body contains a tool_use block.
// This is a lightweight check on the raw JSON to avoid full deserialization
// on every request.
func isToolUseRequest(body []byte) bool {
	return bytes.Contains(body, []byte(`"tool_use"`))
}

// handleToolUse intercepts a tool-use request, records the event, and returns
// a deterministic mock response to the LLM without executing the tool.
func (s *Server) handleToolUse(w http.ResponseWriter, r *http.Request, body []byte) {
	start := time.Now()

	// Extract tool name from the raw payload for the trace log.
	toolName := extractToolName(body)

	event := trace.ToolCallEvent{
		Timestamp:    start.UTC(),
		AgentID:      extractAgentID(r),
		ToolName:     toolName,
		InputParams:  extractInputParams(body),
		MockResponse: mockToolResponse,
		DurationMs:   time.Since(start).Milliseconds(),
	}

	// Best-effort trace recording — do not fail the response on write errors.
	_ = s.recorder.Record(event)

	// Return a well-formed mock response (Anthropic shaped fallback for now, generalize appropriately if needed).
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{
  "id": "simulated-response",
  "type": "message",
  "role": "assistant",
  "content": [{"type": "text", "text": %q}],
  "model": "simulated",
  "stop_reason": "end_turn",
  "usage": {"input_tokens": 0, "output_tokens": 1}
}`, mockToolResponse)
}

// forward passes the request transparently to the target LLM API.
func (s *Server) forward(w http.ResponseWriter, r *http.Request) {
	// Dynamically determine the host to proxy to
	host := r.Host
	if host == "" {
		host = r.URL.Host
	}
	target, _ := url.Parse("https://" + host)
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.Director = func(req *http.Request) {
		req.URL.Scheme = "https"
		req.URL.Host = host
		req.Host = host
	}
	proxy.ServeHTTP(w, r)
}

// extractToolName attempts to parse the tool name from the raw JSON body.
// Returns "unknown" if it cannot be determined, avoiding panics.
func extractToolName(body []byte) string {
	var payload struct {
		Content []struct {
			Type string `json:"type"`
			Name string `json:"name"`
		} `json:"content"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return unknownIdentity
	}
	for _, block := range payload.Content {
		if block.Type == "tool_use" && block.Name != "" {
			return block.Name
		}
	}
	return "unknown"
}

// extractInputParams attempts to parse tool input parameters from the body.
// Returns an empty map on any parse failure instead of panicking.
func extractInputParams(body []byte) map[string]any {
	var payload struct {
		Content []struct {
			Input map[string]any `json:"input"`
			Type  string         `json:"type"`
		} `json:"content"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return map[string]any{}
	}
	for _, block := range payload.Content {
		if block.Type == "tool_use" && block.Input != nil {
			return block.Input
		}
	}
	return map[string]any{}
}

// extractAgentID attempts to derive an agent identifier from a request header.
// Falls back to "unknown" if not present.
func extractAgentID(r *http.Request) string {
	if id := r.Header.Get("X-Xcaffold-Agent"); id != "" {
		return id
	}
	return unknownIdentity
}
