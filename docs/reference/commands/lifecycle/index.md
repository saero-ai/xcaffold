---
title: Lifecycle
description: Commands that govern the end-to-end declarative configuration lifecycle.
---

# Lifecycle Commands

These commands manage the primary states of your project configuration, defining the core lifecycle loop: from bootstrap (`init`), to active execution (`apply`), validation (`validate`), and reverse ingestion (`import`).

| Command | Action |
| --- | --- |
| [`apply`](./apply) | Compile .xcaf resources into provider-native agent files |
| [`import`](./import) | Import existing provider config into project.xcaf |
| [`init`](./init) | Bootstrap a new project.xcaf configuration |
| [`validate`](./validate) | Check .xcaf syntax, cross-references, and structural invariants |
