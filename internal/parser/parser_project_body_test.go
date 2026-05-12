package parser

import (
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

func TestParse_ProjectWithBody_Rejected(t *testing.T) {
	xcaf := `---
kind: project
version: "1.0"
name: test-project
targets: [claude]
---
This body should be rejected.
`
	_, err := Parse(strings.NewReader(xcaf))
	require.Error(t, err)
	require.Contains(t, err.Error(), "kind: project does not support a markdown body")
}
