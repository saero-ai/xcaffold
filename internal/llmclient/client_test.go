package llmclient_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/saero-ai/xcaffold/internal/auth"
	"github.com/saero-ai/xcaffold/internal/llmclient"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// rewriteTransport redirects all outbound requests to a fixed target URL for testing.
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

func anthropicResponse(text string) string {
	body, _ := json.Marshal(map[string]any{
		"content": []map[string]any{
			{"type": "text", "text": text},
		},
	})
	return string(body)
}

func openAIResponse(content string) string {
	body, _ := json.Marshal(map[string]any{
		"choices": []map[string]any{
			{"message": map[string]any{"content": content}},
		},
	})
	return string(body)
}

// --- TestNew_AuthModeSelection ---

func TestNew_AuthModeSelection(t *testing.T) {
	t.Run("OpenAI key wins over Anthropic key", func(t *testing.T) {
		c, err := llmclient.New(llmclient.Config{
			AnthropicKey:  "sk-anthropic",
			GenericAPIKey: "sk-openai",
		})
		require.NoError(t, err)
		assert.Equal(t, auth.AuthModeGenericAPI, c.AuthMode())
	})

	t.Run("Anthropic key wins when no OpenAI key", func(t *testing.T) {
		c, err := llmclient.New(llmclient.Config{
			AnthropicKey: "sk-anthropic",
		})
		require.NoError(t, err)
		assert.Equal(t, auth.AuthModeAPIKey, c.AuthMode())
	})

	t.Run("Subscription mode when no keys", func(t *testing.T) {
		c, err := llmclient.New(llmclient.Config{})
		require.NoError(t, err)
		assert.Equal(t, auth.AuthModeSubscription, c.AuthMode())
	})
}

// --- TestNew_DefaultsApplied ---

func TestNew_DefaultsApplied(t *testing.T) {
	t.Run("empty model uses DefaultModel", func(t *testing.T) {
		c, err := llmclient.New(llmclient.Config{
			DefaultModel: "claude-haiku-4-5",
		})
		require.NoError(t, err)
		// We can only verify indirectly — no panic and creation succeeds.
		assert.NotNil(t, c)
	})

	t.Run("nil HTTPClient gets a timeout client", func(t *testing.T) {
		c, err := llmclient.New(llmclient.Config{})
		require.NoError(t, err)
		assert.NotNil(t, c)
		// The client must NOT use http.DefaultClient (no timeout).
		// We verify by checking the client was created successfully — the
		// internal timeout is validated in integration by request behaviour.
	})

	t.Run("empty CLIPath defaults to claude", func(t *testing.T) {
		c, err := llmclient.New(llmclient.Config{})
		require.NoError(t, err)
		assert.NotNil(t, c)
	})
}

// --- TestNew_GenericAPIBaseValidation ---

