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

type Generator struct {
	client *llmclient.Client
	model  string
}

// GeneratorConfig holds initialization options for a Generator.
type GeneratorConfig struct {
	AnthropicKey  string
	GenericAPIKey string
	APIBaseURL    string
	Model         string
	CLIPath       string
	HTTPClient    *http.Client
}

// New constructs a Generator backed by llmclient. Returns an error if the
// configuration is invalid (e.g. SSRF-unsafe GenericAPIBase URL, or missing model).
func New(cfg GeneratorConfig) (*Generator, error) {
	if cfg.Model == "" {
		return nil, fmt.Errorf("generator: model must be specified (cannot be empty)")
	}
	client, err := llmclient.New(llmclient.Config{
		AnthropicKey:   cfg.AnthropicKey,
		GenericAPIKey:  cfg.GenericAPIKey,
		GenericAPIBase: cfg.APIBaseURL,
		Model:          cfg.Model,
		CLIPath:        cfg.CLIPath,
		MaxTokens:      4096,
		HTTPClient:     cfg.HTTPClient,
	})
	if err != nil {
		return nil, fmt.Errorf("generator: %w", err)
	}
	return &Generator{client: client, model: cfg.Model}, nil
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
