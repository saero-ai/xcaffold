package templates

import (
	"fmt"
	"strings"
)

// allProviders defines the canonical display order for provider columns.
var allProviders = []string{"claude", "cursor", "gemini", "copilot", "antigravity"}

// providerFieldSupport maps kind -> field-label -> providers that support it.
// "dropped" at apply time means the field is silently discarded by the renderer.
// Fields not listed for a provider are dropped.
var providerFieldSupport = map[string][]matrixRow{
	"agent": {
		{"name / description", []string{"claude", "cursor", "gemini", "copilot", "antigravity"}},
		{"model", []string{"claude", "gemini", "antigravity"}},
		{"effort", []string{"claude"}},
		{"permission-mode", []string{"claude"}},
		{"tools", []string{"claude", "cursor", "gemini", "copilot", "antigravity"}},
		{"skills / rules / mcp", []string{"claude", "cursor", "gemini", "antigravity"}},
		{"hooks", []string{"claude"}},
		{"memory", []string{"claude"}},
		{"targets: overrides", []string{"claude", "cursor", "gemini", "copilot", "antigravity"}},
		{"instructions", []string{"claude", "cursor", "gemini", "copilot", "antigravity"}},
	},
	"rule": {
		{"name / description", []string{"claude", "cursor", "gemini", "copilot", "antigravity"}},
		{"activation", []string{"claude", "cursor", "gemini", "copilot", "antigravity"}},
		{"always-apply", []string{"claude", "cursor", "gemini", "copilot", "antigravity"}},
		{"paths", []string{"claude", "cursor", "copilot", "antigravity"}},
		{"exclude-agents", []string{"copilot"}},
		{"instructions", []string{"claude", "cursor", "gemini", "copilot", "antigravity"}},
	},
	"settings": {
		{"permissions", []string{"claude"}},
		{"sandbox", []string{"claude"}},
		{"mcp-servers", []string{"claude", "cursor", "gemini", "antigravity"}},
		{"hooks", []string{"claude"}},
		{"model", []string{"claude", "gemini", "antigravity"}},
		{"env", []string{"claude", "gemini"}},
	},
}

type matrixRow struct {
	field     string
	supported []string // providers that support this field
}

// RenderMatrix returns a YAML comment block showing which fields are
// supported by each of the selectedTargets for the given kind.
// Returns empty string for unknown kinds.
func RenderMatrix(kind string, selectedTargets []string) string {
	rows, ok := providerFieldSupport[kind]
	if !ok || len(selectedTargets) == 0 {
		return ""
	}

	// Filter allProviders to only selected targets (preserving display order).
	var cols []string
	for _, p := range allProviders {
		for _, sel := range selectedTargets {
			if p == sel {
				cols = append(cols, p)
				break
			}
		}
	}

	// Calculate column width: max(len(provider), 7) + 2 padding.
	colWidth := 8
	for _, c := range cols {
		if len(c)+2 > colWidth {
			colWidth = len(c) + 2
		}
	}
	fieldWidth := 24 // fixed width for field column

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# kind: %s - provider field support for your selected targets\n#\n", kind))

	// Header row
	sb.WriteString(fmt.Sprintf("#  %-*s", fieldWidth, "Field"))
	for _, col := range cols {
		sb.WriteString(fmt.Sprintf("%-*s", colWidth, col))
	}
	sb.WriteString("\n")

	// Data rows
	for _, row := range rows {
		supportSet := make(map[string]bool)
		for _, p := range row.supported {
			supportSet[p] = true
		}
		sb.WriteString(fmt.Sprintf("#  %-*s", fieldWidth, row.field))
		for _, col := range cols {
			if supportSet[col] {
				sb.WriteString(fmt.Sprintf("%-*s", colWidth, "YES"))
			} else {
				sb.WriteString(fmt.Sprintf("%-*s", colWidth, "dropped"))
			}
		}
		sb.WriteString("\n")
	}

	return sb.String()
}
