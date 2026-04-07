package state_test

import (
	"testing"

	"github.com/saero-ai/xcaffold/internal/state"
	"github.com/stretchr/testify/assert"
)

func TestLockFilePath_DefaultTarget(t *testing.T) {
	assert.Equal(t, "scaffold.lock", state.LockFilePath("scaffold.lock", ""))
}

func TestLockFilePath_ClaudeTarget(t *testing.T) {
	assert.Equal(t, "scaffold.lock", state.LockFilePath("scaffold.lock", "claude"))
}

func TestLockFilePath_CursorTarget(t *testing.T) {
	assert.Equal(t, "scaffold.cursor.lock", state.LockFilePath("scaffold.lock", "cursor"))
}

func TestLockFilePath_AntigravityTarget(t *testing.T) {
	assert.Equal(t, "scaffold.antigravity.lock", state.LockFilePath("scaffold.lock", "antigravity"))
}
