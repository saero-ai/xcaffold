package generator

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/saero-ai/xcaffold/internal/analyzer"
	"github.com/saero-ai/xcaffold/internal/auth"
	"github.com/saero-ai/xcaffold/internal/llmclient"
)

const defaultGeneratorModel = "claude-3-7-sonnet-20250219" // Sonnet is preferred for generation speed and context

type Generator struct {
	client *llmclient.Client
	model  string
}

// New constructs a Generator backed by llmclient. Returns an error if the
// configuration is invalid (e.g. SSRF-unsafe GenericAPIBase URL).
func New(anthropicKey, genericAPIKey, apiBaseURL, model, cliPath string, httpClient *http.Client) (*Generator, error) {
	if model == "" {
		model = defaultGeneratorModel
	}
	client, err := llmclient.New(llmclient.Config{
		AnthropicKey:   anthropicKey,
		GenericAPIKey:  genericAPIKey,
		GenericAPIBase: apiBaseURL,
		Model:          model,
		DefaultModel:   defaultGeneratorModel,
		CLIPath:        cliPath,
		MaxTokens:      4096,
		HTTPClient:     httpClient,
	})
	if err != nil {
		return nil, fmt.Errorf("generator: %w", err)
	}
	return &Generator{client: client, model: model}, nil
}

// AuthMode returns the active authentication mode.
func (g *Generator) AuthMode() auth.AuthMode {
	return g.client.AuthMode()
}

type GenerationResult struct {
	YAMLConfig string
	AuditJSON  string
}

// Generate generates the raw YAML scaffold file and audit report using the configured auth mode.
func (g *Generator) Generate(ctx context.Context, sig *analyzer.ProjectSignature) (*GenerationResult, error) {
	prompt := buildGeneratorPrompt(sig)
	text, err := g.client.Call(ctx, prompt)
	if err != nil {
		return nil, err
	}
	return parseJSONOutput(text)
}

// parseJSONOutput attempts to pull raw JSON out of markdown codeblocks
// and unmarshals it into the GenerationResult.
func parseJSONOutput(text string) (*GenerationResult, error) {
	text = strings.TrimSpace(text)
	lower := strings.ToLower(text)

	startToken := "```json"
	if !strings.Contains(lower, startToken) {
		startToken = "```"
	}

	start := strings.Index(lower, startToken)
	if start >= 0 {
		contentStart := start + len(startToken)
		end := strings.Index(text[contentStart:], "```")
		if end >= 0 {
			text = strings.TrimSpace(text[contentStart : contentStart+end])
		}
	}

	var output struct {
		YAMLConfig  string          `json:"yaml_config"`
		AuditReport json.RawMessage `json:"audit_report"`
	}

	if err := json.Unmarshal([]byte(text), &output); err != nil {
		return nil, fmt.Errorf("failed to parse output as dual JSON block: %w (raw output:\n%s)", err, text)
	}

	return &GenerationResult{
		YAMLConfig: output.YAMLConfig,
		AuditJSON:  string(output.AuditReport),
	}, nil
}
