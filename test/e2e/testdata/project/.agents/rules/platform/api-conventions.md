---
description: "API design conventions for the platform backend."
trigger: glob
globs: "packages/api/**/*.ts,packages/api/**/*.js"
---
## Endpoint Naming

Use plural nouns for resource collections (`/users`, `/projects`). Use kebab-case for multi-word paths (`/project-members`). Never use verbs in resource URLs — the HTTP method conveys the action.

## Request Validation

Validate all request bodies using DTOs with class-validator decorators. Return 400 with structured error details for invalid input. Never pass raw request bodies to service functions.

## Response Format

All responses use a consistent envelope:
```json
{
  "data": {},
  "meta": { "requestId": "..." }
}
```

Error responses include a machine-readable `code` field alongside the human-readable `message`.

## Authentication

All endpoints except health checks require a valid bearer token. Use the `@Auth()` guard decorator. Never access `req.user` directly — use the typed `@CurrentUser()` parameter decorator.
