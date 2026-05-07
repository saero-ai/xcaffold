# XCAF Golden Manifests

Reference `.xcaf` files that exercise every field for each resource kind.

## Purpose

1. **User reference** — browse these files to see every valid field for a resource kind
2. **CI validation** — `golden_test.go` validates all files parse without error
3. **Schema drift detection** — when a field is removed or renamed in `types.go`, CI breaks here

## Coverage

- **Full coverage** kinds (agent, skill, rule, workflow, mcp, hooks, project, global, settings): every valid field is exercised
- **Preview** kinds (blueprint, policy): parser support is complete but full field documentation is pending
- **Preview — parser pending** kinds (template, system): parser support for these kinds is not yet implemented; CI skips them

## Maintenance

When you add a field to `internal/ast/types.go`:
1. Add it to the corresponding golden manifest
2. Run `go test ./schema/golden/... -v` to verify

When you remove or rename a field:
1. CI will fail because the golden manifest references the old field
2. Update the golden manifest to match the new schema
