package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestExportCmd_OutputFlagRegistered verifies that --output is a registered
// string flag on the export command.
func TestExportCmd_OutputFlagRegistered(t *testing.T) {
	f := exportCmd.Flags().Lookup("output")
	require.NotNil(t, f, "--output flag must be registered on exportCmd")
	assert.Equal(t, "string", f.Value.Type(), "--output flag must be a string type")
}

// TestExportCmd_OutputFlagIsRequired verifies that the export command flags
// carry the cobra required annotation on --output.
func TestExportCmd_OutputFlagIsRequired(t *testing.T) {
	f := exportCmd.Flags().Lookup("output")
	require.NotNil(t, f, "--output flag must be registered on exportCmd")
	// cobra.MarkFlagRequired stores "cobra_annotation_bash_completion_one_required_flag"
	const cobraRequiredAnnotation = "cobra_annotation_bash_completion_one_required_flag"
	_, required := f.Annotations[cobraRequiredAnnotation]
	assert.True(t, required, "--output flag must be marked as required")
}
