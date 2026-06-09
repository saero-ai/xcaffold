# Model Resolution

xcaffold resolves model identifiers when compiling agent and skill configurations. Each provider has a model resolver that maps short aliases to canonical model IDs.

## How It Works

When a `.xcaf` file specifies `model: flash`, xcaffold resolves this alias to the provider's canonical model ID at compile time. Unknown aliases pass through unchanged — the provider's runtime handles validation.

The resolution order:
1. Check provider-specific alias map
2. If matched, emit the canonical ID
3. If unmatched, pass the original string through

## Provider Model Maps

### Claude

| Alias | Resolves To |
|-------|------------|
| `sonnet` | `claude-sonnet-4-6` |
| `opus` | `claude-opus-4-6` |
| `haiku` | `claude-haiku-4-5` |

Default model: `claude-sonnet-4-6`

### Antigravity 2.0

> Antigravity 2.0 supports multi-vendor model selection — Gemini, Claude, and GPT-OSS models are all valid targets.

| Alias | Resolves To | Tier |
|-------|------------|------|
| `flash` | `gemini-3.5-flash` | Gemini |
| `pro` | `gemini-3.1-pro-high` | Gemini |
| `pro-low` | `gemini-3.1-pro-low` | Gemini |
| `sonnet-thinking` | `claude-sonnet-4-6-thinking` | Claude |
| `opus-thinking` | `claude-opus-4-6-thinking` | Claude |
| `gpt-oss` | `gpt-oss-120b` | GPT-OSS |

Full model IDs (pass through unchanged): `gemini-3.5-flash`, `gemini-3.1-pro-high`, `gemini-3.1-pro-low`, `gemini-3-flash`, `claude-sonnet-4-6-thinking`, `claude-opus-4-6-thinking`, `gpt-oss-120b`, `nano-banana-2`.

Default model: `gemini-3.5-flash`

### Antigravity (v1)

> **Deprecated.** The `antigravity` target is deprecated in favor of `antigravity2` (Antigravity 2.0). Existing configurations continue to work but new projects should use `antigravity2`. See [Supported Providers](supported-providers.md).

| Alias | Resolves To |
|-------|------------|
| `pro` | `gemini-2.5-pro` |

Default model: `gemini-2.5-pro`

### Gemini CLI

| Alias | Resolves To |
|-------|------------|
| `flash` | `gemini-2.5-flash` |
| `pro` | `gemini-2.5-pro` |

Default model: `gemini-2.5-flash`

### Cursor

| Alias | Resolves To |
|-------|------------|
| `auto` | `cursor/auto` |

Default model: pass-through (Cursor manages its own model selection).

### GitHub Copilot

| Alias | Resolves To |
|-------|------------|
| `gpt-4o` | `gpt-4o` |
| `claude-sonnet` | `claude-3.5-sonnet` |

Default model: pass-through.

### Codex

Codex uses a fixed model per agent (`model` field in `.codex/agents/*.toml`). No alias resolution — all model strings pass through unchanged.

## Fidelity Notes

When a model alias resolves differently across providers, xcaffold emits a `MODEL_ALIAS_RESOLVED` fidelity note showing the resolved ID. This helps identify cases where the same alias (e.g., `pro`) maps to different models on different providers.
