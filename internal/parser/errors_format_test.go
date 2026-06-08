package parser

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnwrapParseFileError(t *testing.T) {
	inner := errors.New("field 'tools' is not valid for kind 'rule'")
	wrapped := fmt.Errorf("error in %q: %w", "/proj/xcaf/rules/bad/rule.xcaf", inner)
	assert.Equal(t, "field 'tools' is not valid for kind 'rule'", UnwrapParseFileError(wrapped))
}

func TestFormatParseFileError(t *testing.T) {
	inner := fmt.Errorf("error in %q: %s", "/tmp/proj/agents.xcaf", "unknown field")
	err := FormatParseFileError("/tmp/proj", "/tmp/proj/agents.xcaf", inner)
	require.Error(t, err)
	assert.Equal(t, "agents.xcaf: unknown field", err.Error())
}

func TestFormatValidationError_RewritesEmbeddedPath(t *testing.T) {
	msg := `failed to merge: error in "/tmp/proj/xcaf/rules/a/rule.xcaf": duplicate rule ID "a"`
	got := FormatValidationError("/tmp/proj", errors.New(msg))
	assert.Contains(t, got, "xcaf/rules/a/rule.xcaf:")
	assert.NotContains(t, got, "error in \"")
}

func TestFormatDiagnosticLine(t *testing.T) {
	line := FormatDiagnosticLine(Diagnostic{
		FilePath: "xcaf/agents/dev/agent.xcaf",
		Message:  "unknown plugin",
	})
	assert.Equal(t, "xcaf/agents/dev/agent.xcaf: unknown plugin", line)
}
