package analyzer

import (
	"github.com/saero-ai/xcaffold/internal/ast"
)

// Analyzer handles static analysis of the AST without invoking network calls.
// In the future, this will integrate the Anthropic WASM tokenizer for exact
// BPE token counts. Currently uses a conservative heuristic baseline.
type Analyzer struct{}

// New returns a new Analyzer instance.
func New() *Analyzer {
	return &Analyzer{}
}

// AnalyzeTokens estimates the token usage for each agent in the config.
// The heuristic approximates 1 token per 4 characters of instructions,
// which is a conservative estimate sufficient for bloat detection.
// Returns a map of agent ID to estimated token count.
func (a *Analyzer) AnalyzeTokens(config *ast.XcaffoldConfig) map[string]int {
	report := make(map[string]int, len(config.Agents))
	for id, agent := range config.Agents {
		// Heuristic: ~4 chars per token (conservative BPE estimate).
		charCount := len(agent.Instructions) + len(agent.Description)
		report[id] = charCount / 4
	}
	return report
}
