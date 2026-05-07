# Documentation Impact Analysis

## Date: 2026-05-07
## Source Changes Since Last Ground Truth Verification (May 1-2)

### Change Summary

- **361 source files** changed across cmd/, internal/, and providers/
- **Major change areas:** xcaf extension migration, project variables, parser refactoring, ClearableList, manifest-driven provider config, field model, blueprint targets
- **All documentation is impacted** — the xcaf migration alone touches every code example and file reference

### High-Impact Changes by Area

| Change Area | Source Files | Impacted Doc Categories | Priority |
|-------------|-------------|------------------------|----------|
| xcaf extension migration | 183+ files renamed | ALL docs with file references | P1 |
| Project variables system | parser/variables.go, resolver/attribute.go | concepts/config, ref/commands (apply, init, validate) | P1 |
| Two-layer field model | ast/types.go (+xcf:role= markers) | concepts/config/field-model, ref/kinds/* | P1 |
| ClearableList merge semantics | ast/types.go, compiler/override_merge* | concepts/config/layer-precedence, ref/kinds/* | P1 |
| Manifest-driven provider config | providers/registry.go, providers/*.go | concepts/arch/provider-architecture, ref/supported-providers | P1 |
| Blueprint targets | ast/types.go (BlueprintConfig.Targets) | ref/kinds/xcaffold/blueprint, best-practices/blueprint-design | P2 |
| Parser refactoring (5 modules) | parser/*.go (split from monolith) | concepts/arch/architecture, concepts/arch/translation-pipeline | P2 |
| Help --xcaf enhancements | cmd/xcaffold/help.go | ref/commands/utility/help | P2 |
| Validate command changes | cmd/xcaffold/validate.go | ref/commands (lifecycle or utility) | P2 |
| Catalog validation model | parser/parser_validation.go | concepts/config/declarative-compilation | P2 |
| Skill artifacts field | parser/skill_validation.go | ref/kinds/provider/skill, best-practices/skill-organization | P2 |
| kind:context formalization | ast/types.go (ContextConfig) | ref/kinds/provider/context, best-practices/workspace-context | P2 |

### Feature Coverage Gaps (from feature-coverage.json)

| Feature | Type | Current Coverage | Action Needed |
|---------|------|-----------------|---------------|
| registry command | command | missing | Create reference page (already created as utility/registry.md) |
| memory kind | kind | partial | Upgrade to full reference (already created as provider/memory.md) |
| variables concept | config | new | Document (already created as concepts/config/variables.md by develop) |

### Terminology Migration Status

The xcaf migration renamed all .xcf references to .xcaf across the codebase. Documentation must be verified for:
- File extension references (.xcaf not .xcf)
- Directory references (xcaf/ not xcf/)
- Schema references (*.xcaf not *.xcf)
- Command examples (all must use .xcaf)
