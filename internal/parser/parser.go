package parser

import (
	"fmt"
	"io"
	"strings"

	"github.com/saero-ai/xcaffold/internal/ast"
	"gopkg.in/yaml.v3"
)

// Parse reads a .xcf YAML configuration from the given reader and returns a
// validated XcaffoldConfig. It returns a descriptive error if parsing or
// validation fails; it never panics.
func Parse(r io.Reader) (*ast.XcaffoldConfig, error) {
	config := &ast.XcaffoldConfig{}

	decoder := yaml.NewDecoder(r)
	// Strict mode: reject unknown top-level fields so typos in .xcf files
	// surface immediately rather than silently producing incorrect output.
	decoder.KnownFields(true)

	if err := decoder.Decode(config); err != nil {
		return nil, fmt.Errorf("failed to parse .xcf YAML: %w", err)
	}

	if err := validate(config); err != nil {
		return nil, fmt.Errorf("invalid .xcf configuration: %w", err)
	}

	return config, nil
}

// validate performs semantic validation on a parsed config.
func validate(c *ast.XcaffoldConfig) error {
	if c.Version == "" {
		return fmt.Errorf("version is required (e.g. \"1.0\")")
	}

	name := strings.TrimSpace(c.Project.Name)
	if name == "" {
		return fmt.Errorf("project.name is required and must not be empty")
	}

	for id := range c.Agents {
		if strings.ContainsAny(id, "/\\") || strings.Contains(id, "..") {
			return fmt.Errorf("agent id contains invalid characters: %q", id)
		}
	}
	return nil
}
