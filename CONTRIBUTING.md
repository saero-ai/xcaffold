# Contributing to xcaffold

Thank you for considering contributing to xcaffold. This guide covers everything you need to get a pull request merged.

## 1. Getting Started

1. **Fork the repository** on GitHub.
2. **Clone your fork** locally.
3. **Run Setup**: Navigate into the `xcaffold` directory and run `make setup`. This will securely install the correct Git `pre-commit` hooks and download `golangci-lint` globally.

## 2. Core Architectural Mandates (One-Way Compilation)

When contributing code to the engine, you must adhere strictly to the Deterministic Target architecture.
- **Single Source of Truth**: The `project.xcf` file is the definitive state object.
- **No Bi-directional Syncs**: Do NOT introduce synchronization tools. The CLI is a one-way compiler (`.xcf` → provider-specific output directories). Modifications inside generated files are designed to be explicitly overwritten on re-compilation.
- **Framework Independence**: The engine must remain native. Do not integrate deep legacy tools directly into the AST generation boundaries.

## 3. Submission Guidelines

- All Pull Requests must natively pass `make lint` and `make test`.
- We strictly enforce **Conventional Commits** (e.g. `feat(generator): ...`, `fix(cli): ...`, `docs(readme): ...`).
- If you touch the execution boundaries, ensure your logic does not introduce proprietary SDKs or API keys. Keep the CLI platform agnostic.
- Ensure any logical changes remain reflected in the appropriate `docs/` pillar and `CHANGELOG.md`.

## 4. Documentation Protocol

Documentation is structured using the **Diátaxis framework** under `docs/`.
When adding or updating documentation:
1. Documentation MUST be nested in the correct pillar (`tutorials/`, `how-to/`, `concepts/`, `reference/`).
2. Every `.md` file MUST contain the standard YAML frontmatter:
   ```yaml
   ---
   title: "Your Title"
   description: "Brief summary"
   ---
   ```
3. Do NOT add monolithic reference content directly into `README.md`. It must remain a lightweight overview with links pointing to the `docs/` pillars.

## 5. Adding a New Provider Renderer

Adding a target platform is the highest-leverage community contribution. A renderer maps the AST to a target's native file format.

1. Create a new package under `internal/renderer/<target>/`
2. Implement the `renderer.TargetRenderer` interface (`Render`, `OutputDir`, `LockSuffix`)
3. Register the renderer in `internal/renderer/registry.go`
4. Add test fixtures in `internal/renderer/<target>/<target>_test.go`
5. Run `make test` to verify

See `internal/renderer/claude/` for a reference implementation.
