<p align="center">
  <img src="assets/logo.svg" alt="xcaffold logo" height="150" />
  <img src="assets/xaff.svg" alt="Xaff" height="150" />
</p>

# xcaffold

[![CI](https://github.com/saero-ai/xcaffold/actions/workflows/ci.yml/badge.svg)](https://github.com/saero-ai/xcaffold/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/saero-ai/xcaffold)](https://goreportcard.com/report/github.com/saero-ai/xcaffold)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Go Version](https://img.shields.io/badge/go-1.24-blue.svg)](https://golang.org/dl/)

**Your agents, by design.** Agent-as-Code — design, compile, and manage agent blueprints across every AI coding platform. Declare your agents in a single `.xcf` YAML file. `xcaffold` compiles it deterministically into native configurations across multiple providers — with drift detection, behavioral testing, and policy enforcement.

```
                                    ──► (claude)       ──►  .claude/
project.xcf  ──►  xcaffold apply   ──► (cursor)       ──►  .cursor/
                                    ──► (antigravity)  ──►  .agents/
                                    ──► (copilot)      ──►  .github/
                                    ──► (gemini)       ──►  .gemini/
```

## Why xcaffold?

Agent configuration fragments across tools. Each AI coding platform (Claude Code, Cursor, GitHub Copilot, Gemini CLI, Antigravity) has its own format: `.claude/`, `.cursor/`, `.github/`, `.agents/`, etc. Teams either maintain parallel copies — accepting drift and inconsistency — or lock themselves into a single provider.

`xcaffold` treats agent blueprints as a programmable, testable source of truth. A single `.xcf` file defines your agents once, then compiles into every platform's native format:

- **Blueprint-based design** — Declare intent once. Grow configurations over time without rewriting. Switch contexts between providers without losing fidelity.
- **Multi-provider compilation** — Compile to Claude Code, Cursor, GitHub Copilot, Gemini CLI, Antigravity simultaneously. Evaluate and migrate between platforms without rebuilding agents.
- **Drift detection** — SHA-256 state tracking detects when `.claude/`, `.cursor/`, or other generated directories are manually edited, preventing silent overwrites.
- **Behavioral testing** — Define test assertions and use `xcaffold test --judge` to validate agent behavior before deployment, powered by LLM-as-a-Judge.
- **Cross-provider translation** — Automatically map capabilities, constraints, and tools across platform differences. Fidelity reporting shows what translates and what requires customization.
- **Policy enforcement** — Compile-time policy checks prevent unsafe configurations (dangerous tool combinations, missing required constraints) from reaching any platform.
- **Import from existing** — Start from an existing `.claude/`, `.cursor/`, or other provider config. `xcaffold import` extracts your agents and skills into a `.xcf` blueprint.

`xcaffold` is a compiler layer, not a new format. It sits above any source (`.xcf` YAML, community standards like `AGENTS.md`, or imported configs) and produces deterministic, testable, multi-platform agent output.

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
kind: project
version: "1.0"
name: my-project
agents:
  - developer

---
kind: agent
version: "1.0"
name: developer
description: General-purpose development agent
instructions: |
  You are a software developer.
  Write clean, maintainable code.
model: claude-sonnet-4-20250514
tools: [Bash, Read, Write, Edit, Glob, Grep]
```

Then run the full lifecycle:

```bash
xcaffold init --target claude,cursor     # Scaffold a new provider-first project
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
