package cursor

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolveAlias_TierAliases(t *testing.T) {
	r := NewModelResolver()
	tests := []struct {
		alias  string
		wantID string
		wantOK bool
	}{
		{"balanced", "claude-sonnet-4-5", true},
		{"flagship", "gemini-2.5-pro", true},
		{"fast", "cursor-fast", true},
	}
	for _, tt := range tests {
		t.Run(tt.alias, func(t *testing.T) {
			id, ok := r.ResolveAlias(tt.alias)
			assert.Equal(t, tt.wantOK, ok, "ResolveAlias(%q) ok", tt.alias)
			assert.Equal(t, tt.wantID, id, "ResolveAlias(%q) model ID", tt.alias)
		})
	}
}

func TestResolveAlias_Passthrough(t *testing.T) {
	r := NewModelResolver()
	tests := []struct {
		input  string
		wantID string
	}{
		{"claude-sonnet-4-5", "claude-sonnet-4-5"},
		{"gpt-4o", "gpt-4o"},
		{"gemini-2.5-pro", "gemini-2.5-pro"},
		{"cursor-fast", "cursor-fast"},
		{"composer-agent", "composer-agent"},
		{"o1-preview", "o1-preview"},
		{"o3-mini", "o3-mini"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			id, ok := r.ResolveAlias(tt.input)
			assert.True(t, ok, "ResolveAlias(%q) should succeed", tt.input)
			assert.Equal(t, tt.wantID, id, "ResolveAlias(%q) model ID", tt.input)
		})
	}
}

func TestResolveAlias_CaseInsensitive(t *testing.T) {
	r := NewModelResolver()
	tests := []struct {
		input  string
		wantID string
	}{
		{"Claude-Sonnet-4-5", "claude-sonnet-4-5"},
		{"GPT-4o", "gpt-4o"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			id, ok := r.ResolveAlias(tt.input)
			assert.True(t, ok, "ResolveAlias(%q) should succeed (case-insensitive)", tt.input)
			assert.Equal(t, tt.wantID, id, "ResolveAlias(%q) should return lowered form", tt.input)
		})
	}
}

func TestResolveAlias_Rejected(t *testing.T) {
	r := NewModelResolver()
	tests := []struct {
		input string
	}{
		{"unknown-model"},
		{"mistral-large"},
		{""},
	}
	for _, tt := range tests {
		name := tt.input
		if name == "" {
			name = "empty"
		}
		t.Run(name, func(t *testing.T) {
			id, ok := r.ResolveAlias(tt.input)
			assert.False(t, ok, "ResolveAlias(%q) should fail", tt.input)
			assert.Empty(t, id, "ResolveAlias(%q) should return empty ID", tt.input)
		})
	}
}
