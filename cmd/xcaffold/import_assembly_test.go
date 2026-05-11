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