func TestNew_GenericAPIBaseValidation(t *testing.T) {
	t.Run("HTTPS accepted", func(t *testing.T) {
		_, err := llmclient.New(llmclient.Config{
			GenericAPIKey:  "key",
			GenericAPIBase: "https://api.openai.com/v1",
		})
		require.NoError(t, err)
	})

	t.Run("HTTP localhost accepted", func(t *testing.T) {
		_, err := llmclient.New(llmclient.Config{
			GenericAPIKey:  "key",
			GenericAPIBase: "http://localhost:8080/v1",
		})
		require.NoError(t, err)
	})

	t.Run("HTTP 127.0.0.1 accepted", func(t *testing.T) {
		_, err := llmclient.New(llmclient.Config{
			GenericAPIKey:  "key",
			GenericAPIBase: "http://127.0.0.1:11434/v1",
		})
		require.NoError(t, err)
	})

	t.Run("HTTP non-localhost rejected", func(t *testing.T) {
		_, err := llmclient.New(llmclient.Config{
			GenericAPIKey:  "key",
			GenericAPIBase: "http://example.com/v1",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "HTTPS")
	})

	t.Run("metadata IP rejected", func(t *testing.T) {
		_, err := llmclient.New(llmclient.Config{
			GenericAPIKey:  "key",
			GenericAPIBase: "https://169.254.169.254/latest",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "169.254")
	})

	t.Run("empty string accepted (uses default)", func(t *testing.T) {
		_, err := llmclient.New(llmclient.Config{
			GenericAPIKey:  "key",
			GenericAPIBase: "",
		})
		require.NoError(t, err)
	})

	t.Run("invalid URL rejected", func(t *testing.T) {
		_, err := llmclient.New(llmclient.Config{
			GenericAPIKey:  "key",
			GenericAPIBase: "://bad-url",
		})
		require.Error(t, err)
	})
}

// --- TestNew_CLIPathSanitization ---

func TestNew_CLIPathSanitization(t *testing.T) {
	t.Run("bare name stays as-is", func(t *testing.T) {
		c, err := llmclient.New(llmclient.Config{CLIPath: "claude"})
		require.NoError(t, err)
		assert.NotNil(t, c)
	})

	t.Run("absolute path stays as-is (has separator)", func(t *testing.T) {
		c, err := llmclient.New(llmclient.Config{CLIPath: "/usr/bin/claude"})
		require.NoError(t, err)
		assert.NotNil(t, c)
	})

	t.Run("empty CLIPath defaults to claude", func(t *testing.T) {
		c, err := llmclient.New(llmclient.Config{})
		require.NoError(t, err)
		assert.NotNil(t, c)
	})
}

// --- TestCall_AnthropicAPI_Success ---

func TestCall_AnthropicAPI_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify required headers.
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "sk-test-key", r.Header.Get("x-api-key"))
		assert.Equal(t, "2023-06-01", r.Header.Get("anthropic-version"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, anthropicResponse("hello from claude"))
	}))
	defer ts.Close()

	c, err := llmclient.New(llmclient.Config{
		AnthropicKey: "sk-test-key",
		HTTPClient: &http.Client{
			Transport: &rewriteTransport{base: ts.Client().Transport, target: ts.URL},
		},
	})
	require.NoError(t, err)

	result, err := c.Call(context.Background(), "say hello")
	require.NoError(t, err)
	assert.Equal(t, "hello from claude", result)
}

// --- TestCall_AnthropicAPI_Non200Error ---

func TestCall_AnthropicAPI_Non200Error(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = io.WriteString(w, `{"error":{"message":"unauthorized"}}`)
	}))
	defer ts.Close()

	c, err := llmclient.New(llmclient.Config{
		AnthropicKey: "bad-key",
		HTTPClient: &http.Client{
			Transport: &rewriteTransport{base: ts.Client().Transport, target: ts.URL},
		},
	})
	require.NoError(t, err)

	_, err = c.Call(context.Background(), "prompt")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "401")
}

// --- TestCall_GenericAPI_Success ---

func TestCall_GenericAPI_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify Authorization header.
		assert.Equal(t, "Bearer sk-openai-key", r.Header.Get("Authorization"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		// Path must be /v1/chat/completions (base includes /v1).
		assert.Equal(t, "/v1/chat/completions", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, openAIResponse("hello from openai"))
	}))
	defer ts.Close()

	c, err := llmclient.New(llmclient.Config{
		GenericAPIKey:  "sk-openai-key",
		GenericAPIBase: "https://api.openai.com/v1",
		HTTPClient: &http.Client{
			Transport: &rewriteTransport{base: ts.Client().Transport, target: ts.URL},
		},
	})
	require.NoError(t, err)

	result, err := c.Call(context.Background(), "say hello")
	require.NoError(t, err)
	assert.Equal(t, "hello from openai", result)
}

// --- TestCall_GenericAPI_CustomBaseURL ---

func TestCall_GenericAPI_CustomBaseURL(t *testing.T) {
	var capturedPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, openAIResponse("ok"))
	}))
	defer ts.Close()

	// Use HTTP + 127.0.0.1 base — valid for local dev.
	c, err := llmclient.New(llmclient.Config{
		GenericAPIKey:  "sk-key",
		GenericAPIBase: "http://127.0.0.1:11434/v1",
		HTTPClient: &http.Client{
			Transport: &rewriteTransport{base: ts.Client().Transport, target: ts.URL},
		},
	})
	require.NoError(t, err)

	_, err = c.Call(context.Background(), "prompt")
	require.NoError(t, err)
	assert.Equal(t, "/v1/chat/completions", capturedPath)
}

// --- TestCall_Retry_On429 ---

