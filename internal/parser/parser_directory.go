package parser

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/providers"
	"gopkg.in/yaml.v3"
)

// ParsedFile pairs a parsed partial config with its source file path.
type ParsedFile struct {
	Config   *ast.XcaffoldConfig
	FilePath string
}

// ParseDirectory recursively scans the given directory for all *.xcaf files,
// parses them, merges them strictly (erroring on duplicate IDs), and then
// resolves 'extends:' chains.
func ParseDirectory(dir string, opts ...ParseDirOption) (*ast.XcaffoldConfig, error) {
	dirOpts := resolveParseDirOptions(opts)

	vars, err := LoadVariableStack(dir, dirOpts.Target, dirOpts.VarFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load variables: %w", err)
	}

	merged, err := parseDirectoryUnvalidated(dir, dirOpts, vars, nil)
	if err != nil {
		return nil, err
	}

	// Re-load envs now that we have project config
	if merged.Project != nil && len(merged.Project.AllowedEnvVars) > 0 {
		envs := LoadEnv(merged.Project.AllowedEnvVars)
		// We need to re-parse everything if there are env vars?
		// That's expensive. Let's optimize: only re-parse if env vars were actually used.
		// For now, simplicity: re-run unvalidated parse with envs.
		merged, err = parseDirectoryUnvalidated(dir, dirOpts, vars, envs)
		if err != nil {
			return nil, err
		}
	}

	if err := validateMerged(merged); err != nil {
		return nil, fmt.Errorf("validation failed for project configuration: %w", err)
	}

	return merged, nil
}

// ParseDirectoryWithCrossRefWarnings parses a directory and returns the config plus
// any cross-reference validation issues separately. Structural errors still return
// as errors. Cross-reference issues are returned as a separate list for caller handling.
func ParseDirectoryWithCrossRefWarnings(dir string, opts ...ParseDirOption) (*ast.XcaffoldConfig, []CrossReferenceIssue, error) {
	dirOpts := resolveParseDirOptions(opts)

	vars, err := LoadVariableStack(dir, dirOpts.Target, dirOpts.VarFile)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load variables: %w", err)
	}

	merged, err := parseDirectoryUnvalidated(dir, dirOpts, vars, nil)
	if err != nil {
		return nil, nil, err
	}

	// Re-load envs now that we have project config
	if merged.Project != nil && len(merged.Project.AllowedEnvVars) > 0 {
		envs := LoadEnv(merged.Project.AllowedEnvVars)
		merged, err = parseDirectoryUnvalidated(dir, dirOpts, vars, envs)
		if err != nil {
			return nil, nil, err
		}
	}

	// Validate structural rules (base + permissions), but not cross-references
	if err := validateMergedStructural(merged); err != nil {
		return nil, nil, fmt.Errorf("validation failed for project configuration: %w", err)
	}

	// Collect cross-reference issues separately
	issues := validateCrossReferencesAsList(merged)

	return merged, issues, nil
}

// parseableKinds lists the kind values accepted by isParseableFile.
// Every .xcaf document must declare an explicit kind field.
var parseableKinds = map[string]bool{
	"project":   true,
	"agent":     true,
	"skill":     true,
	"rule":      true,
	"workflow":  true,
	"mcp":       true,
	"hooks":     true,
	"settings":  true,
	"global":    true,
	"policy":    true,
	"blueprint": true,
	"context":   true,
	"memory":    true,
}

// isParseableFile reads the kind: field from an .xcaf file to determine if it
// should be parsed by the compiler. Returns true for known resource-kind files.
// Returns false for files with unknown, empty, or removed kinds (such as "config").
func isParseableFile(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	// Extract only the frontmatter portion. Markdown body after the closing
	// --- may contain YAML-invalid syntax (e.g., tables with |) that would
	// cause yaml.Unmarshal on the full file to fail.
	fm, _, _ := extractFrontmatterAndBody(data)
	if len(fm) == 0 {
		fm = data
	}
	var header struct {
		Kind string `yaml:"kind"`
	}
	if err := yaml.Unmarshal(fm, &header); err != nil {
		return false
	}
	return parseableKinds[header.Kind]
}

// canonicalKindFilenames lists the resource kinds that can appear as prefixes in override filenames.
var canonicalKindFilenames = map[string]bool{
	"agent":    true,
	"skill":    true,
	"rule":     true,
	"workflow": true,
	"mcp":      true,
	"hooks":    true,
	"settings": true,
	"policy":   true,
	"template": true,
	"context":  true,
	"memory":   true,
}

