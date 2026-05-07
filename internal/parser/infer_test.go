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
		// Standard convention: <kind>/<name>/<kind>.xcaf
		{"xcaf/agents/xaff/agent.xcaf", "agent", "xaff"},
		{"xcaf/skills/xcaffold/skill.xcaf", "skill", "xcaffold"},
		{"xcaf/rules/xcaf-conventions/rule.xcaf", "rule", "xcaf-conventions"},
		{"xcaf/workflows/deploy/workflow.xcaf", "workflow", "deploy"},
		{"xcaf/mcp/server/mcp.xcaf", "mcp", "server"},

		// Namespaced rules: <kind>/<namespace>/<name>/<kind>.xcaf
		{"xcaf/rules/cli/go-code-quality/rule.xcaf", "rule", "cli/go-code-quality"},
		{"xcaf/rules/platform/api-standards/rule.xcaf", "rule", "platform/api-standards"},
		{"xcaf/rules/core/safety-check/rule.xcaf", "rule", "core/safety-check"},

		// Provider overrides: <kind>/<name>/<kind>.<provider>.xcaf
		{"xcaf/agents/xaff/agent.claude.xcaf", "agent", "xaff"},
		{"xcaf/agents/xaff/agent.cursor.xcaf", "agent", "xaff"},
		{"xcaf/agents/xaff/agent.gemini.xcaf", "agent", "xaff"},
		{"xcaf/rules/cli/go-code-quality/rule.gemini.xcaf", "rule", "cli/go-code-quality"},
		{"xcaf/skills/tdd/skill.claude.xcaf", "skill", "tdd"},

		// Legacy format (backward compat): <kind>/<name>/<name>.xcaf
		{"xcaf/skills/xcaffold/xcaffold.xcaf", "skill", "xcaffold"},
		{"xcaf/rules/xcaf-conventions/xcaf-conventions.xcaf", "rule", "xcaf-conventions"},
		{"xcaf/agents/developer/developer.xcaf", "agent", "developer"},

		// Absolute path prefix (common in error messages and tests)
		{"project-root/xcaf/agents/dev/agent.xcaf", "agent", "dev"},
		{"home/user/xcaf/rules/cli/style/rule.xcaf", "rule", "cli/style"},

		// Invalid cases: no match
		{"some/other/path/file.xcaf", "", ""},
		{"xcaf/unknown-dir/test/test.xcaf", "", ""},
		{"xcaf/rules/rule.xcaf", "", ""},
		{"no-xcaf-dir/agents/test/agent.xcaf", "", ""},
		{"notxcaf/agents/test/agent.xcaf", "", ""},
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
