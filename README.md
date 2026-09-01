<p align="center">
  <img src="assets/banner.svg" alt="PIVOT — Hybrid CLI Orchestrator" width="800"/>
</p>

<p align="center">
  <strong>A universal hybrid CLI orchestrator that combines AI agents and traditional CLI tools in a single workflow</strong>
</p>

<p align="center">
  <a href="https://github.com/Marwanmorsy999/pivot/actions/workflows/ci.yml"><img src="https://github.com/Marwanmorsy999/pivot/actions/workflows/ci.yml/badge.svg" alt="CI"/></a>
  <a href="https://github.com/Marwanmorsy999/pivot/actions/workflows/codeql.yml"><img src="https://github.com/Marwanmorsy999/pivot/actions/workflows/codeql.yml/badge.svg" alt="CodeQL"/></a>
  <a href="https://goreportcard.com/report/github.com/Marwanmorsy999/pivot"><img src="https://goreportcard.com/badge/github.com/Marwanmorsy999/pivot" alt="Go Report Card"/></a>
  <a href="https://pkg.go.dev/github.com/Marwanmorsy999/pivot"><img src="https://pkg.go.dev/badge/github.com/Marwanmorsy999/pivot.svg" alt="PkgGoDev"/></a>
  <a href="https://opensource.org/licenses/MIT"><img src="https://img.shields.io/badge/License-MIT-yellow.svg" alt="License: MIT"/></a>
  <a href="https://github.com/Marwanmorsy999/pivot/releases"><img src="https://img.shields.io/github/v/release/Marwanmorsy999/pivot?include_prereleases" alt="Release"/></a>
  <a href="https://github.com/Marwanmorsy999/pivot/stargazers"><img src="https://img.shields.io/github/stars/Marwanmorsy999/pivot" alt="GitHub Stars"/></a>
</p>

---

## Why PIVOT?

PIVOT bridges the gap between AI agents and traditional command-line tools. Instead of choosing between automation and intelligence, get both — orchestrated through a single dependency graph with real-time visibility.

## Features

| Feature | Description |
|---------|-------------|
| **Hybrid Execution** | Seamlessly mix AI agents (Ollama, Claude Code, Gemini CLI) with traditional CLI tools (grep, jq, curl, etc.) |
| **Real-time TUI** | Beautiful terminal UI with live task tracking, cost monitoring, and event logging |
| **Worktree Isolation** | Run AI agents in isolated git worktrees to prevent unwanted changes |
| **Cost Tracking** | Track token usage and estimated costs across all AI operations |
| **Session Management** | Resume failed or paused sessions with full state persistence |
| **Multi-Provider** | Support for Ollama, OpenAI, Groq, Gemini, and Anthropic |
| **Dependency Graph** | Tasks declare dependencies and execute in topological order |

## Runs Everywhere — Super Easy

PIVOT is built with Go (1.21+) and runs on Windows, macOS, Linux, and inside Docker. No complex setup needed — just clone, build, or download the binary.

```bash
# One-line detect + init (auto-configures everything)
pivot detect
pivot init
```

**Supported platforms:**
- Windows (`pivot.exe` / `go build`)
- macOS / Linux (`go install` or binary)
- Docker (`docker build -t pivot .` / `docker run ...`)

See terminal proof shots: [docs/proofs/](docs/proofs/)

---

## Quick Start

```bash
# Initialize pivot (creates config and state DB)
pivot init

# Run a goal
pivot run "Find all TODO comments in the codebase and create GitHub issues for them"

# Check status of recent sessions
pivot status

# Resume a failed session
pivot resume sess_1234567890
```

## Installation

### From Source

```bash
git clone https://github.com/Marwanmorsy999/pivot.git
cd pivot
go mod tidy
go build -o pivot ./cmd/pivot
```

### Using Go Install

```bash
go install github.com/Marwanmorsy999/pivot/cmd/pivot@latest
```

### Using Docker

```bash
docker build -t pivot .
docker run -it --rm -v ~/.pivot:/root/.pivot pivot run "your goal"
```

### Using Install Script (macOS/Linux)

```bash
curl -fsSL https://raw.githubusercontent.com/Marwanmorsy999/pivot/main/scripts/install.sh | bash
```

### Download Pre-built Binaries

