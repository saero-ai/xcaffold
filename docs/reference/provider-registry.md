---
title: "Provider Registry"
description: "Registered providers, their input/output directories, and kind support matrix"
---

# Provider Registry

## Registered Providers

| Provider | InputDir | OutputDir | Importer | Renderer |
|----------|----------|-----------|----------|----------|
| claude | `.claude` | `.claude` | `internal/importer/claude/` | `internal/renderer/claude/` |
| cursor | `.cursor` | `.cursor` | `internal/importer/cursor/` | `internal/renderer/cursor/` |
| gemini | `.gemini` | `.gemini` | `internal/importer/gemini/` | `internal/renderer/gemini/` |
| copilot | `.github` | `.github` | `internal/importer/copilot/` | `internal/renderer/copilot/` |
| antigravity | `.agents` | `.agents` | `internal/importer/antigravity/` | `internal/renderer/antigravity/` |

## Kind Support Matrix

| Kind | Claude | Cursor | Gemini | Copilot | Antigravity |
|------|--------|--------|--------|---------|-------------|
| Agent | agents/*.md | agents/*.md | agents/*.md | agents/*.md | prompts/*.md |
| Skill | skills/*/SKILL.md | skills/*/SKILL.md | skills/*.md | skills/*.md | skills/*.md |
| Rule | rules/*.md | rules/*.mdc | rules/*.md | instructions/*.instructions.md | rules/*.md |
| MCP | mcp.json | mcp.json | settings.json (embedded) | copilot/mcp-config.json | mcp_config.json |
| Hooks | settings.json (embedded) | hooks.json | settings.json (embedded) | -- | -- |
| Settings | settings.json | -- | settings.json | -- | -- |
| Memory | agent-memory/** | -- | -- | -- | -- |
| Workflow | -- | -- | -- | copilot-setup-steps.yml | workflows/*.md |

## ProviderExtras

Files that don't match any KindMapping pattern are stored in `ProviderExtras`. During `apply`:
- Same-provider extras are restored as-is
- Cross-provider extras are skipped with a FidelityNote

During parse, `ReclassifyExtras()` re-evaluates extras against current importers, graduating files that are now recognized.
