---
name: debugging
description: "Systematic debugging workflow for isolating root causes before applying fixes."
allowed-tools: [Bash, Read, Grep, Glob]
when-to-use: "Use when encountering any bug, test failure, or unexpected behavior."
disable-model-invocation: false
---
## Step 1: Reproduce

Reproduce the failure with a minimal, deterministic test case. If the failure is intermittent, identify the conditions that trigger it. Never guess — observe.

```bash
go test ./... -run TestFailing -v -count=1
```

## Step 2: Isolate

Narrow the scope using binary search. Comment out half the code path. Does the failure persist? Keep narrowing until you find the exact line or function responsible.

Use `grep` to trace function calls:
```bash
grep -rn "functionName" internal/ cmd/
```

## Step 3: Understand

Read the code around the failure point. Trace the data flow from input to the failure. Check recent changes with `git log -p --follow -- path/to/file.go`. Understand WHY it fails, not just WHERE.

## Step 4: Fix

Apply the minimal fix that addresses the root cause. Do not refactor, clean up, or add features in the same change. Write a regression test that would have caught the bug.

## Step 5: Verify

Run the failing test. Run the full test suite. Check for regressions in related functionality. Confirm the fix does not mask a deeper issue.
