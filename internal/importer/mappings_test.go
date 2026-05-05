package importer

import (
	"testing"
)

func TestCommonMappings_HookScripts(t *testing.T) {
	tests := []struct {
		path        string
		shouldMatch bool
	}{
		{"hooks/pre-commit.sh", true},
		{"hooks/post-tool-use.sh", true},
		{"hooks/custom/nested.sh", true},
		{"hooks/dir/test.sh", true},
		{"agents/test.md", false},
	}

	for _, tt := range tests {
		hookPatterns := []string{"hooks/*.sh", "hooks/**"}
		matched := false
		for _, pattern := range hookPatterns {
			if MatchGlob(pattern, tt.path) {
				matched = true
				break
			}
		}
		if matched != tt.shouldMatch {
			t.Errorf("hook patterns: %q should match=%v, got matched=%v", tt.path, tt.shouldMatch, matched)
		}
	}
}

func TestCommonMappings_Agents(t *testing.T) {
	tests := []struct {
		path        string
		shouldMatch bool
	}{
		{"agents/developer.md", true},
		{"agents/reviewer.md", true},
		{"agents/nested/developer.md", false},
		{"skills/test/SKILL.md", false},
	}

	pattern := "agents/*.md"
	for _, tt := range tests {
		matched := MatchGlob(pattern, tt.path)
		if matched != tt.shouldMatch {
			t.Errorf("agent pattern: %q should match=%v, got matched=%v", tt.path, tt.shouldMatch, matched)
		}
	}
}

func TestCommonMappings_Skills(t *testing.T) {
	tests := []struct {
		path        string
		shouldMatch bool
	}{
		{"skills/tdd/SKILL.md", true},
		{"skills/debugging/SKILL.md", true},
		{"skills/tdd/references/guide.md", true},
		{"skills/tdd/references/nested/doc.md", true},
		{"skills/tdd/scripts/helper.sh", true},
		{"skills/tdd/assets/data.json", true},
		{"skills/tdd/something-else.md", false},
		{"agents/test.md", false},
	}

	patterns := []string{"skills/*/SKILL.md", "skills/*/references/**", "skills/*/scripts/**", "skills/*/assets/**"}
	for _, tt := range tests {
		matched := false
		for _, pattern := range patterns {
			if MatchGlob(pattern, tt.path) {
				matched = true
				break
			}
		}
		if matched != tt.shouldMatch {
			t.Errorf("skill patterns: %q should match=%v, got matched=%v", tt.path, tt.shouldMatch, matched)
		}
	}
}

func TestCommonMappings_Rules(t *testing.T) {
	tests := []struct {
		path        string
		shouldMatch bool
	}{
		{"rules/test.md", true},
		{"rules/cli/test.md", true},
		{"rules/cli/testing/guide.md", true},
		{"rules/nested/deep/rule.md", true},
		{"agents/test.md", false},
		{"rules/test.txt", false},
	}

	pattern := "rules/**/*.md"
	for _, tt := range tests {
		matched := MatchGlob(pattern, tt.path)
		if matched != tt.shouldMatch {
			t.Errorf("rule pattern: %q should match=%v, got matched=%v", tt.path, tt.shouldMatch, matched)
		}
	}
}

func TestCommonMappings_MappingsAreCorrect(t *testing.T) {
	if len(CommonMappings) != 8 {
		t.Errorf("CommonMappings has %d entries, want 8", len(CommonMappings))
	}

	// Verify specific mappings
	tests := []struct {
		pattern string
		kind    Kind
		layout  Layout
	}{
		{"hooks/*.sh", KindHookScript, FlatFile},
		{"agents/*.md", KindAgent, FlatFile},
		{"skills/*/SKILL.md", KindSkill, DirectoryPerEntry},
		{"rules/**/*.md", KindRule, FlatFile},
	}

	for _, tt := range tests {
		found := false
		for _, m := range CommonMappings {
			if m.Pattern == tt.pattern {
				found = true
				if m.Kind != tt.kind {
					t.Errorf("Pattern %q: Kind = %v, want %v", tt.pattern, m.Kind, tt.kind)
				}
				if m.Layout != tt.layout {
					t.Errorf("Pattern %q: Layout = %v, want %v", tt.pattern, m.Layout, tt.layout)
				}
				break
			}
		}
		if !found {
			t.Errorf("Pattern %q not found in CommonMappings", tt.pattern)
		}
	}
}
