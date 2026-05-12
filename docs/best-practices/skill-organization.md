---
title: "Skill Organization"
description: "How to structure skill resources with the artifacts field and auto-discovery."
---

# Skill Organization

Skills are reusable bundles of instructions and supporting files. They are organized into directories under `xcaf/skills/`, using the `artifacts` field to enable automatic discovery of supporting content.

## Directory Layout

A skill is defined by a directory containing a manifest and one or more artifact subdirectories.

```
xcaf/skills/<skill-name>/
├── skill.xcaf                # Main definition (frontmatter + body)
├── references/               # Artifact: Background knowledge
├── scripts/                  # Artifact: Executable tools
├── assets/                   # Artifact: Production templates/schemas
└── examples/                 # Artifact: Demonstration outputs
```

### The Manifest File

The manifest can be named either `skill.xcaf` or `<skill-name>.xcaf`. Both are equally supported.

```yaml
---
kind: skill
version: "1.0"
name: git-workflow
description: "Standardized git operations and commit conventions."
artifacts:
  - references
  - scripts
---
# Instructions
Always use the `scripts/commit.sh` tool for commits...
```

### Artifact Discovery

The `artifacts` list tells xcaffold which subdirectories to include. **You do not need to list individual files.** The compiler automatically walks every declared directory and includes all files found within it.

*   **Natural Extensions:** Supporting files keep their native extensions (`.md`, `.sh`, `.json`, `.py`).
*   **Max Depth 1:** Artifact directories must be flat. Nested subdirectories (e.g., `references/docs/intro.md`) are not allowed and will fail validation.
*   **Auto-Detection:** Files in declared artifact directories are automatically available to the agent using their relative paths (e.g., `scripts/deploy.py`).

## Semantic Conventions

While you can name your artifact directories anything (as long as they are declared in the `artifacts` list), using the four recommended names enables provider-specific optimizations:

| Directory | Semantic Role | Provider Handling |
|---|---|---|
| `references/` | **INFORM** — knowledge | Standard context load. |
| `scripts/` | **DO** — execution | Handled as executable tools. |
| `assets/` | **BECOME** — production | Templates and schemas. |
| `examples/` | **DEMONSTRATE** — output | Some providers (Claude) flatten these to the skill root for better learning; others (Cursor/Gemini) collapse them into `references/`. |

## Validator Enforcement

The `xcaffold validate` command enforces the following structural rules:

1.  **Declared Artifacts:** Any subdirectory on disk that is *not* in the `artifacts` list will trigger a warning, as it will be ignored during compilation.
2.  **Missing Artifacts:** Any folder listed in `artifacts` that does not exist on disk is a hard error.
3.  **Stray Files:** Non-manifest files (like `notes.txt` or `README.md`) at the skill root are not allowed. They must be moved into an artifact subdirectory.
4.  **Flat Manifests:** Manifest files placed directly in `xcaf/skills/` without a parent directory are flagged. Skills must always live in a dedicated subdirectory.
