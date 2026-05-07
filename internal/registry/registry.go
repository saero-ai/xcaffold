package registry

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// GlobalScanIterator is set by providers/registry.go during init to break
// the import cycle. It iterates registered providers and calls each one's
// global scanner.
var GlobalScanIterator func(userHome string, r *GlobalScanResult)

// Config represents the global user preferences.
type Config struct {
	DefaultTarget string `yaml:"default_target,omitempty"`
}

// Project represents a single registered xcaffold project.
type Project struct {
	Registered  time.Time `yaml:"registered"`
	LastApplied time.Time `yaml:"last_applied,omitempty"`
	Path        string    `yaml:"path"`
	Name        string    `yaml:"name"`
	ConfigDir   string    `yaml:"config_directory,omitempty"`
	Targets     []string  `yaml:"targets,omitempty"`
}

// GlobalHome returns the absolute path to the xcaffold home directory.
// If XCAFFOLD_HOME is set, it is used directly (primarily for test isolation).
// Otherwise it falls back to ~/.xcaffold/.
func GlobalHome() (string, error) {
	if override := os.Getenv("XCAFFOLD_HOME"); override != "" {
		return override, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not determine home directory: %w", err)
	}
	return filepath.Join(home, ".xcaffold"), nil
}

// DefaultGlobalXCAFContent is a minimal starter template used only when no
// real global agent configs are found on disk.
const DefaultGlobalXCAFContent = `kind: global
version: "1.0"
# No global agents were detected. Add agents here and run:
#   xcaffold apply --global
agents: {}
`

// RebuildGlobalXCAF re-scans all registered platform providers and rewrites
// ~/.xcaffold/global.xcaf. Call this after installing a new provider or after
// adding new global agents, skills, or rules to an existing one.
func RebuildGlobalXCAF() error {
	home, err := GlobalHome()
	if err != nil {
		return err
	}
	data := buildGlobalXCAF()
	return os.WriteFile(filepath.Join(home, "global.xcaf"), data, 0600)
}

// EnsureGlobalHome creates ~/.xcaffold/ and its seed files if they don't exist.
func EnsureGlobalHome() error {
	home, err := GlobalHome()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(home, 0755); err != nil {
		return fmt.Errorf("could not create global home: %w", err)
	}

	// Remove legacy settings.xcaf — preferences now live in global.xcaf's settings: block.
	_ = os.Remove(filepath.Join(home, "settings.xcaf"))

	projectsPath := filepath.Join(home, "registry.xcaf")
	if _, err := os.Stat(projectsPath); os.IsNotExist(err) {
		wrapper := map[string]interface{}{
			"kind":     "registry",
			"projects": []Project{},
		}
		out, _ := yaml.Marshal(wrapper)
		_ = os.WriteFile(projectsPath, out, 0600)
	}

	// TODO: auto-generate global.xcaf from detected provider directories.
	// Global scope is under development; bootstrap deferred to avoid emitting
	// fields the current parser cannot consume.

	return nil
}

// ── Resource entry types ──────────────────────────────────────────────────
//
// Each type represents one manageable resource class in global.xcaf.
//
// HOW TO ADD A NEW RESOURCE TYPE (e.g. "templates"):
//  1. Add a struct below (e.g. GlobalTemplateEntry).
//  2. Add a map field to GlobalScanResult (e.g. Templates map[string]GlobalTemplateEntry).
//  3. Initialise the map in NewScanResult().
//  4. Scan it inside the relevant provider Scan function(s).
//  5. Emit it in marshalGlobalXCAF().
//  No other code changes are needed.

type GlobalAgentEntry struct{ InstructionsFile string }
type GlobalSkillEntry struct{ InstructionsFile string }
type GlobalRuleEntry struct{ InstructionsFile string }
type GlobalWorkflowEntry struct{ InstructionsFile string }
type GlobalMCPEntry struct {
	Command string // command-type server (e.g. "npx @modelcontextprotocol/…")
	URL     string // http/sse-type server URL
	Args    []string
}

