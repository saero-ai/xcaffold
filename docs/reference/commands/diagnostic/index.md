---
title: Diagnostic
description: Commands that inspect state, topologies, and compilation health.
---

# Diagnostic Commands

Diagnostic commands never alter active workspaces or emit provider artifact configurations. They natively read your schemas and render topology analytics, calculate programmatic drift, and inspect execution health.

| Command | Action |
| --- | --- |
| [\`graph\`](/docs/cli/reference/commands/diagnostic/graph) | Visualize dependency topologies and execution chains cross-referencing your agents. |
| [\`list\`](/docs/cli/reference/commands/diagnostic/list) | Scan configurations and display discovered resources grouped natively by type. |
| [\`status\`](/docs/cli/reference/commands/diagnostic/status) | Audit current synchronization state and calculate physical drift against the ledger. |
