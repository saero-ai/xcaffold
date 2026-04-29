---
title: "xcaffold apply"
description: "Deterministically compile your AST definitions into provider-native artifacts."
---

# xcaffold apply

Compiles your `.xcf` logic into native target outputs (`.claude/`, `.cursor/`, etc.).

The `apply` command reads your centralized configuration (AST), cross-references agents with their defined tools, enforces required structural policies, and invokes the specific renderer API tied to the compilation target. 

This is a strictly one-way generation process. Any manual modifications making their way directly into the compiled output directory will be violently overwritten.

## Usage

```bash
xcaffold apply [flags]
```

## Options

| Flag | Default | Description |
|---|---|---|
| `--target <string>` | `"claude"` | The compilation target platform. Valid options: `claude`, `cursor`, `antigravity`, `copilot`, `gemini`. |
| `--blueprint <string>` | `""` | Compile only a subset of resources specified by the designated blueprint logic. |
| `--backup` | `false` | Take a differential backup of the existing output directory to a timestamped archive before writing compiled output. |
| `--check` | `false` | Validate the configuration syntax and logic without rendering physical formats. |
| `--check-permissions` | `false` | Validate the specific platform capabilities. It will report dropped capabilities and emit error exits if explicit policy contradictions are detected. |
| `--dry-run` | `false` | Prints a simulated execution plan to standard output without persisting any artifacts to disk. |
| `--force` | `false` | Force an overwrite of the native directories, bypassing the safety mechanisms that prevent compilation when downstream physical artifacts have drifted. |
| `--project <name>` | `""` | Provide the namespace or exact path to external project configuration maintained by the global configuration registry. |
| `-g, --global` | `false` | Compile the user-wide environment global registry (`~/.xcaffold/global.xcf`). |

## Behavior

### Compilation Phases
1. **Parsing:** Evaluates all root level declarations via the `internal/parser`. Enforces Xcaffold strict structural schemas alongside recursive dependency checks. Unrecognized keys throw immediately.
2. **Translation:** Builds the provider-neutral Graph AST representations. Cross pollinates instructions with corresponding resources.
3. **Validation Checkpoints:** Ensures the current generated tree passes local project guidelines along with internal governance `Policies`.
4. **Rendering Execution:** Transmutes standard AST objects into formatted provider artifacts. Generates markdown schemas for text inputs, `.json` descriptors natively supported by LLM providers, and hooks scripts.
5. **State Ledger:** Triggers drift detection routines and subsequently writes the `.xcaffold/project.xcf.state` snapshot referencing exactly what was scaffolded alongside active cryptography signatures. Orphaned files remaining from a previous AST are purged.

### Output Constraints
Xcaffold acts as the single source of truth for your configuration. To prevent configuration silos and fragmentation, running `xcaffold apply` explicitly enforces parity. A direct implication means that manually edited configuration files native to your specific agentic workspace (e.g., tweaking `clauderc.json` within `.claude/`) will be eradicated cleanly during the subsequent update loop to fulfill declarative definitions.

## Examples

**Compile your AST to target the Claude Code platform:**
```bash
xcaffold apply --target claude
```

**Validate syntactical compliance without execution:**
```bash
xcaffold apply --check
```

**Run a preview against the Cursor implementation mapping, skipping outputs:**
```bash
xcaffold apply --target cursor --dry-run
```
