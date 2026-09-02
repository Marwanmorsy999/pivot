# Contributing to PIVOT

Thank you for contributing to PIVOT. The project favors small, reviewable changes with explicit tests and a clean automated validation trail.

## Before You Start

For bugs and feature requests, check existing [issues](https://github.com/Marwanmorsy999/pivot/issues) first. Security vulnerabilities must be reported privately through the process in [`SECURITY.md`](SECURITY.md).

## Development Setup

### Prerequisites

- Go `1.21+` for the module baseline.
- A working C compiler/toolchain for SQLite (`github.com/mattn/go-sqlite3`).
- Git.

Clone the repository and verify the working tree:

```bash
git clone https://github.com/Marwanmorsy999/pivot.git
cd pivot
git status
```

Download dependencies and run the test suite:

```bash
go mod download
go test ./...
go test -race ./...
```

Build the CLI and verify it starts:

```bash
go build -o pivot ./cmd/pivot
./pivot --help
```

## Coding Standards

Use idiomatic Go and keep changes focused. Format code with `gofmt`, handle errors explicitly, avoid unnecessary global state, and add tests for behavior that changes or is introduced.

Before opening a pull request, run the checks that can be reproduced locally:

```bash
gofmt -w .
go vet ./...
go test -race ./...
go build ./...
```

Run `golangci-lint` locally when available:

```bash
golangci-lint run ./...
```

## Pull Requests

1. Create a focused branch from `main`.
2. Keep the change set small and explain the motivation in the PR description.
3. Add or update tests for functional changes.
4. Update documentation when user-visible behavior, configuration, or development workflow changes.
5. Check that CI, security scanning, and CodeQL are green before merge.
6. Do not merge around a failing required check; investigate the failure or document a verified infrastructure issue.

## Commit Messages

Use a clear conventional-style prefix:

```text
feat: add parallel task execution
fix: handle planner timeout errors
docs: clarify configuration setup
test: cover worktree cleanup
refactor: simplify task scheduling
chore: update build tooling
```

Keep the subject concise and describe the user/developer impact where useful.

## Branch and Release Process

PIVOT uses GitHub Flow:

```text
main
  └── focused branch → pull request → required checks → merge
```

Do not rewrite the history of `main`. Release versions are created from Git tags matching `v*`; the release workflow then tests, builds, smoke-tests, generates an SPDX SBOM, computes SHA-256 checksums, and publishes the artifacts.

## Testing Expectations

At minimum, functional changes should pass:

```bash
go test ./...
go test -race ./...
go build ./...
```

Changes involving concurrency, state persistence, command execution, networking, or security-sensitive behavior should include targeted regression tests.

## Documentation

Keep these project documents synchronized with reality:

- [`README.md`](README.md) — user-facing overview and quick start.
- [`SECURITY.md`](SECURITY.md) — security reporting and operational guidance.
- [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) — architecture reference.
- [`docs/ROADMAP.md`](docs/ROADMAP.md) — planned work.
- [`docs/VERIFICATION.md`](docs/VERIFICATION.md) — dated verification evidence.
- [`CHANGELOG.md`](CHANGELOG.md) — notable user-facing changes.

## Code Review Standard

Reviewers should verify correctness, test coverage, backward compatibility where relevant, security implications, and documentation impact. A clean diff and green automation are part of the definition of done.

## Questions

Open a discussion through GitHub Issues for project questions that are not security-sensitive. Be specific about the command, environment, expected behavior, and observed behavior when reporting a problem.
