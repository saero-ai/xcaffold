package generator

import (
	"strings"

	"github.com/saero-ai/xcaffold/internal/analyzer"
)

// buildGeneratorPrompt creates the prompt for the generator based on
// the extracted project signature.
func buildGeneratorPrompt(sig *analyzer.ProjectSignature) string {
	var sb strings.Builder

	sb.WriteString("You are an expert platform architect and DevSecOps engineer. ")
	sb.WriteString("Your objective is to generate an `xcaffold` YAML configuration file that defines AI agent workflows for the given project.\n\n")

	sb.WriteString("## Project Context Context (Deterministic Signature)\n")
	sb.WriteString("Below is the highly compressed deterministic signature of the project structure and dependencies:\n")
	sb.WriteString("```json\n")
	sb.WriteString(sig.String())
	sb.WriteString("\n```\n\n")

	sb.WriteString("## Architectural Philosophy & Rules\n")
	sb.WriteString("1. **No Hallucinations**: Rely ONLY on the provided JSON schema. Do not guess file names or logic that wasn't exported in the signature.\n")
	sb.WriteString("2. **Actionable Steps**: Generate discrete, robust CLI terminal commands in the agent `instructions` block.\n")

	sb.WriteString("\n## Adversarial Verification Requirements\n")
	sb.WriteString("The agent instructions MUST include a `test` block. The test block contains `assertions`.\n")
	sb.WriteString("Because this configuration will be tested by an Adversarial LLM-as-a-Judge, you must author High-Quality, Adversarial check statements, such as:\n")
	sb.WriteString("- Boundary checks (e.g. 'Must elegantly handle empty inputs without crashing')\n")
	sb.WriteString("- Idempotency (e.g. 'Running the command twice does not create duplicate entries')\n")
	sb.WriteString("- Concrete state (e.g. 'File build/output.bin exists and is greater than 0 bytes')\n")
	sb.WriteString("Avoid 'happy path' checks like 'The command returned 200 OK'.\n\n")

	sb.WriteString("## Output Format Requirement\n")
	sb.WriteString("You must return ONLY a raw JSON strictly adhering to the schema below. Do not wrap it in markdown code blocks. Do not add conversational text.\n")
	sb.WriteString("```json\n")
	sb.WriteString("{\n")
	sb.WriteString("  \"yaml_config\": \"[The raw YAML string adhering to the .xcf schema]\",\n")
	sb.WriteString("  \"audit_report\": {\n")
	sb.WriteString("    \"type\": \"[greenfield|brownfield]\",\n")
	sb.WriteString("    \"scores\": {\n")
	sb.WriteString("      \"security\": [0-100],\n")
	sb.WriteString("      \"prompt_quality\": [0-100],\n")
	sb.WriteString("      \"tool_restrictions\": [0-100]\n")
	sb.WriteString("    },\n")
	sb.WriteString("    \"feedback\": \"[A detailed paragraph explaining why this architecture was chosen or what was corrected from the previous setup]\"\n")
	sb.WriteString("  }\n")
	sb.WriteString("}\n")
	sb.WriteString("```\n\n")
	sb.WriteString("The `yaml_config` string must contain raw YAML adhering strictly to the `.xcf` schema. CRITICAL: Because this is JSON, you MUST escape all newlines (`\\n`) and quotes (`\\\"`) inside the `yaml_config` string so the overall JSON remains valid:\n")
	sb.WriteString(`
project:
  name: "[Inferred Project Name]"
version: "1.0.0"

agents:
  "agent_id_here":
    description: "Brief summary"
    model: "claude-3-5-sonnet-latest"
    instructions: >
      1. Step one
      2. Step two
    assertions:
      - "Concrete adversarial assertion 1"
      - "Concrete adversarial assertion 2"
`)
	sb.WriteString("\n")

	return sb.String()
}
