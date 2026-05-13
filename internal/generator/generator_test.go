package generator

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/saero-ai/xcaffold/internal/analyzer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

func mockAnthropicServer(responseBody string, statusCode int) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		_, _ = io.WriteString(w, responseBody)
	}))
}

func TestNew_EmptyModel_ReturnsError(t *testing.T) {
	g, err := New(GeneratorConfig{
		AnthropicKey:  "test-key",
		GenericAPIKey: "",
		APIBaseURL:    "",
		Model:         "",
		CLIPath:       "",
		HTTPClient:    nil,
	})
	require.Error(t, err)
	assert.Nil(t, g)
	assert.Contains(t, err.Error(), "model must be specified")
}

func TestParseJSONOutput(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		checkConfig string
		expectError bool
	}{
		{
			name:        "No markdown block",
			input:       `{"yaml_config": "project: raw", "audit_report": {}}`,
			checkConfig: "project: raw",
		},
		{
			name:        "Markdown json block",
			input:       "Here is it:\n```json\n{\"yaml_config\": \"project: json\"}\n```\nDone.",
			checkConfig: "project: json",
		},
		{
			name:        "Markdown unlabeled block",
			input:       "```\n{\"yaml_config\": \"project: unlabeled\"}\n```",
			checkConfig: "project: unlabeled",
		},
		{
			name:        "Invalid JSON",
			input:       "invalid json",
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			res, err := parseJSONOutput(tc.input)
			if tc.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.NotNil(t, res)
				assert.Equal(t, tc.checkConfig, res.YAMLConfig)
			}
		})
	}
}

func TestGenerateViaAPI(t *testing.T) {
	mockResponse := map[string]any{
		"content": []map[string]any{
			{
				"type": "text",
				"text": "```json\n{\"yaml_config\": \"project: generated\", \"audit_report\": {}}\n```",
			},
		},
	}
	respBytes, _ := json.Marshal(mockResponse)
	ts := mockAnthropicServer(string(respBytes), http.StatusOK)
	defer ts.Close()

	g, err := New(GeneratorConfig{
		AnthropicKey:  "test-key",
		GenericAPIKey: "",
		APIBaseURL:    "",
		Model:         "claude-sonnet-4-6",
		CLIPath:       "",
		HTTPClient: &http.Client{
			Transport: &rewriteTransport{base: ts.Client().Transport, target: ts.URL},
		},
	})
	require.NoError(t, err)

	sig := &analyzer.ProjectSignature{
		Files: []string{"package.json"},
	}

	res, err := g.Generate(context.Background(), sig)
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.Equal(t, "project: generated", res.YAMLConfig)
}

func TestBuildGeneratorPrompt(t *testing.T) {
	sig := &analyzer.ProjectSignature{
		Files: []string{"package.json", "src/"},
		DependencyManifests: map[string]string{
			"package.json": "{}",
		},
	}

	prompt := buildGeneratorPrompt(sig)

	// Check that we inject the signature
	assert.Contains(t, prompt, "package.json")
	// Check that we enforce adversarial checks
	assert.Contains(t, prompt, "High-Quality, Adversarial check statements")
	// Check that we assert the dual output format
	assert.Contains(t, prompt, "audit_report")
}
