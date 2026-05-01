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
	restoreBodyFields(config, &cp)
	return &cp
}

// restoreBodyFields copies Body fields that are lost during YAML
// round-trip because they are tagged yaml:"-".
func restoreBodyFields(src, dst *ast.XcaffoldConfig) {
	for k, s := range src.Agents {
		if d, ok := dst.Agents[k]; ok {
			d.Body = s.Body
			dst.Agents[k] = d
		}
	}
	for k, s := range src.Skills {
		if d, ok := dst.Skills[k]; ok {
			d.Body = s.Body
			dst.Skills[k] = d
		}
	}
	for k, s := range src.Rules {
		if d, ok := dst.Rules[k]; ok {
			d.Body = s.Body
			dst.Rules[k] = d
		}
	}
	for k, s := range src.Workflows {
		if d, ok := dst.Workflows[k]; ok {
			d.Body = s.Body
			dst.Workflows[k] = d
		}
	}
	for k, s := range src.Contexts {
		if d, ok := dst.Contexts[k]; ok {
			d.Body = s.Body
			dst.Contexts[k] = d
		}
	}
	if src.Project != nil && dst.Project != nil {
		dst.Project.Body = src.Project.Body
	}
}
