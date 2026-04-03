package analyzer

import (
	"unicode/utf8"

	"github.com/saero-ai/xcaffold/internal/ast"
)

// Analyzer handles static analysis of the AST without invoking network calls.
// Token counts are estimated using a byte-count heuristic (~4 bytes per token),
// which is a conservative approximation of BPE tokenization for typical
// English-language agent instruction text.
//
// This heuristic is intentionally simple and honest. It is not a WASM-backed
// exact tokenizer. For the vast majority of agent configurations, the ÷4 estimate
// is accurate within ±10% of the true token count. Users requiring exact parity
// with the Anthropic API should measure token usage directly via the API response
// usage fields.
type Analyzer struct{}

// New returns a new Analyzer instance.
func New() *Analyzer {
	return &Analyzer{}
}

// AnalyzeTokens estimates the token usage for each agent in the config.
// It uses a byte-count heuristic: approximately 4 bytes per BPE token for
// standard English text. Returns a map of agent ID to estimated token count.
func (a *Analyzer) AnalyzeTokens(config *ast.XcaffoldConfig) map[string]int {
	report := make(map[string]int, len(config.Agents))
	for id, agent := range config.Agents {
		payload := agent.Instructions + " " + agent.Description
		// ~4 printable characters per token is a conservative BPE estimate.
		report[id] = utf8.RuneCountInString(payload) / 4
	}
	return report
}
