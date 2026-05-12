---
title: "xcaffold export"
description: "Repackaging natively compiled artifacts into portable, standardized distributions."
---

> **Preview command** — This command is hidden from `xcaffold --help` output. It may change without notice.

# xcaffold export

Repackages compiled output into a standardized, portable plugin-compliant directory format.

The `export` command bypasses specific localized integration patterns for IDE dependencies and translates rendered architectures directly into distributable bundles (currently focused entirely around the `plugin` format specification), enabling teams to share complete topologies agnostically.

## Usage

```bash
xcaffold export [flags]
```

## Options

| Flag | Required | Default | Description |
|---|---|---|---|
| `--output <string>` | Yes | | Output directory for the exported plugin. |
| `--format <string>` | | `"plugin"` | Export format. Currently only `plugin` is supported. |
| `--target <string>` | | `""` | Compilation target provider (e.g., `claude`, `cursor`, `gemini`). |
| `--var-file <path>` | | `""` | Load additional variables from a custom file (layered on top of `xcaf/project.vars`). |

## Behavior

1. **Parse**: Reads the project configuration from `project.xcaf` and any `xcaf/` sources.
2. **Compile**: Runs the full compilation pipeline for the specified `--target` (or provider-agnostic if omitted).
3. **Optimize**: Applies the optimizer pass for the target provider.
4. **Package**: Writes the plugin directory structure to `--output`.

## Examples

**Package your compiled configuration into a plugin distribution:**
```bash
xcaffold export --format plugin --output ./my-plugin/
```

**Export targeting a specific provider:**
```bash
xcaffold export --format plugin --output ./my-plugin/ --target claude
```

**Export with a custom variable file:**
```bash
xcaffold export --format plugin --output ./my-plugin/ --target gemini --var-file ./custom.vars
```
