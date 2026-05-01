package templates

// RenderCLICheatsheet returns a concise command reference for the xcaffold CLI.
//
// The generated content is written to .xcaffold/schemas/cli-cheatsheet.reference
// and is NOT parsed by xcaffold. It serves as a quick-reference for all
// xcaffold commands, flags, and global options.
func RenderCLICheatsheet() string {
	return `# ============================================================
# xcaffold CLI — Command Reference
# ============================================================
# This file is NOT parsed by xcaffold.
# ============================================================

# ── Global Flags ─────────────────────────────────────────────
#   --config PATH    Path to project.xcf (default: ./.xcaffold/project.xcf)
#   --global         Operate on user-wide global config (~/.xcaffold/global.xcf)
#   --no-color       Disable color output

# ── apply ────────────────────────────────────────────────────
# Compile .xcf resources into provider-native agent files.
# Scope: project, global
#
#   xcaffold apply
#   xcaffold apply --target claude
#   xcaffold apply --target cursor
#   xcaffold apply --target gemini
#   xcaffold apply --target copilot
#   xcaffold apply --target antigravity
#   xcaffold apply --dry-run              # preview changes without writing
#   xcaffold apply --check                # check syntax only
#   xcaffold apply --check-permissions    # report permission drops, then exit
#   xcaffold apply --force                # overwrite customized files
#   xcaffold apply --backup               # backup target directory first
#   xcaffold apply --blueprint NAME       # compile a specific blueprint

# ── validate ─────────────────────────────────────────────────
# Check .xcf syntax, cross-references, and structural invariants.
# Scope: project, global
#
#   xcaffold validate
#   xcaffold validate --structural        # run structural invariant checks

# ── status ───────────────────────────────────────────────────
# Show compilation state and check for drift across all providers.
# Scope: project, global
#
#   xcaffold status
#   xcaffold status --target claude       # focus on a single provider
#   xcaffold status --all                 # show all files (default: drifted only)

# ── import ───────────────────────────────────────────────────
# Import existing provider config into project.xcf.
# Scope: project
#
#   xcaffold import
#   xcaffold import --plan                # preview without making changes
#   xcaffold import --target claude       # import from a specific provider only
#   xcaffold import --filter-agent NAME
#   xcaffold import --filter-skill NAME
#   xcaffold import --filter-rule NAME
#   xcaffold import --filter-workflow NAME
#   xcaffold import --filter-mcp NAME
#   xcaffold import --filter-hooks
#   xcaffold import --filter-settings
#   xcaffold import --filter-memory

# ── init ─────────────────────────────────────────────────────
# Bootstrap a new project.xcf configuration.
# Scope: project, global
#
#   xcaffold init
#   xcaffold init --yes                   # accept all defaults (CI/CD mode)
#   xcaffold init --target claude         # set target explicitly
#   xcaffold init --target claude,cursor  # multiple targets
#   xcaffold init --no-policies           # skip starter policies
#   xcaffold init --json                  # machine-readable manifest output

# ── graph ────────────────────────────────────────────────────
# Visualize the resource dependency graph.
# Scope: project, global
#
#   xcaffold graph
#   xcaffold graph --format terminal      # default
#   xcaffold graph --format mermaid
#   xcaffold graph --format dot
#   xcaffold graph --format json
#   xcaffold graph --agent NAME           # target a specific agent
#   xcaffold graph --full                 # show fully expanded topology
#   xcaffold graph --scan-output          # scan for undeclared artifacts

# ── list ─────────────────────────────────────────────────────
# List discovered resources and blueprints.
# Scope: project, global
#
#   xcaffold list
#   xcaffold list --verbose               # show memory entry names per agent

# ── Workflow ─────────────────────────────────────────────────
# Typical development loop:
#
#   xcaffold init --target claude         # 1. bootstrap
#   # edit xcf/ files
#   xcaffold validate                     # 2. validate
#   xcaffold apply --dry-run              # 3. preview
#   xcaffold apply                        # 4. compile
#   xcaffold status                       # 5. verify drift

# ── Generated Layout ─────────────────────────────────────────
#
#   my-project/
#     .xcaffold/
#       project.xcf                   # kind: project (targets, resource refs)
#       schemas/
#         agent.xcf.reference         # agent field reference (this file's siblings)
#         skill.xcf.reference
#         rule.xcf.reference
#         workflow.xcf.reference
#         mcp.xcf.reference
#         hooks.xcf.reference
#         memory.xcf.reference
#         cli-cheatsheet.reference    # THIS FILE
#     xcf/
#       agents/
#         xaff/
#           agent.xcf                 # base agent (universal)
#           agent.claude.xcf          # per-provider override
#       skills/
#         xcaffold/
#           xcaffold.xcf              # xcaffold authoring skill
#       rules/
#         xcf-conventions/
#           xcf-conventions.xcf       # xcaffold authoring conventions rule
#       policies/
#         require-agent-description.xcf
#         require-agent-instructions.xcf
#       settings.xcf                  # kind: settings (MCP, permissions, hooks)
`
}
