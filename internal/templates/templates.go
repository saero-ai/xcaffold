package templates

import (
	"strings"
)

// RenderXaffOverrideXCF generates a per-provider override file for the Xaff agent.
// The override file supplements the base agent.xcf with provider-specific settings.
func RenderXaffOverrideXCF(target string) string {
	var sb strings.Builder

	sb.WriteString("---\n")
	sb.WriteString("kind: agent\n")
	sb.WriteString("version: \"1.0\"\n")
	sb.WriteString("name: xaff\n")
	sb.WriteString("\n")

	switch target {
	case "claude":
		sb.WriteString("effort: \"high\"\n")
		sb.WriteString("permission-mode: default\n")
		sb.WriteString("---\n")
	case "cursor":
		sb.WriteString("readonly: false\n")
		sb.WriteString("---\n")
	case "gemini":
		sb.WriteString("---\n")
	case "copilot":
		sb.WriteString("disable-model-invocation: false\n")
		sb.WriteString("---\n")
	case "antigravity":
		sb.WriteString("---\n")
	default:
		sb.WriteString("---\n")
	}

	return sb.String()
}
