package policy

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatViolations_SingleError(t *testing.T) {
	violations := []Violation{
		{
			PolicyName:   "require-model",
			Severity:     SeverityError,
			ResourceName: "backend-dev",
			Message:      `field "model" value "gpt-4o" is not in approved list [sonnet opus]`,
		},
	}
	out := FormatViolations(violations)
	assert.Contains(t, out, "POLICY VIOLATION [error] require-model")
	assert.Contains(t, out, "resource: backend-dev")
	assert.Contains(t, out, `field "model" value "gpt-4o"`)
}

func TestFormatViolations_OutputTarget(t *testing.T) {
	violations := []Violation{
		{
			PolicyName: "path-safety",
			Severity:   SeverityError,
			FilePath:   ".claude/agents/../../../etc/passwd",
			Message:    `output path contains forbidden string ".."`,
		},
	}
	out := FormatViolations(violations)
	assert.Contains(t, out, "file: .claude/agents/../../../etc/passwd")
	assert.NotContains(t, out, "resource:")
}

func TestFormatViolations_Empty(t *testing.T) {
	out := FormatViolations(nil)
	assert.Empty(t, out)
}
