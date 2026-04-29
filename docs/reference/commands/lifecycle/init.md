---
title: "xcaffold init"
description: "Bootstrap the agentic environment and set up the xcaffold compiler."
---

# xcaffold init

Scaffolds the environment to prepare it for Agent-as-Code compilation.

The `init` command bootstraps a repository with a `.xcaffold/project.xcf` file and the base directories needed for defining custom agents, rules, and skills. If it detects existing provider directories (e.g., `.claude/` or `.cursor/`), it will proactively prompt you to run `xcaffold import` instead, enabling a smooth migration path into the Xcaffold ecosystem.

## Usage

```bash
xcaffold init [flags]
```

## Options

| Flag | Default | Description |
|---|---|---|
| `-y, --yes` | `false` | Accept all defaults non-interactively. Perfect for CI/CD injection. |
| `--template <name>` | `""` | Scaffold a predefined topology template. Disables the interactive wizard. Values: `rest-api`, `cli-tool`, `frontend-app`. |
| `--no-policies` | `false` | Skip the generation of starter policy `.xcf` files during bootstrap. |
| `--target <strings>`| `""` | Comma-separated list to immediately compile specific target platforms (e.g., `claude,cursor`) alongside configuration creation. |
| `--json` | `false` | Output machine-readable JSON manifests instead of rendering interactive setup logs to stdout. |
| `-g, --global` | `false` | Bootstrap the user-wide environment instead of project-wide. Generates `~/.xcaffold/global.xcf`. |

## Behavior

### The Bootstrap Phase
1. **Detection:** Identifies if you are starting from a blank slate, or if legacy AI provider configurations currently exist.
2. **Interactive Wizard:** Presents a step-by-step CLI prompt to collect the Project ID, default model provider configuration, and desired AI personas (or agents). 
3. **Synthesis:** Renders the `.xcaffold/project.xcf` entrypoint alongside an initial `agents.xcf` file based on your selections. 
4. **Directory Structure:** Emits the base directory structure `xcf/` (if using split-file architectures) containing `agents/`, `skills/`, `rules/`, and `workflows/`.

### Templates
You can bypass the interactive wizard fully by specifying a `--template`. This option will automatically pre-populate your `.xcf` configuration with a battle-tested architecture corresponding to your project type.

## Examples

**Start the interactive initialization wizard:**
```bash
xcaffold init
```

**Initialize a blank project silently (CI/CD mode):**
```bash
xcaffold init --yes
```

**Initialize an advanced Full-Stack template and immediately target Claude:**
```bash
xcaffold init --template rest-api --target claude
```

**Initialize the Global registry:**
```bash
xcaffold init --global
```
