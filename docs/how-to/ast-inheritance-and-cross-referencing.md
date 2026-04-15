# Inheriting Configuration with `extends:` and Linking with `references:`

`extends:` lets a project `.xcf` file inherit agents, skills, rules, MCP servers, workflows, and hooks from a base config. The child selectively overrides individual resources; everything else is carried forward from the base. `references:` lets a skill declare supplementary files that are copied verbatim into the skill's output directory at compile time.

> **Note:** Configuration inheritance is **strictly an Xcaffold parser-layer construct**. Target providers (like Claude Code and Cursor) do not natively support cross-project inheritance loops or `extends:` directives. Xcaffold resolves the inheritance graph securely during compilation and physically emits the inherited resources into the local project so the providers can read them seamlessly.

Both mechanisms are target-agnostic: the same inheritance chain produces correct output for `--target claude`, `--target cursor`, `--target antigravity`, and `--target agentsmd`.

---

## When to use `extends:`

- Shared organization-wide rules or agents that every project inherits without duplication.
- A "base" agent template that individual projects customize by overriding a single field.
- A global hook library that all projects adopt additively.

---

## The `extends:` field

`extends:` is a `string` — single inheritance only. Arrays are not supported.

```yaml
version: "1.0"
project:
  name: my-service

extends: ../shared/base.xcf
```

The value is resolved relative to the directory that contains the `.xcf` file (its `baseDir`), not the working directory at the time `xcaffold` is invoked.

An absolute path is also accepted:

```yaml
extends: /opt/corp/standards/base.xcf
```

### The `global` alias

The string `"global"` is a reserved alias that resolves to `~/.xcaffold/global.xcf`:

```yaml
extends: global
```

This is equivalent to specifying the full path. The alias is only recognised during xcaffold's parse phase — the underlying provider (Claude Code, Cursor) receives the physically merged output, not the alias string.

`extends: global` does not affect compiled output. xcaffold strips global resources from the project's compiled output directory (e.g., `.claude/`) because they are already compiled separately via `xcaffold apply -g`. The alias is for visualization (`xcaffold graph`) and cross-reference validation only.

---

## Circular dependency detection

The parser detects cycles and fails immediately. If `project.xcf` extends `base.xcf` and `base.xcf` extends `project.xcf`, xcaffold fails with:

```
circular dependency detected: project.xcf → base.xcf → project.xcf
```

The cycle chain is printed in dependency order. xcaffold refuses to compile until the cycle is broken.

---


## Skill resources with `references:`, `scripts:`, and `assets:`

`references:`, `scripts:`, and `assets:` are optional `[]string` array fields on `SkillConfig` that align with the Agent Skills open standard. You are not required to provide them; they simply allow skills to bundle supplementary files for the agent's context when needed:

- **`references:`**: Static documents, guidelines, or code snippets the agent should read (e.g., `api-spec.yaml`, `style-guide.md`).
- **`scripts:`**: Executable tooling or helper scripts the agent can run (e.g., `validate.sh`, `generate-client.py`).
- **`assets:`**: Supporting media, models, or binary artifacts the skill depends on.

Each entry is a path (or glob pattern) pointing to a supplementary file. The compiler copies each matched file into its corresponding directory (`references/`, `scripts/`, or `assets/`) inside the skill's output directory.

```yaml
skills:
  db-schema:
    name: DB Schema Helper
    description: Assists with Drizzle schema authoring
    instructions: "Use the reference schemas below as examples."
    references:
      - docs/schema/users.sql
      - docs/schema/projects.sql
      - docs/examples/*.md
    scripts:
      - scripts/db-helper.sh
    assets:
      - assets/schema-diagram.png
```

Glob patterns are expanded at compile time. If a non-glob path matches no file, compilation fails with an error.

### Output location

For the `claude` target (and all other targets following the open standard), the above compiles to:

```
.claude/skills/db-schema/SKILL.md
.claude/skills/db-schema/references/users.sql
.claude/skills/db-schema/references/projects.sql
.claude/skills/db-schema/references/<each matched .md file>
.claude/skills/db-schema/scripts/db-helper.sh
.claude/skills/db-schema/assets/schema-diagram.png
```

> **Path safety.** Any file path that resolves to start with `..` is rejected immediately:
>
> ```
> scripts path "../../etc/passwd" traverses above the project root
> ```
>
> Paths are resolved relative to `baseDir` (the directory of the `.xcf` file), not the current working directory. This applies equally to `instructions-file:` paths across all resource types.

---

## Path safety callout

Both `instructions-file:` and `references:` enforce the following constraints at parse and compile time:

- Absolute paths are rejected for `instructions-file:` — only relative paths are accepted.
- Any path containing `..` is rejected.
- Paths that resolve into compiler output directories (`.claude/`, `.cursor/`, `.agents/`, `.antigravity/`) are rejected to prevent circular read-write dependencies.
- All output paths pass through `filepath.Clean` before being written.

---

## Dual-target output comparison

The same `.xcf` source — including its inherited rules — produces target-specific output for each renderer.

**Source (after inheritance resolution):**

```yaml
rules:
  linting:
    description: Enforce project lint standards
    instructions: "Run golangci-lint --fix before every commit."
    paths: ["**/*.go"]
```

**`--target claude` output: `.claude/rules/linting.md`**

```markdown
---
description: Enforce project lint standards
paths: [**/*.go]
---

Run golangci-lint --fix before every commit.
```

**`--target cursor` output: `.cursor/rules/linting.mdc`**

```markdown
---
description: Enforce project lint standards
globs: [**/*.go]
---

Run golangci-lint --fix before every commit.
```

The key normalization: the cursor renderer translates `paths:` to `globs:` in frontmatter. A rule with no `paths:` receives `always-apply: true` instead.

The `antigravity` and `agentsmd` targets follow the same source; their renderers apply their own target-specific normalizations.

---

## Verify the merged topology with `xcaffold graph --full`

After applying `extends:`, use `xcaffold graph --full` to inspect the fully-merged configuration before compiling. The `--full` flag renders the expanded topology tree including all inherited resources:

```
xcaffold graph --full
xcaffold graph --full --format mermaid > topology.md
xcaffold graph --full --format json | jq .
```

Without `--full`, `xcaffold graph` renders a summary. With `--full` (or when `--agent` targets a specific agent), the complete inheritance-resolved tree is printed — agents, skills, rules, MCP servers, and hooks as they will actually be compiled.
