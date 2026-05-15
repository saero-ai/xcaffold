package importer

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/saero-ai/xcaffold/internal/ast"
	"gopkg.in/yaml.v3"
)

// ParseFrontmatter splits YAML frontmatter from a markdown file's body.
// If the data does not start with "---\n", the entire content is returned as
// the body with zero-value metadata. Malformed YAML in the frontmatter block
// returns an error — callers that need graceful fallback should use
// ParseFrontmatterLenient.
func ParseFrontmatter(data []byte, v interface{}) (body string, err error) {
	content := string(data)
	if !strings.HasPrefix(content, "---\n") {
		return strings.TrimSpace(content), nil
	}
	parts := strings.SplitN("\n"+content[4:], "\n---", 2)
	if len(parts) < 2 {
		return strings.TrimSpace(content), nil
	}
	if err := yaml.Unmarshal([]byte(parts[0]), v); err != nil {
		return "", fmt.Errorf("frontmatter: %w", err)
	}
	return strings.TrimSpace(strings.TrimPrefix(parts[1], "\n")), nil
}

// ParseFrontmatterLenient is like ParseFrontmatter but returns the body with
// zero-value metadata when the YAML is malformed, instead of an error. Use
// for user-edited files where frontmatter may contain unquoted special chars.
func ParseFrontmatterLenient(data []byte, v interface{}) (body string, err error) {
	content := string(data)
	if !strings.HasPrefix(content, "---\n") {
		return strings.TrimSpace(content), nil
	}
	parts := strings.SplitN("\n"+content[4:], "\n---", 2)
	if len(parts) < 2 {
		return strings.TrimSpace(content), nil
	}
	if err := yaml.Unmarshal([]byte(parts[0]), v); err != nil {
		return strings.TrimSpace(strings.TrimPrefix(parts[1], "\n")), nil
	}
	return strings.TrimSpace(strings.TrimPrefix(parts[1], "\n")), nil
}

// MatchGlob matches a relative path against a glob pattern.
// Supports "*" (any single segment) and "**" (any number of segments).
func MatchGlob(pattern, rel string) bool {
	patParts := strings.Split(pattern, "/")
	relParts := strings.Split(rel, "/")
	return matchSegments(patParts, relParts)
}

func matchSegments(pat, rel []string) bool {
	for len(pat) > 0 && len(rel) > 0 {
		switch pat[0] {
		case "**":
			for i := 0; i <= len(rel); i++ {
				if matchSegments(pat[1:], rel[i:]) {
					return true
				}
			}
			return false
		default:
			ok, err := filepath.Match(pat[0], rel[0])
			if err != nil || !ok {
				return false
			}
			pat, rel = pat[1:], rel[1:]
		}
	}
	return len(pat) == 0 && len(rel) == 0
}

// ReadFile reads a file from disk. Thin wrapper over os.ReadFile provided
// so that provider importers use a single import path for file I/O.
func ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// AppendUnique appends s to slice only if it is not already present.
func AppendUnique(slice []string, s string) []string {
	for _, v := range slice {
		if v == s {
			return slice
		}
	}
	return append(slice, s)
}

// FileVisitor is called for each regular file found during a provider directory walk.
// rel is the slash-separated path relative to the walk root. data is the file contents.
type FileVisitor func(rel string, data []byte) error

// walkOptions holds internal parameters for the recursive walk.
type walkOptions struct {
	root    string
	current string
	prefix  string
	visited map[string]bool
}

// WalkProviderDir walks dir recursively, calling visitor for each regular file.
// Symlinks to directories are followed with cycle detection. Directories are
// skipped. rel paths use forward slashes.
func WalkProviderDir(dir string, visitor FileVisitor) error {
	return walkProviderDir(walkOptions{
		root:    dir,
		current: dir,
		prefix:  "",
		visited: make(map[string]bool),
	}, visitor)
}