Visit the [Releases page](https://github.com/Marwanmorsy999/pivot/releases) for pre-built binaries for:
- Linux (amd64, arm64)
- macOS (Intel & Apple Silicon)
- Windows (amd64, arm64)

## Configuration

Edit `~/.pivot/config.yaml`:

```yaml
planner:
  provider: ollama  # ollama, openai, groq, gemini, anthropic
  model: llama3.2:3b
  api_key: ""       # For OpenAI/Groq/Gemini/Anthropic
  endpoint: http://localhost:11434

worktree:
  enabled: false
  base_dir: ""      # Defaults to temp dir

cost:
  enabled: true
```

See [config.example.yaml](config.example.yaml) for a full example.

## Architecture

```
pivot/
├── cmd/pivot/            # CLI entry point (cobra)
├── internal/
│   ├── config/           # Configuration management
│   ├── state/            # SQLite-backed session & journal storage
│   ├── planner/          # AI-powered task planning (Ollama, OpenAI-compatible)
│   ├── core/
│   │   ├── task.go       # Task type definition
│   │   ├── graph.go      # Dependency graph with topological sort
│   │   ├── executor.go   # Task execution (tool + agent)
│   │   └── orchestrator.go # Main orchestration loop
│   ├── tui/              # Bubble Tea terminal UI
│   ├── worktree/         # Git worktree isolation
│   └── cost/             # Token/cost estimation
├── scripts/              # Build and install scripts
├── examples/             # Usage examples and workflows
├── assets/               # Project assets (logo, banner)
├── .github/              # GitHub templates and CI
├── docs/                 # Additional documentation
├── Makefile              # Build automation
├── Dockerfile            # Container build
├── LICENSE               # MIT License
├── CONTRIBUTING.md       # Contribution guidelines
├── SECURITY.md           # Security policy
└── CHANGELOG.md          # Version history
```

## Task Model

The planner generates a JSON task graph:

```json
{
  "tasks": [
    {
      "id": "a",
      "type": "tool",
      "tool": "grep",
      "args": ["-r", "TODO", "."],
      "depends_on": [],
      "description": "Find all TODO comments"
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

## Examples

See [examples/](examples/) for real-world usage patterns:
- [Basic Usage](examples/basic-usage.md) - Quick start and common workflows
- Finding and fixing code issues
- Automated refactoring workflows
- Multi-step research tasks

### Common Workflows

```bash
# Code review
pivot run "Review the last commit for potential bugs and security issues"

# Bug investigation
pivot run "Find the root cause of the null pointer exception in auth module"

# Feature implementation
pivot run "Implement rate limiter middleware with configurable limits"

# Documentation generation
pivot run "Generate API documentation for all exported functions"

# Security audit
pivot run "Audit codebase for SQL injection and XSS vulnerabilities"
```

## Troubleshooting

Common issues and solutions:

| Issue | Solution |
|-------|----------|
| **Ollama connection failed** | Ensure `ollama serve` is running: `ollama serve` |
| **Permission denied on worktree** | Check git worktree permissions: `chmod -R 755 ~/.pivot/worktrees` |
| **Model not found** | Pull the model: `ollama pull llama3.2:3b` |
| **API key errors** | Verify your API key in `~/.pivot/config.yaml` |
| **SQLite database locked** | Close other pivot instances, remove `~/.pivot/state.db.lock` |
| **TUI rendering issues** | Try a different terminal emulator or set `TERM=xterm-256color` |

Still having issues? [Open an issue](https://github.com/Marwanmorsy999/pivot/issues/new) with:
- Your OS and Go version
- Full error message
- Steps to reproduce

## Development

```bash
# Run tests
make test

# Run linter
make lint

# Build for all platforms
make build-all

# Run with race detector
go test -race ./...
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for detailed contribution guidelines.

## Trust & Security

- ✅ All tests pass with race detection (`go test -race ./...`)
- ✅ CI runs on every push and PR with multi-version Go testing
- ✅ CodeQL security scanning enabled
- ✅ Dependencies audited via Dependabot
- ✅ SBOM generated for each release
- ✅ HTTP clients have timeouts to prevent hanging connections
- ✅ API keys never logged or exposed

Security vulnerabilities can be reported via [SECURITY.md](SECURITY.md).

## Contributing

We welcome contributions! See our [Contributing Guide](CONTRIBUTING.md) for:
- How to report bugs
- Suggesting features
- Submitting pull requests
- Code standards and testing

Check out [MAINTAINERS.md](MAINTAINERS.md) to learn about our team and review process.

## Roadmap

See [ROADMAP.md](docs/ROADMAP.md) for upcoming features and improvements.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

Built with:
- [Cobra](https://github.com/spf13/cobra) — CLI framework
- [Bubble Tea](https://github.com/charmbracelet/bubbletea) — Terminal UI
- [Lipgloss](https://github.com/charmbracelet/lipgloss) — Styling
- [SQLite](https://github.com/mattn/go-sqlite3) — State persistence

---

<p align="center">
  <sub>Built with ❤️ by <a href="https://github.com/Marwanmorsy999">Marwan Morsy</a></sub>
</p>
