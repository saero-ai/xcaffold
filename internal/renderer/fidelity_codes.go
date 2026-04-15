package renderer

// Code catalog for FidelityNote.Code. Codes are stable identifiers:
// once published in a release they are never renamed or removed.
// New codes must be added here and documented in the spec.
const (
	// CodeRendererKindUnsupported is emitted when an entire resource kind
	// (e.g. workflow) has no representation in the target. All fields are dropped.
	CodeRendererKindUnsupported = "RENDERER_KIND_UNSUPPORTED"

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

	// CodeMemoryBodyEmpty is emitted when a memory entry has neither
	// instructions nor a resolvable instructions-file body.
	CodeMemoryBodyEmpty = "MEMORY_BODY_EMPTY"

	// CodeMemorySeedSkipped is emitted when a seed-once memory file already
	// exists on disk and --reseed was not set; the existing file is preserved.
	CodeMemorySeedSkipped = "MEMORY_SEED_SKIPPED"

	// CodeMemoryIndexUpdateFailed is emitted when writing the MEMORY.md
	// project index fails but the memory file itself was written successfully.
	CodeMemoryIndexUpdateFailed = "MEMORY_INDEX_UPDATE_FAILED"

	// CodeWorkflowLoweredToRulePlusSkill is emitted when a workflow was compiled
	// to a rule + skill pair because the target has no first-class workflow primitive.
	CodeWorkflowLoweredToRulePlusSkill = "WORKFLOW_LOWERED_TO_RULE_PLUS_SKILL"

	// CodeWorkflowLoweredToPromptFile is emitted when a workflow was compiled to
	// a static prompt file. Dynamic branching steps are lost.
	CodeWorkflowLoweredToPromptFile = "WORKFLOW_LOWERED_TO_PROMPT_FILE"

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

	// CodeAgentSecurityFieldsDropped is emitted when one or more security-related
	// agent fields (permissionMode, disallowedTools, isolation) have no equivalent
	// and were dropped. Security constraints will NOT be enforced on this target.
	CodeAgentSecurityFieldsDropped = "AGENT_SECURITY_FIELDS_DROPPED"

	// CodeSkillScriptsDropped is emitted when a skill's scripts/ directory
	// reference has no equivalent in the target and was dropped.
	CodeSkillScriptsDropped = "SKILL_SCRIPTS_DROPPED"

	// CodeSkillAssetsDropped is emitted when a skill's assets/ directory
	// reference has no equivalent in the target and was dropped.
	CodeSkillAssetsDropped = "SKILL_ASSETS_DROPPED"

	// CodeRuleActivationUnsupported is emitted when a rule's activation mode
	// has no native equivalent in the target. The rule is emitted as always-loaded.
	CodeRuleActivationUnsupported = "RULE_ACTIVATION_UNSUPPORTED"

	// CodeRuleExcludeAgentsDropped is emitted when a rule's exclude-agents list
	// has no native equivalent in the target and was dropped.
	CodeRuleExcludeAgentsDropped = "RULE_EXCLUDE_AGENTS_DROPPED"
)

// AllCodes returns every code defined in this catalog. Used by tests to verify
// catalog completeness and by tooling that needs to enumerate known codes.
func AllCodes() []string {
	return []string{
		CodeRendererKindUnsupported,
		CodeFieldUnsupported,
		CodeFieldTransformed,
		CodeActivationDegraded,
		CodeInstructionsFlattened,
		CodeInstructionsClosestWinsForcedConcat,
		CodeMemoryNoNativeTarget,
		CodeMemoryPartialFidelity,
		CodeMemoryBodyEmpty,
		CodeMemorySeedSkipped,
		CodeMemoryIndexUpdateFailed,
		CodeWorkflowLoweredToRulePlusSkill,
		CodeWorkflowLoweredToPromptFile,
		CodeReservedOutputPathRejected,
		CodeSettingsFieldUnsupported,
		CodeHookInterpolationRequiresEnvSyntax,
		CodeAgentModelUnmapped,
		CodeAgentSecurityFieldsDropped,
		CodeSkillScriptsDropped,
		CodeSkillAssetsDropped,
		CodeRuleActivationUnsupported,
		CodeRuleExcludeAgentsDropped,
	}
}
