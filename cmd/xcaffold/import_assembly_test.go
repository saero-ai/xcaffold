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
			name: "references only",
			cfg: ast.SkillConfig{
				References: ast.ClearableList{Values: []string{"ref.md"}},
			},
			want: 1,
		},
		{
			name: "multiple fields",
			cfg: ast.SkillConfig{
				AllowedTools: ast.ClearableList{Values: []string{"Read"}},
				References:   ast.ClearableList{Values: []string{"ref.md"}},
				WhenToUse:    "always",
				ArgumentHint: "hint",
			},
			want: 4,
		},
		{
			name: "scripts and assets",
			cfg: ast.SkillConfig{
				Scripts: ast.ClearableList{Values: []string{"run.sh"}},
				Assets:  ast.ClearableList{Values: []string{"icon.svg"}},
			},
			want: 2,
		},
		{
			name: "all fields populated",
			cfg: ast.SkillConfig{
				AllowedTools:           ast.ClearableList{Values: []string{"Read", "Write"}},
				References:             ast.ClearableList{Values: []string{"ref.md"}},
				Scripts:                ast.ClearableList{Values: []string{"run.sh"}},
				Assets:                 ast.ClearableList{Values: []string{"icon.svg"}},
				Examples:               ast.ClearableList{Values: []string{"example.md"}},
				DisableModelInvocation: boolPtr(true),
				WhenToUse:              "when needed",
				ArgumentHint:           "hint text",
			},
			want: 8,
		},
		{
			name: "empty lists don't count",
			cfg: ast.SkillConfig{
				AllowedTools: ast.ClearableList{Values: []string{}},
				References:   ast.ClearableList{Values: []string{}},
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
