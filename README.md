<p align="center">
  <img src="assets/logo.svg" alt="xcaffold logo" height="150" />
  <img src="assets/xaff.svg" alt="Xaff" height="150" />
</p>

# xcaffold

[![CI](https://github.com/saero-ai/xcaffold/actions/workflows/ci.yml/badge.svg)](https://github.com/saero-ai/xcaffold/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/saero-ai/xcaffold)](https://goreportcard.com/report/github.com/saero-ai/xcaffold)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Go Version](https://img.shields.io/badge/go-1.24-blue.svg)](https://golang.org/dl/)

**deterministic agent configuration compiler for AI coding platforms.** Declare your AI agents in a single `.xcf` YAML file. `xcaffold` compiles it deterministically into native AI coding agent configurations — with drift detection, policy enforcement, and sandboxed simulation.

```
                                    ──► (claude)       ──►  .claude/
scaffold.xcf  ──►  xcaffold apply   ──► (cursor)       ──►  .cursor/
                                    ──► (antigravity)  ──►  .agents/
                                    ──► (agentsmd)     ──►  AGENTS.md
```

## Why xcaffold?

The agentic development ecosystem is fragmented. Every AI coding tool ships its own configuration format, its own rules directory, its own agent file structure. Teams maintaining agents across multiple tools end up with multiple independent configuration trees — no shared source of truth, no consistency guarantee, and no systematic way to move between them.

`xcaffold` treats agent configuration as code. A single `.xcf` file is compiled deterministically into the native format each platform expects. As the ecosystem grows — and new AI coding tools continue to emerge — xcaffold's renderer model means your existing configuration adapts without being rewritten from scratch.

- **No portability** — Configurations written for one AI coding tool cannot be compiled to another. Evaluating an alternative or migrating to a different provider means recreating every agent, skill, and rule from scratch.
- **No reproducible onboarding** — New team members configure agents from memory or copy-paste. There is no `git clone && xcaffold apply` for agent setup.
- **No drift detection** — Changes made directly to generated files (`.claude/`, `.cursor/`) go unnoticed and are silently overwritten on the next compile.
- **No fleet visibility** — Teams with multiple projects have no single view of what agents are deployed, which targets are active, or when a project was last compiled.
- **No policy enforcement** — Nothing prevents an agent from being configured with dangerous permissions or missing required constraints before it runs against production code.
- **No testability** — Agent behavior is never validated against declared constraints before deployment.

`xcaffold` is not a format — it is the compilation and governance layer above any format, including community standards like `AGENTS.md`. Version control, drift detection, policy evaluation, and behavioral testing apply regardless of which platform you compile to.

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
./xcaffold --help      # Run locally
# OR
make install           # Install globally to $GOPATH/bin
```

## Example Usage

```yaml
kind: config
version: "1.1"
project:
  name: "acme-web-platform"
  test:
    judge_model: "claude-haiku-4-5-20251001"

  agents:
    developer:
      description: "Expert React developer."
      model: claude-sonnet-4-6
      tools: [Read, Write, Bash, Glob]
      disallowedTools: [WebFetch]
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

  mcp:
    sqlite:
      command: npx
      args: ["-y", "@modelcontextprotocol/server-sqlite", "data.db"]
```

Then run the full lifecycle:

```bash
xcaffold list                            # View all registered projects and their status
xcaffold graph                           # Visualize agent topology
xcaffold apply                           # Compile to target (--dry-run to preview)
xcaffold diff                            # Check for manual drift in output dirs
xcaffold validate                        # Validate syntax without compiling
xcaffold test --agent developer --judge  # Simulate and evaluate agent behavior
```

## Documentation

To learn more about how to use `xcaffold`, explore our documentation:

- [Tutorials](docs/tutorials/index.md) — End-to-end guides for agent configuration setups.
- [How-To Guides](docs/how-to/index.md) — Targeted recipes for common workflows.
- [Concepts](docs/concepts/index.md) — Deep dives into architecture, compilation scopes, and best practices.
- [Reference](docs/reference/index.md) — Exhaustive `.xcf` schema references, CLI flags, and diagnostics APIs.


## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).
