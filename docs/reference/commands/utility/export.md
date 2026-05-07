---
title: "xcaffold export"
description: "Repackaging natively compiled artifacts into portable, standardized distributions."
---

> **Note:** This command is in **Preview**. It is available natively in the binary but its execution schemas subject to changes.

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
| `--target <string>` | | `""` | Compilation target provider (e.g., `claude`, `cursor`, `gemini`). When omitted, exports without provider-specific optimizations. |
| `--var-file <path>` | | `""` | Load variables from a custom file instead of the default `xcaf/project.vars`. |

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
