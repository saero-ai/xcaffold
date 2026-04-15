// Package copilot compiles an XcaffoldConfig AST into GitHub Copilot instruction
// files. Rules are written as .instructions.md files under .github/instructions/
// with Copilot-compatible YAML frontmatter.
//
// Key normalizations applied during compilation:
//   - Output path: .github/instructions/<id>.instructions.md
//   - activation: always → applyTo: "**"
//   - activation: path-glob → applyTo: "<comma-joined paths>"
//   - activation: manual-mention / model-decided / explicit-invoke → applyTo: "**" + FidelityNote
//   - exclude-agents → excludeAgent: <list> (Copilot singular key name)
package copilot

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/output"
	"github.com/saero-ai/xcaffold/internal/renderer"
)

const targetName = "copilot"

// Renderer compiles an XcaffoldConfig AST into GitHub Copilot instruction files.
// It targets the ".github/instructions/" directory structure.
type Renderer struct{}

// New returns a new Renderer instance.
func New() *Renderer {
	return &Renderer{}
}

// Target returns the identifier for this renderer's target platform.
func (r *Renderer) Target() string {
	return targetName
}

// OutputDir returns the output directory prefix for this renderer.
func (r *Renderer) OutputDir() string {
	return ".github/instructions"
}

// Render wraps a files map in an output.Output. This is an identity
// operation — no additional path rewriting is needed at this layer.
func (r *Renderer) Render(files map[string]string) *output.Output {
	return &output.Output{Files: files}
}

// Compile translates an XcaffoldConfig AST into its Copilot output representation.
// baseDir is the directory that contains the scaffold.xcf file; it is used to
// resolve instructions-file: paths. The second return is a slice of fidelity
// notes describing information loss relative to the native Claude target.
// Compile returns an error if any resource fails to compile. It never panics.
func (r *Renderer) Compile(config *ast.XcaffoldConfig, baseDir string) (*output.Output, []renderer.FidelityNote, error) {
	out := &output.Output{Files: make(map[string]string)}
	var notes []renderer.FidelityNote

	for id, rule := range config.Rules {
		md, ruleNotes, err := compileCopilotRule(id, rule)
		if err != nil {
			return nil, nil, fmt.Errorf("copilot: failed to compile rule %q: %w", id, err)
		}
		safePath := filepath.Clean(fmt.Sprintf(".github/instructions/%s.instructions.md", id))
		out.Files[safePath] = md
		notes = append(notes, ruleNotes...)
	}

	return out, notes, nil
}

// compileCopilotRule renders a single RuleConfig as a Copilot .instructions.md file.
func compileCopilotRule(id string, rule ast.RuleConfig) (string, []renderer.FidelityNote, error) {
	var notes []renderer.FidelityNote

	activation := renderer.ResolvedActivation(rule)

	var applyTo string
	switch activation {
	case ast.RuleActivationAlways:
		applyTo = `"**"`
	case ast.RuleActivationPathGlob:
		if len(rule.Paths) > 0 {
			applyTo = fmt.Sprintf("%q", strings.Join(rule.Paths, ", "))
		} else {
			applyTo = `"**"`
		}
	case ast.RuleActivationManualMention, ast.RuleActivationModelDecided, ast.RuleActivationExplicitInvoke:
		applyTo = `"**"`
		notes = append(notes, renderer.NewNote(
			renderer.LevelWarning,
			targetName,
			"rule",
			id,
			"activation",
			renderer.CodeRuleActivationUnsupported,
			fmt.Sprintf("rule %q: activation %q has no Copilot-native equivalent; emitted as applyTo: \"**\"", id, activation),
			"Use activation: always or activation: path-glob for full Copilot compatibility.",
		))
	default:
		applyTo = `"**"`
	}

	var sb strings.Builder
	sb.WriteString("---\n")

	if rule.Description != "" {
		sb.WriteString(fmt.Sprintf("description: %s\n", rule.Description))
	}

	sb.WriteString(fmt.Sprintf("applyTo: %s\n", applyTo))

	if len(rule.ExcludeAgents) > 0 {
		sb.WriteString("excludeAgent:\n")
		for _, agent := range rule.ExcludeAgents {
			sb.WriteString(fmt.Sprintf("  - %s\n", agent))
		}
	}

	sb.WriteString("---\n")

	if rule.Instructions != "" {
		sb.WriteString("\n")
		sb.WriteString(rule.Instructions)
		sb.WriteString("\n")
	}

	return sb.String(), notes, nil
}
