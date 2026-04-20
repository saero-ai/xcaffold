<p align="center">
  <img src="assets/logo.svg" alt="xcaffold logo" height="150" />
  <img src="assets/xaff.svg" alt="Xaff" height="150" />
</p>

# xcaffold

[![CI](https://github.com/saero-ai/xcaffold/actions/workflows/ci.yml/badge.svg)](https://github.com/saero-ai/xcaffold/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/saero-ai/xcaffold)](https://goreportcard.com/report/github.com/saero-ai/xcaffold)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Go Version](https://img.shields.io/badge/go-1.24-blue.svg)](https://golang.org/dl/)

**Your agents, by design.** Agent-as-Code — design, compile, and manage agent blueprints across every AI coding platform. Declare your agents once in a `.xcf` YAML file. `xcaffold` compiles deterministically into native configurations for Claude Code, Cursor, GitHub Copilot, Gemini CLI, and Antigravity — with drift detection, policy enforcement, and behavioral testing.

```
                                   ──► claude        ──►  .claude/
project.xcf  ──►  xcaffold apply  ──► cursor        ──►  .cursor/
                                   ──► antigravity   ──►  .agents/
                                   ──► copilot       ──►  .github/
                                   ──► gemini        ──►  .gemini/
```

## Why xcaffold?

The agentic development ecosystem is fragmented. Every AI coding tool ships its own configuration format, directory structure, and file conventions. Teams maintaining agents across multiple tools end up with multiple independent, unmaintained configuration trees.

`xcaffold` eliminates this fragmentation by treating agent configuration as code — declarative, deterministic, and version-controlled. A single `project.xcf` YAML file is your source of truth. It compiles to native formats for every supported platform.

### Core capabilities

- **Blueprint-based agent configuration** — Define agents, skills, rules, and hooks once in `.xcf` YAML. Target-specific details emerge at compile time, not in your source tree.
- **Multi-provider compilation** — Single source, native output. Compile to Claude Code, Cursor, GitHub Copilot, Gemini CLI, Antigravity — all from one `.xcf` file.
- **Drift detection via SHA-256 state** — Track compiled state in `project.lock`. Detect when `.claude/`, `.cursor/`, or other output directories have been manually edited. `xcaffold diff` shows exactly what changed and why.
- **Policy enforcement at compile time** — Define policies (require, deny, match constraints). Violations block compilation with precise error messages. No unsafe agent configurations reach production.
- **Behavioral testing with `xcaffold test`** — Simulate agent behavior against declared test cases. Use `--judge` to evaluate behavior against assertions with LLM-backed reasoning.
- **Cross-provider translation with fidelity reporting** — When a capability cannot be expressed in a target's native format, `xcaffold` reports the translation loss. Migrate between providers with full visibility into behavioral gaps.
- **Import existing configs** — Have agents already configured in Claude Code, Cursor, or GitHub Copilot? `xcaffold import` reads existing agent/skill/rule directories and reconstructs the `.xcf` blueprint.

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

Define your agents in `project.xcf`:

```yaml
kind: project
version: "1.0"
name: my-agents
agents:
  - name: backend
    description: Backend development agent
    instructions: |
      You are a Golang backend engineer. 
      Write production-grade code.
    model: claude-opus-4-1-20250805
    tools: [Bash, Read, Write, Edit, Glob, Grep]
    targets:
      - claude
      - cursor
      
  - name: frontend
    description: Frontend development agent
    instructions: |
      You are a TypeScript/React engineer.
      Prioritize accessibility and performance.
    model: claude-opus-4-1-20250805
    tools: [Bash, Read, Write, Edit, Glob, Grep]
    targets:
      - claude
```

Run the lifecycle:

```bash
xcaffold init                      # Initialize a new project.xcf
xcaffold apply                     # Compile to .claude/, .cursor/, etc.
xcaffold diff                      # Detect manual drift in output directories
xcaffold validate                  # Check syntax without compiling
xcaffold test --agent backend      # Simulate agent behavior
xcaffold import --provider claude  # Read existing .claude/ and generate .xcf
```

## Documentation

To learn more about how to use `xcaffold`, explore our documentation:

- [Tutorials](docs/tutorials/index.md) — End-to-end guides for agent configuration setups.
- [How-To Guides](docs/how-to/index.md) — Targeted recipes for common workflows.
- [Concepts](docs/concepts/index.md) — Deep dives into architecture, compilation scopes, and best practices.
- [Reference](docs/reference/index.md) — Exhaustive `.xcf` schema references, CLI flags, and diagnostics APIs.


## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).
