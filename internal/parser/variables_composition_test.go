package parser

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestVariableComposition(t *testing.T) {
	files := map[string]string{
		"project.vars": "project_name = my-app\nbase_dir = /opt/${var.project_name}\ndescription = App installed at ${var.base_dir}\n",
	}
	tmpDir, cleanup := SetupTestEnv(t, files)
	defer cleanup()

	vars, err := LoadVariableStack(tmpDir, "", "")
	require.NoError(t, err)

	assert.Equal(t, "my-app", vars["project_name"])
	assert.Equal(t, "/opt/my-app", vars["base_dir"])
	assert.Equal(t, "App installed at /opt/my-app", vars["description"])
}
