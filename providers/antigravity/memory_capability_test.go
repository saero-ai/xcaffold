package antigravity_test

import (
	"testing"

	"github.com/saero-ai/xcaffold/providers/antigravity"
	"github.com/stretchr/testify/require"
)

func TestAntigravity_CapabilitySet_MemoryDeferred(t *testing.T) {
	r := antigravity.New()
	caps := r.Capabilities()
	require.False(t, caps.Memory, "Antigravity memory rendering is deferred — capability must be false until native format is implemented")
}
