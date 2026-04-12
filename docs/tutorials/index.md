# Tutorials

xcaffold tutorials are learning-oriented, step-by-step guides that take you from zero to a working result. Each tutorial prioritizes a successful first experience over comprehensive coverage — consult the reference documentation once the workflow is familiar.

## Prerequisites

- xcaffold installed: `brew install saero-ai/tap/xcaffold` or `go install github.com/saero-ai/xcaffold/cmd/xcaffold@latest`
- A terminal
- Any text editor

No AI subscription or API key is required for `init`, `apply`, `diff`, or `validate`.

## Tutorials

Work through these in order. Getting Started is a prerequisite for the other two.

| Order | Tutorial | Description | Time |
|-------|----------|-------------|------|
| 1 | [Getting Started](getting-started.md) | Initialize a project, compile your first agent, and understand the `.xcf` → output pipeline | ~10 min |
| 2 | [Multi-Agent Workspace](multi-agent-workspace.md) | Configure a team of differentiated agents with distinct tool permissions, shared rules and skills, and validated output | ~15 min |
| 3 | [Drift Remediation](drift-remediation.md) | Detect, diagnose, and restore managed files when compiled output has been modified directly | ~10 min |

## Reading Order

**Getting Started** must be completed first — it establishes the project structure and compilation workflow that the other two tutorials build on.

**Multi-Agent Workspace** and **Drift Remediation** are independent of each other and can be read in either order after Getting Started.

## Next Steps

- [`Concepts`](../concepts/index.md) — deep dives into architecture and compilation scopes
- [`How-To Guides`](../how-to/index.md) — task-oriented guides for specific operations
- [`Reference`](../reference/index.md) — full schema reference and CLI command reference
