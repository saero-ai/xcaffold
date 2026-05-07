package parser

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func SetupTestEnv(t *testing.T, files map[string]string) (string, func()) {
	tmpDir, err := os.MkdirTemp("", "var_test_")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	xcafDir := filepath.Join(tmpDir, "xcaf")
	if err := os.MkdirAll(xcafDir, 0755); err != nil {
		t.Fatalf("failed to create xcaf dir: %v", err)
	}

	for name, content := range files {
		var filePath string
		if filepath.IsAbs(name) {
			filePath = name
		} else {
			filePath = filepath.Join(xcafDir, name)
		}
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			os.RemoveAll(tmpDir)
			t.Fatalf("failed to write file %s: %v", filePath, err)
		}
	}

	return tmpDir, func() {
		os.RemoveAll(tmpDir)
	}
}

func TestLoadVariableStack_Layering(t *testing.T) {
	tests := []struct {
		name        string
		target      string
		customFile  string
		files       map[string]string
		expected    map[string]interface{}
		expectedErr bool
	}{
		{
			name:   "only project.vars",
			target: "default",
			files: map[string]string{
				"project.vars": "key1 = value1\nkey2 = 123\nkey3 = true\n",
			},
			expected: map[string]interface{}{
				"key1": "value1",
				"key2": 123,
				"key3": true,
			},
		},
		{
			name:   "project.<target>.vars overrides project.vars",
			target: "prod",
			files: map[string]string{
				"project.vars":      "key1 = value1\nkey2 = 123\nkey3 = true\n",
				"project.prod.vars": "key1 = overridden_value1\nkey4 = new_value4\n",
			},
			expected: map[string]interface{}{
				"key1": "overridden_value1",
				"key2": 123,
				"key3": true,
				"key4": "new_value4",
			},
		},
		{
			name:   "project.vars.local overrides all",
			target: "prod",
			files: map[string]string{
				"project.vars":       "key1 = value1\nkey2 = 123\n",
				"project.prod.vars":  "key1 = overridden_value1\nkey3 = true\n",
				"project.vars.local": "key1 = local_value1\nkey4 = 456\n",
			},
			expected: map[string]interface{}{
				"key1": "local_value1",
				"key2": 123,
				"key3": true,
				"key4": 456,
			},
		},
		{
			name:   "typed values",
			target: "default",
			files: map[string]string{
				"project.vars": "str = hello world\nint = 12345\nbool-true = true\nlist = [item1, item2]\n",
			},
			expected: map[string]interface{}{
				"str":       "hello world",
				"int":       12345,
				"bool-true": true,
				"list":      []interface{}{"item1", "item2"},
			},
		},
		{
			name:   "malformed key rejected",
			target: "default",
			files: map[string]string{
				"project.vars": "1variable = value\n",
			},
			expectedErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, cleanup := SetupTestEnv(t, tt.files)
			defer cleanup()

			vars, err := LoadVariableStack(tmpDir, tt.target, tt.customFile)

			if tt.expectedErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.expected, vars)
		})
	}
}

func TestLoadEnv(t *testing.T) {
	os.Setenv("TEST_VAR_1", "env_value_1")
	os.Setenv("TEST_VAR_2", "env_value_2")
	defer os.Unsetenv("TEST_VAR_1")
	defer os.Unsetenv("TEST_VAR_2")

	allowed := []string{"TEST_VAR_1", "TEST_VAR_2", "NON_EXISTENT"}
	expected := map[string]string{
		"TEST_VAR_1": "env_value_1",
		"TEST_VAR_2": "env_value_2",
	}

	result := LoadEnv(allowed)
	assert.Equal(t, expected, result)
}
