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
    project.xcf              # kind: project (targets, resource refs)
    schemas/
      agent.xcf.reference    # annotated agent field catalog
      skill.xcf.reference    # annotated skill field catalog
      rule.xcf.reference     # annotated rule field catalog
      workflow.xcf.reference # annotated workflow field catalog
      mcp.xcf.reference      # annotated MCP field catalog
      hooks.xcf.reference    # annotated hooks field catalog
      memory.xcf.reference   # annotated memory field catalog
      cli-cheatsheet.reference  # CLI command reference
  xcf/
    agents/
      xaff/
        agent.xcf            # base Xaff agent (universal)
        agent.claude.xcf     # per-provider override
    skills/
      xcaffold/
        xcaffold.xcf         # THIS SKILL
    rules/
      xcf-conventions/
        xcf-conventions.xcf  # xcaffold authoring conventions
    policies/
      require-agent-description.xcf
      require-agent-instructions.xcf
    settings.xcf             # kind: settings (MCP, permissions)
```

## Checking compilation state

```bash
xcaffold status              # show drift across all providers
xcaffold status --target claude  # focus on one provider
```

## Importing existing provider config

```bash
xcaffold import --plan       # preview what will be imported
xcaffold import              # write to xcf/
xcaffold validate            # verify the imported config
xcaffold apply --dry-run     # preview compiled output
```

## Applying and validating

```bash
xcaffold validate            # schema + policy validation (no file writes)
xcaffold apply --target claude  # compile xcf/ to .claude/ output
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
