package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/saero-ai/xcaffold/internal/parser"
	"github.com/saero-ai/xcaffold/pkg/schema"
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

func TestHelpXcf_EmptyKind_ReturnsError(t *testing.T) {
	err := runHelpXcf(rootCmd, "", "", false)
	assert.Error(t, err, "empty kind should return error")
	assert.Contains(t, err.Error(), "unknown kind")
}

func TestHelpXcf_Out_WritesTemplate(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "agent.xcf")

	ks, _ := schema.LookupKind("agent")
	err := generateTemplate(rootCmd, ks, "agent", dest)
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
	err := generateTemplate(rootCmd, ks, "skill", dir)
	require.NoError(t, err)

	expected := filepath.Join(dir, "skill.xcf")
	assert.FileExists(t, expected)
}

func TestHelpXcf_Out_MissingDir_Errors(t *testing.T) {
	ks, _ := schema.LookupKind("agent")
	err := generateTemplate(rootCmd, ks, "agent", "/tmp/xcf-nonexistent-dir-12345/test.xcf")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "directory does not exist")
}

func TestHelpXcf_Out_InvalidExtension_Errors(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "agent.yaml")

	ks, _ := schema.LookupKind("agent")
	err := generateTemplate(rootCmd, ks, "agent", dest)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must end in .xcf")
}

func TestHelpXcf_FieldsMatchParser(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "agent.xcf")

	ks, _ := schema.LookupKind("agent")
	err := generateTemplate(rootCmd, ks, "agent", dest)
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

func TestHelpXCF_ShowsProviderSupport(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	noColorFlag = true
	defer func() { noColorFlag = false }()

	ks, ok := schema.LookupKind("agent")
	require.True(t, ok)

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	defer rootCmd.SetOut(os.Stdout)

	displayKindSchema(rootCmd, ks)
	output := buf.String()

	// Description field should show provider annotations if it has provider data
	hasProviderData := false
	for _, f := range ks.Fields {
		if f.YAMLKey == "description" && len(f.Provider) > 0 {
			hasProviderData = true
			break
		}
	}

	if hasProviderData {
		assert.Contains(t, output, "Providers:", "output should show Providers annotation for fields with provider data")
		// Check that provider names appear with proper capitalization
		assert.Regexp(t, `\b[A-Z][a-z]+\(`, output, "provider names should be capitalized with optional (required) suffix")
	}

	// Envelope fields (like name, kind) should NOT show Providers even if they're required
	lines := strings.Split(output, "\n")
	for i, line := range lines {
		if strings.Contains(line, "name") && strings.Contains(line, "required") {
			// Check that the next line(s) don't have "Providers:" for the envelope field
			nextLineHasProviders := false
			if i+1 < len(lines) && strings.HasPrefix(lines[i+1], "                                                        Providers:") {
				nextLineHasProviders = true
			}
			if nextLineHasProviders {
				t.Errorf("envelope field 'name' should not show Providers annotation")
			}
			break
		}
	}
}
