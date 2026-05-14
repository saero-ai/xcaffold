package renderer

// Code catalog for FidelityNote.Code. Codes are stable identifiers:
// once published in a release they are never renamed or removed.
// New codes must be added here and documented in the spec.
const (
	// CodeRendererKindUnsupported is emitted when an entire resource kind
	// (e.g. workflow) has no representation in the target. All fields are dropped.
	CodeRendererKindUnsupported = "RENDERER_KIND_UNSUPPORTED"

	// CodeRendererKindDowngraded is emitted when an entire resource kind has no
	// native representation in the target, and was instead rendered as its
	// closest equivalent (e.g. agent rendered as a specialist note).
	CodeRendererKindDowngraded = "RENDERER_KIND_DOWNGRADED"

	// CodeFieldUnsupported is emitted when a specific field on a resource
	// has no equivalent in the target and was dropped silently.
	CodeFieldUnsupported = "FIELD_UNSUPPORTED"

	// CodeFieldTransformed is emitted when a field was translated to a
	// different name or structure. No information was lost.
	CodeFieldTransformed = "FIELD_TRANSFORMED"

	// CodeActivationDegraded is emitted when a rule's activation value was
	// lowered to the closest supported equivalent. Behavior may differ from intent.
	CodeActivationDegraded = "ACTIVATION_DEGRADED"

	// CodeInstructionsFlattened is emitted when multiple instructions sources
	// were merged into a single string. Structural distinction is lost.
	CodeInstructionsFlattened = "INSTRUCTIONS_FLATTENED"

	// CodeInstructionsClosestWinsForcedConcat is emitted when no exact semantic
	// match existed; the closest mechanism was used and content was appended
	// rather than replacing.
	CodeInstructionsClosestWinsForcedConcat = "INSTRUCTIONS_CLOSEST_WINS_FORCED_CONCAT"

	// CodeMemoryNoNativeTarget is emitted when the memory kind has no native
	// storage target in this renderer. Content was dropped or embedded in a fallback.
	CodeMemoryNoNativeTarget = "MEMORY_NO_NATIVE_TARGET"

	// CodeMemoryPartialFidelity is emitted when a memory entry is appended to a
	// single flat file (e.g. GEMINI.md), losing per-entry file granularity.
	CodeMemoryPartialFidelity = "MEMORY_PARTIAL_FIDELITY"

	// CodeMemoryIndexUpdateFailed is emitted when writing the MEMORY.md
	// project index fails but the memory file itself was written successfully.
	CodeMemoryIndexUpdateFailed = "MEMORY_INDEX_UPDATE_FAILED"

	// CodeMemoryDriftDetected is emitted when a tracked memory entry's on-disk
	// hash diverges from the hash recorded in the state file after the last seed.
	// This is an error-level note: the drift aborts the memory pass for that
	// entry. Emitted alongside the hard error so tooling consuming FidelityNotes
	// (e.g. CI drift reports) sees a structured event rather than only stderr text.
	CodeMemoryDriftDetected = "MEMORY_DRIFT_DETECTED"

	// CodeWorkflowLoweredToRulePlusSkill is emitted when a workflow was compiled
	// to a rule + skill pair because the target has no first-class workflow primitive.
	CodeWorkflowLoweredToRulePlusSkill = "WORKFLOW_LOWERED_TO_RULE_PLUS_SKILL"

	// CodeWorkflowLoweredToPromptFile is emitted when a workflow was compiled to
	// a static prompt file. Dynamic branching steps are lost.
	CodeWorkflowLoweredToPromptFile = "WORKFLOW_LOWERED_TO_PROMPT_FILE"

	// CodeWorkflowLoweredToCustomCommand is emitted when a workflow was compiled to
	// a custom shell command or script. Native workflow semantics are not preserved.
	CodeWorkflowLoweredToCustomCommand = "WORKFLOW_LOWERED_TO_CUSTOM_COMMAND"

	// CodeWorkflowLoweredToNative is emitted when a workflow was lowered to the
	// native workflow model but with reduced fidelity compared to the source.
	CodeWorkflowLoweredToNative = "WORKFLOW_LOWERED_TO_NATIVE"

	// CodeWorkflowNoNativeTarget is emitted when a workflow has no native representation
	// in the target. The workflow was either dropped or converted to an alternative form.
	CodeWorkflowNoNativeTarget = "WORKFLOW_NO_NATIVE_TARGET"

	// CodeReservedOutputPathRejected is emitted when a generated output path
	// conflicts with a reserved path. The file was not written.
	CodeReservedOutputPathRejected = "RESERVED_OUTPUT_PATH_REJECTED"

	// CodeSettingsFieldUnsupported is emitted when a settings-level field
	// (e.g. permissions, sandbox) has no equivalent and was dropped.
	CodeSettingsFieldUnsupported = "SETTINGS_FIELD_UNSUPPORTED"

	// CodeHookInterpolationRequiresEnvSyntax is emitted when a hook or MCP value
	// uses ${VAR} interpolation; the target requires ${env:VAR} syntax.
	CodeHookInterpolationRequiresEnvSyntax = "HOOK_INTERPOLATION_REQUIRES_ENV_SYNTAX"

	// CodeAgentModelUnmapped is emitted when an agent's model value could not be
	// mapped to a known target-specific string and was omitted.
	CodeAgentModelUnmapped = "AGENT_MODEL_UNMAPPED"

	// Deprecated: CodeAgentSecurityFieldsDropped is superseded by per-field
	// FIELD_UNSUPPORTED notes from CheckFieldSupport. Retained for code stability.
	CodeAgentSecurityFieldsDropped = "AGENT_SECURITY_FIELDS_DROPPED"

	// CodeAgentToolsDropped is emitted when an agent's tools list contains
	// Claude-native tools (e.g. "Bash", "Read") but the target renderer is not Claude.
	// Only MCP tools (mcp_*) and explicitly wildcarded/supported tools are kept.
	CodeAgentToolsDropped = "AGENT_TOOLS_DROPPED"

	// CodeSkillScriptsDropped is emitted when a skill's scripts/ directory
	// reference has no equivalent in the target and was dropped.
	CodeSkillScriptsDropped = "SKILL_SCRIPTS_DROPPED"

	// CodeSkillAssetsDropped is emitted when a skill's assets/ directory
	// reference has no equivalent in the target and was dropped.
	CodeSkillAssetsDropped = "SKILL_ASSETS_DROPPED"

	// CodeSkillReferencesDropped is emitted when a skill's references/ directory
	// has no equivalent in the target and was dropped.
	CodeSkillReferencesDropped = "SKILL_REFERENCES_DROPPED"

	// CodeSkillExamplesDropped is emitted when a skill's examples/ directory
	// could not be compiled for the target provider.
	CodeSkillExamplesDropped = "SKILL_EXAMPLES_DROPPED"

	// CodeRuleActivationUnsupported is emitted when a rule's activation mode
	// has no native equivalent in the target. The rule is emitted as always-loaded.
	CodeRuleActivationUnsupported = "RULE_ACTIVATION_UNSUPPORTED"

	// CodeRuleExcludeAgentsDropped is emitted when a rule's exclude-agents list
	// has no native equivalent in the target and was dropped.
	CodeRuleExcludeAgentsDropped = "RULE_EXCLUDE_AGENTS_DROPPED"

	// CodeInstructionsImportInlined is emitted when InstructionsImports entries
	// are inlined into the rendered output because the target lacks native @-import
	// support (e.g. Cursor, Copilot, Gemini).
	CodeInstructionsImportInlined = "INSTRUCTIONS_IMPORT_INLINED"

	// CodeReconciliationUnionLossy is emitted when a union merge of variant
	// instructions-scopes drops one or more lines that exist in one variant but
	// not all others.
	CodeReconciliationUnionLossy = "RECONCILIATION_UNION_LOSSY"

	// CodeReconciliationDriftDetected is emitted when the on-disk content of
	// an instructions-scope file diverges from all known variants and the
	// reconciliation strategy cannot determine a canonical source.
	CodeReconciliationDriftDetected = "RECONCILIATION_DRIFT_DETECTED"

	// CodeOptimizerPassReordered is emitted when the optimizer reordered compilation
	// passes to meet target constraints. Semantic equivalence is maintained but
	// the sequence differs from the source declaration.
	CodeOptimizerPassReordered = "OPTIMIZER_PASS_REORDERED"

	// CodeMCPGlobalConfigOnly is emitted when MCP servers are declared but the
	// target only reads MCP configuration from a global user-level path. No
	// project-local MCP config file is written. The user must configure MCP
	// servers via the provider UI or by editing the global config file directly.
	CodeMCPGlobalConfigOnly = "MCP_GLOBAL_CONFIG_ONLY"

	// CodeClaudeNativePassthrough is emitted by the Copilot renderer when a
	// resource (agent, skill, rule, or root project instruction) is skipped
	// because a .claude/ directory is detected in baseDir at apply time.
	// GitHub Copilot natively loads .claude/agents/, .claude/skills/,
	// .claude/rules/, and root CLAUDE.md automatically, making re-translation
	// into .github/ redundant when .claude/ is already present.
	CodeClaudeNativePassthrough = "CLAUDE_NATIVE_PASSTHROUGH"

	// CodeFieldRequiredForTarget is emitted when a compilation target requires
	// a field that is absent from the source resource. The compiled output will
	// be incomplete or rejected by the target provider.
	CodeFieldRequiredForTarget = "FIELD_REQUIRED_FOR_TARGET"

	// CodeWorkflowRoutedToSingleSkill is emitted when a body-only workflow
	// is rendered as a single routing skill.
	CodeWorkflowRoutedToSingleSkill = "WORKFLOW_ROUTED_TO_SINGLE_SKILL"

	// CodeWorkflowChainedToOrchestrator is emitted when a workflow with
	// skill-ref steps is rendered as an orchestrator skill.
	CodeWorkflowChainedToOrchestrator = "WORKFLOW_CHAINED_TO_ORCHESTRATOR"

	// CodeWorkflowSimpleToSections is emitted when a workflow with inline
	// body steps is rendered as a single skill with step sections.
	CodeWorkflowSimpleToSections = "WORKFLOW_SIMPLE_TO_SECTIONS"

	// CodeWorkflowDefaultChanged is emitted as a migration warning when a
	// workflow that would have produced rule+skill under the old default now
	// produces a single skill under the new structure-based inference.
	CodeWorkflowDefaultChanged = "WORKFLOW_DEFAULT_CHANGED"

	// CodeWorkflowBodyIgnored is emitted when a workflow has both a top-level
	// body and steps; the body is dropped in chained and simple modes.
	CodeWorkflowBodyIgnored = "WORKFLOW_BODY_IGNORED"

	// CodeWorkflowMixedSteps is emitted when a workflow has a mix of
	// skill-ref steps and inline-body steps.
	CodeWorkflowMixedSteps = "WORKFLOW_MIXED_STEPS"
)

