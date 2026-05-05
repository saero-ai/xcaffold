package parser

import (
	"testing"

	"github.com/saero-ai/xcaffold/providers"
	"github.com/stretchr/testify/assert"
)

func TestNewParseFilter_IncludesGenericDirs(t *testing.T) {
	filter := newParseFilter(".")

	// Standard ignore directories should always be present
	expected := []string{".git", ".worktrees", "node_modules", "vendor", ".venv", "dist", "build", "coverage"}
	for _, dir := range expected {
		assert.True(t, filter[dir], "expected %q to be in parse filter", dir)
	}
}

func TestNewParseFilter_IncludesProviderDirs(t *testing.T) {
	// Get the registered provider input directories
	providerDirs := providers.RegisteredInputDirs()
	assert.NotEmpty(t, providerDirs, "expected at least one registered provider")

	filter := newParseFilter(".")

	// Each provider's input directory should be in the filter
	for _, providerDir := range providerDirs {
		assert.True(t, filter[providerDir], "expected provider dir %q to be in parse filter", providerDir)
	}
}

func TestNewParseFilter_HasKnownProviders(t *testing.T) {
	filter := newParseFilter(".")

	// We expect at least the core providers to be excluded
	knownDirs := []string{".claude", ".cursor", ".agents", ".gemini"}
	for _, dir := range knownDirs {
		assert.True(t, filter[dir], "expected known provider dir %q in parse filter", dir)
	}
}
