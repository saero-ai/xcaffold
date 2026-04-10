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

// GlobalHome returns the absolute path to ~/.xcaffold/.
func GlobalHome() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not determine home directory: %w", err)
	}
	return filepath.Join(home, ".xcaffold"), nil
}

// DefaultGlobalXCFContent is a minimal starter template used only when no
// real global agent configs are found on disk.
const DefaultGlobalXCFContent = `version: "1.0"
project:
  name: "global"
  description: "User-wide agent configuration."

# No global agents were detected. Add agents here and run:
#   xcaffold apply --scope global
agents: {}
`

// RebuildGlobalXCF re-scans all registered platform providers and rewrites
// ~/.xcaffold/global.xcf. Call this after installing a new provider or after
// adding new global agents, skills, or rules to an existing one.
func RebuildGlobalXCF() error {
	home, err := GlobalHome()
	if err != nil {
		return err
	}
	data := buildGlobalXCF()
	return os.WriteFile(filepath.Join(home, "global.xcf"), data, 0600)
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

	configPath := filepath.Join(home, "settings.xcf")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		cfg := Config{DefaultTarget: "claude"}
		out, _ := yaml.Marshal(cfg)
		_ = os.WriteFile(configPath, out, 0600)
	}

	projectsPath := filepath.Join(home, "registry.xcf")
	if _, err := os.Stat(projectsPath); os.IsNotExist(err) {
		out, _ := yaml.Marshal([]Project{})
		_ = os.WriteFile(projectsPath, out, 0600)
	}

	// Auto-bootstrap global.xcf only when it doesn't exist yet.
	globalXcfPath := filepath.Join(home, "global.xcf")
	if _, err := os.Stat(globalXcfPath); os.IsNotExist(err) {
		data := buildGlobalXCF()
		_ = os.WriteFile(globalXcfPath, data, 0600)
	}

	return nil
}

// ── Resource entry types ──────────────────────────────────────────────────
//
// Each type represents one manageable resource class in global.xcf.
//
// HOW TO ADD A NEW RESOURCE TYPE (e.g. "templates"):
//  1. Add a struct below (e.g. globalTemplateEntry).
//  2. Add a map field to globalScanResult (e.g. templates map[string]globalTemplateEntry).
//  3. Initialise the map in newScanResult().
//  4. Scan it inside the relevant provider Scan function(s).
//  5. Emit it in marshalGlobalXCF().
//  No other code changes are needed.

type globalAgentEntry struct{ instructionsFile string }
type globalSkillEntry struct{ instructionsFile string }
type globalRuleEntry struct{ instructionsFile string }
type globalWorkflowEntry struct{ instructionsFile string }
type globalMCPEntry struct {
	command string // command-type server (e.g. "npx @modelcontextprotocol/…")
	url     string // http/sse-type server URL
	args    []string
}

// globalScanResult is the aggregated output of all platform scanners.
// Maps are deduplicated: the first provider to register a key for a given
// resource type wins. Later providers that discover the same key are skipped.
type globalScanResult struct {
	agents    map[string]globalAgentEntry
	skills    map[string]globalSkillEntry
	rules     map[string]globalRuleEntry
	workflows map[string]globalWorkflowEntry
	mcp       map[string]globalMCPEntry
	// memoryFile tracks the first global "memory" instructions file discovered
	// (informational, used for display purposes only).
	memoryFile string
}

func newScanResult() globalScanResult {
	return globalScanResult{
		agents:    make(map[string]globalAgentEntry),
		skills:    make(map[string]globalSkillEntry),
		rules:     make(map[string]globalRuleEntry),
		workflows: make(map[string]globalWorkflowEntry),
		mcp:       make(map[string]globalMCPEntry),
	}
}

// ── Provider registry ─────────────────────────────────────────────────────
//
// platformProvider describes the global configuration layout of one agentic
// IDE platform. Each provider's Scan function discovers whatever global
// resources that platform exposes on disk, then merges them into the shared
// globalScanResult (first-seen wins for deduplication).
//
// HOW TO ADD A NEW PROVIDER:
//  1. Implement a scan function: funcscanProvider<Name>(userHome string, r *globalScanResult).
//  2. Document which on-disk paths this provider uses (see existing examples).
//  3. Append an entry to globalProviders below.
//  No other code changes are needed.

type platformProvider struct {
	// Scan discovers global resources for this provider and merges them into r.
	// Func field is placed first to satisfy fieldalignment requirements.
	Scan func(userHome string, r *globalScanResult)
	// Name is a human-readable label used in YAML comments and CLI output.
	Name string
}

