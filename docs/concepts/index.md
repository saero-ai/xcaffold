# Concepts

Understanding-oriented explanations of xcaffold's architectural and philosophical foundations. These documents explain *why* the system is designed the way it is — not how to use it.

## Documents

- [Architecture](architecture.md) — Under the hood of the Xcaffold compiler engine
- [Declarative Compilation](declarative-compilation.md) — Why xcaffold uses a one-way deterministic compiler instead of bidirectional sync
- [Drift Detection and State](drift-detection-state.md) — How SHA-256 lock manifests provide deterministic state synchronization
- [Sandboxing](sandboxing.md) — Runtime sandbox configuration and compile-time evaluation sandbox
- [Multi-Target Rendering](multi-target-rendering.md) — How the AST enables the same configuration to compile to different platform formats
- [Configuration Scopes](configuration-scopes.md) — Understanding Implicit Global Inheritance and runtime scopes.
- [Best Practices](best-practices.md) — Configuration and structural best practices
