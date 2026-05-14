# ============================================================
# xcaffold CLI — Command Reference
# ============================================================
# This file is NOT parsed by xcaffold.
# ============================================================

# ── Global Flags ─────────────────────────────────────────────
#   --config PATH    Path to project.xcaf (default: ./project.xcaf)
#   --global         Operate on user-wide global config (~/.xcaffold/global.xcaf)
#   --no-color       Disable color output

# ── init ─────────────────────────────────────────────────────
# Bootstrap a new project.xcaf configuration.
# Scope: project, global
#
#   xcaffold init
#   xcaffold init --yes                   # accept all defaults (CI/CD mode)
#   xcaffold init --target claude         # set target explicitly
#   xcaffold init --target claude,cursor  # multiple targets
#   xcaffold init --json                  # machine-readable manifest output

# ── apply ────────────────────────────────────────────────────
# Compile .xcaf resources into provider-native agent files.
# Scope: project, global
#
#   xcaffold apply
#   xcaffold apply --target claude
#   xcaffold apply --target cursor
#   xcaffold apply --target gemini
#   xcaffold apply --target copilot
#   xcaffold apply --target antigravity
#   xcaffold apply --dry-run              # preview changes without writing
#   xcaffold apply --force                # overwrite customized files
#   xcaffold apply --backup               # backup target directory first
#   xcaffold apply --project NAME         # apply to a registered project
#   xcaffold apply --blueprint NAME       # compile a specific blueprint

# ── validate ─────────────────────────────────────────────────
# Check .xcaf syntax, cross-references, and structural invariants.
# Scope: project, global
#
#   xcaffold validate
#   xcaffold validate --target claude     # validate field support for a specific provider

# ── status ───────────────────────────────────────────────────
# Show compilation state and check for drift across all providers.
# Scope: project, global
#
#   xcaffold status
#   xcaffold status --target claude       # focus on a single provider
#   xcaffold status --all                 # show all files (default: drifted only)

# ── import ───────────────────────────────────────────────────
# Import existing provider config into project.xcaf.
# Scope: project
#
#   xcaffold import
#   xcaffold import --plan                # preview without making changes
#   xcaffold import --target claude       # import from a specific provider only
#   xcaffold import --agent NAME          # import agents (filter by name)
#   xcaffold import --skill NAME          # import skills (filter by name)
#   xcaffold import --rule NAME           # import rules (filter by name)
#   xcaffold import --workflow NAME       # import workflows (filter by name)
#   xcaffold import --mcp NAME            # import MCP servers (filter by name)
#   xcaffold import --hook                # import hooks
#   xcaffold import --setting             # import settings
#   xcaffold import --memory              # import memory

# ── graph ────────────────────────────────────────────────────
# Visualize the resource dependency graph.
# Scope: project, global
#
#   xcaffold graph
#   xcaffold graph --format terminal      # default
#   xcaffold graph --format mermaid
#   xcaffold graph --format dot
#   xcaffold graph --format json
#   xcaffold graph --agent NAME           # focus on a specific agent
#   xcaffold graph --project NAME         # focus on a registered project
#   xcaffold graph --full                 # show fully expanded topology
#   xcaffold graph --all                  # show global topology + all projects
#   xcaffold graph --scan-output          # scan for undeclared artifacts

# ── list ─────────────────────────────────────────────────────
# List discovered resources.
# Scope: project, global
#
#   xcaffold list
#   xcaffold list --verbose               # show memory entry names per agent

# ── Workflow ─────────────────────────────────────────────────
# Typical development loop:
#
#   xcaffold init --target claude         # 1. bootstrap
#   # edit xcaf/ files
#   xcaffold validate                     # 2. validate
#   xcaffold apply --dry-run              # 3. preview
#   xcaffold apply                        # 4. compile
#   xcaffold status                       # 5. verify drift

# ── Generated Layout ─────────────────────────────────────────
#
#   my-project/
#     .xcaffold/
#       project.xcaf                   # kind: project (targets, resource refs)
#     xcaf/
#       agents/
#         xaff/
#           agent.xcaf                 # base agent (universal)
#           agent.claude.xcaf          # per-provider override
#       skills/
#         xcaffold/
#           xcaffold.xcaf         # THIS SKILL
#           references/
#             agent-reference.md      # agent field catalog
#             skill-reference.md      # skill field catalog
#             rule-reference.md       # rule field catalog
#             workflow-reference.md   # workflow field catalog
#             mcp-reference.md        # MCP field catalog
#             hooks-reference.md      # hooks field catalog
#             memory-reference.md     # memory field catalog
#             cli-cheatsheet.md       # THIS FILE — CLI command reference
#             authoring-guide.md      # xcaf manifest authoring guide
#             operating-guide.md      # xcaffold CLI operating guide
#       rules/
#         xcaf-conventions/
#           xcaf-conventions.xcaf       # xcaffold authoring conventions rule
#       policies/
#         require-agent-description.xcaf
#         require-agent-instructions.xcaf
#       settings.xcaf                  # kind: settings (MCP, permissions, hooks)
