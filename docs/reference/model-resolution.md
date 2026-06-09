---
title: "Model Resolution"
description: "How xcaffold resolves model tier aliases and literal model IDs per provider"
---

# Model Resolution

xcaffold supports a `model` field on agents that accepts either a **tier alias** (`balanced`, `flagship`, `fast`) or a **literal model ID** (e.g., `gpt-5.5`, `claude-sonnet-4-6`). Each provider's model resolver translates these into the correct output value.

## Tier Aliases

Tier aliases let you write provider-agnostic `.xcaf` files. Each provider maps the alias to its recommended model for that tier.

| Alias | Intent | Claude Code | Cursor | Copilot | Gemini CLI | Codex |
|-------|--------|-------------|--------|---------|------------|-------|
| `balanced` | General-purpose default | `claude-sonnet-4-5` | `claude-sonnet-4-6` | `claude-sonnet-4-6` | `gemini-2.5-flash` | `gpt-5.4` |
| `flagship` | Most capable model | `claude-opus-4-7` | `gpt-5.5` | `claude-opus-4-7` | `gemini-2.5-pro` | `gpt-5.5` |
| `fast` | Fastest / cheapest | `claude-haiku-4-5` | `composer-2.5` | `claude-haiku-4-5` | `gemini-2.5-flash` | `gpt-5.4-mini` |

> **Last verified:** 2026-06-09 against official provider documentation.
> Tier mappings are defaults. Override them per-provider using `${var.model-default}` in `project.<provider>.vars` or `agent.<provider>.xcaf` overrides.

## Literal Model IDs (Pass-Through)

When the `model` field contains a literal model ID instead of a tier alias, each provider checks whether the ID matches a known prefix family and passes it through unchanged.

| Provider | Accepted Prefixes |
|----------|------------------|
| Claude Code | `claude-`, bare aliases (`sonnet`, `opus`, `haiku`) |
| Cursor | `claude-`, `gpt-`, `gemini-`, `cursor-`, `composer-`, `o1-`, `o3-`, `grok-`, `kimi-` |
| Copilot | `claude-`, `gpt-` |
| Gemini CLI | `gemini-` |
| Codex | `gpt-` |
| Antigravity | *(no model support)* |

Pass-through is case-insensitive. `Claude-Sonnet-4-6` resolves to `claude-sonnet-4-6`.

Unrecognized model IDs (no matching prefix) produce an `AGENT_MODEL_UNMAPPED` warning and are omitted from output.

## Fidelity Notes

| Note Code | Level | When |
|-----------|-------|------|
| `FIELD_TRANSFORMED` | Info | A literal model ID was passed through (not a tier alias) |
| `AGENT_MODEL_UNMAPPED` | Warning | The model ID could not be resolved (unknown prefix) |

## Per-Provider Override

To set a different model for a specific provider without changing the base `.xcaf`:

```yaml
# xcaf/agents/researcher/agent.cursor.xcaf
model: gpt-5.5
```

Or use variables:

```yaml
# xcaf/agents/researcher/agent.xcaf
model: ${var.model-deep}
```

```ini
# project.cursor.vars
model-deep = gpt-5.5

# project.claude.vars
model-deep = claude-opus-4-7
```

## Cursor Model Catalog

Featured models available in Cursor as of 2026-06-09:

| Model | Provider | Slug | Context | Tier |
|-------|----------|------|---------|------|
| Claude 4.6 Sonnet | Anthropic | `claude-sonnet-4-6` | 200K / 1M | balanced |
| Claude Opus 4.8 | Anthropic | `claude-opus-4-8` | 300K / 1M | flagship |
| Composer 2.5 | Cursor | `composer-2.5` | 200K | fast |
| Gemini 3.1 Pro | Google | `gemini-3.1-pro` | 200K / 1M | flagship |
| Gemini 3.5 Flash | Google | `gemini-3.5-flash` | 200K / 1M | fast |
| GPT-5.3 Codex | OpenAI | `gpt-5.3-codex` | 272K | balanced |
| GPT-5.5 | OpenAI | `gpt-5.5` | 272K / 1M | flagship |
| Grok Build 0.1 | xAI | `grok-build-0.1` | 256K | balanced |

> **Source:** [Cursor Docs — Models](https://cursor.com/docs) (mined 2026-06-09).
> Model availability and slugs may change. Use literal pass-through for models not listed here.
