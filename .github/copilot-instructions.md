# GitHub Copilot Workspace Instructions

You are working in the `xcaffold` codebase.

## What is xcaffold?

`xcaffold` is an open-source Go CLI that compiles `.xcf` YAML configuration deterministically into Claude Code's native `.claude/` directory structure. It provides drift detection, token budgeting, and sandboxed agent simulation.

**Lifecycle:** `init → analyze → plan → apply → diff → test`

## Quick Commands

```bash
make build     # Compile binary → ./xcaffold
make test      # go test ./... — must always be zero failures before declaring done
make lint      # golangci-lint (falls back to go vet)
```

## Core Architectural Rules

1. **One-way compilation only.** `.xcf` → `.claude/`. Never edit `.claude/` files directly. `xcaffold apply` overwrites them. `xcaffold diff` detects manual changes.

2. **Strict YAML parsing.** Always use `yaml.KnownFields(true)`. Adding a new AST field in `internal/ast/types.go` **requires updating all `testing/fixtures/*.xcf` files** or the parser tests will fail.

3. **Path safety is mandatory.** All output paths use `filepath.Clean`. `instructions_file:` and `references:` paths must be validated against `..` traversal. Use the `validateID` helper pattern in `internal/parser/parser.go`.

4. **Skills compile to directories.** Skills output to `skills/<id>/SKILL.md`, never flat `skills/<id>.md`. `apply.go` creates parent directories via `os.MkdirAll(filepath.Dir(absPath))`.

5. **Compiler takes a `baseDir`.** `compiler.Compile(config, baseDir)` receives the directory containing `scaffold.xcf` for resolving `instructions_file:` references. Never use `os.Getwd()` inside the compiler.

6. **Never reference the platform.** This is open-source only. Do not mention `platform`, internal SaaS features, or enterprise pricing in any code, comment, or commit.

7. **Security invariants — do not touch:**
   - `http.MaxBytesReader()` in `proxy.go`
   - `filepath.Base()` validation on `claudePath` in `generator.go` / `judge.go`
   - All `filepath.Clean()` calls on output paths

## AST Type Reference

All types are in `internal/ast/types.go`. Key fields:

**AgentConfig:** `name, description, instructions, instructions_file, model, effort, memory, maxTurns, tools, blocked_tools, skills, rules, mcp, assertions`

**SkillConfig:** `name, type, description, instructions, instructions_file, tools, allowed-tools, paths, references`

**RuleConfig:** `description, paths, instructions, instructions_file`

**SettingsConfig:** `env, statusLine (object), enabledPlugins (map[string]bool), alwaysThinkingEnabled, effortLevel, skipDangerousModePermissionPrompt, permissions, mcpServers`

> `instructions` and `instructions_file` are mutually exclusive per resource. Both set → parse error.

## Current Implementation Status

### Resolved bugs (do not re-introduce)
- Bugs 1-13 resolved — see `xcaffold/llms.txt` for full list

### Features in planning (check `FEATURES.md`)
- `instructions_file:` support — P0
- `references:` for skill files — P0
- `xcaffold graph` command — P1
- Settings type fixes (`statusLine` object, `enabledPlugins` map) — P1
- Faithful `xcaffold import` — P1

## Test Pattern

All compiler tests live in `internal/compiler/`. Follow the naming convention:
```
TestCompile_<Feature>_<Scenario>
```

All parser edge tests live in `internal/parser/parser_edge_test.go`. Always add a negative test (invalid input) for every new validation rule.

## Additional Context

- `CLAUDE.md` — Claude Code–specific deep context
- `AGENTS.md` — Universal AI agent rules (NEVER list)
- `FEATURES.md` — planned CLI features
- `planned platform features (out of scope for this repo)
