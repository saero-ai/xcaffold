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
| Antigravity | `GEMINI.md` | project root |
| GitHub Copilot | `copilot-instructions.md` | `.github/` |

> [!NOTE]
> Antigravity reads `GEMINI.md` from the project root for project context — it does not have its own separate context file. Both `gemini` and `antigravity` targets write to the same `GEMINI.md`.

Because these files are loaded on **every interaction** (check with the provider for any changes), their token cost is unconditional — paid regardless of which agent is active or what task is being performed. Keep them short and focused on absolute invariants.

---

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
  - antigravity
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

> [!IMPORTANT]
> The `kind: project` document does **not** carry workspace instructions. Project metadata (`name`, `targets`, `extends`) goes in `kind: project`. Workspace prose goes in `kind: context`. Mixing the two will cause a parse error — the parser enforces strict field validation (`KnownFields(true)`).

---

## Rule 1 — Document Only Absolute Laws

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

---

## Rule 2 — Isolate Tactical Rules to Their Scope

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

---

## Rule 3 — Write Instructions, Not Documentation

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

---

## Rule 4 — Use Multiple Context Documents for Multi-Provider Projects

If providers need different framing for the same information — for example, Claude has MCP tool access while Copilot operates only through the editor — use separate `kind: context` documents with scoped `targets:` rather than writing provider-specific conditionals in a single document:

```
---
kind: context
version: "1.0"
name: root-claude
targets:
  - claude
  - antigravity
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

---

