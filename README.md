<p align="center">
  <img src="assets/logo.svg" alt="xcaffold logo" height="150" />
  <img src="assets/xaff.svg" alt="Xaff" height="150" />
</p>

# xcaffold

[![CI](https://github.com/saero-ai/xcaffold/actions/workflows/ci.yml/badge.svg)](https://github.com/saero-ai/xcaffold/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/saero-ai/xcaffold)](https://goreportcard.com/report/github.com/saero-ai/xcaffold)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Go Version](https://img.shields.io/badge/go-1.25-blue.svg)](https://go.dev/dl/)

**Your agents, by design.** Every AI coding tool your team uses ships its own configuration format, directory structure, and file conventions. Use three tools and you maintain three separate configuration trees — `.claude/`, `.cursor/`, `.gemini/` — that drift from each other silently. When someone updates the rules in one, the others are forgotten.

`xcaffold` gives you `.xcaf` manifests as your single source of truth — compiling deterministically into native configuration for every provider your team uses.

```
project.xcaf  ──►  xcaffold apply  ──►  claude       ──►  .claude/
                                   ──►  cursor       ──►  .cursor/
                                   ──►  gemini       ──►  .gemini/
                                   ──►  copilot      ──►  .github/
                                   ──►  antigravity  ──►  .agents/
```

This is **Harness-as-Code**: the complete agent harness — system prompts, tools, rules, memory, hooks, MCP servers, and policies — declared once in version-controlled `.xcaf` manifests, compiled deterministically, with drift detection and compile-time policy enforcement.

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

**Build from source**
```bash
git clone https://github.com/saero-ai/xcaffold
cd xcaffold
make build
./xcaffold --help
# or: make install  (installs to $GOPATH/bin)
```

Pre-built binaries for Linux (amd64/arm64), macOS (amd64/arm64), and Windows (amd64) are available on the [Releases page](https://github.com/saero-ai/xcaffold/releases).

## Quick Start

**Already have a `.claude/`, `.cursor/`, or `.gemini/` directory?** Import your existing configuration in seconds:

```bash
xcaffold import --target claude    # reads .claude/ → generates .xcaf manifests
```

**Starting from scratch:**

```bash
xcaffold init                      # scaffold a new project.xcaf
```

**Core workflow:**

```bash
xcaffold apply                     # compile .xcaf → .claude/, .cursor/, etc.
xcaffold status                    # detect drift in output directories
xcaffold validate                  # validate manifests without compiling
xcaffold graph                     # visualize resource scope and dependencies
xcaffold list                      # list all resources across providers
```

## What xcaffold Manages

Each `.xcaf` manifest declares one resource in the agent harness. xcaffold compiles the full set to the appropriate native format per provider:

| Kind | Purpose |
|------|---------|
| `agent` | Identity, system prompt, model selection, tool declarations |
| `skill` | Reusable capability modules with scoped tool access |
| `rule` | Constraints and standards enforced at the provider level |
| `hooks` | Lifecycle hooks — pre/post tool use, session events |
| `mcp` | MCP server declarations and connection configuration |
| `memory` | Persistent memory definitions |
| `settings` | Provider-level permissions and behavior settings |
| `policy` | Compile-time enforcement; violations block `xcaffold apply` |
| `workflow` | Multi-step agent procedures |
| `blueprint` | Resource subset selectors for multi-environment targeting |
| `context` | Formal workspace context declarations |

## Provider Support

| Resource | Claude Code | Cursor | GitHub Copilot | Gemini CLI | Antigravity |
|----------|:-----------:|:------:|:--------------:|:----------:|:-----------:|
| Agents | ✓ | ✓ | ✓ | ✓ | — |
| Skills | ✓ | ✓ | ✓ | ✓ | ✓ |
| Rules | ✓ | ✓ | ✓ | ✓ | ✓ |
| Workflows | ✓* | ✓* | ✓* | ✓* | ✓ |
| Hooks | ✓ | ✓ | ✓ | ✓ | — |
| MCP Servers | ✓ | ✓ | ✓ | ✓ | ✓ |
| Memory | ✓ | —** | —** | —** | —** |
| Settings | ✓ | ✓ | ✓ | ✓ | ✓ |

*Compiled as rules + skills for providers without a native workflow format.
**Persistent context can be delivered through `context`, `rule`, or `hooks` kinds. See [memory reference](docs/reference/kinds/provider/memory.md#cross-provider-memory-patterns).

When a feature cannot be expressed in a target's native format, xcaffold emits a structured fidelity report rather than silently dropping configuration. You always know exactly what was and was not applied.

The provider architecture is open. Adding a new target requires implementing two Go interfaces (`TargetRenderer` and `ProviderImporter`). Agent SDKs with declarative configuration formats are natural expansion targets. See [CONTRIBUTING.md](CONTRIBUTING.md).

## Documentation

- [Tutorials](docs/tutorials/index.md) — End-to-end setup guides
- [Best Practices](docs/best-practices/index.md) — Task-oriented recipes
- [Concepts](docs/concepts/index.md) — Architecture, compilation, field model
- [Reference](docs/reference/index.md) — CLI commands, `.xcaf` schema, provider matrix

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

Apache 2.0 — see [LICENSE](LICENSE).
