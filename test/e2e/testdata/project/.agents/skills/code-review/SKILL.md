---
name: code-review
description: "Structured code review process covering correctness, security, and performance."
---
## Before Reviewing

Pull the latest changes and read the PR description. Understand the intent before reading the code. Check if tests exist and pass.

## Review Dimensions

### Correctness
- Does the code do what the PR description claims?
- Are edge cases handled (empty input, nil pointers, boundary values)?
- Do error paths return meaningful errors, not silent failures?

### Security
- Is user input validated before use?
- Are SQL queries parameterized?
- Are file paths sanitized against traversal attacks?
- Are secrets kept out of logs and error messages?

### Performance
- Are there N+1 query patterns in database access?
- Are large collections processed with streaming, not in-memory loading?
- Are expensive operations (HTTP calls, DB queries) cached where appropriate?

### Maintainability
- Are function names self-documenting?
- Is there unnecessary abstraction or premature generalization?
- Would a new team member understand this code without external context?

## Output

For each finding, provide: file path, line number, severity (error/warning/info), description, and a concrete suggestion. Group findings by file.
