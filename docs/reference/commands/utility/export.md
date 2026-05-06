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

| Flag | Default | Description |
|---|---|---|
| `--format <string>` | `"plugin"` | Desired distribution format logic. Currently solely supports `plugin` repackaging mappings. |
| `--output <string>` | `""` | Output directory destination for processing generated exported constraints. |
| `--target <string>` | `"claude"` | Underlying compilation target provider platform for translating native artifacts into distribution wrappers. Valid ranges identically align with standard target variables (`claude`, `cursor`, `gemini`, etc.). |
| `--var-file <path>` | `""` | Load variables from a custom file instead of the default `xcf/project.vars`. |

## Behavior

The export routine performs the following lifecycle behaviors:
1. **Source Discovery**: Connects directly against compiled physical artifacts associated immediately with `--target` requirements inside your core configuration mapping variables. 
2. **Translation Verification**: Audits structural capabilities surrounding your local output layer resolving missing artifact mappings required exclusively for independent distribution logic.
3. **Format Packaging**: Builds the requested formatting wrappers natively into your requested `--output` directories.

## Examples

**Package your compiled configuration into a standardized plugin distribution:**
```bash
xcaffold export --format plugin --output ./my-plugin/
```
