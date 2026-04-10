package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGraphAll_MutualExclusion_WithGlobal(t *testing.T) {
	graphAll = true
	globalFlag = true
	defer func() {
		graphAll = false
		globalFlag = false
	}()

	err := runGraph(nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mutually exclusive")
}

func TestGraphAll_MutualExclusion_WithProject(t *testing.T) {
	graphAll = true
	graphProject = "some-project"
	defer func() {
		graphAll = false
		graphProject = ""
	}()

	err := runGraph(nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mutually exclusive")
}
