package parser

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeTestXCF(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	require.NoError(t, os.MkdirAll(filepath.Dir(p), 0755))
	require.NoError(t, os.WriteFile(p, []byte(content), 0600))
	return p
}

func TestParseDirectory_DuplicateAgentID_ReportsBothFiles(t *testing.T) {
	dir := t.TempDir()

	writeTestXCF(t, dir, "project.xcf", `
version: "1.0"
project:
  name: "test-project"
`)
	writeTestXCF(t, dir, "agents.xcf", `
agents:
  developer:
    description: "First developer"
    instructions: "Do stuff"
`)
	writeTestXCF(t, dir, "tools.xcf", `
agents:
  developer:
    description: "Duplicate developer"
    instructions: "Do other stuff"
`)

	_, err := ParseDirectory(dir)
	require.Error(t, err, "duplicate agent ID across files must error")
	assert.Contains(t, err.Error(), "developer")
	assert.Contains(t, err.Error(), "agents.xcf")
	assert.Contains(t, err.Error(), "tools.xcf")
}
