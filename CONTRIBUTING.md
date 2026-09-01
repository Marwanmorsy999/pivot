# Contributing to PIVOT

Thank you for your interest in contributing to PIVOT! This document provides guidelines for contributing to the project.

## Code of Conduct

By participating in this project, you are expected to uphold a respectful and inclusive environment.

## How to Contribute

### Reporting Bugs

1. Check if the bug has already been reported in [Issues](https://github.com/Marwanmorsy999/pivot/issues).
2. If not, open a new issue using the Bug Report template.
3. Include as much detail as possible: steps to reproduce, expected behavior, environment info.

### Suggesting Features

1. Open a new issue using the Feature Request template.
2. Clearly describe the problem and your proposed solution.
3. Be open to discussion and feedback from maintainers.

### Pull Requests

1. Fork the repository and create your branch from `main`.
2. Make your changes in a focused, single-purpose branch.
3. Write or update tests for any changed functionality.
4. Run `go test ./...` and `go test -race ./...` locally.
5. Run `go fmt ./...` and `go vet ./...` before opening the PR.
6. Run `golangci-lint run ./...` when available.
7. Update documentation if needed (README, CHANGELOG, etc.).
8. Open a pull request using the PR template.

## Development Setup

### Prerequisites

- Go 1.21 or later
- Git
- A C compiler/toolchain (required by `github.com/mattn/go-sqlite3`)

### Getting Started

```bash
# Clone the repository
git clone https://github.com/Marwanmorsy999/pivot.git
cd pivot

# Install dependencies
go mod tidy

# Run tests
go test ./...

# Build the CLI
go build -o pivot ./cmd/pivot
```

### Project Structure

```
pivot/
├── cmd/
│   └── pivot/
│       └── main.go           # CLI entry point
├── internal/
│   ├── config/               # Configuration management
│   ├── core/                 # Graph, executor, orchestrator
│   ├── cost/                 # Token/cost estimation
│   ├── planner/              # AI planners (Ollama, OpenAI)
│   ├── state/                # SQLite persistence
│   ├── tui/                  # Terminal UI
│   └── worktree/             # Git worktree isolation
├── docs/                     # Architecture, roadmap, proofs
├── examples/                 # Usage examples
└── scripts/                  # Development/release helpers
```

## Coding Standards

- Follow standard Go conventions and idioms
- Use `gofmt` for code formatting
- Write meaningful commit messages
- Add tests for new functionality
- Document exported functions and types
- Handle errors properly — never silently ignore them

## Testing

```bash
# Run all tests
go test ./...

# Run tests with race detector
go test -race ./...

# Run tests with coverage
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out

# Run tests for a specific package
go test ./internal/core/...
```

## Commit Message Format

Use clear, descriptive commit messages:

```
<type>: <description>

[optional body]
```

Types:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `test`: Adding or updating tests
- `refactor`: Code refactoring
- `chore`: Maintenance tasks

Example:

```
feat: add support for Anthropic Claude planner

Implements a new planner using the Anthropic API with
proper error handling and timeout configuration.
```

## Branch and Release Process

Use a lightweight GitHub Flow:

1. Create a focused feature or fix branch from `main`.
2. Open a pull request and wait for CI/security checks to pass.
3. Merge using a single linear history where practical.
4. Create version tags in the form `vX.Y.Z` for releases.
5. Pushing a `v*` tag triggers the release workflow.

Before releasing, update `CHANGELOG.md` and verify the release artifacts and SBOM.

## Questions?

Feel free to open an issue for any questions or concerns.