// GlobalScanResult is the aggregated output of all platform scanners.
// Maps are deduplicated: the first provider to register a key for a given
// resource type wins. Later providers that discover the same key are skipped.
type GlobalScanResult struct {
	Agents    map[string]GlobalAgentEntry
	Skills    map[string]GlobalSkillEntry
	Rules     map[string]GlobalRuleEntry
	Workflows map[string]GlobalWorkflowEntry
	MCP       map[string]GlobalMCPEntry
	// MemoryFile tracks the first global "memory" instructions file discovered
	// (informational, used for display purposes only).
	MemoryFile string
}

func NewScanResult() GlobalScanResult {
	return GlobalScanResult{
		Agents:    make(map[string]GlobalAgentEntry),
		Skills:    make(map[string]GlobalSkillEntry),
		Rules:     make(map[string]GlobalRuleEntry),
		Workflows: make(map[string]GlobalWorkflowEntry),
		MCP:       make(map[string]GlobalMCPEntry),
	}
}

// ── Core bootstrap ────────────────────────────────────────────────────────

// buildGlobalXCAF generates global.xcaf content by iterating registered provider
// scanners. Falls back to the minimal starter template when nothing is found on disk.
func buildGlobalXCAF() []byte {
	userHome, err := os.UserHomeDir()
	if err != nil {
		return []byte(DefaultGlobalXCAFContent)
	}

	// Migrate verbatim from a legacy ~/.claude/global.xcaf if one exists.
	if data, err := os.ReadFile(filepath.Join(userHome, ".claude", "global.xcaf")); err == nil {
		return data
	}

	r := NewScanResult()
	if GlobalScanIterator != nil {
		GlobalScanIterator(userHome, &r)
	}

	// Nothing found — emit the empty starter template.
	if len(r.Agents) == 0 && len(r.Skills) == 0 &&
		len(r.Rules) == 0 && len(r.MCP) == 0 {
		return []byte(DefaultGlobalXCAFContent)
	}

	return marshalGlobalXCAF(&r)
}

// ── Exported scan helpers ─────────────────────────────────────────────────
//
// Provider scan functions compose these helpers rather than duplicating logic.
// They are intentionally generic (dir-agnostic) so any provider can reuse them.

// ScanMarkdownFilesAsAgents reads *.md files from dir and populates out.
// It is suitable for flat-file agent directories (e.g. ~/.claude/agents/).
func ScanMarkdownFilesAsAgents(dir string, out map[string]GlobalAgentEntry) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		id := strings.TrimSuffix(e.Name(), ".md")
		if _, exists := out[id]; !exists {
			out[id] = GlobalAgentEntry{
				InstructionsFile: filepath.ToSlash(filepath.Join(dir, e.Name())),
			}
		}
	}
}

// ScanMarkdownFilesAsRules reads *.md files from dir and populates out.
// It is suitable for flat-file rule directories (e.g. ~/.claude/rules/).
func ScanMarkdownFilesAsRules(dir string, out map[string]GlobalRuleEntry) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		id := strings.TrimSuffix(e.Name(), ".md")
		if _, exists := out[id]; !exists {
			out[id] = GlobalRuleEntry{
				InstructionsFile: filepath.ToSlash(filepath.Join(dir, e.Name())),
			}
		}
	}
}

// ScanSkillDirs reads sub-directories from dir. Each sub-directory that
// contains a SKILL.md file is registered as a skill entry. This matches
// xcaffold's canonical skill-as-directory model.
func ScanSkillDirs(dir string, out map[string]GlobalSkillEntry) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		skillMD := filepath.Join(dir, e.Name(), "SKILL.md")
		if _, err := os.Stat(skillMD); err == nil {
			if _, exists := out[e.Name()]; !exists {
				out[e.Name()] = GlobalSkillEntry{
					InstructionsFile: filepath.ToSlash(skillMD),
				}
			}
		}
	}
}