func parseDirectoryUnvalidated(dir string, dirOpts parseDirConfig, vars map[string]interface{}, envs map[string]string) (*ast.XcaffoldConfig, error) {
	var files []string
	var overrideFiles []overrideFileEntry
	ignored := newParseFilter(dir)
	providerDir := filepath.Join("xcaf", "provider")
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if path != dir && (strings.HasPrefix(name, ".") || ignored[name]) {
				return filepath.SkipDir
			}
			if rel, relErr := filepath.Rel(dir, path); relErr == nil && rel == providerDir {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(d.Name(), ".xcaf") {
			if kind, provider, ok := classifyOverrideFile(d.Name()); ok {
				if !providers.IsRegistered(provider) {
					return fmt.Errorf("override file %s: unknown provider %q; valid providers: %s", d.Name(), provider, strings.Join(providers.RegisteredNames(), ", "))
				}
				overrideFiles = append(overrideFiles, overrideFileEntry{
					Path:     path,
					Kind:     kind,
					Provider: provider,
				})
			} else if isParseableFile(path) {
				files = append(files, path)
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to scan directory %q: %w", dir, err)
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no *.xcaf files found in directory %q", dir)
	}

	var parsedFiles []ParsedFile
	for _, f := range files {
		cfg, err := ParseFileExact(f, withVars(vars), withEnvs(envs))
		if err != nil {
			return nil, err
		}
		parsedFiles = append(parsedFiles, ParsedFile{Config: cfg, FilePath: f})
	}

	var globalConfig *ast.XcaffoldConfig
	if dirOpts.skipGlobal {
		globalConfig = &ast.XcaffoldConfig{}
	} else {
		var loadErr error
		globalConfig, loadErr = loadGlobalBase()
		if loadErr != nil {
			return nil, fmt.Errorf("failed to load implicit global configuration: %w", loadErr)
		}
	}

	merged, err := mergeAllStrict(parsedFiles)
	if err != nil {
		return nil, fmt.Errorf("failed to merge config files in %q: %w", dir, err)
	}

	if merged.Extends != "" {
		merged, err = resolveExtends(dir, merged, vars, envs)
		if err != nil {
			return nil, err
		}
	}

	// Implicitly overlay the project configuration on top of the global base
	merged = mergeConfigOverride(globalConfig, merged)

	// Parse override files
	for _, of := range overrideFiles {
		if err := parseOverrideFile(of, merged, vars, envs); err != nil {
			return nil, err
		}
	}

	// Validate that every override has a corresponding base
	if err := validateOverrideBasesExist(merged); err != nil {
		return nil, err
	}

	if err := loadExtras(dir, merged); err != nil {
		return nil, fmt.Errorf("failed to load extras: %w", err)
	}

	return merged, nil
}

func parseDirectoryRaw(dir string, vars map[string]interface{}, envs map[string]string, opts ...parseOptionFunc) (*ast.XcaffoldConfig, error) {
	var files []string
	var overrideFiles []overrideFileEntry
	ignored := newParseFilter(dir)
	providerDir := filepath.Join("xcaf", "provider")
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if path != dir && (strings.HasPrefix(name, ".") || ignored[name]) {
				return filepath.SkipDir
			}
			if rel, relErr := filepath.Rel(dir, path); relErr == nil && rel == providerDir {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(d.Name(), ".xcaf") {
			if kind, provider, ok := classifyOverrideFile(d.Name()); ok {
				if !providers.IsRegistered(provider) {
					return fmt.Errorf("override file %s: unknown provider %q; valid providers: %s", d.Name(), provider, strings.Join(providers.RegisteredNames(), ", "))
				}
				overrideFiles = append(overrideFiles, overrideFileEntry{
					Path:     path,
					Kind:     kind,
					Provider: provider,
				})
			} else if isParseableFile(path) {
				files = append(files, path)
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to scan directory %q: %w", dir, err)
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no *.xcaf files found in directory %q", dir)
	}

	var parsedFiles []ParsedFile
	for _, f := range files {
		cfg, err := ParseFileExact(f, withVars(vars), withEnvs(envs))
		if err != nil {
			return nil, err
		}
		parsedFiles = append(parsedFiles, ParsedFile{Config: cfg, FilePath: f})
	}

	merged, err := mergeAllStrict(parsedFiles)
	if err != nil {
		return nil, fmt.Errorf("failed to merge config files in %q: %w", dir, err)
	}

	// Parse override files
	for _, of := range overrideFiles {
		if err := parseOverrideFile(of, merged, vars, envs); err != nil {
			return nil, err
		}
	}

	// Validate that every override has a corresponding base
	if err := validateOverrideBasesExist(merged); err != nil {
		return nil, err
	}

	if err := loadExtras(dir, merged); err != nil {
		return nil, fmt.Errorf("failed to load extras: %w", err)
	}

	return merged, nil
}
