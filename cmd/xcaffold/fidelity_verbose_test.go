package main

import (
	"bytes"
	"testing"

	"github.com/saero-ai/xcaffold/internal/renderer"
	"github.com/stretchr/testify/assert"
)

// TestPrintFidelityNotes_VerboseFalse_FilterNonErrorNotes verifies that when
// verbose is false, only error-level notes are printed.
func TestPrintFidelityNotes_VerboseFalse_FilterNonErrorNotes(t *testing.T) {
	notes := []renderer.FidelityNote{
		{
			Level:  renderer.LevelError,
			Target: "agent:dev",
			Reason: "required field missing",
		},
		{
			Level:  renderer.LevelWarning,
			Target: "agent:dev",
			Reason: "field not recommended",
		},
		{
			Level:  renderer.LevelInfo,
			Target: "agent:dev",
			Reason: "field is optional",
		},
	}

	var buf bytes.Buffer
	printed := printFidelityNotes(&buf, notes, false)

	// When verbose=false, only errors should be printed
	assert.Equal(t, 1, printed, "only error-level notes should be printed")

	output := buf.String()
	assert.Contains(t, output, "ERROR (agent:dev): required field missing")
	assert.NotContains(t, output, "WARNING (agent:dev): field not recommended")
	assert.NotContains(t, output, "INFO (agent:dev): field is optional")
}

// TestPrintFidelityNotes_VerboseTrue_PrintAllNotes verifies that when verbose
// is true, all notes (error, warning, info) are printed.
func TestPrintFidelityNotes_VerboseTrue_PrintAllNotes(t *testing.T) {
	notes := []renderer.FidelityNote{
		{
			Level:  renderer.LevelError,
			Target: "agent:dev",
			Reason: "required field missing",
		},
		{
			Level:  renderer.LevelWarning,
			Target: "agent:dev",
			Reason: "field not recommended",
		},
		{
			Level:  renderer.LevelInfo,
			Target: "agent:dev",
			Reason: "field is optional",
		},
	}

	var buf bytes.Buffer
	printed := printFidelityNotes(&buf, notes, true)

	// When verbose=true, all notes should be printed
	assert.Equal(t, 3, printed, "all notes should be printed")

	output := buf.String()
	assert.Contains(t, output, "ERROR (agent:dev): required field missing")
	assert.Contains(t, output, "WARNING (agent:dev): field not recommended")
	assert.Contains(t, output, "INFO (agent:dev): field is optional")
}

// TestPrintFidelityNotes_VerboseFalse_ErrorsOnly verifies that when verbose
// is false and only warnings exist, nothing is printed.
func TestPrintFidelityNotes_VerboseFalse_NoOutput_WarningsOnly(t *testing.T) {
	notes := []renderer.FidelityNote{
		{
			Level:  renderer.LevelWarning,
			Target: "agent:dev",
			Reason: "field not recommended",
		},
		{
			Level:  renderer.LevelInfo,
			Target: "agent:dev",
			Reason: "field is optional",
		},
	}

	var buf bytes.Buffer
	printed := printFidelityNotes(&buf, notes, false)

	assert.Equal(t, 0, printed, "no notes should be printed when verbose=false and only warnings/info exist")
	assert.Empty(t, buf.String(), "output should be empty")
}

// TestPrintFidelityNotes_VerboseTrue_WarningsAndInfo verifies that when
// verbose is true, warnings and info are included in the count.
func TestPrintFidelityNotes_VerboseTrue_IncludeWarningsAndInfo(t *testing.T) {
	notes := []renderer.FidelityNote{
		{
			Level:  renderer.LevelWarning,
			Target: "agent:dev",
			Reason: "field not recommended",
		},
		{
			Level:  renderer.LevelInfo,
			Target: "agent:dev",
			Reason: "field is optional",
		},
	}

	var buf bytes.Buffer
	printed := printFidelityNotes(&buf, notes, true)

	assert.Equal(t, 2, printed, "both warnings and info should be counted")

	output := buf.String()
	assert.Contains(t, output, "WARNING (agent:dev): field not recommended")
	assert.Contains(t, output, "INFO (agent:dev): field is optional")
}