// ScanMCPFromJSONFile reads a JSON or YAML config file at path and extracts
// MCP server entries from the top-level map key serversKey (typically
// "mcpServers"). cmdKey and urlKey are the field names used for the command
// and URL within each server object — these differ between providers
// (Claude uses "url", Antigravity uses "serverUrl").
func ScanMCPFromJSONFile(path, serversKey, cmdKey, urlKey string, out map[string]GlobalMCPEntry) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	// yaml.Unmarshal handles JSON as a strict superset, so this works for
	// both .json and .yaml files without importing encoding/json.
	var raw map[string]interface{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return
	}
	servers, _ := raw[serversKey].(map[string]interface{})
	for name, srv := range servers {
		if _, exists := out[name]; exists {
			continue // first provider wins; skip duplicates
		}
		srvMap, _ := srv.(map[string]interface{})
		entry := GlobalMCPEntry{}
		if cmd, _ := srvMap[cmdKey].(string); cmd != "" {
			entry.Command = cmd
		}
		if u, _ := srvMap[urlKey].(string); u != "" {
			entry.URL = u
		}
		if argsRaw, _ := srvMap["args"].([]interface{}); len(argsRaw) > 0 {
			for _, a := range argsRaw {
				if s, ok := a.(string); ok {
					entry.Args = append(entry.Args, s)
				}
			}
		}
		out[name] = entry
	}
}

// ── YAML serializer ───────────────────────────────────────────────────────
//
// marshalGlobalXCAF emits all non-empty resource sections in stable order.
//
// HOW TO ADD A NEW SECTION: add an if-block for the new resource map here,
// following the same pattern as existing sections.

func marshalGlobalXCAF(r *GlobalScanResult) []byte {
	var buf bytes.Buffer

	buf.WriteString("kind: global\n")
	buf.WriteString("version: \"1.0\"\n")
	buf.WriteString("# User-wide agent configuration (auto-discovered).\n\n")
	buf.WriteString("# All instructions-file paths are absolute so they resolve correctly\n")
	buf.WriteString("# regardless of the current working directory.\n")

	if len(r.Agents) > 0 {
		buf.WriteString("\nagents:\n")
		for id, a := range r.Agents {
			buf.WriteString("  " + id + ":\n")
			buf.WriteString("    instructions-file: \"" + a.InstructionsFile + "\"\n")
		}
	}

	if len(r.Skills) > 0 {
		buf.WriteString("\nskills:\n")
		for id, s := range r.Skills {
			buf.WriteString("  " + id + ":\n")
			buf.WriteString("    instructions-file: \"" + s.InstructionsFile + "\"\n")
		}
	}

	if len(r.Rules) > 0 {
		buf.WriteString("\nrules:\n")
		for id, ru := range r.Rules {
			buf.WriteString("  " + id + ":\n")
			buf.WriteString("    instructions-file: \"" + ru.InstructionsFile + "\"\n")
		}
	}

	if len(r.Workflows) > 0 {
		buf.WriteString("\nworkflows:\n")
		for id, w := range r.Workflows {
			buf.WriteString("  " + id + ":\n")
			buf.WriteString("    instructions-file: \"" + w.InstructionsFile + "\"\n")
		}
	}

	if len(r.MCP) > 0 {
		buf.WriteString("\n# MCP servers auto-detected from platform config files.\n")
		buf.WriteString("mcp:\n")
		for name, m := range r.MCP {
			buf.WriteString("  " + name + ":\n")
			if m.Command != "" {
				buf.WriteString("    command: \"" + m.Command + "\"\n")
			}
			if m.URL != "" {
				buf.WriteString("    url: \"" + m.URL + "\"\n")
			}
			if len(m.Args) > 0 {
				buf.WriteString("    args:\n")
				for _, a := range m.Args {
					buf.WriteString("      - \"" + a + "\"\n")
				}
			}
		}
	}

	return buf.Bytes()
}

