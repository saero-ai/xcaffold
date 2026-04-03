package generator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"

	"github.com/saero-ai/xcaffold/internal/analyzer"
)

const defaultGeneratorModel = "claude-3-5-sonnet-20241022" // Sonnet is preferred for generation speed and context

type AuthMode string

const (
	AuthModeAPIKey       AuthMode = "api_key"
	AuthModeSubscription AuthMode = "subscription"
)

type Generator struct {
	apiKey     string
	model      string
	claudePath string
	authMode   AuthMode
	httpClient *http.Client
}

// New returns a Generator. Automatically selects AuthMode based on apiKey presence.
func New(apiKey, model, claudePath string, httpClient *http.Client) *Generator {
	if model == "" {
		model = defaultGeneratorModel
	}
	if claudePath == "" {
		claudePath = "claude"
	}
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	mode := AuthModeSubscription
	if apiKey != "" {
		mode = AuthModeAPIKey
	}

	return &Generator{
		apiKey:     apiKey,
		model:      model,
		claudePath: claudePath,
		authMode:   mode,
		httpClient: httpClient,
	}
}

// AuthMode returns the active authentication mode.
func (g *Generator) AuthMode() AuthMode {
	return g.authMode
}

type GenerationResult struct {
	YAMLConfig string
	AuditJSON  string
}

// Generate generates the raw YAML scaffold file and audit report using the specified dual-auth mode.
func (g *Generator) Generate(sig *analyzer.ProjectSignature) (*GenerationResult, error) {
	prompt := buildGeneratorPrompt(sig)

	switch g.authMode {
	case AuthModeAPIKey:
		return g.generateViaAPI(prompt)
	case AuthModeSubscription:
		return g.generateViaCLI(prompt)
	default:
		return nil, fmt.Errorf("generator: unknown auth mode %q", g.authMode)
	}
}

func (g *Generator) generateViaAPI(prompt string) (*GenerationResult, error) {
	reqBody, err := json.Marshal(map[string]any{
		"model":      g.model,
		"max_tokens": 4096,
		"messages": []map[string]any{
			{"role": "user", "content": prompt},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("generator: failed to build API request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, "https://api.anthropic.com/v1/messages", bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("generator: failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", g.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("generator: API call failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("generator: failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("generator: API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var apiResp struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("generator: failed to parse API response: %w", err)
	}
	if len(apiResp.Content) == 0 {
		return nil, fmt.Errorf("generator: empty content in API response")
	}

	return parseJSONOutput(apiResp.Content[0].Text)
}

func (g *Generator) generateViaCLI(prompt string) (*GenerationResult, error) {
	// Let the output stream directly to the terminal so the user sees the
	// spinning Claude UI during the potentially long generation phase.
	// We use standard input/output pipes to print cleanly.
	cmd := exec.Command(g.claudePath, "-p", prompt) //nolint:gosec

	// Unlike the judge (which wants silence), generator might take 10+ seconds.
	// However, using -p guarantees raw output. If we pipe it, we might get terminal artifacts.
	// Let's capture stdout cleanly, and only pipe stderr which holds the progress spinner for `claude`.
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		stderrStr := strings.TrimSpace(stderr.String())
		if stderrStr != "" {
			return nil, fmt.Errorf("generator: claude CLI failed: %w — %s", err, stderrStr)
		}
		return nil, fmt.Errorf("generator: claude CLI failed: %w", err)
	}

	text := strings.TrimSpace(stdout.String())
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
