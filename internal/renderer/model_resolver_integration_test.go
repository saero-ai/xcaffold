package renderer_test

import (
	"testing"

	"github.com/saero-ai/xcaffold/internal/renderer"
	_ "github.com/saero-ai/xcaffold/providers/all"
)

func TestAllProvidersHaveModelResolversRegistered(t *testing.T) {
	providers := []string{"claude", "cursor", "gemini", "copilot", "antigravity"}

	for _, p := range providers {
		resolver := renderer.LookupModelResolver(p)
		if resolver == nil {
			t.Errorf("provider %q has no ModelResolver registered", p)
		}
	}
}
