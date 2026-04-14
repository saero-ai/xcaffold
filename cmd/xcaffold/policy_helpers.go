package main

import (
	"github.com/saero-ai/xcaffold/internal/ast"
	"gopkg.in/yaml.v3"
)

func deepCopyConfig(config *ast.XcaffoldConfig) *ast.XcaffoldConfig {
	data, err := yaml.Marshal(config)
	if err != nil {
		return config
	}
	var cp ast.XcaffoldConfig
	if err := yaml.Unmarshal(data, &cp); err != nil {
		return config
	}
	return &cp
}
