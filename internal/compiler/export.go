package compiler

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/output"
	"github.com/saero-ai/xcaffold/providers"
)

// pluginDirForTarget returns the plugin output directory for the given target.
// Returns an error if the target is unknown or does not support plugin export.
func pluginDirForTarget(target string) (string, error) {
	if target == "" {
		return "", fmt.Errorf("--target is required for export")
	}
	m, ok := providers.ManifestFor(target)
	if !ok {
		return "", fmt.Errorf("unknown target %q; supported: %s", target, strings.Join(providers.PrimaryNames(), ", "))
	}
	if m.PluginDir == "" {
		return "", fmt.Errorf("target %q does not support plugin export", target)
	}
	return m.PluginDir, nil
}

// ExportPlugin repackages a compiled Output into the standard plugin format.
// target selects the output platform.
// Settings files are excluded (they are environment-specific), and the hooks
// file is relocated under a hooks/ subdirectory.
func ExportPlugin(config *ast.XcaffoldConfig, compiled *output.Output, target string) (*output.Output, error) {
	if config.Project == nil {
		return nil, fmt.Errorf("ExportPlugin requires a project configuration")
	}

	pluginDir, err := pluginDirForTarget(target)
	if err != nil {
		return nil, err
	}

	out := &output.Output{Files: make(map[string]string)}

	manifestJSON, err := buildPluginManifest(config)
	if err != nil {
		return nil, err
	}
	out.Files[pluginDir+"/plugin.json"] = manifestJSON

	copyCompiledFiles(compiled, out)

	if err := appendPluginHooks(config, out); err != nil {
		return nil, err
	}

	return out, nil
}

// buildPluginManifest serialises the project metadata fields into the plugin.json
// manifest format. Only non-empty optional fields are included.
func buildPluginManifest(config *ast.XcaffoldConfig) (string, error) {
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
	b, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal plugin manifest: %w", err)
	}
	return string(b), nil
}

// copyCompiledFiles copies content files from compiled into out, skipping hooks
// and settings files that must not appear in a plugin distribution.
func copyCompiledFiles(compiled *output.Output, out *output.Output) {
	for path, content := range compiled.Files {
		switch {
		case path == "hooks.json", path == "hooks/hooks.json":
			continue // skip any lingering standalone hooks files
		case path == "settings.json", path == "settings.local.json":
			continue // settings are environment-specific; exclude from plugin exports
		default:
			out.Files[path] = content
		}
	}
}

// appendPluginHooks writes the config's hooks into the plugin's hooks/hooks.json
// file. Does nothing when there are no hooks.
func appendPluginHooks(config *ast.XcaffoldConfig, out *output.Output) error {
	if len(config.Hooks) == 0 {
		return nil
	}
	wrapper := map[string]any{"hooks": config.Hooks}
	b, err := json.MarshalIndent(wrapper, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to format plugin hooks: %w", err)
	}
	out.Files["hooks/hooks.json"] = string(b)
	return nil
}
