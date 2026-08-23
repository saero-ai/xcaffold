package antigravity_test

import (
	"testing"

	"github.com/saero-ai/xcaffold/providers/antigravity"
	"github.com/stretchr/testify/require"
)

func TestAntigravity_CapabilitySet_MemoryUnsupported(t *testing.T) {
	r := antigravity.New()
	caps := r.Capabilities()
	require.False(t, caps.Memory, "Antigravity does not support persistent memory per ground truth (db/memory.json) — capability is false")
}
