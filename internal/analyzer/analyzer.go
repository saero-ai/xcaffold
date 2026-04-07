package analyzer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/resolver"
)

// TokenEntry represents the token estimate for a single artifact.
type TokenEntry struct {
	ID     string `json:"id"`
	Kind   string `json:"kind"`
	Source string `json:"source"`
	Tokens int    `json:"tokens"`
}

// TokenReport aggregates token estimates across all artifact categories.
type TokenReport struct {
	ByKind   map[string]int `json:"by_kind"`
	Entries  []TokenEntry   `json:"entries"`
	Warnings []string       `json:"warnings,omitempty"`
	Total    int            `json:"total"`
}

// Analyzer handles static analysis of the AST without invoking network calls.
// Token counts are estimated using a byte-count heuristic (~4 bytes per token),
// which is a conservative approximation of BPE tokenization for typical
// English-language agent instruction text.
type Analyzer struct{}

// New returns a new Analyzer instance.
func New() *Analyzer {
	return &Analyzer{}
}

func estimateTokens(payload string) int {
	// ~4 printable characters per token is a conservative BPE estimate.
	return utf8.RuneCountInString(payload) / 4
}

func sizeTokens(b []byte) int {
	return estimateTokens(string(b))
}

// AnalyzeTokens estimates tokens for all artifacts declared in the XCF config.
// It resolves instructions_file content from disk using baseDir.
// Errors on missing files are collected as warnings, not fatal.
//
//nolint:gocyclo
func (a *Analyzer) AnalyzeTokens(config *ast.XcaffoldConfig, baseDir string) *TokenReport {
	report := &TokenReport{
		ByKind: make(map[string]int),
	}

	addEntry := func(id, kind string, tokens int) {
		report.Entries = append(report.Entries, TokenEntry{
			ID:     id,
			Kind:   kind,
			Tokens: tokens,
			Source: "xcf",
		})
		report.Total += tokens
		report.ByKind[kind] += tokens
	}

	for id, agent := range config.Agents {
		body, err := resolver.ResolveInstructions(agent.Instructions, agent.InstructionsFile, "", baseDir)
		if err != nil {
			report.Warnings = append(report.Warnings, fmt.Sprintf("agent %s: %v", id, err))
		}
		addEntry(id, "agent", estimateTokens(body+" "+agent.Description))
	}

	for id, skill := range config.Skills {
		body, err := resolver.ResolveInstructions(skill.Instructions, skill.InstructionsFile, "", baseDir)
		if err != nil {
			report.Warnings = append(report.Warnings, fmt.Sprintf("skill %s: %v", id, err))
		}

		// Add reference file contents
		var refsContent strings.Builder
		for _, ref := range skill.References {
			var refPath string
			if filepath.IsAbs(ref) {
				refPath = ref
			} else {
				refPath = filepath.Join(baseDir, ref)
			}
			b, err := os.ReadFile(refPath)
			if err == nil {
				refsContent.Write(b)
			} else {
				report.Warnings = append(report.Warnings, fmt.Sprintf("skill %s reference %s: %v", id, ref, err))
			}
		}

		addEntry(id, "skill", estimateTokens(body+" "+skill.Description+" "+refsContent.String()))
	}

	for id, rule := range config.Rules {
		body, err := resolver.ResolveInstructions(rule.Instructions, rule.InstructionsFile, "", baseDir)
		if err != nil {
			report.Warnings = append(report.Warnings, fmt.Sprintf("rule %s: %v", id, err))
		}
		addEntry(id, "rule", estimateTokens(body+" "+rule.Description))
	}

	for id, wf := range config.Workflows {
		body, err := resolver.ResolveInstructions(wf.Instructions, wf.InstructionsFile, "", baseDir)
		if err != nil {
			report.Warnings = append(report.Warnings, fmt.Sprintf("workflow %s: %v", id, err))
		}
		addEntry(id, "workflow", estimateTokens(body+" "+wf.Description))
	}

	for id, mcp := range config.MCP {
		b, _ := json.Marshal(mcp)
		addEntry(id, "mcp", sizeTokens(b))
	}

	if len(config.Hooks) > 0 {
		b, _ := json.Marshal(config.Hooks)
		hookTokens := sizeTokens(b)

		// Follow hook scripts
		for _, group := range config.Hooks {
			for _, grp := range group {
				for _, h := range grp.Hooks {
					if h.Type == "command" {
						cmd := h.Command
						if strings.HasSuffix(cmd, ".sh") {
							// Crude extraction: resolve the last token if it looks like a script path
							parts := strings.Fields(cmd)
							if len(parts) > 0 {
								scriptPath := parts[len(parts)-1]
								scriptPath = strings.ReplaceAll(scriptPath, "$CLAUDE_PROJECT_DIR", baseDir)
								b, err := os.ReadFile(scriptPath)
								if err == nil {
									hookTokens += sizeTokens(b)
								}
							}
						}
					}
				}
			}
		}
		addEntry("hooks", "hook", hookTokens)
	}

	// Settings
	b, _ := json.Marshal(config.Settings)
	if len(b) > 4 { // "{}" is 2 bytes
		addEntry("settings", "settings", sizeTokens(b))
	}

	// Sort entries for deterministic output
	sort.Slice(report.Entries, func(i, j int) bool {
		if report.Entries[i].Kind != report.Entries[j].Kind {
			return report.Entries[i].Kind < report.Entries[j].Kind
		}
		return report.Entries[i].ID < report.Entries[j].ID
	})

	return report
}

