# Layer Precedence

When compiling resources, xcaffold resolves configuration through a
layered hierarchy. Each layer narrows or overrides the previous.

## The Resolution Hierarchy

From lowest to highest priority:

1. **Global config** (`~/.xcaffold/global.xcf`) — project-wide defaults
   inherited by all resources via `extends: global`.
2. **Resource definition** — the base `.xcf` file for the resource
   (`xcf/agents/<id>.xcf`, `xcf/skills/<id>.xcf`, etc.).
3. **Blueprint targets** — when `--blueprint` is active and the named
   blueprint declares a `targets:` list, that list controls which
   providers are compiled.
4. **Project targets** (`project.xcf`) — the project-wide compilation
   target list under `project.targets`.
5. **Override files** (`<resource>.<provider>.xcf`) — provider-specific
   field values that replace or clear base values for one provider only.
6. **`--target` flag** — CLI imperative override; highest priority.

## Target Resolution

`xcaffold apply` determines which providers to compile for with this
four-tier precedence (first match wins):

```
--target flag (if set by the caller)
  └─ blueprint.targets (if --blueprint is active and blueprint has targets)
      └─ project.targets (from project.xcf)
          └─ error: "no compilation targets configured"
```

Blueprints with their own `targets:` list compile for exactly those
providers — they do **not** fall through to project targets. A blueprint
without `targets:` falls through normally.

When no `--target` flag is set and no targets are configured anywhere,
`xcaffold apply` exits with an error rather than defaulting to any
provider.

## Override Merge Rules

When an override file exists for a resource (e.g., `developer.cursor.xcf`
for a `developer` agent), the compiler merges it with the base resource
before passing to the renderer:

| Field Type | Merge Behavior |
|------------|---------------|
| Scalar (`name`, `model`, `description`) | Override value replaces base |
| Boolean (`readonly`, `background`) | Override value replaces base |
| List (`tools`, `skills`, `rules`, `allowed-tools`) | See ClearableList semantics below |
| Map (`mcp-servers`, `hooks`) | Deep merge — override keys win on conflict |

**ClearableList merge rules** (list fields only):

| Override Value | Result |
|---------------|--------|
| Absent in override | Inherit base value |
| `[]` or `~` | Clear — base value is removed |
| `[a, b]` | Replace — override values used, base discarded |

See [field-model.md](./field-model.md) for the full `ClearableList`
specification.

## Examples

### Single-Provider Project

```yaml
# .xcaffold/project.xcf
kind: project
version: "1.0"
name: my-project
targets: [claude]
```

All resources compile for Claude only. Running `xcaffold apply` without
`--target` uses this list.

### Multi-Provider Project

```yaml
# .xcaffold/project.xcf
kind: project
version: "1.0"
name: my-project
targets: [claude, gemini]
```

Each resource compiles twice — once per target. Override files
(`agent.gemini.xcf`) customize fields per provider without duplicating
the base definition.

### Blueprint with Its Own Targets

```yaml
# .xcaffold/project.xcf
kind: project
version: "1.0"
name: my-project
targets: [claude]

blueprints:
  mobile:
    name: mobile
    targets: [cursor, copilot]
    agents: [mobile-dev]
```

Running `xcaffold apply --blueprint mobile` compiles `mobile-dev` for
`cursor` and `copilot` only, regardless of the project-level `targets:
[claude]`. The blueprint targets take precedence.

### Provider-Specific Override

```yaml
# xcf/agents/reviewer.xcf
kind: agent
version: "1.0"
name: reviewer
model: sonnet
tools: [Read, Grep, Glob]
```

```yaml
# xcf/agents/reviewer.gemini.xcf
kind: agent
version: "1.0"
name: reviewer
tools: []   # cleared — gemini reviewer gets no tools list
```

When compiling for `gemini`, the `tools` field is cleared. When compiling
for `claude`, the base `[Read, Grep, Glob]` is used.
