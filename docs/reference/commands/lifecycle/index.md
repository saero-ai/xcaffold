---
title: Lifecycle
description: Commands that govern the end-to-end declarative configuration lifecycle.
---

# Lifecycle Commands

These commands manage the primary states of your project configuration, defining the core lifecycle loop: from bootstrap (`init`), to active execution (`apply`), validation (`validate`), and reverse ingestion (`import`).

| Command | Action |
| --- | --- |
| [`apply`](./apply) | Compile .xcaf resources into provider-native agent configuration files. |
| [`import`](./import) | Scans and reverse-engineers legacy workspace inputs into isolated xcaffold ASTs. |
| [`init`](./init) | Interactively bootstrap environment contexts creating initial schemas. |
| [`validate`](./validate) | Validate .xcaf manifests against the schema without compiling. |
