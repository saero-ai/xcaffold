package bir

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/renderer"
	"gopkg.in/yaml.v3"
)

// xXcaffoldMarker is the shape of the x-xcaffold provenance block emitted by
// translator.TranslateWorkflow inside a ```yaml code fence in the rule body.
type xXcaffoldMarker struct {
	CompiledFrom string   `yaml:"compiled-from"`
	WorkflowName string   `yaml:"workflow-name"`
	ApiVersion   string   `yaml:"api-version"`
	StepOrder    []string `yaml:"step-order"`
	StepSkills   []string `yaml:"step-skills"`
}

// yamlCodeFenceRE matches a ```yaml … ``` block anywhere in the rule body.
var yamlCodeFenceRE = regexp.MustCompile("(?s)```yaml\n(.*?)```")

// ReassembleWorkflow reconstructs a WorkflowConfig from a compiled Claude rule
// file plus its companion skill files, using the x-xcaffold: provenance marker
// emitted by translator.TranslateWorkflow.
//
// The rule file is expected at:
//
//	<dir>/.claude/rules/<workflowName>-workflow.md
//
// Each skill file is expected at:
//
//	<dir>/.claude/skills/<skillID>/SKILL.md
//
// Returns (nil, nil, nil) when:
//   - the rule file does not exist
//   - the rule body contains no x-xcaffold marker
//   - compiled-from is not "workflow"
//   - one or more referenced skill files are missing (graceful fallback)
//
// Returns (*WorkflowConfig, []FidelityNote, nil) on successful reassembly.
func ReassembleWorkflow(dir, workflowName string) (*ast.WorkflowConfig, []renderer.FidelityNote, error) {
	rulePath := filepath.Join(dir, ".claude", "rules", workflowName+"-workflow.md")
	data, err := os.ReadFile(rulePath)
	if err != nil {
		// File missing — no marker, not an error.
		return nil, nil, nil
	}

	marker, description, err := extractXXcaffoldMarker(string(data))
	if err != nil {
		return nil, nil, err
	}
	if marker == nil {
		return nil, nil, nil
	}
	if marker.CompiledFrom != "workflow" {
		return nil, nil, nil
	}

	apiVersion := marker.ApiVersion
	if apiVersion == "" {
		apiVersion = "workflow/v1"
	}

	// Validate parallel slices.
	if len(marker.StepOrder) != len(marker.StepSkills) {
		return nil, nil, fmt.Errorf("provenance marker step-order and step-skills length mismatch in %s", rulePath)
	}

	// Load each skill file and extract the body.
	steps := make([]ast.WorkflowStep, 0, len(marker.StepOrder))
	for i, stepName := range marker.StepOrder {
		skillID := marker.StepSkills[i]
		skillPath := filepath.Join(dir, ".claude", "skills", skillID, "SKILL.md")
		skillData, readErr := os.ReadFile(skillPath)
		if readErr != nil {
			// Missing skill — graceful fallback.
			return nil, nil, nil
		}
		body := stripFrontmatter(string(skillData))
		steps = append(steps, ast.WorkflowStep{
			Name: stepName,
			Body: strings.TrimSpace(body),
		})
	}

	wf := &ast.WorkflowConfig{
		ApiVersion:  apiVersion,
		Name:        workflowName,
		Description: description,
		Steps:       steps,
	}

	note := renderer.NewNote(
		renderer.LevelInfo,
		"claude",
		"workflow",
		workflowName,
		"",
		renderer.CodeWorkflowLoweredToRulePlusSkill,
		fmt.Sprintf("workflow %q reassembled from rule+skill provenance marker", workflowName),
		"",
	)

	return wf, []renderer.FidelityNote{note}, nil
}

// extractXXcaffoldMarker parses the x-xcaffold provenance block from a rule body.
// The provenance block is written as a ```yaml code fence by translator.TranslateWorkflow.
// It also extracts a description from YAML frontmatter if present.
// Returns (nil, "", nil) when no marker is found.
func extractXXcaffoldMarker(content string) (*xXcaffoldMarker, string, error) {
	// Extract YAML frontmatter description if present.
	description := extractDescriptionFromFrontmatter(content)

	// Find a ```yaml code block anywhere in the content.
	match := yamlCodeFenceRE.FindStringSubmatch(content)
	if match == nil {
		return nil, description, nil
	}
	yamlBlock := match[1]

	// The block may start with "x-xcaffold:" — unmarshal as a map first.
	var raw map[string]xXcaffoldMarker
	if err := yaml.Unmarshal([]byte(yamlBlock), &raw); err != nil {
		return nil, description, fmt.Errorf("parsing x-xcaffold block: %w", err)
	}

	m, ok := raw["x-xcaffold"]
	if !ok {
		return nil, description, nil
	}
	return &m, description, nil
}

// extractDescriptionFromFrontmatter returns the description field from YAML
// frontmatter if the content starts with ---. Returns "" if absent or unparseable.
func extractDescriptionFromFrontmatter(content string) string {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	if !strings.HasPrefix(content, "---\n") {
		return ""
	}
	end := strings.Index(content[4:], "\n---")
	if end == -1 {
		return ""
	}
	fm := content[4 : 4+end]
	var meta struct {
		Description string `yaml:"description"`
	}
	if err := yaml.Unmarshal([]byte(fm), &meta); err != nil {
		return ""
	}
	return meta.Description
}
