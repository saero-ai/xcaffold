package parser

import (
	"testing"
)

func TestInferKindAndName(t *testing.T) {
	tests := []struct {
		path     string
		wantKind string
		wantName string
	}{
		// Standard convention: <kind>/<name>/<kind>.xcf
		{"xcf/agents/xaff/agent.xcf", "agent", "xaff"},
		{"xcf/skills/xcaffold/skill.xcf", "skill", "xcaffold"},
		{"xcf/rules/xcf-conventions/rule.xcf", "rule", "xcf-conventions"},
		{"xcf/workflows/deploy/workflow.xcf", "workflow", "deploy"},
		{"xcf/mcp/server/mcp.xcf", "mcp", "server"},

		// Namespaced rules: <kind>/<namespace>/<name>/<kind>.xcf
		{"xcf/rules/cli/go-code-quality/rule.xcf", "rule", "cli/go-code-quality"},
		{"xcf/rules/platform/api-standards/rule.xcf", "rule", "platform/api-standards"},
		{"xcf/rules/core/safety-check/rule.xcf", "rule", "core/safety-check"},

		// Provider overrides: <kind>/<name>/<kind>.<provider>.xcf
		{"xcf/agents/xaff/agent.claude.xcf", "agent", "xaff"},
		{"xcf/agents/xaff/agent.cursor.xcf", "agent", "xaff"},
		{"xcf/agents/xaff/agent.gemini.xcf", "agent", "xaff"},
		{"xcf/rules/cli/go-code-quality/rule.gemini.xcf", "rule", "cli/go-code-quality"},
		{"xcf/skills/tdd/skill.claude.xcf", "skill", "tdd"},

		// Legacy format (backward compat): <kind>/<name>/<name>.xcf
		{"xcf/skills/xcaffold/xcaffold.xcf", "skill", "xcaffold"},
		{"xcf/rules/xcf-conventions/xcf-conventions.xcf", "rule", "xcf-conventions"},
		{"xcf/agents/developer/developer.xcf", "agent", "developer"},

		// Absolute path prefix (common in error messages and tests)
		{"project-root/xcf/agents/dev/agent.xcf", "agent", "dev"},
		{"home/user/xcf/rules/cli/style/rule.xcf", "rule", "cli/style"},

		// Invalid cases: no match
		{"some/other/path/file.xcf", "", ""},
		{"xcf/unknown-dir/test/test.xcf", "", ""},
		{"xcf/rules/rule.xcf", "", ""},
		{"no-xcf-dir/agents/test/agent.xcf", "", ""},
		{"notxcf/agents/test/agent.xcf", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			gotKind, gotName := inferKindAndName(tt.path)
			if gotKind != tt.wantKind || gotName != tt.wantName {
				t.Errorf("inferKindAndName(%q) = (%q, %q), want (%q, %q)",
					tt.path, gotKind, gotName, tt.wantKind, tt.wantName)
			}
		})
	}
}
