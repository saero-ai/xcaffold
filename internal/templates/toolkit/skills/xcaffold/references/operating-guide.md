# xcaffold CLI Operating Guide

## Starting a new project

```bash
# Interactive — prompts for providers, generates Xaff authoring toolkit
xcaffold init

# Non-interactive — explicit targets, machine-readable manifest
xcaffold init --target claude,cursor --yes --json
```

The `--json` flag emits a machine-readable manifest of all generated files.

Generated layout:

```
my-project/
  .xcaffold/
    project.xcaf              # kind: project (targets, resource refs)
  xcaf/
    agents/
      xaff/
        agent.xcaf            # base Xaff agent (universal)
        agent.claude.xcaf     # per-provider override
    skills/
      xcaffold/
        xcaffold.xcaf         # THIS SKILL
        references/
          agent-reference.md        # agent field catalog
          skill-reference.md        # skill field catalog
          rule-reference.md         # rule field catalog
          workflow-reference.md     # workflow field catalog
          mcp-reference.md          # MCP field catalog
          hooks-reference.md        # hooks field catalog
          memory-reference.md       # memory field catalog
          cli-cheatsheet.md         # CLI command reference
          authoring-guide.md        # xcaf manifest authoring guide
          operating-guide.md        # xcaffold CLI operating guide
    rules/
      xcaf-conventions/
        xcaf-conventions.xcaf  # xcaffold authoring conventions
    policies/
      require-agent-description.xcaf
      require-agent-instructions.xcaf
    settings.xcaf             # kind: settings (MCP, permissions)
```

## CLI reference

See the cli-cheatsheet reference (installed alongside this guide) for the complete command and flag reference, including all flags for `init`, `apply`, `validate`, `status`, `import`, `graph`, `list`, `export`, and `test`.

## Checking compilation state

```bash
xcaffold status              # show drift across all providers
xcaffold status --target claude  # focus on one provider
```

## Importing existing provider config

```bash
xcaffold import --plan       # preview what will be imported
xcaffold import              # write to xcaf/
xcaffold validate            # verify the imported config
xcaffold apply --dry-run     # preview compiled output
```

## Applying and validating

```bash
xcaffold validate            # schema + policy validation (no file writes)
xcaffold apply --target claude  # compile xcaf/ to .claude/ output
xcaffold apply --dry-run     # preview output without writing
```

## Listing and exploring

```bash
xcaffold list                # enumerate declared resources
xcaffold list --verbose      # show memory entries per agent
```

## Visualizing dependencies

```bash
xcaffold graph               # visualize agent/skill/rule topology
xcaffold graph --agent reviewer  # focus on one agent
```
