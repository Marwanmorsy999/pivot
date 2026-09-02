# ⚡ Pivot — Hybrid CLI Orchestrator

> Turn any goal into a parallel, AI-powered task graph. One binary. Zero Python.

[![CI](https://github.com/Marwanmorsy999/pivot/actions/workflows/ci.yml/badge.svg)](https://github.com/Marwanmorsy999/pivot/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/Marwanmorsy999/pivot)](https://goreportcard.com/report/github.com/Marwanmorsy999/pivot)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

## What Is Pivot?

Pivot lets you describe a goal in plain English. It calls an LLM to design a
dependency graph of tasks (CLI tools + AI agents), validates the plan, then
executes it **in parallel** — showing live status in a Bubble Tea TUI.

```
pivot run "find all TODO comments in this repo, summarise them with Claude, and save to todos.md"
```

Pivot then:
1. **Plans** — asks your LLM to design the task graph
2. **Validates** — checks types, tools, dependencies before touching anything
3. **Executes** — runs independent tasks concurrently (wave scheduler)
4. **Tracks** — real cost per task, per-model pricing, full journal in SQLite
5. **Recovers** — `pivot resume <session>` picks up exactly where it left off

## Quick Start

```bash
# Prerequisites: Go 1.22+, gcc (for SQLite)
git clone https://github.com/Marwanmorsy999/pivot
cd pivot

# Set your API key
export ANTHROPIC_API_KEY=sk-ant-...   # or OPENAI_API_KEY, GROQ_API_KEY

# Auto-detect providers and initialise
CGO_ENABLED=1 go run ./cmd/pivot detect
CGO_ENABLED=1 go run ./cmd/pivot init

# Run a goal
CGO_ENABLED=1 go run ./cmd/pivot run "list all Go files and count lines of code"

# Preview without executing
CGO_ENABLED=1 go run ./cmd/pivot run --dry-run "deploy my app to k8s"

# Run with more parallelism
CGO_ENABLED=1 go run ./cmd/pivot run --parallel 8 "analyse all log files"
```

## Commands

| Command | Description |
|---------|-------------|
| `pivot detect` | Auto-detect all AI providers and local tools |
| `pivot init` | Initialise config + state DB from detected setup |
| `pivot run "goal"` | Execute a goal (AI plans → validates → runs) |
| `pivot run --dry-run "goal"` | Show task plan without executing |
| `pivot run --parallel N` | Max concurrent tasks (default 4) |
| `pivot resume <session-id>` | Resume failed session from saved plan |
| `pivot status` | List recent sessions |

## Supported AI Providers

| Provider | Models | Env Var |
|----------|--------|---------|
| Anthropic | claude-opus-4-5, claude-3-5-sonnet | `ANTHROPIC_API_KEY` |
| OpenAI | gpt-4o, gpt-4o-mini | `OPENAI_API_KEY` |
| Groq | llama-3.1-8b-instant, llama-3.1-70b | `GROQ_API_KEY` |
| Gemini | gemini-1.5-flash, gemini-2.0-flash | `GEMINI_API_KEY` |
| Ollama | llama3.2:3b (local, free) | _(just run Ollama)_ |

## Supported Tools (40+)

Pivot validates every tool before running it. The allowlist includes:

**Unix core:** `find grep awk sed cat echo wc sort uniq head tail xargs tar zip cut tr tee diff patch ls cp mv rm mkdir chmod touch stat`

**Shell:** `sh bash`

**Network/data:** `jq curl wget ssh rsync`

**Dev:** `git docker kubectl make python3 node go npm npx pip cargo rustc terraform helm`

**Cloud:** `aws gcloud az`

**AI agents:** `ollama claude-code gemini-cli`

## Output Piping

Reference one task's output in another task's args:

```json
{
  "tasks": [
    {"id": "fetch", "type": "tool", "tool": "curl", "args": ["https://api.example.com/data"]},
    {"id": "parse", "type": "tool", "tool": "jq",   "args": [".[0].name", "$OUTPUT[fetch]"], "depends_on": ["fetch"]},
    {"id": "save",  "type": "tool", "tool": "tee",  "args": ["output.txt"],                  "depends_on": ["parse"],
     "args": ["$OUTPUT[parse]"]}
  ]
}
```

Use `$OUTPUT[task-id]` for named references (multiple deps supported).
The legacy bare `$OUTPUT` still works and maps to `DependsOn[0]`.

## Task Options

```json
{
  "id": "slow-analysis",
  "type": "agent",
  "tool": "claude-code",
  "args": ["analyse this codebase"],
  "timeout_sec": 600,
  "retries": 2
}
```

- `timeout_sec` — per-task deadline (default 300s)
- `retries` — retry on failure with exponential backoff (1s → 2s → 4s)

## Resume

Every run persists its task plan. If tasks fail, resume with the exact same plan:

```bash
pivot status                          # find your session ID
pivot resume sess_1234567890123456    # re-run only failed tasks
```

## Cost Tracking

Pivot tracks real cost per task using per-model rates:

| Model | Input | Output |
|-------|-------|--------|
| claude-opus-4-5 | $3/M | $15/M |
| gpt-4o-mini | $0.15/M | $0.60/M |
| gemini-1.5-flash | $0.075/M | $0.30/M |
| groq llama-3.1-8b | $0.05/M | $0.08/M |
| ollama (local) | $0 | $0 |

## Architecture

See [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) for the full data flow diagram
and package descriptions.

## Configuration

`~/.pivot/config.yaml` (created by `pivot init`):

```yaml
planner:
  provider: anthropic
  model: claude-opus-4-5
  api_key: ""          # leave empty — use ANTHROPIC_API_KEY env var instead
  endpoint: ""         # leave empty for default
worktree:
  enabled: false       # enable for git-isolated agent execution
  base_dir: /tmp/pivot-worktrees
cost:
  enabled: true
```

Set `PIVOT_HOME` to override the `~/.pivot` directory.

## Windows

```bat
run.bat "list all Python files"
```

Requires GCC (e.g. via [MSYS2](https://www.msys2.org/)) for SQLite.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md). PRs welcome — tests required.
