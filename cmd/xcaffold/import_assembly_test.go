package main

import (
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/stretchr/testify/assert"
)

func TestScoreSkillSpecificity(t *testing.T) {
	tests := []struct {
		name string
		cfg  ast.SkillConfig
		want int
	}{
		{
			name: "empty",
			cfg:  ast.SkillConfig{},
			want: 0,
		},
		{
			name: "tools only",
			cfg: ast.SkillConfig{
				AllowedTools: ast.ClearableList{Values: []string{"Read"}},
			},
			want: 1,
		},
		{
			name: "disable-model-invocation only",
			cfg: ast.SkillConfig{
				DisableModelInvocation: boolPtr(true),
			},
			want: 1,
		},
		{
			name: "multiple fields",
			cfg: ast.SkillConfig{
				AllowedTools: ast.ClearableList{Values: []string{"Read"}},
				WhenToUse:    "always",
				ArgumentHint: "hint",
			},
			want: 3,
		},
		{
			name: "disable-model-invocation and when-to-use",
			cfg: ast.SkillConfig{
				DisableModelInvocation: boolPtr(true),
				WhenToUse:              "when needed",
			},
			want: 2,
		},
		{
			name: "all fields populated",
			cfg: ast.SkillConfig{
				AllowedTools:           ast.ClearableList{Values: []string{"Read", "Write"}},
				DisableModelInvocation: boolPtr(true),
				WhenToUse:              "when needed",
				ArgumentHint:           "hint text",
			},
			want: 4,
		},
		{
			name: "empty lists don't count",
			cfg: ast.SkillConfig{
				AllowedTools: ast.ClearableList{Values: []string{}},
				WhenToUse:    "when",
			},
			want: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := scoreSkillSpecificity(tt.cfg)
			assert.Equal(t, tt.want, got)
		})
	}
}

func boolPtr(b bool) *bool {
	return &b
}

func TestSelectBodyAwareBase(t *testing.T) {
	tests := []struct {
		name    string
		scores  map[string]int
		hasBody map[string]bool
		want    string
	}{
		{
			name:    "prefers body provider over bodyless stub",
			scores:  map[string]int{"claude": 12, "gemini": 0},
			hasBody: map[string]bool{"claude": true, "gemini": false},
			want:    "claude",
		},
		{
			name:    "both have body falls back to lowest score",
			scores:  map[string]int{"claude": 12, "gemini": 0},
			hasBody: map[string]bool{"claude": true, "gemini": true},
			want:    "gemini",
		},
		{
			name:    "neither has body falls back to lowest score",
			scores:  map[string]int{"claude": 12, "gemini": 0},
			hasBody: map[string]bool{"claude": false, "gemini": false},
			want:    "gemini",
		},
		{
			name:    "body wins over alphabetical order",
			scores:  map[string]int{"alpha": 0, "beta": 0},
			hasBody: map[string]bool{"alpha": false, "beta": true},
			want:    "beta",
		},
		{
			name:    "multiple body providers uses scoring tiebreak",
			scores:  map[string]int{"claude": 12, "cursor": 1, "gemini": 0},
			hasBody: map[string]bool{"claude": true, "cursor": true, "gemini": false},
			want:    "cursor",
		},
		{
			name:    "single provider with body",
			scores:  map[string]int{"claude": 5},
			hasBody: map[string]bool{"claude": true},
			want:    "claude",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := selectBodyAwareBase(tt.scores, tt.hasBody)
			assert.Equal(t, tt.want, got)
		})
	}
}
