---
title: "Workspace Context"
description: "Best practices for using kind: context to anchor agent awareness with project-level workspace instructions."
---

# Workspace Context

In xcaffold, workspace-level instructions — the root prompt read by all AI providers when they open your project — are declared using `kind: context` documents. These compile to the following provider-specific files at your project root:

| Provider | Output file | Location |
|---|---|---|
| Claude Code | `CLAUDE.md` | project root |
| Cursor | `AGENTS.md` | project root |
| Gemini CLI | `GEMINI.md` | project root |
| Antigravity (deprecated) | `GEMINI.md` | project root |
| Antigravity 2 | `GEMINI.md` | project root |
| GitHub Copilot | `copilot-instructions.md` | `.github/` |

> [!NOTE]
> Both Antigravity v1 and Antigravity 2 read `GEMINI.md` from the project root for project context — they do not have their own separate context files. The `gemini`, `antigravity`, and `antigravity2` targets all write to the same `GEMINI.md`. If targeting more than one of these, be aware that the latter target compiled will overwrite the file.

Because these files are loaded on **every interaction** (check with the provider for any changes), their token cost is unconditional — paid regardless of which agent is active or what task is being performed. Keep them short and focused on absolute invariants.

## Declaring Workspace Context

Create a `kind: context` file under `xcaf/`. The markdown body becomes the compiled workspace instruction:

```
---
kind: context
version: "1.0"
name: root
targets:
  - claude
  - cursor
  - gemini
  - antigravity2
  - copilot
---

## Repository Invariants

- All source lives in `src/`. Never write files to the project root.
- Package management: use `pnpm` exclusively. Running `npm install` or `yarn` is forbidden.
- Version control: all commits must follow Conventional Commits (`feat:`, `fix:`, `chore:`, etc.).
- Never push directly to `main`. Always open a pull request.

## Technology Stack

- Frontend: React 19 with Next.js 15 App Router only. No Pages Router.
- Backend: Node.js 22 with Hono. No Express.
- Database: PostgreSQL 16 with Drizzle ORM. Raw SQL queries are forbidden outside migration files.
```

The `targets:` field scopes which providers receive this context. Omitting `targets:` causes the context to compile for all declared project targets. A project may have multiple `kind: context` documents — one per provider, or one shared across several.


## Subdirectory Contexts

By default, context files compile to the project root: `CLAUDE.md`, `GEMINI.md`, etc. When your project is a monorepo or has distinct subsystems, you can scope a context to a subdirectory by setting the `path:` field:

```yaml
---
kind: context
version: "1.0"
name: backend-context
targets: [claude]
path: "backend/"
---

## Backend-Only Instructions

This workspace covers the API server and database layer.
Use the database-tools MCP server for schema exploration.
Never modify client-side code from the backend services.
```

When compiled with `path: "backend/"`, this context renders to `backend/CLAUDE.md` instead of the project root. This is useful for:

- **Monorepos:** Different subdirectories need different instructions (backend teams read `backend/CLAUDE.md`, frontend teams read `frontend/CLAUDE.md`).
- **Layered Services:** A multi-service architecture where each service has its own agent configuration needs.
- **Shared Infrastructure:** Separate context for infrastructure code, CI/CD, and shared libraries.

### Nested Context Example

For a monorepo with distinct service areas:

```
xcaf/
├── contexts/
│   ├── root.xcaf                  # project-wide (compiles to CLAUDE.md)
│   ├── backend-context.xcaf       # backend team (compiles to backend/CLAUDE.md)
│   └── frontend-context.xcaf      # frontend team (compiles to frontend/CLAUDE.md)
```

Each context compiles to its own location. The uniqueness constraint is per **(target, path) pair** — you can have two Claude contexts as long as they target different paths:

```yaml
---
kind: context
version: "1.0"
name: api-backend
targets: [claude]
path: "services/api/"
---
```

and

```yaml
---
kind: context
version: "1.0"
name: auth-backend
targets: [claude]
path: "services/auth/"
---
```

