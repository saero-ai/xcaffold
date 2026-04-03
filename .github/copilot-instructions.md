# GitHub Copilot Workspace Instructions

You are working in the `xcaffold` codebase.

- `xcaffold` is an open-source Go CLI that compiles `.xcf` configuration directly into `.claude/` directories.
- Always run `make test` before declaring a task complete.
- When writing tests, strictly follow Test-Driven Development (TDD) via standard Go `testing` loops.
- NEVER manually add files to `.claude/`. Modifying those outputs breaks the deterministic One-Way Compilation architecture.
- NEVER use generic marketing terminology or mention proprietary platforms like "platform".
- Check `AGENTS.md` and `CLAUDE.md` for deeper instructions.