// ── Project registry ──────────────────────────────────────────────────────

func readProjects() ([]Project, error) {
	home, err := GlobalHome()
	if err != nil {
		return nil, err
	}
	projectsPath := filepath.Join(home, "registry.xcaf")
	data, err := os.ReadFile(projectsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []Project{}, nil
		}
		return nil, err
	}

	// Try new format first: {kind: registry, projects: [...]}
	var wrapper struct {
		Kind     string    `yaml:"kind"`
		Projects []Project `yaml:"projects"`
	}
	if err := yaml.Unmarshal(data, &wrapper); err == nil && wrapper.Kind == "registry" {
		if wrapper.Projects == nil {
			return []Project{}, nil
		}
		return wrapper.Projects, nil
	}

	// Fallback: legacy bare []Project array
	var projects []Project
	if err := yaml.Unmarshal(data, &projects); err != nil {
		return nil, err
	}
	return projects, nil
}

func writeProjects(projects []Project) error {
	home, err := GlobalHome()
	if err != nil {
		return err
	}
	projectsPath := filepath.Join(home, "registry.xcaf")
	wrapper := map[string]interface{}{
		"kind":     "registry",
		"projects": projects,
	}
	data, err := yaml.Marshal(wrapper)
	if err != nil {
		return err
	}
	return os.WriteFile(projectsPath, data, 0600)
}

// Register adds a new project or updates an existing one by path.
func Register(projectPath, name string, targets []string, configDir string) error {
	abs, err := filepath.Abs(projectPath)
	if err != nil {
		return err
	}

	projects, err := readProjects()
	if err != nil {
		return err
	}

	found := false
	for i, p := range projects {
		if p.Path == abs {
			projects[i].Name = name
			projects[i].ConfigDir = configDir
			if len(targets) > 0 {
				projects[i].Targets = targets
			}
			found = true
			break
		}
	}

	if !found {
		for _, p := range projects {
			if p.Name == name {
				name = fmt.Sprintf("%s-%s", filepath.Base(filepath.Dir(abs)), name)
				break
			}
		}
		projects = append(projects, Project{
			Path:       abs,
			Name:       name,
			ConfigDir:  configDir,
			Registered: time.Now().UTC(),
			Targets:    targets,
		})
	}

	return writeProjects(projects)
}

// Unregister removes a project by name or path.
func Unregister(nameOrPath string) error {
	projects, err := readProjects()
	if err != nil {
		return err
	}

	abs, _ := filepath.Abs(nameOrPath)
	var filtered []Project
	for _, p := range projects {
		if p.Name != nameOrPath && p.Path != abs {
			filtered = append(filtered, p)
		}
	}

	return writeProjects(filtered)
}

// List returns all registered projects.
func List() ([]Project, error) {
	return readProjects()
}

// Resolve looks up a project by its registered name or absolute path.
func Resolve(nameOrPath string) (Project, error) {
	projects, err := readProjects()
	if err != nil {
		return Project{}, err
	}

	abs, _ := filepath.Abs(nameOrPath)

	for _, p := range projects {
		if p.Name == nameOrPath || p.Path == abs || p.Path == nameOrPath {
			return p, nil
		}
	}

	return Project{}, fmt.Errorf("project not found: %s", nameOrPath)
}

// UpdateLastApplied updates the LastApplied timestamp for a project.
func UpdateLastApplied(projectPath string) error {
	abs, err := filepath.Abs(projectPath)
	if err != nil {
		return err
	}

	projects, err := readProjects()
	if err != nil {
		return err
	}

	for i, p := range projects {
		if p.Path == abs {
			projects[i].LastApplied = time.Now().UTC()
			return writeProjects(projects)
		}
	}
	return nil
}
