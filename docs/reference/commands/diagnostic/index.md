---
title: Diagnostic
description: Commands that inspect state, topologies, and compilation health.
---

# Diagnostic Commands

Diagnostic commands never alter active workspaces or emit provider artifact configurations. They natively read your schemas and render topology analytics, calculate programmatic drift, and inspect execution health.

| Command | Action |
| --- | --- |
| [`graph`](./graph) | Visualize the resource dependency graph |
| [`list`](./list) | List discovered resources and blueprints |
| [`status`](./status) | Show compilation state and check for drift across all providers |
