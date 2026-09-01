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
4. Ensure all tests pass: `go test ./...`
5. Run `go fmt ./...` to format your code.
6. Update documentation if needed (README, CHANGELOG, etc.).
7. Open a pull request using the PR template.

## Development Setup

### Prerequisites

- Go 1.21 or later
- Git

### Getting Started

```bash
# Clone the repository
git clone https://github.com/Marwanmorsy999/pivot.git
cd pivot

# Install dependencies
go mod tidy

# Run tests
go test ./...

# Build
go build -o pivot .
```

### Project Structure

```
pivot/
├── main.go                 # CLI entry point
├── internal/
│   ├── config/             # Configuration management
│   ├── core/               # Graph, executor, orchestrator
│   ├── cost/               # Token/cost estimation
│   ├── planner/            # AI planners (Ollama, OpenAI)
│   ├── state/              # SQLite persistence
│   ├── tui/                # Terminal UI
│   └── worktree/           # Git worktree isolation
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

## Release Process

1. Update CHANGELOG.md with the new version's changes
2. Create a git tag: `git tag v2.1.0`
3. Push the tag: `git push origin v2.1.0`
4. GitHub Actions will build and create a release

## Questions?

Feel free to open an issue for any questions or concerns.
