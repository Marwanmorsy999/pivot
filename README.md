<p align="center">
  <img src="assets/banner.svg" alt="PIVOT — Hybrid CLI Orchestrator" width="800"/>
</p>

<p align="center">
  <strong>Hybrid CLI orchestration for AI agents and traditional command-line tools.</strong>
</p>

<p align="center">
  <a href="https://github.com/Marwanmorsy999/pivot/actions/workflows/ci.yml"><img src="https://github.com/Marwanmorsy999/pivot/actions/workflows/ci.yml/badge.svg" alt="CI"/></a>
  <a href="https://github.com/Marwanmorsy999/pivot/actions/workflows/codeql.yml"><img src="https://github.com/Marwanmorsy999/pivot/actions/workflows/codeql.yml/badge.svg" alt="CodeQL"/></a>
  <a href="https://goreportcard.com/report/github.com/Marwanmorsy999/pivot"><img src="https://goreportcard.com/badge/github.com/Marwanmorsy999/pivot" alt="Go Report Card"/></a>
  <a href="https://pkg.go.dev/github.com/Marwanmorsy999/pivot"><img src="https://pkg.go.dev/badge/github.com/Marwanmorsy999/pivot.svg" alt="PkgGoDev"/></a>
  <a href="https://github.com/Marwanmorsy999/pivot/releases"><img src="https://img.shields.io/github/v/release/Marwanmorsy999/pivot?include_prereleases" alt="Latest release"/></a>
  <a href="https://opensource.org/licenses/MIT"><img src="https://img.shields.io/badge/License-MIT-yellow.svg" alt="MIT License"/></a>
</p>

> **Project status:** active development. The latest published release is `v0.1.0-rc.1`. The `main` branch is continuously validated by automated testing, linting, build checks, dependency vulnerability scanning, and CodeQL.

## Overview

PIVOT is a Go-based CLI orchestrator that combines AI agents with traditional command-line tools inside one dependency-aware workflow. It is designed for repeatable, inspectable automation rather than one-off shell commands.

The core execution model is a task graph: tasks can call normal CLI tools or supported AI providers, declare dependencies, and persist session state so work can be inspected and resumed.

## What PIVOT Provides

| Capability | What it does |
| --- | --- |
| **Hybrid execution** | Mix AI agents and conventional CLI commands in one workflow. |
| **Dependency graph** | Order tasks by explicit dependencies and execute them topologically. |
| **Terminal UI** | Monitor sessions, tasks, events, and execution progress in a Bubble Tea TUI. |
| **Worktree isolation** | Run agent work in isolated Git worktrees to reduce accidental repository changes. |
| **Session persistence** | Store execution state and journals in SQLite and resume interrupted work. |
| **Cost tracking** | Track token usage and estimated AI costs when provider metadata is available. |
| **Multi-provider planning** | Support Ollama, OpenAI-compatible endpoints, Groq, Gemini, and Anthropic workflows. |

## Quick Start

### Build from source

PIVOT requires Go `1.21+` and a working C toolchain because it uses `github.com/mattn/go-sqlite3`.

```bash
git clone https://github.com/Marwanmorsy999/pivot.git
cd pivot
go mod download
go build -o pivot ./cmd/pivot
```

### Initialize and run

```bash
./pivot detect
./pivot init
./pivot run "Find all TODO comments in the codebase and summarize the highest-priority items"
./pivot status
```

### Install with Go

```bash
go install github.com/Marwanmorsy999/pivot/cmd/pivot@latest
```

### Docker

```bash
docker build -t pivot .
docker run -it --rm -v ~/.pivot:/root/.pivot pivot run "your goal"
```

For macOS/Linux, the repository also includes an install script at [`scripts/install.sh`](scripts/install.sh).

## Configuration

PIVOT reads configuration from `~/.pivot/config.yaml` by default.

```yaml
planner:
  provider: ollama
  model: llama3.2:3b
  api_key: ""
  endpoint: http://localhost:11434

worktree:
  enabled: false
  base_dir: ""

cost:
  enabled: true
```

See [`config.example.yaml`](config.example.yaml) for the complete example.

## Architecture

