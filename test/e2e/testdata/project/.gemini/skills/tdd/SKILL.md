---
name: tdd
description: "Test-driven development workflow using the Red-Green-Refactor cycle."
---
## Red Phase

Write a test that describes the expected behavior. The test must fail before any implementation code is written. Run the test and confirm the failure message is clear and specific.

```bash
go test ./path/to/package -run TestNew -v
```

Verify the test fails with a meaningful message, not a compile error.

## Green Phase

Write the minimal code required to make the failing test pass. Do not add extra functionality, error handling for hypothetical cases, or premature abstractions. Run the test and confirm it passes.

## Refactor Phase

Improve the code structure without changing behavior. Extract functions, rename variables, remove duplication. Run the full test suite after each refactoring step to ensure nothing breaks.

## Rules

- Never skip the red phase. If the test passes before implementation, the test is wrong.
- One assertion per test when possible. Multiple assertions obscure which behavior failed.
- Test names describe the scenario: `TestCompile_MissingField_ReturnsError`.
- Use `t.Helper()` in all test helper functions so failures report the caller's line number.
