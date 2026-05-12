package resolver

import (
	"os"
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpandVariables(t *testing.T) {
	// Setup environment variable for testing
	os.Setenv("USER", "bob")
	defer os.Unsetenv("USER")

	tests := []struct {
		name      string
		input     []byte
		vars      map[string]interface{}
		envs      map[string]string
		expected  []byte
		wantErr   bool
		errString string
	}{
		{
			name:     "String replacement",
			input:    []byte("description: ${var.org}"),
			vars:     map[string]interface{}{"org": "acme"},
			envs:     nil,
			expected: []byte("description: acme"),
			wantErr:  false,
		},
		{
			name:     "Int replacement",
			input:    []byte("turns: ${var.turns}"),
			vars:     map[string]interface{}{"turns": 10},
			envs:     nil,
			expected: []byte("turns: 10"),
			wantErr:  false,
		},
		{
			name:     "Bool replacement",
			input:    []byte("debug: ${var.debug}"),
			vars:     map[string]interface{}{"debug": true},
			envs:     nil,
			expected: []byte("debug: true"),
			wantErr:  false,
		},
		{
			name:     "List replacement (interface slice)",
			input:    []byte("items: ${var.items}"),
			vars:     map[string]interface{}{"items": []interface{}{1, "two", true}},
			envs:     nil,
			expected: []byte("items: [1, two, true]"),
			wantErr:  false,
		},
		{
			name:     "Env replacement",
			input:    []byte("user: ${env.USER}"),
			vars:     nil,
			envs:     map[string]string{"USER": "bob"},
			expected: []byte("user: bob"),
			wantErr:  false,
		},
		{
			name:      "Error: Undefined variable",
			input:     []byte("config: ${var.missing}"),
			vars:      map[string]interface{}{"org": "acme"},
			envs:      nil,
			expected:  nil,
			wantErr:   true,
			errString: "unresolved variable: ${var.missing}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExpandVariables(tt.input, tt.vars, tt.envs)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errString)
				assert.Nil(t, got)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, got)
			}
		})
	}
}

func TestResolveAttributes_StringField(t *testing.T) {
	config := &ast.XcaffoldConfig{}
	config.Skills = map[string]ast.SkillConfig{
		"tdd": {Description: "Test-driven development"},
	}
	config.Agents = map[string]ast.AgentConfig{
		"developer": {Description: "${skill.tdd.description}"},
	}
	err := ResolveAttributes(config)
	require.NoError(t, err)
	assert.Equal(t, "Test-driven development", config.Agents["developer"].Description)
}

func TestResolveAttributes_StringSliceField(t *testing.T) {
	config := &ast.XcaffoldConfig{}
	config.Skills = map[string]ast.SkillConfig{
		"tdd": {AllowedTools: ast.ClearableList{Values: []string{"Bash", "Read", "Write"}}},
	}
	config.Agents = map[string]ast.AgentConfig{
		"developer": {Tools: ast.ClearableList{Values: []string{"${skill.tdd.allowed-tools}"}}},
	}
	err := ResolveAttributes(config)
	require.NoError(t, err)
	assert.Equal(t, []string{"Bash", "Read", "Write"}, config.Agents["developer"].Tools.Values)
}

func TestResolveAttributes_StringInterpolation(t *testing.T) {
	config := &ast.XcaffoldConfig{}
	config.Skills = map[string]ast.SkillConfig{
		"tdd": {Description: "TDD"},
	}
	config.Agents = map[string]ast.AgentConfig{
		"developer": {Description: "Developer using ${skill.tdd.description} workflow"},
	}
	err := ResolveAttributes(config)
	require.NoError(t, err)
	assert.Equal(t, "Developer using TDD workflow", config.Agents["developer"].Description)
}

func TestResolveAttributes_MissingResource_Error(t *testing.T) {
	config := &ast.XcaffoldConfig{}
	config.Agents = map[string]ast.AgentConfig{
		"developer": {Tools: ast.ClearableList{Values: []string{"${skill.nonexistent.tools}"}}},
	}
	err := ResolveAttributes(config)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nonexistent")
}

func TestResolveAttributes_MissingField_Error(t *testing.T) {
	config := &ast.XcaffoldConfig{}
	config.Skills = map[string]ast.SkillConfig{
		"tdd": {Description: "TDD"},
	}
	config.Agents = map[string]ast.AgentConfig{
		"developer": {Description: "${skill.tdd.nonexistent}"},
	}
	err := ResolveAttributes(config)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nonexistent")
}

