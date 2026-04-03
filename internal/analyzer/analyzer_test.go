package analyzer

import (
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
)

func TestAnalyzeTokens_HeuristicIsCorrect(t *testing.T) {
	a := New()

	config := &ast.XcaffoldConfig{
		Agents: map[string]ast.AgentConfig{
			"agent-one": {
				Instructions: "This is a test instruction.",
				Description:  "This is a test description.",
			},
			"agent-two": {
				Instructions: "Short context block.",
			},
		},
	}

	report := a.AnalyzeTokens(config)

	// Verify the ÷4 heuristic produces the expected result for each agent.
	payloadOne := config.Agents["agent-one"].Instructions + " " + config.Agents["agent-one"].Description
	expectedTokensOne := len(payloadOne) / 4
	if report["agent-one"] != expectedTokensOne {
		t.Errorf("agent-one: expected %d tokens, got %d", expectedTokensOne, report["agent-one"])
	}

	payloadTwo := config.Agents["agent-two"].Instructions + " " + config.Agents["agent-two"].Description
	expectedTokensTwo := len(payloadTwo) / 4
	if report["agent-two"] != expectedTokensTwo {
		t.Errorf("agent-two: expected %d tokens, got %d", expectedTokensTwo, report["agent-two"])
	}
}

func TestAnalyzeTokens_EmptyConfig(t *testing.T) {
	a := New()
	config := &ast.XcaffoldConfig{}
	report := a.AnalyzeTokens(config)
	if len(report) != 0 {
		t.Errorf("expected empty report for empty config, got %v", report)
	}
}
