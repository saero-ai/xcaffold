# Project

## Architecture

This project uses a compiler pipeline that transforms declarative configuration into provider-native output. The pipeline stages are: parse, validate, compile, render.

## Conventions

- All configuration uses YAML frontmatter in Markdown files
- Tests use the `TestCommand_Feature_Scenario` naming convention
- Error messages include context: `fmt.Errorf("parse %s: %w", path, err)`
- Functions stay under 50 lines; extract helpers for complex logic

## Development Workflow

1. Write the failing test
2. Implement minimal code to pass
3. Refactor without changing behavior
4. Run `make test` before committing