Both compile successfully because they target different paths.

### Provider Support for Nested Contexts

| Provider | Nested Path Support | Output Pattern |
|---|---|---|
| Claude Code | `✓` Yes | `{path}/CLAUDE.md` |
| Cursor | `✓` Yes | `{path}/AGENTS.md` |
| Gemini CLI | `✓` Yes | `{path}/GEMINI.md` |
| GitHub Copilot | `✓` Yes (via glob) | `.github/instructions/{name}.instructions.md` with `applyTo: {path}/**` glob |
| Antigravity v1 | ✗ No | Deprecated provider; consolidates at `GEMINI.md` |
| Antigravity 2 | `✓` Yes | `{path}/GEMINI.md` |
| Codex | ✗ No | Consolidates at `AGENTS.md` (path ignored) |

Gemini CLI and Antigravity targets consolidate all contexts at the project root regardless of the `path:` field — they do not support nested instructions.

### Variable Interpolation

The `path:` field supports variable interpolation, allowing configuration-driven subdirectory selection:

```yaml
---
kind: context
version: "1.0"
name: custom-service
targets: [claude]
path: "${var.service-dir}"
---
```

Pass the value via `--var-file`:

```bash
xcaf/project.vars.local:
service-dir = "services/my-service/"

# Then compile with:
xcaffold apply --var-file xcaf/project.vars.local
```

> [!IMPORTANT]
> The `kind: project` document does **not** support a markdown body. Project metadata (`name`, `targets`, `extends`) goes in `kind: project`. Workspace prose goes in `kind: context`. Adding a markdown body after the closing `---` in a project manifest will cause a parse error.


## Document Only Absolute Laws

Reserve workspace context for immutable ground rules that span all disciplines and all agents. Ask yourself: "Is this true for every file in the repository, every agent, and every task?" If not, it belongs somewhere narrower.

**Good candidates for workspace context:**
- Repository layout invariants (`"all source lives in src/"`)
- Build tooling and package managers (`"use pnpm, never npm"`)
- Branch and commit hygiene (`"Conventional Commits mandatory"`)
- Cross-cutting security constraints (`"never log credentials or tokens"`)

**Poor candidates for workspace context:**
- CSS conventions (frontend-only)
- Database migration patterns (backend-only)
- Testing framework usage (language-specific)
- Agent-specific behavioral guidance


## Isolate Tactical Rules to Their Scope

**Anti-pattern:** Embedding a CSS architecture guide in the root `kind: context`.

When your backend agent adjusts a Redis polling loop, the compiler still passes the Tailwind token guide to the LLM on every interaction — polluting the context window with irrelevant knowledge and wasting tokens.

**Best practice:** Move scope-specific guidance to a `kind: rule` with a `paths:` glob. A rule scoped to frontend paths is only injected when the agent's active file matches that pattern:

```
---
kind: rule
version: "1.0"
name: frontend-css-conventions
activation: path-glob
paths:
  - "src/ui/**/*.tsx"
  - "src/ui/**/*.css"
---

Use Tailwind utility classes exclusively. Never write inline styles or custom CSS modules.
When adding a new component, check `src/ui/design-tokens.ts` for available color and spacing values before inventing new ones.
```

Similarly, move agent-specific guidance directly into that agent's body, not into the workspace context.


## Write Instructions, Not Documentation

The most common mistake when authoring a context file is writing it like a README or a wiki page — describing what the project is, listing technologies, explaining architecture. That content may be accurate, but it does not change how an AI agent behaves.

**Workspace context is a behavioral contract, not a project description.** Anthropic, Google, and OpenAI all converge on the same principle for effective system prompts: every line must directly constrain or enable an action the model will take. If removing a line wouldn't change how the agent behaves, the line doesn't belong here.

### What makes an instruction effective

**Use imperative voice.** Tell the model what to do, not what the project does.