```text
pivot/
├── cmd/pivot/            # CLI entry point
├── internal/
│   ├── config/           # Configuration management
│   ├── state/            # SQLite-backed session and journal storage
│   ├── planner/          # AI planner integrations
│   ├── core/             # Tasks, dependency graph, executor, orchestrator
│   ├── tui/              # Bubble Tea terminal UI
│   ├── worktree/         # Git worktree isolation
│   └── cost/             # Token/cost estimation
├── scripts/              # Build and installation helpers
├── examples/             # Example workflows
├── assets/               # Branding assets
├── docs/                 # Architecture, roadmap, and verification records
├── .github/              # CI, CodeQL, release automation, and templates
├── Dockerfile
├── Makefile
├── LICENSE
├── CONTRIBUTING.md
├── SECURITY.md
└── CHANGELOG.md
```

## Task Model

The planner produces a dependency graph that can mix tool and agent tasks:

```json
{
  "tasks": [
    {
      "id": "a",
      "type": "tool",
      "tool": "grep",
      "args": ["-r", "TODO", "."],
      "depends_on": [],
      "description": "Find TODO comments"
    },
    {
      "id": "b",
      "type": "agent",
      "tool": "ollama",
      "args": ["--prompt", "Summarize $OUTPUT into actionable items"],
      "depends_on": ["a"],
      "description": "Summarize findings with AI"
    }
  ]
}
```

## Common Workflows

```bash
# Code review
pivot run "Review the last commit for bugs and security issues"

# Bug investigation
pivot run "Find the root cause of the null pointer exception in auth module"

# Feature implementation
pivot run "Implement rate limiter middleware with configurable limits"

# Documentation generation
pivot run "Generate API documentation for exported functions"

# Security audit
pivot run "Audit the codebase for SQL injection and command injection risks"
```

More examples are available in [`examples/`](examples/), including [`examples/basic-usage.md`](examples/basic-usage.md).

## Development & Quality Gates

The repository treats automated validation as a required part of normal development.

```bash
# Unit tests
go test ./...

# Race detection
go test -race ./...

# Build
go build ./...

# Formatting
gofmt -w .

# Static analysis
go vet ./...
```

GitHub Actions also runs the project's CI matrix against Go `1.26` and `1.27`, with linting, build validation, `gosec`, and `govulncheck`. CodeQL runs independently for static security analysis.

See [`docs/VERIFICATION.md`](docs/VERIFICATION.md) for a dated verification record with direct links to the GitHub checks.

## Releases

PIVOT currently publishes release candidates through Git tags. The release workflow builds platform binaries, smoke-tests them, generates an SPDX SBOM, computes SHA-256 checksums, and attaches the resulting artifacts to the GitHub release.

The latest published release is [`v0.1.0-rc.1`](https://github.com/Marwanmorsy999/pivot/releases/tag/v0.1.0-rc.1).

## Security

PIVOT is security-conscious by design, but it executes commands and can invoke AI-generated task plans. Review generated work before execution when appropriate, prefer worktree isolation for agent-driven changes, and run the CLI with the minimum permissions necessary.

Security-sensitive reports should follow [`SECURITY.md`](SECURITY.md). Do not publish undisclosed vulnerabilities in normal GitHub issues.

## Contributing

Contributions are welcome. Start with [`CONTRIBUTING.md`](CONTRIBUTING.md), use the issue/PR templates, keep changes focused, and wait for the required checks to pass before merge.

## Documentation

- [`CONTRIBUTING.md`](CONTRIBUTING.md) — contributor workflow and quality expectations
- [`SECURITY.md`](SECURITY.md) — vulnerability reporting and security practices
- [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) — system architecture
- [`docs/ROADMAP.md`](docs/ROADMAP.md) — planned work
- [`docs/VERIFICATION.md`](docs/VERIFICATION.md) — evidence-based repository verification
- [`CHANGELOG.md`](CHANGELOG.md) — notable changes

## License

PIVOT is licensed under the MIT License. See [`LICENSE`](LICENSE).

<p align="center">
  <sub>Built in Go by <a href="https://github.com/Marwanmorsy999">Marwan Morsy</a>.</sub>
</p>
