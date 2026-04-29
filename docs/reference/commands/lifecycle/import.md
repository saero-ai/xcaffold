---
title: "xcaffold import"
description: "Import existing provider configurations into Xcaffold project structure."
---

# xcaffold import

Adopts existing fragmented AI configurations from provider directories (`.claude/`, `.cursor/`, `.gemini/`, etc.) into a centralized Xcaffold `.xcf` project specification.

The `import` command:
1. **Detects** available provider directories in the project root
2. **Imports** resources from each provider using the `ProviderImporter` interface
3. **Filters** resources by kind (agents, skills, rules, etc.) via `--<kind>` flags
4. **Merges** multi-provider resources into base files + provider-specific override files
5. **Writes** the result as `project.xcf` + directory-per-resource layout in `xcf/`

## Usage

```bash
xcaffold import [flags]
```

## Options

| Flag | Type | Default | Description |
|---|---|---|---|
| `--target` | string | (auto-detect) | Import from a specific provider: `claude`, `gemini`, `cursor`, `antigravity`, `copilot`. Without this flag, auto-detects all available providers. |
| `--agent` | string | (all) | Import agents. Optionally filter by name. |
| `--skill` | string | (all) | Import skills. Optionally filter by name. |
| `--rule` | string | (all) | Import rules. Optionally filter by name. |
| `--workflow` | string | (all) | Import workflows. Optionally filter by name. |
| `--mcp` | string | (all) | Import MCP server definitions. Optionally filter by name. |
| `--hooks` | bool | false | Import hook definitions. |
| `--settings` | bool | false | Import settings configuration. |
| `--memory` | bool | false | Import agent-written memory snapshots to `xcf/agents/<id>/memory/` sidecars. |
| `--plan` | bool | false | Dry-run: print import plan without writing files. |

## Behavior

### Single-Provider Import (with --target)

When `--target` is specified:
- Scans only the target provider's directory (e.g., `.claude/` for `--target claude`)
- Imports all matching resources (filtered by `--<kind>` flags if provided)
- Tags all resources with `targets: [<provider>]` to indicate their source
- Writes `project.xcf` + directory-per-resource layout

**Example:** `xcaffold import --target claude --agent` imports all agents from `.claude/` and tags them with `targets: [claude]`.

### Multi-Provider Import (without --target)

When `--target` is not specified:
- Auto-detects all available provider directories
- Imports matching resources from each provider
- Applies **smart assembly** to merge multi-provider variants:
  - **Agents, Skills, Rules, Workflows**: Field-by-field comparison across providers
    - Identical resources (same name, frontmatter, body) from multiple providers are merged into a single base file
    - Divergent fields (e.g., same agent with different instructions) are extracted into provider-specific override files
    - Override files use `<kind>.<provider>.xcf` naming (e.g., `agent.claude.xcf`, `agent.cursor.xcf`)
  - **Memory**: Union merge across all provider-specific memory directories. Within a single agent's memory, first-seen document wins on key collision.
  - **Hooks, MCP, Settings**: All variants merged; provider-specific differences preserved in `target-options` where applicable

**Example:** `xcaffold import` (no flags) detects `.claude/`, `.cursor/`, and `.gemini/` directories, imports all resources from each, and produces:
- `xcf/agents/researcher/agent.xcf` (base, common to multiple providers)
- `xcf/agents/researcher/agent.claude.xcf` (Claude-specific overrides)
- `xcf/agents/researcher/agent.cursor.xcf` (Cursor-specific overrides)

### Resource Filtering

Per-kind flags (`--agent`, `--skill`, etc.) control which resource types are imported. Without any flags, all types are imported.

- `--agent` imports all agents
- `--agent "dev*"` imports agents matching the pattern
- `--rule --skill` imports only rules and skills (omitting agents, workflows, etc.)

## Directory Layout

After import, the project structure is organized as directory-per-resource with canonical filenames:

```
xcf/
├── agents/
│   └── researcher/
│       ├── agent.xcf              # base definition
│       ├── agent.claude.xcf       # Claude-specific overrides
│       ├── agent.cursor.xcf       # Cursor-specific overrides
│       └── memory/                # agent memory sidecars
├── skills/
│   └── code-review/
│       ├── skill.xcf
│       └── skill.claude.xcf
├── rules/
│   └── security/
│       └── rule.xcf
├── workflows/
│   └── ci-pipeline/
│       └── workflow.xcf
├── hooks/
│   └── hooks.xcf
├── mcp/
│   └── github-mcp/
│       ├── mcp.xcf
│       └── mcp.claude.xcf
└── settings/
    └── settings.xcf
```

## Examples

**Import from all detected providers:**
```bash
xcaffold import
```

**Import only from Gemini:**
```bash
xcaffold import --target gemini
```

**Import only agents from Claude:**
```bash
xcaffold import --target claude --agent
```

**Import a specific agent from Claude:**
```bash
xcaffold import --target claude --agent developer
```

**Dry-run preview (no files written):**
```bash
xcaffold import --plan
```

## Output

After a successful import, `xcaffold import` prints:
- Resource summary (count by kind)
- **Targets tagging** for single-provider imports: "Resources tagged with `targets: [claude]`. Remove the `targets` field to make universal."
- **Conflict reporting** for multi-provider imports: "N conflicts detected. Run `xcaffold validate` to review."
- Next steps: "Run `xcaffold apply` when ready to assume management."

## Merge Semantics

When multi-provider resources conflict:

| Scenario | Resolution |
|---|---|
| Same resource name, identical content across providers | Single base file, no overrides needed |
| Same resource name, divergent frontmatter fields | Base file + provider-specific override files |
| Same resource name, divergent body text | Base file with common content + provider-specific override files |
| Resource exists in only one provider | Single file tagged with that provider in `targets` |

## Limitations

- Multi-document `.xcf` files (multiple resources per file) are not supported during import; resources are split into directory-per-resource layout
- Imported resources must have a valid `name` (inferred from filename or declared in YAML); unnamed resources are skipped with a warning
- Circular memory references across agents are flattened; direct circular imports are not supported

## Next Steps

After import completes:
1. Review the generated `project.xcf` and override files
2. Run `xcaffold validate` to verify the project structure
3. Run `xcaffold apply` to compile to target provider directories

