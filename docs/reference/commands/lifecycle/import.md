---
title: "xcaffold import"
description: "Reverse engineer existing provider configs and migrate them into Xcaffold ASTs."
---

# xcaffold import

Manages adopting existing, fragmented AI configurations into a centralized Xcaffold `.xcf` specification.

The `import` command reads pre-existing workspaces (e.g. `.claude/`, `.cursor/rules/`, `.github/`) and reverse engineers them. It attempts to classify undocumented markdown files by utilizing generative semantic analysis and translates inferred intents into structured Xcaffold primitives (like `skills`, `rules`, `agents`, `workflows`).

## Usage

```bash
xcaffold import [flags]
```

## Options

| Flag | Default | Description |
|---|---|---|
| `--source <string>` | `""` | File or directory of workflow markdown files to translate. Triggers cross-platform translation mode. |
| `--from <string>` | `"auto"` | Specify the source platform of input files (e.g., `antigravity`, `claude`, `cursor`, `gemini`, `copilot`). |
| `--with-memory` | `false` | Take a snapshot of dynamically agent-written memory tracks and attach them as `xcf/agents/<id>/memory/` sidecars. |
| `--auto-merge <string>`| `""` | Automatically merge divergent resource variants during decomposition (value: `union`). |
| `--plan` | `false` | Run a dry-run translation without executing file generation, printing the decomposition plan to standard output. |

## Behavior

### Native Import (Default)
When run without `--source`, `import` assumes it is operating over a raw project directory and scans for supported `.claude/`, `.cursor/`, and `.gemini/` legacy directories. It extracts these configurations, maps their properties natively (such as `mcp-servers`), generates standard frontmatter structures inside the `.xcf` specification, and points content toward newly organized isolated directories (`xcf/rules`, `xcf/skills`, etc.).

### Cross-Platform Translation (--source)
When pointing to a file or directory via `--source`, `import` pivots to its _Translation Mode_. Let's say you have an `antigravity` workflow defined within `./workflows/`. Providing this path alongside `--from antigravity` directs the internal translation engine to deeply analyze the markdown content, classify its functional intents, and map equivalent constraints into discrete primitives.

As you migrate workflows into the robust constraints of Xcaffold `project.xcf` definitions, we heavily advise utilizing the `--plan` flag to audit translation predictions before disk persistence.

### Multi-Directory Import
When multiple provider directories are detected in the project root (e.g., `.claude/`, `.cursor/`, `.gemini/`), `import` automatically imports from each directory and merges the results. All resource types are supported: agents, skills, rules, workflows, memory, hooks, MCP servers, settings, and project instructions.

Merge behavior is deterministic:
- **Agents, Skills, Rules, Workflows**: When the same resource is found in multiple provider directories, the variant with richer instructions is retained. If instruction detail is equivalent, the first-seen variant is kept.
- **Memory**: Union merge across all provider-specific memory directories (e.g., `.claude/agent-memory/`, `.cursor/agent-memory/`). Within a single agent's memory, the first document for a given key wins on collision.
- **Hooks**: Per-event merge. Hook definitions from all providers are combined; events are not overwritten.
- **MCP Servers, Settings, Project Instructions**: Merged across all provider configurations. Provider-specific variants are preserved in `target-options` where applicable.

Agent memory from provider-specific directories (e.g., `.claude/agent-memory/`, `.cursor/agent-memory/`) is automatically included in multi-directory imports and integrated into the corresponding `xcf/agents/<id>/memory/` structure.

## Examples

**Automatically snapshot and import native .claude structures alongside contextual memory:**
```bash
xcaffold import --with-memory
```

**Dry-run the translation of existing Cursor rules:**
```bash
xcaffold import --source .cursor/rules/ --from cursor --plan
```

**Translate a broad set of workflow architectures from Gemini:**
```bash
xcaffold import --source .gemini/ --from gemini
```
