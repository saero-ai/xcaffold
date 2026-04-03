# Contributing to xcaffold

First off, thank you for considering contributing to `xcaffold`! It's engineering leaders like you that make `xcaffold` a powerful, deterministic xcaffold ecosystem for everyone.

## 1. Getting Started

1. **Fork the repository** on GitHub.
2. **Clone your fork** locally.
3. **Run Setup**: Navigate into the `xcaffold` directory and run `make setup`. This will securely install the correct Git `pre-commit` hooks and download `golangci-lint` globally.

## 2. Core Architectural Mandates (One-Way Compilation)

When contributing code to the engine, you must adhere strictly to the Deterministic Target architecture.
- **Single Source of Truth**: The `scaffold.xcf` YAML file is the definitive state object.
- **No Bi-directional Syncs**: Do NOT introduce synchronization tools. The CLI is a one-way compiler (`.xcf` -> `.claude/` files). Modifications inside `.claude/` are designed to be explicitly overwritten.
- **Framework Independence**: The engine must remain native. Do not integrate deep legacy tools directly into the AST generation boundaries.

## 3. Submission Guidelines

- All Pull Requests must natively pass `make lint` and `make test`.
- We strictly enforce **Conventional Commits** (e.g. `feat(generator): ...`, `fix(cli): ...`, `docs(readme): ...`).
- If you touch the execution boundaries, ensure your logic does not introduce proprietary SDKs or API keys. Keep the CLI platform agnostic.
- Ensure any logical changes remain reflected in the `README.md` and `CHANGELOG.md` files structurally.
