---
title: "Model Resolution"
description: "How xcaffold resolves model tier aliases and literal model IDs per provider"
---

# Model Resolution

xcaffold supports a `model` field on agents that accepts either a **tier alias** (`balanced`, `flagship`, `fast`) or a **literal model ID** (e.g., `gpt-5.5`, `claude-sonnet-4-6`). Each provider's model resolver translates these into the correct output value.

## Tier Aliases

Tier aliases let you write provider-agnostic `.xcaf` files. Each provider maps the alias to its recommended model for that tier.

| Alias | Intent | Claude Code | Cursor | Copilot | Gemini CLI | Codex | Antigravity |
|-------|--------|-------------|--------|---------|------------|-------|-------------|
| `balanced` | General-purpose default | `sonnet` | `claude-sonnet-4-6` | `claude-sonnet-4-6` | `gemini-3.5-flash` | `gpt-5.4` | `gemini-3.5-flash` |
| `flagship` | Most capable model | `opus` | `gpt-5.5` | `claude-opus-4-8` | `gemini-2.5-pro` | `gpt-5.5` | `gemini-3.1-pro-high` |
| `fast` | Fastest / cheapest | `haiku` | `composer-2.5` | `claude-haiku-4-5` | `gemini-3.5-flash` | `gpt-5.4-mini` | `gemini-2.5-flash` |

Claude Code uses bare aliases (`sonnet`, `opus`, `haiku`) that resolve at runtime to the latest version. This means the compiled output always targets the current model without needing resolver updates.

> **Last verified:** 2026-08-23 against official provider documentation.
> Tier mappings are defaults. Override them per-provider using `model-tier-<alias>` entries in `project.<provider>.vars`, or use `agent.<provider>.xcaf` overrides for individual agents.

## Literal Model IDs (Pass-Through)

When the `model` field contains a literal model ID instead of a tier alias, each provider checks whether the ID matches a known prefix family and passes it through unchanged.

| Provider | Accepted Prefixes / Pass-Through |
|----------|----------------------------------|
| Claude Code | `claude-`, bare aliases (`sonnet`, `opus`, `haiku`) |
| Cursor | `claude-`, `gpt-`, `gemini-`, `cursor-`, `composer-`, `o1-`, `o3-`, `grok-`, `kimi-` |
| Copilot | `claude-`, `gpt-` |
| Gemini CLI | `gemini-` |
| Codex | `gpt-` |
| Antigravity | Multi-vendor model IDs, short aliases, and pass-through strings — see [Antigravity Models](#antigravity-models) |

Pass-through is case-insensitive. `Claude-Sonnet-4-6` resolves to `claude-sonnet-4-6`.

Unrecognized model IDs (no matching prefix) produce an `AGENT_MODEL_UNMAPPED` warning and are omitted from output.

## Antigravity Models

> Antigravity supports multi-vendor model selection across Gemini, Claude, and GPT-OSS tiers.

| Alias | Resolves To | Vendor |
|-------|------------|--------|
| `flagship` | `gemini-3.1-pro-high` | Gemini |
| `balanced` | `gemini-3.5-flash` | Gemini |
| `fast` | `gemini-2.5-flash` | Gemini |
| `flash-3.7` / `flash-latest` | `gemini-3.7-flash` | Gemini |
| `flash-3.6` | `gemini-3.6-flash` | Gemini |
| `flash` | `gemini-3.5-flash` | Gemini |
| `pro` | `gemini-3.1-pro-high` | Gemini |
| `pro-low` | `gemini-3.1-pro-low` | Gemini |
| `sonnet-thinking` | `claude-sonnet-4-6-thinking` | Claude |
| `opus-thinking` | `claude-opus-4-6-thinking` | Claude |
| `gpt-oss` | `gpt-oss-120b` | GPT-OSS |

Full model IDs accepted directly: `gemini-3.7-flash`, `gemini-3.6-flash`, `gemini-3.5-flash`, `gemini-3.1-pro-high`, `gemini-3.1-pro-low`, `gemini-3.1-pro`, `gemini-3-flash`, `gemini-2.5-pro`, `gemini-2.5-flash`, `claude-sonnet-4-6-thinking`, `claude-opus-4-6-thinking`, `gpt-oss-120b`, `nano-banana-2`.

Default model: `gemini-3.1-pro`.

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

## User-Configurable Tier Mapping

To override what a tier alias resolves to for a specific provider, add `model-tier-<alias>` entries to the target's variable file:

`xcaf/project.cursor.vars`:
```ini
model-tier-balanced = composer-2.5-custom
model-tier-flagship = gpt-5.5
model-tier-fast = composer-2.5-turbo
```

When present, these override the built-in alias map for the compilation target. Any agent using `model: balanced` compiles to `composer-2.5-custom` on Cursor instead of the default. When absent, the built-in defaults apply.

This is useful when a team wants to standardize model selection across all agents without per-agent overrides or variable references.

| Variable | Overrides Alias |
|----------|----------------|
| `model-tier-balanced` | `balanced` |
| `model-tier-flagship` | `flagship` |
| `model-tier-fast` | `fast` |

Tier overrides apply only to the three canonical aliases. Literal model IDs and provider-native aliases are unaffected.

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

## Antigravity Model Catalog

Available models in Antigravity (CLI, IDE, and runtime) as of 2026-08-23:

| Model | Provider | Model ID | Context | Tier / Capabilities |
|-------|----------|----------|---------|---------------------|
| Gemini 3.7 Flash | Google | `gemini-3.7-flash` | 1M tokens | default, reasoning (thinking) |
| Gemini 3.6 Flash | Google | `gemini-3.6-flash` | 1M tokens | fast, reasoning (thinking) |
| Gemini 3.5 Flash | Google | `gemini-3.5-flash` | 1M tokens | balanced / fast, thinking |
| Gemini 3.1 Pro (high) | Google | `gemini-3.1-pro-high` | 1M tokens | flagship, deep reasoning |
| Gemini 3.1 Pro (low) | Google | `gemini-3.1-pro-low` | 1M tokens | flagship, low thinking |
| Gemini 3 Flash | Google | `gemini-3-flash` | 1M tokens | fast |
| Gemini 2.5 Pro | Google | `gemini-2.5-pro` | 1M tokens | flagship |
| Gemini 2.5 Flash | Google | `gemini-2.5-flash` | 1M tokens | fast, efficient |
| Claude Sonnet 4.6 (thinking) | Anthropic | `claude-sonnet-4-6-thinking` | 200K tokens | reasoning, extended thinking |
| Claude Opus 4.6 (thinking) | Anthropic | `claude-opus-4-6-thinking` | 200K tokens | flagship reasoning |
| GPT-OSS-120b | Open Source | `gpt-oss-120b` | 128K tokens | open-source reasoning |
| Nano Banana 2 | Internal | `nano-banana-2` | N/A | image generation |

> **Source:** Ground Truth Model Registry (verified 2026-08-23).
> Model availability and aliases are supported across Antigravity CLI (`agy`), Antigravity IDE, and Antigravity 2.0. Unlisted or custom model IDs pass through directly.

