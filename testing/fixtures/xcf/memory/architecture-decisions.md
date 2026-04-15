xcaffold uses a one-way compiler model. scaffold.xcf is the source of truth.
The .claude/ directory is compiler output and must never be edited directly.
SHA-256 drift detection is performed via scaffold.lock on every apply.
