package main

import (
	"encoding/json"

	"github.com/saero-ai/xcaffold/internal/ast"
)

func deepCopyConfig(config *ast.XcaffoldConfig) *ast.XcaffoldConfig {
	// Use JSON round-trip instead of YAML to preserve Body fields
	// (Body fields are tagged yaml:"-" so YAML round-trip drops them).
	// Body fields do NOT have json:"-", so they survive JSON marshaling.
	data, err := json.Marshal(config)
	if err != nil {
		return config
	}
	var cp ast.XcaffoldConfig
	if err := json.Unmarshal(data, &cp); err != nil {
		return config
	}
	return &cp
}
