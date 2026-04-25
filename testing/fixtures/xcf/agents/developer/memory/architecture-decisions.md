xcaffold uses a one-way compiler model. project.xcf is the source of truth.
The .claude/ directory is compiler output and must never be edited directly.
SHA-256 drift detection is performed via .xcaffold/project.xcf.state on every apply.
