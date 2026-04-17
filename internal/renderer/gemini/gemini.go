// Package gemini compiles an XcaffoldConfig AST into Gemini CLI output files.
// Project instructions are written to GEMINI.md using concat-nested semantics with
// native @-import preservation. Rules are written to .gemini/rules/<id>.md and
// referenced via @-import lines in GEMINI.md.
package gemini

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/output"
	"github.com/saero-ai/xcaffold/internal/renderer"
)

const targetName = "gemini"

// Renderer compiles an XcaffoldConfig AST into Gemini CLI output files.
type Renderer struct{}

// New returns a new Renderer instance.
func New() *Renderer { return &Renderer{} }

// Target returns the canonical name of this renderer.
func (r *Renderer) Target() string { return targetName }

// OutputDir returns the base output directory for Gemini CLI.
func (r *Renderer) OutputDir() string { return ".gemini" }

// Render wraps a files map in an output.Output. This is an identity
// operation — no additional path rewriting is needed at this layer.
func (r *Renderer) Render(files map[string]string) *output.Output {
	return &output.Output{Files: files}
}

// Compile translates an XcaffoldConfig AST into Gemini CLI output files.
// baseDir is the directory containing the scaffold.xcf file; it is used to
// resolve instructions-file paths. Compile returns an error if any resource
// fails to compile. It never panics.
func (r *Renderer) Compile(config *ast.XcaffoldConfig, baseDir string) (*output.Output, []renderer.FidelityNote, error) {
	out := &output.Output{Files: make(map[string]string)}
	var notes []renderer.FidelityNote

	if config.Project != nil {
		instrNotes := r.renderProjectInstructions(config, baseDir, out.Files)
		notes = append(notes, instrNotes...)
	}

	ruleNotes, err := r.renderRules(config, out.Files)
	if err != nil {
		return nil, nil, err
	}
	notes = append(notes, ruleNotes...)

	return out, notes, nil
}

// renderProjectInstructions writes project root instructions to GEMINI.md and
// emits per-scope nested GEMINI.md files. @-import lines are preserved verbatim
// since Gemini natively supports them.
func (r *Renderer) renderProjectInstructions(config *ast.XcaffoldConfig, baseDir string, files map[string]string) []renderer.FidelityNote {
	p := config.Project
	if p.Instructions == "" && p.InstructionsFile == "" {
		return nil
	}

	rootContent := resolveInstructionsContent(p.Instructions, p.InstructionsFile, baseDir)

	var sb strings.Builder
	sb.WriteString(rootContent)

	// Append @-import lines — Gemini supports native @-imports.
	for _, imp := range p.InstructionsImports {
		if !strings.HasSuffix(sb.String(), "\n") {
			sb.WriteString("\n")
		}
		fmt.Fprintf(&sb, "@%s\n", imp)
	}

	files["GEMINI.md"] = sb.String()

	// Emit per-scope GEMINI.md files.
	for _, scope := range p.InstructionsScopes {
		scopeContent := resolveScopeContent(scope, targetName, baseDir)
		if scopeContent == "" {
			continue
		}
		scopePath := filepath.Join(scope.Path, "GEMINI.md")
		safePath := filepath.Clean(scopePath)
		files[safePath] = scopeContent
	}

	return nil
}

// renderRules writes each rule to .gemini/rules/<id>.md and appends @-import
// lines to GEMINI.md. Rules with unsupported activation modes emit a fidelity note
// but are still written (Gemini treats all imported rules as always-active).
func (r *Renderer) renderRules(config *ast.XcaffoldConfig, files map[string]string) ([]renderer.FidelityNote, error) {
	if len(config.Rules) == 0 {
		return nil, nil
	}

	var notes []renderer.FidelityNote
	var importLines []string

	for _, id := range sortedKeys(config.Rules) {
		rule := config.Rules[id]

		activation := renderer.ResolvedActivation(rule)
		if activation != ast.RuleActivationAlways && activation != ast.RuleActivationPathGlob {
			notes = append(notes, renderer.NewNote(
				renderer.LevelWarning,
				targetName,
				"rule",
				id,
				"activation",
				renderer.CodeRuleActivationUnsupported,
				fmt.Sprintf("rule %q activation %q has no native equivalent in Gemini; rule will be always-loaded via @-import", id, activation),
				"Remove the activation field or use 'always' or 'path-glob' for Gemini targets",
			))
		}

		body := buildRuleBody(rule)
		rulePath := fmt.Sprintf(".gemini/rules/%s.md", id)
		safePath := filepath.Clean(rulePath)
		files[safePath] = body
		importLines = append(importLines, fmt.Sprintf("@%s", safePath))
	}

	if len(importLines) > 0 {
		existing := files["GEMINI.md"]
		if existing != "" && !strings.HasSuffix(existing, "\n") {
			existing += "\n"
		}
		files["GEMINI.md"] = existing + strings.Join(importLines, "\n") + "\n"
	}

	return notes, nil
}

// buildRuleBody constructs the markdown content for a rule file.
func buildRuleBody(rule ast.RuleConfig) string {
	var sb strings.Builder
	if rule.Description != "" {
		fmt.Fprintf(&sb, "# %s\n\n", rule.Description)
	}
	body := strings.TrimRight(rule.Instructions, "\n")
	if body != "" {
		sb.WriteString(body)
		sb.WriteString("\n")
	}
	return sb.String()
}

// resolveInstructionsContent returns inline instructions or reads InstructionsFile
// relative to baseDir. Returns empty string on any read error.
func resolveInstructionsContent(inline, file, baseDir string) string {
	if inline != "" {
		return inline
	}
	if file == "" {
		return ""
	}
	data, err := os.ReadFile(filepath.Join(baseDir, file))
	if err != nil {
		return ""
	}
	return string(data)
}

// resolveScopeContent returns the effective content for a scope, preferring
// a gemini-specific variant when one is declared.
func resolveScopeContent(scope ast.InstructionsScope, provider, baseDir string) string {
	if v, ok := scope.Variants[provider]; ok {
		return resolveInstructionsContent("", v.InstructionsFile, baseDir)
	}
	return resolveInstructionsContent(scope.Instructions, scope.InstructionsFile, baseDir)
}

// sortedKeys returns a sorted slice of keys from a map.
func sortedKeys[K ~string, V any](m map[K]V) []K {
	keys := make([]K, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	return keys
}
