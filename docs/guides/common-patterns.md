---
title: "Common configuration patterns"
description: "Copy-pasteable .xcaf recipes for multi-provider projects, shared rules, workspace context, and monorepos."
---

# Common configuration patterns

Recipes for problems that come up often when adopting Harness-as-Code with xcaffold. Each pattern includes a short problem statement, a minimal `.xcaf` layout, and what you should see after `xcaffold apply`.

For field-level detail, see the [resource kind reference](../reference/kinds/index.md). For deeper treatment of multi-target compilation, see [multi-target compilation](../best-practices/multi-target-compilation.md).

## Pattern 1: Multi-provider setup (Claude + Cursor from one source)

### Problem

Your team uses both Claude Code and Cursor on the same repository. You want one set of `.xcaf` manifests — not two hand-maintained trees under `.claude/` and `.cursor/`.

### `.xcaf` snippet

Declare every provider you compile in `project.xcaf`, then write agents and skills once:

```yaml
# project.xcaf
kind: project
version: "1.0"
name: platform
targets:
  - claude
  - cursor
```

```yaml
# xcaf/agents/developer/agent.xcaf
kind: agent
version: "1.0"
name: developer
description: "Implements features and runs tests."
model: sonnet
tools: [Read, Write, Edit, Bash, Glob, Grep]
skills: [tdd]
---
Implement the requested change, read existing code first, and stop when tests pass.
```

```yaml
# xcaf/skills/tdd/skill.xcaf
kind: skill
version: "1.0"
name: tdd
description: "Red-green-refactor workflow."
allowed-tools: [Bash, Read, Write, Edit]
---
Write a failing test first, implement the minimum code to pass, then refactor.
```

Apply all targets in one command:

```bash
xcaffold apply
```

To compile a single provider during development:

```bash
xcaffold apply --target claude
```

### Expected compiled output

After `xcaffold apply`, xcaffold writes provider-native trees from the same sources:

| Target | Output directory | Example artifacts |
|--------|------------------|-------------------|
| `claude` | `.claude/` | `agents/developer.md`, skill folders under `.claude/skills/` |
| `cursor` | `.cursor/` | Cursor-native agent and rule files mapped from the same manifests |

Fields a provider does not support are dropped; run `xcaffold validate --target <provider>` or `xcaffold apply --verbose` to see fidelity notes on stderr. Warnings do not block compilation.

---

## Pattern 2: Shared rules referenced by multiple agents

### Problem

Several agents must follow the same commit format and security rules. You want one rule definition, referenced by name, instead of duplicating prose in every agent body.

### `.xcaf` snippet

Define each rule once under `xcaf/rules/<name>/rule.xcaf`:

```yaml
# xcaf/rules/commit-format/rule.xcaf
kind: rule
version: "1.0"
name: commit-format
description: "Conventional Commits for all agents."
activation: always
---
All commit messages use `type(scope): description` with types feat, fix, docs, test, or chore.
```

```yaml
# xcaf/rules/secure-coding/rule.xcaf
kind: rule
version: "1.0"
name: secure-coding
description: "Baseline secure coding constraints."
activation: always
---
Never log secrets or tokens. Validate all external input before use.
```

Reference the same rule IDs from multiple agents:

```yaml
# xcaf/agents/developer/agent.xcaf
kind: agent
version: "1.0"
name: developer
description: "Feature implementation."
model: sonnet
tools: [Read, Write, Edit, Bash, Glob, Grep]
rules: [commit-format, secure-coding]
---
Implement features and run tests before finishing.
```

```yaml
# xcaf/agents/reviewer/agent.xcaf
kind: agent
version: "1.0"
name: reviewer
description: "Read-only code review."
model: sonnet
tools: [Read, Glob, Grep]
rules: [commit-format, secure-coding]
readonly: true
---
Review diffs for correctness and test coverage. Do not modify files.
```

### Expected compiled output

For each target, xcaffold injects the shared rules into every agent that lists them:

- **Claude** — rules appear under `.claude/rules/` (or as provider-native rule entries linked to agents, depending on activation mode).
- **Cursor** — the same rule bodies compile into Cursor rule files; both agents receive `commit-format` and `secure-coding` without duplicating YAML.

Edit a rule once, run `xcaffold apply`, and all referencing agents pick up the change on the next compile.

---

## Pattern 3: Workspace context plus focused agents

### Problem

Some instructions apply to every agent in every session (repo layout, package manager). Others belong only on specialized agents. You want a thin workspace context and richer per-agent bodies.

### `.xcaf` snippet