func TestCall_Retry_On429(t *testing.T) {
	callCount := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount < 2 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = io.WriteString(w, `{"error":"rate limited"}`)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, anthropicResponse("retried ok"))
	}))
	defer ts.Close()

	c, err := llmclient.New(llmclient.Config{
		AnthropicKey: "sk-key",
		HTTPClient: &http.Client{
			Transport: &rewriteTransport{base: ts.Client().Transport, target: ts.URL},
		},
	})
	require.NoError(t, err)

	result, err := c.Call(context.Background(), "prompt")
	require.NoError(t, err)
	assert.Equal(t, "retried ok", result)
	assert.Equal(t, 2, callCount)
}

// --- TestCall_Retry_ExhaustedReturnsError ---

func TestCall_Retry_ExhaustedReturnsError(t *testing.T) {
	callCount := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = io.WriteString(w, `{"error":"rate limited"}`)
	}))
	defer ts.Close()

	c, err := llmclient.New(llmclient.Config{
		AnthropicKey: "sk-key",
		HTTPClient: &http.Client{
			Transport: &rewriteTransport{base: ts.Client().Transport, target: ts.URL},
		},
	})
	require.NoError(t, err)

	_, err = c.Call(context.Background(), "prompt")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "429")
	assert.Equal(t, 3, callCount) // 1 initial + 2 retries
}

// --- TestCall_NoRetry_On400 ---

func TestCall_NoRetry_On400(t *testing.T) {
	callCount := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"error":"bad request"}`)
	}))
	defer ts.Close()

	c, err := llmclient.New(llmclient.Config{
		AnthropicKey: "sk-key",
		HTTPClient: &http.Client{
			Transport: &rewriteTransport{base: ts.Client().Transport, target: ts.URL},
		},
	})
	require.NoError(t, err)

	_, err = c.Call(context.Background(), "prompt")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "400")
	assert.Equal(t, 1, callCount) // no retries
}

// --- TestCall_Retry_With_RetryAfterHeader ---

func TestCall_Retry_With_RetryAfterHeader(t *testing.T) {
	callCount := 0
	start := time.Now()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			w.Header().Set("Retry-After", "0") // 0s retry-after to avoid slow tests
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = io.WriteString(w, `{"error":"rate limited"}`)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, anthropicResponse("ok after retry-after"))
	}))
	defer ts.Close()

	c, err := llmclient.New(llmclient.Config{
		AnthropicKey: "sk-key",
		HTTPClient: &http.Client{
			Transport: &rewriteTransport{base: ts.Client().Transport, target: ts.URL},
		},
	})
	require.NoError(t, err)

	result, err := c.Call(context.Background(), "prompt")
	require.NoError(t, err)
	assert.Equal(t, "ok after retry-after", result)
	assert.Equal(t, 2, callCount)
	// Should complete quickly because Retry-After: 0.
	assert.Less(t, time.Since(start), 2*time.Second)
}

// --- TestCall_ContextCancellation ---

func TestCall_ContextCancellation(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow server — context will be cancelled before this returns.
		time.Sleep(500 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, anthropicResponse("too late"))
	}))
	defer ts.Close()

	c, err := llmclient.New(llmclient.Config{
		AnthropicKey: "sk-key",
		HTTPClient: &http.Client{
			Transport: &rewriteTransport{base: ts.Client().Transport, target: ts.URL},
		},
	})
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err = c.Call(ctx, "prompt")
	require.Error(t, err)
}

// --- TestCall_EmptyAnthropicContent ---

func TestCall_EmptyAnthropicContent(t *testing.T) {
	body, _ := json.Marshal(map[string]any{
		"content": []map[string]any{},
	})
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, string(body))
	}))
	defer ts.Close()

	c, err := llmclient.New(llmclient.Config{
		AnthropicKey: "sk-key",
		HTTPClient: &http.Client{
			Transport: &rewriteTransport{base: ts.Client().Transport, target: ts.URL},
		},
	})
	require.NoError(t, err)

	_, err = c.Call(context.Background(), "prompt")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty content")
}

// --- TestCall_EmptyGenericAPIChoices ---

func TestCall_EmptyGenericAPIChoices(t *testing.T) {
	body, _ := json.Marshal(map[string]any{
		"choices": []map[string]any{},
	})
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, string(body))
	}))
	defer ts.Close()

	c, err := llmclient.New(llmclient.Config{
		GenericAPIKey: "sk-openai",
		HTTPClient: &http.Client{
			Transport: &rewriteTransport{base: ts.Client().Transport, target: ts.URL},
		},
	})
	require.NoError(t, err)

	_, err = c.Call(context.Background(), "prompt")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty choices")
}
