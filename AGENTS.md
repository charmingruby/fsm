# AGENTS.md

> Generic finite state machine library for Go.

Small library that models workflows as states and transitions driven by
events. Keep it simple and keep the public API stable.

## Layout

- `fsm/` is the library core.
- `examples/` holds runnable demos and is a separate Go module.
- `Taskfile.yml` defines the official commands.

## Commands

Use `task` instead of raw `go` commands when available:

- `task test`: run all tests with race detector (same as CI).
- `task lint`: format check + lint. Run before finishing work.
- `task lint-fix`: lint with auto-fix.
- `task release V=X.Y.Z`: tag and push a release.

For focused work you can still use `go test` on a single package and
`go run` inside `examples/` for manual checks.

## Working style

1. Read the core package and its tests before changing behavior.
2. Prefer small, backward-compatible changes. The builder-style API is
   the public contract: avoid breaking it.
3. Update the example or docs when behavior changes.
4. Always finish with `task test` and `task lint`.

## Tests

Tests are **table-driven with testify**:

- Add new behavior as a new table case, not a new test function.
- Cover happy path, failure path, and edge cases (no transition,
  fallback, canceled context).
- Use `require` for blocking assertions and `assert` for the rest.
- Keep tests parallel and hermetic (stub loggers / stores, no stdout).
- On error cases, also assert the partial output, not just the error.

## Docs and style

- Public symbols need clear godoc comments.
- Follow standard Go style: `gofmt`, short functions, no unnecessary
  new files or dependencies.
- If the linter complains, fix the code: don't add suppressions by default.