// globalProviders is the ordered registry of supported agentic platforms.
// Providers are scanned in declaration order; first entry to claim a key wins.
var globalProviders = []platformProvider{
	{Scan: scanProviderClaude, Name: "Claude Code"},
	{Scan: scanProviderAntigravity, Name: "Antigravity"},
	// Cursor: User Rules live in the Cursor app Settings UI — not on disk.
	// Project-scoped rules (.cursor/rules/) are handled separately per-project.
	//
	// Add new providers here, e.g.:
	// {Scan:scanProvider<Name>, Name: "<Name>"},
}

// ── Core bootstrap ────────────────────────────────────────────────────────

// buildGlobalXCF generates global.xcf content by iterating globalProviders.
// Falls back to the minimal starter template when nothing is found on disk.
func buildGlobalXCF() []byte {
	userHome, err := os.UserHomeDir()
	if err != nil {
		return []byte(DefaultGlobalXCFContent)
	}

	// Migrate verbatim from a legacy ~/.claude/global.xcf if one exists.
	if data, err := os.ReadFile(filepath.Join(userHome, ".claude", "global.xcf")); err == nil {
		return data
	}

	r := newScanResult()
	for _, p := range globalProviders {
		p.Scan(userHome, &r)
	}

	// Nothing found — emit the empty starter template.
	if len(r.agents) == 0 && len(r.skills) == 0 &&
		len(r.rules) == 0 && len(r.mcp) == 0 {
		return []byte(DefaultGlobalXCFContent)
	}

	return marshalGlobalXCF(&r)
}

// ── Claude Code provider ──────────────────────────────────────────────────
//
// Global configuration layout:
//   ~/.claude/agents/*.md                    → agents
//   ~/.claude/skills/<name>/SKILL.md         → skills
//   ~/.claude/rules/*.md                     → rules
//   ~/.claude/CLAUDE.md                      → rule "claude-memory" + memoryFile
//   ~/.claude.json  (key: mcpServers)        → mcp  (url key: "url")

func scanProviderClaude(userHome string, r *globalScanResult) {
	claudeDir := filepath.Join(userHome, ".claude")

	scanMarkdownFilesAsAgents(filepath.Join(claudeDir, "agents"), r.agents)
	scanSkillDirs(filepath.Join(claudeDir, "skills"), r.skills)
	scanMarkdownFilesAsRules(filepath.Join(claudeDir, "rules"), r.rules)

	// CLAUDE.md is the global memory/instructions file for Claude Code.
	// We surface it as a rule so xcaffold can track and diff it.
	claudeMD := filepath.Join(claudeDir, "CLAUDE.md")
	if _, err := os.Stat(claudeMD); err == nil {
		if _, exists := r.rules["claude-memory"]; !exists {
			r.rules["claude-memory"] = globalRuleEntry{
				instructionsFile: filepath.ToSlash(claudeMD),
			}
		}
		if r.memoryFile == "" {
			r.memoryFile = filepath.ToSlash(claudeMD)
		}
	}

	// ~/.claude.json holds user-scoped and local-scoped MCP server configs.
	scanMCPFromJSONFile(
		filepath.Join(userHome, ".claude.json"),
		"mcpServers", "command", "url",
		r.mcp,
	)
}

// ── Antigravity provider ──────────────────────────────────────────────────
//
// Global configuration layout:
//   ~/.gemini/antigravity/skills/<name>/SKILL.md  → skills
//   ~/.gemini/GEMINI.md                           → rule "gemini-global"
//   ~/.gemini/antigravity/mcp_config.json
//     (key: mcpServers, url key: "serverUrl")     → mcp

func scanProviderAntigravity(userHome string, r *globalScanResult) {
	agDir := filepath.Join(userHome, ".gemini", "antigravity")

	// Antigravity global skills live at ~/.gemini/antigravity/skills/.
	scanSkillDirs(filepath.Join(agDir, "skills"), r.skills)

	// GEMINI.md is Antigravity's single global rule file (not a directory).
	geminiMD := filepath.Join(userHome, ".gemini", "GEMINI.md")
	if _, err := os.Stat(geminiMD); err == nil {
		if _, exists := r.rules["gemini-global"]; !exists {
			r.rules["gemini-global"] = globalRuleEntry{
				instructionsFile: filepath.ToSlash(geminiMD),
			}
		}
	}

	// Antigravity uses "serverUrl" instead of the standard "url" key in its
	// MCP config file at ~/.gemini/antigravity/mcp_config.json.
	scanMCPFromJSONFile(
		filepath.Join(agDir, "mcp_config.json"),
		"mcpServers", "command", "serverUrl",
		r.mcp,
	)
}

// ── Generic scan helpers ──────────────────────────────────────────────────
//
// Provider scan functions compose these helpers rather than duplicating logic.
// They are intentionally generic (dir-agnostic) so any provider can reuse them.

