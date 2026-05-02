package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/saero-ai/xcaffold/internal/parser"
	"github.com/saero-ai/xcaffold/internal/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHelpXcf_Agent_ShowsAllFields(t *testing.T) {
	ks, ok := schema.LookupKind("agent")
	require.True(t, ok)

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	defer rootCmd.SetOut(os.Stdout)

	displayKindSchema(rootCmd, ks)
	output := buf.String()

	for _, f := range ks.Fields {
		assert.Contains(t, output, f.YAMLKey, "missing field: %s", f.YAMLKey)
	}
	assert.Contains(t, output, "kind: agent")
	assert.Contains(t, output, "frontmatter+body")
}

func TestHelpXcf_Skill_ShowsAllFields(t *testing.T) {
	ks, ok := schema.LookupKind("skill")
	require.True(t, ok)

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	defer rootCmd.SetOut(os.Stdout)

	displayKindSchema(rootCmd, ks)
	output := buf.String()

	for _, f := range ks.Fields {
		assert.Contains(t, output, f.YAMLKey, "missing field: %s", f.YAMLKey)
	}
	assert.Contains(t, output, "kind: skill")
}

func TestHelpXcf_UnknownKind_ReturnsError(t *testing.T) {
	err := runHelpXcf(rootCmd, "nonexistent", "", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown kind: nonexistent")
	assert.Contains(t, err.Error(), "Available:")
}

func TestHelpXcf_NoKind_ShowsNormalHelp(t *testing.T) {
	err := runHelpXcf(rootCmd, "", "", false)
	assert.Error(t, err, "empty kind should return error")
	assert.Contains(t, err.Error(), "unknown kind")
}

func TestHelpXcf_Out_WritesTemplate(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "agent.xcf")

	ks, _ := schema.LookupKind("agent")
	err := generateTemplate(ks, "agent", dest)
	require.NoError(t, err)

	content, err := os.ReadFile(dest)
	require.NoError(t, err)

	s := string(content)
	assert.True(t, strings.HasPrefix(s, "---\n"))
	assert.Contains(t, s, "kind: agent")
	assert.Contains(t, s, "version: \"1.0\"")
	assert.Contains(t, s, "name:")
	assert.Contains(t, s, "# Instructions go here.")
}

func TestHelpXcf_Out_ExistingDir_AppendsKind(t *testing.T) {
	dir := t.TempDir()

	ks, _ := schema.LookupKind("skill")
	err := generateTemplate(ks, "skill", dir)
	require.NoError(t, err)

	expected := filepath.Join(dir, "skill.xcf")
	assert.FileExists(t, expected)
}

func TestHelpXcf_Out_MissingDir_Errors(t *testing.T) {
	ks, _ := schema.LookupKind("agent")
	err := generateTemplate(ks, "agent", "/tmp/xcf-nonexistent-dir-12345/test.xcf")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "directory does not exist")
}

func TestHelpXcf_Out_InvalidExtension_Errors(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "agent.yaml")

	ks, _ := schema.LookupKind("agent")
	err := generateTemplate(ks, "agent", dest)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must end in .xcf")
}

func TestHelpXcf_FieldsMatchParser(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "agent.xcf")

	ks, _ := schema.LookupKind("agent")
	err := generateTemplate(ks, "agent", dest)
	require.NoError(t, err)

	f, err := os.Open(dest)
	require.NoError(t, err)
	defer f.Close()

	_, parseErr := parser.Parse(f)
	assert.NoError(t, parseErr, "parser rejected generated template")
}

func TestHelpXcf_GoldenOutput(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	noColorFlag = true
	defer func() { noColorFlag = false }()

	ks, ok := schema.LookupKind("agent")
	require.True(t, ok)

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	defer rootCmd.SetOut(os.Stdout)

	displayKindSchema(rootCmd, ks)
	actual := buf.String()

	golden, err := os.ReadFile("testdata/help_xcf_agent.golden")
	require.NoError(t, err)

	if actual != string(golden) {
		t.Errorf("output differs from golden file.\nTo update: NO_COLOR=1 ./xcaffold help --xcf agent > cmd/xcaffold/testdata/help_xcf_agent.golden")
		lines := strings.Split(actual, "\n")
		goldenLines := strings.Split(string(golden), "\n")
		for i := 0; i < len(lines) && i < len(goldenLines); i++ {
			if lines[i] != goldenLines[i] {
				t.Logf("first diff at line %d:\n  got:  %q\n  want: %q", i+1, lines[i], goldenLines[i])
				break
			}
		}
	}
}