| ❌ Documentation (passive, descriptive) | ✅ Instruction (imperative, behavioral) |
|---|---|
| "This project uses pnpm for package management." | "Use `pnpm` for all package operations. Never run `npm install` or `yarn`." |
| "The codebase follows Conventional Commits." | "All commit messages must start with a type prefix: `feat:`, `fix:`, `chore:`, `docs:`, `refactor:`." |
| "We have a PostgreSQL database managed by Drizzle ORM." | "Query the database only through Drizzle ORM. Never write raw SQL outside of `packages/db/migrations/`." |
| "The frontend is built with React 19 and Next.js 15." | "Use the App Router exclusively. The Pages Router (`pages/`) is not used and must not be created." |

**Put critical constraints first.** Models exhibit a primacy effect — instructions near the top of a document carry more weight than those buried at the bottom. Put your highest-stakes rules (security, tooling, commit hygiene) before nice-to-haves.

**Be specific about the boundary.** Vague instructions produce inconsistent behavior. "Follow best practices" is not an instruction. "Never write inline styles — use Tailwind utility classes only" is.

**Include the consequence, not just the rule.** When a constraint has a clear reason, stating it reduces the chance of the model making an exception: "Never commit `.env` files. Environment variables are managed per-environment in the hosting platform, not in source control."

### The 200-line signal

A well-written context file is typically under 200 lines. If yours exceeds that, it's usually a signal that some content has drifted into documentation mode or into territory that belongs in a narrower kind. When auditing, ask of each section:

- **Is this true for every file in the repo?** If not → `kind: rule` with `paths:`
- **Is this specific to one agent's job?** If yes → move it to that agent's body
- **Is this a multi-step procedure?** If yes → `kind: skill` or `kind: workflow`
- **Is this describing the system rather than instructing the model?** If yes → remove it

### Split decision reference

| Content type | Where it belongs |
|---|---|
| Project-wide invariants (tooling, commit format, security) | `kind: context` |
| Path-specific coding conventions | `kind: rule` with `paths:` |
| Agent-specific behavioral guidance | That agent's `.xcaf` body |
| Multi-step procedures or workflows | `kind: skill` or `kind: workflow` |
| Background context that doesn't drive behavior | Remove it entirely |


## Use Multiple Context Documents for Multi-Provider Projects

If providers need different framing for the same information — for example, Claude has MCP tool access while Copilot operates only through the editor — use separate `kind: context` documents with scoped `targets:` rather than writing provider-specific conditionals in a single document:

```
---
kind: context
version: "1.0"
name: root-claude
targets:
  - claude
  - antigravity2
---

## Repository Invariants

- Source lives in `src/`. Use pnpm for package management.
- Conventional Commits are required on all commits.

## Tool Usage

Use the `mcp__xcaffold__apply` tool when asked to change agent configurations.
Use the `mcp__db__query` tool to inspect the database schema before writing migrations.
```

```
---
kind: context
version: "1.0"
name: root-copilot
targets:
  - copilot
---

## Repository Invariants

- Source lives in `src/`. Use pnpm for package management.
- Conventional Commits are required on all commits.

## Working With This Repository

Run `xcaffold apply` from the terminal to regenerate agent configs after editing `.xcaf` files.
Database schema is in `packages/db/schema/`. Refer to it before writing migrations.
```

## Decision Guide

| Content Type | Where It Belongs |
|---|---|
| Universal ground rules (repo layout, tooling, commit format) | `kind: context` — workspace-level instructions |
| File-type or directory-scoped conventions (CSS style, test standards) | `kind: rule` with `activation: path-glob` |
| Agent-specific behavioral guidance | Agent body (system prompt) |
| Reusable procedural instructions | `kind: skill` with step-by-step body |
| Per-provider behavioral differences | `kind: context` with provider-scoped `targets:` |

## Related

- [Context Reference](../reference/kinds/provider/context.md) — field-level documentation for context resources
- [Rule Organization](rule-organization.md) — behavioral guidance with activation modes
- [Project Structure](project-structure.md) — how to organize your xcaf/ directory
- [Multi-Target Compilation](multi-target-compilation.md) — compiling context for multiple providers
