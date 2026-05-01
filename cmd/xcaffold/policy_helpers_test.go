package main

import (
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/stretchr/testify/assert"
)

func TestDeepCopyConfig_PreservesBody(t *testing.T) {
	config := &ast.XcaffoldConfig{}
	config.Agents = map[string]ast.AgentConfig{
		"dev": {Name: "dev", Body: "You are a developer."},
	}
	config.Skills = map[string]ast.SkillConfig{
		"tdd": {Name: "tdd", Body: "Follow TDD."},
	}
	config.Rules = map[string]ast.RuleConfig{
		"sec": {Name: "sec", Body: "No secrets."},
	}
	config.Workflows = map[string]ast.WorkflowConfig{
		"deploy": {Name: "deploy", Body: "Deploy steps."},
	}
	config.Contexts = map[string]ast.ContextConfig{
		"main": {Name: "main", Body: "Project context."},
	}

	cp := deepCopyConfig(config)

	assert.Equal(t, "You are a developer.", cp.Agents["dev"].Body)
	assert.Equal(t, "Follow TDD.", cp.Skills["tdd"].Body)
	assert.Equal(t, "No secrets.", cp.Rules["sec"].Body)
	assert.Equal(t, "Deploy steps.", cp.Workflows["deploy"].Body)
	assert.Equal(t, "Project context.", cp.Contexts["main"].Body)
}

func TestDeepCopyConfig_PreservesProjectBody(t *testing.T) {
	config := &ast.XcaffoldConfig{}
	config.Project = &ast.ProjectConfig{Name: "myproj", Body: "Project instructions."}

	cp := deepCopyConfig(config)

	assert.NotNil(t, cp.Project)
	assert.Equal(t, "Project instructions.", cp.Project.Body)
}

func TestDeepCopyConfig_EmptyBody_StaysEmpty(t *testing.T) {
	config := &ast.XcaffoldConfig{}
	config.Skills = map[string]ast.SkillConfig{
		"empty": {Name: "empty", Body: ""},
	}

	cp := deepCopyConfig(config)

	assert.Equal(t, "", cp.Skills["empty"].Body)
}