// ScanOutputDir walks a compiled output directory (.claude/ or .agents/)
// and estimates tokens for on-disk artifacts not declared in the XCF.
//
//nolint:gocyclo
func (a *Analyzer) ScanOutputDir(dir string, declared map[string]bool) ([]TokenEntry, error) {
	var entries []TokenEntry

	source := "disk"
	if strings.Contains(dir, ".claude") {
		source = "disk-claude"
	} else if strings.Contains(dir, ".agents") {
		source = "disk-agents"
	}

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return nil
		}

		parts := strings.Split(filepath.ToSlash(rel), "/")
		if len(parts) == 0 {
			return nil
		}

		b, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		tokens := sizeTokens(b)

		// .agents/rules/foo.md -> kind="rule", id="foo"
		// .agents/skills/foo/SKILL.md -> kind="skill", id="foo"

		var kind string
		var id string

		//nolint:goconst
		switch parts[0] {
		case "agents":
			if len(parts) == 2 && strings.HasSuffix(parts[1], ".md") {
				kind, id = "agent", strings.TrimSuffix(parts[1], ".md")
			}
		case "rules":
			if len(parts) == 2 && strings.HasSuffix(parts[1], ".md") {
				kind, id = "rule", strings.TrimSuffix(parts[1], ".md")
			}
		case "skills":
			if len(parts) == 3 && parts[2] == "SKILL.md" {
				kind, id = "skill", parts[1]
			} else if len(parts) >= 3 {
				// E: count references under parent skill if they are in the references dir
				// We won't try to parse out individual undeclared references here simply,
				// we'll just skip them, as they're too granular.
				return nil
			}
		case "workflows":
			if len(parts) == 2 && strings.HasSuffix(parts[1], ".md") {
				kind, id = "workflow", strings.TrimSuffix(parts[1], ".md")
			}
		case "hooks":
			if len(parts) == 2 && strings.HasSuffix(parts[1], ".sh") {
				kind, id = "hook", parts[1] // ID is the script name
			}
		default:
			return nil
		}

		// Skip if this artifact is managed by XCF
		if kind != "" && id != "" && !declared[fmt.Sprintf("%s:%s", kind, id)] {
			entries = append(entries, TokenEntry{
				ID:     id,
				Kind:   kind,
				Tokens: tokens,
				Source: source,
			})
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Kind != entries[j].Kind {
			return entries[i].Kind < entries[j].Kind
		}
		return entries[i].ID < entries[j].ID
	})

	return entries, nil
}