// walkProviderDir walks current, computing relative paths from root.
// prefix is prepended when walking a symlinked directory that resolves
// outside root — it maps files back to the symlink's position in the
// original tree.
func walkProviderDir(opts walkOptions, visitor FileVisitor) error {
	return filepath.WalkDir(opts.current, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if d.Type()&fs.ModeSymlink != 0 {
			target, err := filepath.EvalSymlinks(path)
			if err != nil {
				return nil
			}
			info, err := os.Stat(target)
			if err != nil {
				return nil
			}
			if info.IsDir() {
				if opts.visited[target] {
					return nil
				}
				opts.visited[target] = true
				symlinkRel, relErr := filepath.Rel(opts.root, path)
				if relErr != nil {
					symlinkRel = filepath.Base(path)
				}
				if opts.prefix != "" {
					symlinkRel = filepath.Join(opts.prefix, symlinkRel)
				}
				return walkProviderDir(walkOptions{
					root:    opts.root,
					current: target,
					prefix:  symlinkRel,
					visited: opts.visited,
				}, visitor)
			}
			path = target
		}
		var rel string
		if opts.prefix != "" {
			fileRel, relErr := filepath.Rel(opts.current, path)
			if relErr != nil {
				return fmt.Errorf("rel path: %w", relErr)
			}
			rel = filepath.Join(opts.prefix, fileRel)
		} else {
			var relErr error
			rel, relErr = filepath.Rel(opts.root, path)
			if relErr != nil {
				return fmt.Errorf("rel path: %w", relErr)
			}
		}
		rel = filepath.ToSlash(rel)
		data, err := os.ReadFile(path)
		if err != nil {
			return nil // non-fatal: skip unreadable files
		}
		return visitor(rel, data)
	})
}

// ExtractHookScript stores a hook shell script for passthrough to xcaf/hooks/.
// Scripts are stored in config.ProviderExtras["xcaf"]["hooks/<basename>"]
// and emitted to xcaf/hooks/<basename> by WriteSplitFiles.
func ExtractHookScript(rel string, data []byte, config *ast.XcaffoldConfig) error {
	dest := "hooks/" + filepath.Base(rel)
	if config.ProviderExtras == nil {
		config.ProviderExtras = make(map[string]map[string][]byte)
	}
	if config.ProviderExtras["xcaf"] == nil {
		config.ProviderExtras["xcaf"] = make(map[string][]byte)
	}
	config.ProviderExtras["xcaf"][dest] = data
	return nil
}

// DefaultExtractRule parses a markdown file with YAML frontmatter as a rule.
// Derives the rule ID from the path by stripping "rules/" prefix and ".md" extension.
// Supports nested rule paths like "rules/cli/testing.md" → ID "cli/testing".
func DefaultExtractRule(rel string, data []byte, provider string, config *ast.XcaffoldConfig) error {
	var front struct {
		Name          string                        `yaml:"name"`
		Description   string                        `yaml:"description"`
		AlwaysApply   *bool                         `yaml:"always-apply"`
		Activation    string                        `yaml:"activation"`
		Paths         []string                      `yaml:"paths"`
		ExcludeAgents []string                      `yaml:"exclude-agents"`
		Targets       map[string]ast.TargetOverride `yaml:"targets"`
	}

	body, err := ParseFrontmatterLenient(data, &front)
	if err != nil {
		return fmt.Errorf("rule %q: %w", rel, err)
	}

	rulesPrefix := "rules/"
	relFromRules := strings.TrimPrefix(filepath.ToSlash(rel), rulesPrefix)
	id := strings.TrimSuffix(relFromRules, ".md")

	name := front.Name
	if name == "" {
		name = id
	}

	if config.Rules == nil {
		config.Rules = make(map[string]ast.RuleConfig)
	}
	config.Rules[id] = ast.RuleConfig{
		Name:           name,
		Description:    front.Description,
		AlwaysApply:    front.AlwaysApply,
		Activation:     front.Activation,
		Paths:          ast.ClearableList{Values: front.Paths},
		ExcludeAgents:  ast.ClearableList{Values: front.ExcludeAgents},
		Targets:        front.Targets,
		Body:           body,
		SourceProvider: provider,
	}
	return nil
}

// DefaultExtractSkillAsset records skill companion subdirectories (references/, scripts/, assets/, examples/)
// by adding the subdirectory name to the artifacts list.
// The file path must follow the pattern: skills/<skillId>/<subDir>/<rest>.
func DefaultExtractSkillAsset(rel string, _ []byte, config *ast.XcaffoldConfig) error {
	parts := strings.SplitN(filepath.ToSlash(rel), "/", 4)
	if len(parts) < 4 {
		return fmt.Errorf("skill asset path too short: %q", rel)
	}
	skillID := parts[1]
	subDir := parts[2]

	if config.Skills == nil {
		config.Skills = make(map[string]ast.SkillConfig)
	}
	skill := config.Skills[skillID]
	switch subDir {
	case "references", "scripts", "assets", "examples":
		if !slices.Contains(skill.Artifacts, subDir) {
			skill.Artifacts = append(skill.Artifacts, subDir)
		}
	default:
		return fmt.Errorf("skill asset: unknown subdirectory %q in %q", subDir, rel)
	}
	config.Skills[skillID] = skill
	return nil
}

// DefaultExtractHookScript delegates to ExtractHookScript.
func DefaultExtractHookScript(rel string, data []byte, config *ast.XcaffoldConfig) error {
	return ExtractHookScript(rel, data, config)
}