Put cross-cutting invariants in `kind: context` (not in `kind: project` — project metadata has no markdown body):

```yaml
# xcaf/context/root/context.xcaf
kind: context
version: "1.0"
name: root
targets:
  - claude
  - cursor
---
## Repository invariants

- Application code lives under `apps/` and `packages/`.
- Use `pnpm` only; do not run `npm install` or `yarn`.
- Never commit directly to `main`.
```

Compose agents that add role-specific instructions and reference skills:

```yaml
# xcaf/agents/platform-dev/agent.xcaf
kind: agent
version: "1.0"
name: platform-dev
description: "Works on the Next.js platform app."
model: sonnet
tools: [Read, Write, Edit, Bash, Glob, Grep]
skills: [frontend-review]
rules: [frontend-css]
---
You work in `apps/platform/`. Read existing components before adding new ones.
```

```yaml
# xcaf/skills/frontend-review/skill.xcaf
kind: skill
version: "1.0"
name: frontend-review
description: "Checklist for UI changes."
allowed-tools: [Read, Glob, Grep]
---
Verify loading states, accessibility labels, and Storybook stories for new components.
```

Optional path-scoped rule (loaded only when matching files are active):

```yaml
# xcaf/rules/frontend-css/rule.xcaf
kind: rule
version: "1.0"
name: frontend-css
description: "Tailwind conventions for UI code."
activation: path-glob
paths:
  - "apps/platform/**/*.tsx"
  - "packages/ui/**/*.tsx"
---
Use Tailwind utility classes. Do not add inline styles or new CSS modules.
```

### Expected compiled output

| Resource | Compiles to (Claude example) |
|----------|------------------------------|
| `kind: context` `root` | `CLAUDE.md` at project root (workspace-wide prose) |
| `kind: agent` `platform-dev` | `.claude/agents/platform-dev.md` (role-specific system prompt) |
| `kind: skill` `frontend-review` | `.claude/skills/frontend-review/` |
| `kind: rule` `frontend-css` | `.claude/rules/frontend-css.md` (path-scoped activation) |

Every agent session loads `CLAUDE.md` automatically; `platform-dev` adds its own prompt; `frontend-css` applies only when working under the configured globs.

---

## Pattern 4: Monorepo package with `--config`

### Problem

A monorepo contains multiple apps, each with its own harness. The xcaffold project for the API service lives at `services/api/`, but developers run commands from the repo root or from sibling packages.

### `.xcaf` snippet

Place the project manifest inside the service directory:

```
monorepo/
  project.xcaf              # optional root meta-project (omit if not needed)
  services/
    api/
      project.xcaf          # kind: project for the API harness
      xcaf/
        agents/
          api-dev/
            agent.xcaf
        skills/
          ...
  apps/
    web/
      project.xcaf          # separate harness for the web app
      xcaf/
        ...
```

```yaml
# services/api/project.xcaf
kind: project
version: "1.0"
name: api-service
targets:
  - claude
```

```yaml
# services/api/xcaf/agents/api-dev/agent.xcaf
kind: agent
version: "1.0"
name: api-dev
description: "Implements HTTP handlers in the API service."
model: sonnet
tools: [Read, Write, Edit, Bash, Glob, Grep]
---
Work only under `services/api/`. Do not modify `apps/web/` or other packages.
```

Run xcaffold against that package from anywhere using `--config`:

```bash
# From repository root
xcaffold apply --config services/api/project.xcaf

# Or from the package directory (walk finds project.xcaf automatically)
cd services/api && xcaffold apply
```

Other commands accept the same flag:

```bash
xcaffold validate --config services/api/project.xcaf
xcaffold list --config services/api/project.xcaf
xcaffold status --config services/api/project.xcaf
```

### Expected compiled output

With `--config services/api/project.xcaf`, compilation uses `services/api/` as the project root:

- State and drift tracking write under `services/api/.xcaffold/`.
- Claude output lands in `services/api/.claude/` (not at the monorepo root).
- `xcaffold list` shows only resources discovered under `services/api/xcaf/`.

Repeat with `apps/web/project.xcaf` for a separate web harness without merging unrelated agents into one project file.

---

## Next steps

- [`xcaffold init`](../reference/commands/lifecycle/init.md) — scaffold a starter project to try these patterns locally.
- [`xcaffold validate`](../reference/commands/lifecycle/validate.md) — catch schema and cross-reference errors before apply.
- [Agent design patterns](../best-practices/agent-design-patterns.md) — composition, tool scoping, and provider overrides in depth.