func TestResolveAttributes_CircularReference_Error(t *testing.T) {
	config := &ast.XcaffoldConfig{}
	config.Agents = map[string]ast.AgentConfig{
		"dev": {Description: "${agent.ops.description}"},
		"ops": {Description: "${agent.dev.description}"},
	}
	err := ResolveAttributes(config)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "circular")
}

func TestResolveAttributes_NoReferences_Passthrough(t *testing.T) {
	config := &ast.XcaffoldConfig{}
	config.Agents = map[string]ast.AgentConfig{
		"developer": {Description: "Plain text, no refs", Model: "sonnet"},
	}
	err := ResolveAttributes(config)
	require.NoError(t, err)
	assert.Equal(t, "Plain text, no refs", config.Agents["developer"].Description)
	assert.Equal(t, "sonnet", config.Agents["developer"].Model)
}

func TestResolveAttributes_PolicyField(t *testing.T) {
	config := &ast.XcaffoldConfig{}
	config.Policies = map[string]ast.PolicyConfig{
		"require-desc": {Name: "require-desc", Severity: "error"},
	}
	config.Agents = map[string]ast.AgentConfig{
		"dev": {Description: "Policy severity: ${policy.require-desc.severity}"},
	}
	err := ResolveAttributes(config)
	require.NoError(t, err)
	assert.Equal(t, "Policy severity: error", config.Agents["dev"].Description)
}

func TestResolveAttributes_MemoryField(t *testing.T) {
	config := &ast.XcaffoldConfig{}
	config.Memory = map[string]ast.MemoryConfig{
		"project-context": {Name: "project-context", Description: "Project memory"},
	}
	config.Agents = map[string]ast.AgentConfig{
		"dev": {Description: "Memory: ${memory.project-context.description}"},
	}
	err := ResolveAttributes(config)
	require.NoError(t, err)
	assert.Equal(t, "Memory: Project memory", config.Agents["dev"].Description)
}

func TestResolveAttributes_ContextField(t *testing.T) {
	config := &ast.XcaffoldConfig{}
	config.Contexts = map[string]ast.ContextConfig{
		"coding-rules": {Name: "coding-rules", Description: "Coding standards"},
	}
	config.Agents = map[string]ast.AgentConfig{
		"dev": {Description: "Context: ${context.coding-rules.description}"},
	}
	err := ResolveAttributes(config)
	require.NoError(t, err)
	assert.Equal(t, "Context: Coding standards", config.Agents["dev"].Description)
}

func TestResolveAttributes_TemplateField(t *testing.T) {
	config := &ast.XcaffoldConfig{}
	config.Templates = map[string]ast.TemplateConfig{
		"scaffold": {Name: "scaffold", DefaultTarget: "claude"},
	}
	config.Agents = map[string]ast.AgentConfig{
		"dev": {Description: "Template target: ${template.scaffold.default-target}"},
	}
	err := ResolveAttributes(config)
	require.NoError(t, err)
	assert.Equal(t, "Template target: claude", config.Agents["dev"].Description)
}

func TestResolveAttributes_HooksField(t *testing.T) {
	config := &ast.XcaffoldConfig{}
	config.Hooks = map[string]ast.NamedHookConfig{
		"pre-commit": {Name: "pre-commit", Description: "Git hooks"},
	}
	config.Agents = map[string]ast.AgentConfig{
		"dev": {Description: "Hook: ${hooks.pre-commit.description}"},
	}
	err := ResolveAttributes(config)
	require.NoError(t, err)
	assert.Equal(t, "Hook: Git hooks", config.Agents["dev"].Description)
}

func TestResolveAttributes_SettingsField(t *testing.T) {
	config := &ast.XcaffoldConfig{}
	config.Settings = map[string]ast.SettingsConfig{
		"default": {Name: "default", Model: "opus"},
	}
	config.Agents = map[string]ast.AgentConfig{
		"dev": {Description: "Settings model: ${settings.default.model}"},
	}
	err := ResolveAttributes(config)
	require.NoError(t, err)
	assert.Equal(t, "Settings model: opus", config.Agents["dev"].Description)
}

func TestResolveAttributes_PolicyContainsRef(t *testing.T) {
	config := &ast.XcaffoldConfig{}
	config.Agents = map[string]ast.AgentConfig{
		"dev": {Model: "opus"},
	}
	config.Policies = map[string]ast.PolicyConfig{
		"check": {Name: "check", Description: "Agent uses ${agent.dev.model}"},
	}
	err := ResolveAttributes(config)
	require.NoError(t, err)
	assert.Equal(t, "Agent uses opus", config.Policies["check"].Description)
}
