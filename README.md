<p align="center">
  <img src="assets/logo.svg" alt="xcaffold logo" height="150" />
  <img src="assets/xaff.svg" alt="Xaff" height="150" />
</p>

# xcaffold

[![CI](https://github.com/saero-ai/xcaffold/actions/workflows/ci.yml/badge.svg)](https://github.com/saero-ai/xcaffold/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/saero-ai/xcaffold)](https://goreportcard.com/report/github.com/saero-ai/xcaffold)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Go Version](https://img.shields.io/badge/go-1.24-blue.svg)](https://golang.org/dl/)

**deterministic agent configuration compiler for AI coding platforms.** Declare your AI agents in a single `.xcf` YAML file. `xcaffold` compiles it deterministically into native AI coding agent constraints (e.g., `.claude/`, `.cursor/rules/`, `.agents/`) — with drift detection, token budgeting, and sandboxed simulation.

```
                                    ──► (claude)       ──►  .claude/
scaffold.xcf  ──►  xcaffold apply   ──► (cursor)       ──►  .cursor/rules/
                                    ──► (antigravity)  ──►  .agents/
                                    ──► (agentsmd)     ──►  AGENTS.md
```

## Why xcaffold?

Most teams hand-edit their AI agent configurations manually. This means:
- **No drift detection** — edits made outside the IDE go unnoticed.
- **No token budgeting** — agents silently exceed context limits.
- **No auditability** — no SHA manifest, no CI/CD integration.
- **No testability** — agent behavior is never validated against declared assertions.

`xcaffold` applies GitOps discipline to agent configuration. The `.xcf` file is the only source of truth. Generated configurations are strict compilation artifacts.

## Installation

**Homebrew (macOS/Linux)**
```bash
brew install saero-ai/tap/xcaffold
```

**Scoop (Windows)**
```powershell
scoop bucket add saero-ai https://github.com/saero-ai/scoop-bucket.git
scoop install xcaffold
```

**Go install (All Platforms)**
```bash
go install github.com/saero-ai/xcaffold/cmd/xcaffold@latest
```

**Binary releases**

Pre-built binaries for Linux (amd64/arm64), macOS (amd64/arm64), and Windows (amd64) are available on the [Releases page](https://github.com/saero-ai/xcaffold/releases). Windows binaries are packaged as `.zip` while Unix binaries are `.tar.gz`.

**Build from source**
```bash
git clone https://github.com/saero-ai/xcaffold
cd xcaffold
make build
```

## Example Usage

```yaml
version: "1.0"
project:
  name: "acme-web-platform"

agents:
  developer:
    description: "Expert React developer."
    model: claude-3-7-sonnet-20250219
    tools: [Read, Write, Bash, Glob]
    blocked_tools: [WebFetch]
    skills: [git]
    instructions: |
      You are a frontend developer specializing in standard React.
      Always run tests before marking a task complete.
    assertions:
      - "The agent must not write files outside the project directory."
      - "The agent must run tests before marking a task complete."

skills:
  git:
    description: "Git commit conventions."
    instructions: "Always use conventional commits (feat:, fix:, chore:)."

rules:
  typescript:
    paths: ["src/**/*.ts"]
    instructions: "Prefer type aliases over interfaces. No any."

hooks:
  pre-commit:
    run: "make lint"

mcp:
  sqlite:
    command: npx
    args: ["-y", "@modelcontextprotocol/server-sqlite", "data.db"]

test:
  judge_model: "claude-3-5-haiku-20241022"
```

Then run the full lifecycle:

```bash
xcaffold graph             # View agent topology maps
xcaffold apply             # Compile to targets (or --check to validate syntax)
xcaffold diff              # Check for manual drift in output dirs
xcaffold test --agent developer --judge  # Simulate + evaluate
```

## Documentation

To learn more about how to use `xcaffold`, explore our documentation:

- [Tutorials](docs/tutorials/index.md) — End-to-end guides for agent configuration setups.
- [How-To Guides](docs/how-to/index.md) — Targeted recipes for common workflows.
- [Concepts](docs/concepts/index.md) — Deep dives into architecture, compilation scopes, and best practices.
- [Reference](docs/reference/index.md) — Exhaustive `.xcf` schema references, CLI flags, and diagnostics APIs.


## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).
