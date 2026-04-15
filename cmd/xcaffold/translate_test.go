package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTranslateCmd_RequiresFrom(t *testing.T) {
	translateFrom = ""
	translateTo = "claude"
	err := runTranslate(translateCmd, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "from")
}

func TestTranslateCmd_RequiresTo(t *testing.T) {
	translateFrom = "antigravity"
	translateTo = ""
	err := runTranslate(translateCmd, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "to")
}

func TestTranslateCmd_InvalidFromProvider(t *testing.T) {
	translateFrom = "invalid-provider"
	translateTo = "claude"
	err := runTranslate(translateCmd, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid-provider")
}

func TestTranslateCmd_InvalidToProvider(t *testing.T) {
	translateFrom = "claude"
	translateTo = "unknown"
	err := runTranslate(translateCmd, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown")
}

func TestTranslateCmd_FidelityModeValidation(t *testing.T) {
	translateFrom = "claude"
	translateTo = "cursor"
	translateFidelity = "invalid"
	err := runTranslate(translateCmd, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "fidelity")
}

func TestTranslateCmd_DiffFormatValidation(t *testing.T) {
	translateFrom = "claude"
	translateTo = "cursor"
	translateFidelity = "warn"
	translateDiff = true
	translateDiffFormat = "xml"
	err := runTranslate(translateCmd, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "diff-format")
}
