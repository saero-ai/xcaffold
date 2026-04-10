package compiler

import (
	"encoding/json"
	"fmt"

	"github.com/saero-ai/xcaffold/internal/ast"
)

// ExportPlugin repackages a compiled Output into the standard plugin format.
// Settings files are excluded (they are environment-specific), and the hooks
// file is relocated under a hooks/ subdirectory.
func ExportPlugin(config *ast.XcaffoldConfig, compiled *Output) (*Output, error) {
	if config.Project == nil {
		return nil, fmt.Errorf("ExportPlugin requires a project configuration")
	}

	out := &Output{
		Files: make(map[string]string),
	}

	manifest := map[string]string{
		"name":        config.Project.Name,
		"description": config.Project.Description,
	}
	if config.Project.Version != "" {
		manifest["version"] = config.Project.Version
	}
	if config.Project.Author != "" {
		manifest["author"] = config.Project.Author
	}
	if config.Project.Homepage != "" {
		manifest["homepage"] = config.Project.Homepage
	}
	if config.Project.Repository != "" {
		manifest["repository"] = config.Project.Repository
	}
	if config.Project.License != "" {
		manifest["license"] = config.Project.License
	}

	manifestJSON, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal plugin manifest: %w", err)
	}
	out.Files[".claude-plugin/plugin.json"] = string(manifestJSON)

	for path, content := range compiled.Files {
		switch {
		case path == "hooks.json", path == "hooks/hooks.json":
			continue // Skip any lingering standalone hooks files
		case path == "settings.json", path == "settings.local.json":
			// settings files are environment-specific; exclude from plugin exports
			continue
		default:
			out.Files[path] = content
		}
	}

	if len(config.Hooks) > 0 {
		wrapper := map[string]any{
			"hooks": config.Hooks,
		}
		b, err := json.MarshalIndent(wrapper, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("failed to format plugin hooks: %w", err)
		}
		out.Files["hooks/hooks.json"] = string(b)
	}

	return out, nil
}
