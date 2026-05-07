---
title: Lifecycle
description: Commands that govern the end-to-end declarative configuration lifecycle.
---

# Lifecycle Commands

These commands manage the primary states of your project configuration, defining the core lifecycle loop: from bootstrap (`init`), to active execution (`apply`), and reverse ingestion (`import`).

| Command | Action |
| --- | --- |
| [\`apply\`](/docs/cli/reference/commands/lifecycle/apply) | Compile .xcaf resources into provider-native agent configuration files. |
| [\`import\`](/docs/cli/reference/commands/lifecycle/import) | Scans and reverse-engineers legacy workspace inputs into isolated Xcaffold ASTs. |
| [\`init\`](/docs/cli/reference/commands/lifecycle/init) | Interactively bootstrap environment contexts creating initial schemas. |