// AllCodes returns every code defined in this catalog. Used by tests to verify
// catalog completeness and by tooling that needs to enumerate known codes.
func AllCodes() []string {
	return []string{
		CodeRendererKindUnsupported,
		CodeRendererKindDowngraded,
		CodeFieldUnsupported,
		CodeFieldTransformed,
		CodeActivationDegraded,
		CodeInstructionsFlattened,
		CodeInstructionsClosestWinsForcedConcat,
		CodeMemoryNoNativeTarget,
		CodeMemoryPartialFidelity,
		CodeMemoryIndexUpdateFailed,
		CodeWorkflowLoweredToRulePlusSkill,
		CodeWorkflowLoweredToPromptFile,
		CodeWorkflowLoweredToCustomCommand,
		CodeWorkflowLoweredToNative,
		CodeWorkflowNoNativeTarget,
		CodeReservedOutputPathRejected,
		CodeSettingsFieldUnsupported,
		CodeHookInterpolationRequiresEnvSyntax,
		CodeAgentModelUnmapped,
		CodeAgentSecurityFieldsDropped,
		CodeAgentToolsDropped,
		CodeSkillScriptsDropped,
		CodeSkillAssetsDropped,
		CodeSkillReferencesDropped,
		CodeSkillExamplesDropped,
		CodeRuleActivationUnsupported,
		CodeRuleExcludeAgentsDropped,
		CodeInstructionsImportInlined,
		CodeMemoryDriftDetected,
		CodeReconciliationUnionLossy,
		CodeReconciliationDriftDetected,
		CodeOptimizerPassReordered,
		CodeMCPGlobalConfigOnly,
		CodeClaudeNativePassthrough,
		CodeFieldRequiredForTarget,
		CodeWorkflowRoutedToSingleSkill,
		CodeWorkflowChainedToOrchestrator,
		CodeWorkflowSimpleToSections,
		CodeWorkflowDefaultChanged,
		CodeWorkflowBodyIgnored,
		CodeWorkflowMixedSteps,
	}
}
