package ast_test

import (
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestProviderExtras_NotSerializedToYAML(t *testing.T) {
	cfg := ast.XcaffoldConfig{
		Version: "1.0",
		ProviderExtras: map[string]map[string][]byte{
			"claude": {"agents/foo.md": []byte("hello")},
		},
	}
	out, err := yaml.Marshal(cfg)
	require.NoError(t, err)
	assert.NotContains(t, string(out), "ProviderExtras")
	assert.NotContains(t, string(out), "provider-extras")
	assert.NotContains(t, string(out), "claude")
}

func TestSourceProvider_NotSerializedToYAML(t *testing.T) {
	agent := ast.AgentConfig{Name: "foo", SourceProvider: "claude"}
	out, err := yaml.Marshal(agent)
	require.NoError(t, err)
	assert.NotContains(t, string(out), "source-provider")
	assert.NotContains(t, string(out), "SourceProvider")
}
