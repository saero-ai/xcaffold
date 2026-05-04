package renderer_test

import (
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/renderer"
	"github.com/stretchr/testify/require"
)

func TestResolvedActivation_ExplicitActivation(t *testing.T) {
	rule := ast.RuleConfig{Activation: ast.RuleActivationManualMention}
	require.Equal(t, ast.RuleActivationManualMention, renderer.ResolvedActivation(rule))
}

func TestResolvedActivation_LegacyAlwaysApplyTrue(t *testing.T) {
	truthy := true
	rule := ast.RuleConfig{AlwaysApply: &truthy}
	require.Equal(t, ast.RuleActivationAlways, renderer.ResolvedActivation(rule))
}

func TestResolvedActivation_LegacyAlwaysApplyFalse(t *testing.T) {
	falsy := false
	rule := ast.RuleConfig{AlwaysApply: &falsy}
	require.Equal(t, ast.RuleActivationManualMention, renderer.ResolvedActivation(rule))
}

func TestResolvedActivation_PathsPresent_NoActivation(t *testing.T) {
	rule := ast.RuleConfig{Paths: ast.ClearableList{Values: []string{"src/**"}}}
	require.Equal(t, ast.RuleActivationPathGlob, renderer.ResolvedActivation(rule))
}

func TestResolvedActivation_NothingSet_DefaultsToAlways(t *testing.T) {
	rule := ast.RuleConfig{}
	require.Equal(t, ast.RuleActivationAlways, renderer.ResolvedActivation(rule))
}

func TestResolvedActivation_ExplicitActivationTakesPrecedenceOverLegacy(t *testing.T) {
	falsy := false
	rule := ast.RuleConfig{
		Activation:  ast.RuleActivationPathGlob,
		AlwaysApply: &falsy,
		Paths:       ast.ClearableList{Values: []string{"src/**"}},
	}
	require.Equal(t, ast.RuleActivationPathGlob, renderer.ResolvedActivation(rule))
}
