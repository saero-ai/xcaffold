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
	"os"
	"path/filepath"
	"sort"
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

	if config.Project != nil {
		instrNotes := r.renderProjectInstructions(config, baseDir, out.Files)
		notes = append(notes, instrNotes...)
	}

	return out, notes, nil
}

// instructionsMode returns the effective instructions-mode for the Copilot renderer.
// Reads project.target-options.copilot.instructions-mode; defaults to "flat".
func instructionsMode(config *ast.XcaffoldConfig) string {
	if config.Project == nil {
		return "flat"
	}
	if opts, ok := config.Project.TargetOptions[targetName]; ok {
		switch opts.InstructionsMode {
		case "nested":
			return "nested"
		}
	}
	return "flat"
}

// renderProjectInstructions dispatches to the flat or nested implementation
// based on the effective instructions-mode for this compilation.
func (r *Renderer) renderProjectInstructions(config *ast.XcaffoldConfig, baseDir string, files map[string]string) []renderer.FidelityNote {
	if instructionsMode(config) == "nested" {
		return r.renderProjectInstructionsNested(config, baseDir, files)
	}
	return r.renderProjectInstructionsFlat(config, baseDir, files)
}

// renderProjectInstructionsFlat emits a single flat-singleton file at
// .github/copilot-instructions.md containing the project root instructions
// followed by each InstructionsScope entry, each wrapped in
// xcaffold:scope HTML provenance markers (path, merge-strategy, origin).
//
// Copilot has no multi-file nesting model, so all scopes are merged into one
// file. Structural distinction is preserved via provenance markers only. One
// INSTRUCTIONS_FLATTENED info note is emitted per scope.
func (r *Renderer) renderProjectInstructionsFlat(config *ast.XcaffoldConfig, baseDir string, files map[string]string) []renderer.FidelityNote {
	p := config.Project
	if p.Instructions == "" && p.InstructionsFile == "" {
		return nil
	}

	var notes []renderer.FidelityNote

	rootContent := renderer.ResolveInstructionsContent(p.Instructions, p.InstructionsFile, baseDir)

	// Inline @-imports — Copilot has no native @-import mechanism.
	for _, imp := range p.InstructionsImports {
		data, err := os.ReadFile(filepath.Join(baseDir, imp))
		if err == nil {
			rootContent += "\n\n" + string(data)
		}
	}

	var sb strings.Builder
	sb.WriteString(rootContent)

	// Sort scopes: depth ascending (fewer path separators first), then alphabetical.
	scopes := make([]ast.InstructionsScope, len(p.InstructionsScopes))
	copy(scopes, p.InstructionsScopes)
	sort.SliceStable(scopes, func(i, j int) bool {
		di := strings.Count(scopes[i].Path, "/")
		dj := strings.Count(scopes[j].Path, "/")
		if di != dj {
			return di < dj
		}
		return scopes[i].Path < scopes[j].Path
	})

	for _, scope := range scopes {
		scopeContent := copilotResolveScopeContent(scope, baseDir)

		if scopeContent != "" {
			name := strings.ReplaceAll(filepath.Clean(scope.Path), string(filepath.Separator), "-")
			filename := fmt.Sprintf("instructions/%s.instructions.md", name)

			var scb strings.Builder
			scb.WriteString("---\n")
			fmt.Fprintf(&scb, "applyTo: %q\n", scope.Path+"/**")
			scb.WriteString("---\n\n")
			scb.WriteString(scopeContent)
			files[filename] = scb.String()
		}

		// Build provenance marker attributes — the A-6 parser uses
		// `(\w[\w-]*)="([^"]*)"`, so any double quote inside an attribute value
		// would terminate the match early. Replace all double quotes with single
		// quotes before embedding.
		safePath := strings.ReplaceAll(scope.Path, `"`, `'`)
		mergeStrategy := scope.MergeStrategy
		if mergeStrategy == "" {
			mergeStrategy = "concat"
		}

		origin := ""
		if scope.SourceProvider != "" || scope.SourceFilename != "" {
			safeProvider := strings.ReplaceAll(scope.SourceProvider, `"`, `'`)
			safeFilename := strings.ReplaceAll(scope.SourceFilename, `"`, `'`)
			origin = fmt.Sprintf(` origin="%s:%s"`, safeProvider, safeFilename)
		}

		fmt.Fprintf(&sb, "\n\n<!-- xcaffold:scope path=\"%s\" merge=\"%s\"%s -->\n",
			safePath, mergeStrategy, origin)
		sb.WriteString(scopeContent)
		sb.WriteString("\n<!-- xcaffold:/scope -->\n")

		notes = append(notes, renderer.NewNote(
			renderer.LevelInfo,
			targetName,
			"instructions",
			scope.Path,
			"merge-strategy",
			renderer.CodeInstructionsFlattened,
			fmt.Sprintf("scope %q flattened into single copilot-instructions.md file with provenance marker", scope.Path),
			"Use a target that supports nested instruction files (e.g. claude) if scope isolation is required",
		))
	}

	safePath := filepath.Clean(".github/copilot-instructions.md")
	files[safePath] = sb.String()
	return notes
}

// renderProjectInstructionsNested emits per-directory AGENTS.md files instead
// of the flat singleton. This mirrors the closest-wins-nested class used by
// the cursor renderer. Root instructions go to AGENTS.md;
// each InstructionsScope produces <scope.Path>/AGENTS.md.
// concat-tagged scopes are pre-flattened (root + scope), emitting a
// INSTRUCTIONS_CLOSEST_WINS_FORCED_CONCAT warning.
func (r *Renderer) renderProjectInstructionsNested(config *ast.XcaffoldConfig, baseDir string, files map[string]string) []renderer.FidelityNote {
	p := config.Project
	if p.Instructions == "" && p.InstructionsFile == "" {
		return nil
	}

	var notes []renderer.FidelityNote

	rootContent := renderer.ResolveInstructionsContent(p.Instructions, p.InstructionsFile, baseDir)

	// Inline @-imports — AGENTS.md has no native @-import mechanism.
	for _, imp := range p.InstructionsImports {
		data, err := os.ReadFile(filepath.Join(baseDir, imp))
		if err == nil {
			rootContent += "\n\n" + string(data)
		}
	}

	files[filepath.Clean("AGENTS.md")] = rootContent

	for _, scope := range p.InstructionsScopes {
		scopeContent := copilotResolveScopeContent(scope, baseDir)
		safePath := filepath.Clean(scope.Path + "/AGENTS.md")

		if scope.MergeStrategy == "concat" {
			files[safePath] = rootContent + "\n\n" + scopeContent
			notes = append(notes, renderer.NewNote(
				renderer.LevelWarning,
				targetName,
				"instructions",
				scope.Path,
				"merge-strategy",
				renderer.CodeInstructionsClosestWinsForcedConcat,
				fmt.Sprintf("concat scope %q pre-flattened into closest-wins AGENTS.md", scope.Path),
				"Use merge-strategy: closest-wins or flat for Copilot nested mode",
			))
		} else {
			files[safePath] = scopeContent
		}
	}

	return notes
}

func copilotResolveScopeContent(scope ast.InstructionsScope, baseDir string) string {
	if v, ok := scope.Variants[targetName]; ok {
		return renderer.ResolveInstructionsContent("", v.InstructionsFile, baseDir)
	}
	return renderer.ResolveInstructionsContent(scope.Instructions, scope.InstructionsFile, baseDir)
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