// scanMarkdownFilesAsAgents reads *.md files from dir and populates out.
// It is suitable for flat-file agent directories (e.g. ~/.claude/agents/).
func scanMarkdownFilesAsAgents(dir string, out map[string]globalAgentEntry) {
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
			out[id] = globalAgentEntry{
				instructionsFile: filepath.ToSlash(filepath.Join(dir, e.Name())),
			}
		}
	}
}

// scanMarkdownFilesAsRules reads *.md files from dir and populates out.
// It is suitable for flat-file rule directories (e.g. ~/.claude/rules/).
func scanMarkdownFilesAsRules(dir string, out map[string]globalRuleEntry) {
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
			out[id] = globalRuleEntry{
				instructionsFile: filepath.ToSlash(filepath.Join(dir, e.Name())),
			}
		}
	}
}

// scanSkillDirs reads sub-directories from dir. Each sub-directory that
// contains a SKILL.md file is registered as a skill entry. This matches
// xcaffold's canonical skill-as-directory model.
func scanSkillDirs(dir string, out map[string]globalSkillEntry) {
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
				out[e.Name()] = globalSkillEntry{
					instructionsFile: filepath.ToSlash(skillMD),
				}
			}
		}
	}
}

// scanMCPFromJSONFile reads a JSON or YAML config file at path and extracts
// MCP server entries from the top-level map key serversKey (typically
// "mcpServers"). cmdKey and urlKey are the field names used for the command
// and URL within each server object — these differ between providers
// (Claude uses "url", Antigravity uses "serverUrl").
func scanMCPFromJSONFile(path, serversKey, cmdKey, urlKey string, out map[string]globalMCPEntry) {
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
		entry := globalMCPEntry{}
		if cmd, _ := srvMap[cmdKey].(string); cmd != "" {
			entry.command = cmd
		}
		if u, _ := srvMap[urlKey].(string); u != "" {
			entry.url = u
		}
		if argsRaw, _ := srvMap["args"].([]interface{}); len(argsRaw) > 0 {
			for _, a := range argsRaw {
				if s, ok := a.(string); ok {
					entry.args = append(entry.args, s)
				}
			}
		}
		out[name] = entry
	}
}

// ── YAML serializer ───────────────────────────────────────────────────────
//
// marshalGlobalXCF emits all non-empty resource sections in stable order.
//
// HOW TO ADD A NEW SECTION: add an if-block for the new resource map here,
// following the same pattern as existing sections.

func marshalGlobalXCF(r *globalScanResult) []byte {
	var buf bytes.Buffer

	buf.WriteString("version: \"1.0\"\n")
	buf.WriteString("project:\n")
	buf.WriteString("  name: \"global\"\n")
	buf.WriteString("  description: \"User-wide agent configuration (auto-discovered).\"\n\n")
	buf.WriteString("# All instructions_file paths are absolute so they resolve correctly\n")
	buf.WriteString("# regardless of the current working directory.\n")

	if len(r.agents) > 0 {
		buf.WriteString("\nagents:\n")
		for id, a := range r.agents {
			buf.WriteString("  " + id + ":\n")
			buf.WriteString("    instructions_file: \"" + a.instructionsFile + "\"\n")
		}
	}

	if len(r.skills) > 0 {
		buf.WriteString("\nskills:\n")
		for id, s := range r.skills {
			buf.WriteString("  " + id + ":\n")
			buf.WriteString("    instructions_file: \"" + s.instructionsFile + "\"\n")
		}
	}

	if len(r.rules) > 0 {
		buf.WriteString("\nrules:\n")
		for id, ru := range r.rules {
			buf.WriteString("  " + id + ":\n")
			buf.WriteString("    instructions_file: \"" + ru.instructionsFile + "\"\n")
		}
	}

	if len(r.workflows) > 0 {
		buf.WriteString("\nworkflows:\n")
		for id, w := range r.workflows {
			buf.WriteString("  " + id + ":\n")
			buf.WriteString("    instructions_file: \"" + w.instructionsFile + "\"\n")
		}
	}

	if len(r.mcp) > 0 {
		buf.WriteString("\n# MCP servers auto-detected from platform config files.\n")
		buf.WriteString("mcp:\n")
		for name, m := range r.mcp {
			buf.WriteString("  " + name + ":\n")
			if m.command != "" {
				buf.WriteString("    command: \"" + m.command + "\"\n")
			}
			if m.url != "" {
				buf.WriteString("    url: \"" + m.url + "\"\n")
			}
			if len(m.args) > 0 {
				buf.WriteString("    args:\n")
				for _, a := range m.args {
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
	projectsPath := filepath.Join(home, "registry.xcf")
	data, err := os.ReadFile(projectsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []Project{}, nil
		}
		return nil, err
	}
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
	projectsPath := filepath.Join(home, "registry.xcf")
	data, err := yaml.Marshal(projects)
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
