<p align="center">
  <img src="assets/banner.svg" alt="Pivot — Hybrid CLI Orchestrator" width="900"/>
</p>

<p align="center">
  <a href="https://github.com/Marwanmorsy999/pivot/actions/workflows/ci.yml"><img src="https://github.com/Marwanmorsy999/pivot/actions/workflows/ci.yml/badge.svg" alt="CI"/></a>
  <a href="https://goreportcard.com/report/github.com/Marwanmorsy999/pivot"><img src="https://goreportcard.com/badge/github.com/Marwanmorsy999/pivot" alt="Go Report Card"/></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/License-MIT-yellow.svg" alt="License: MIT"/></a>
  <img src="https://img.shields.io/badge/Go-1.23+-00ADD8?logo=go" alt="Go 1.23+"/>
</p>

> Turn any plain-English goal into a validated, parallel AI-agent + CLI task graph. One binary. Zero Python.

## What Is Pivot?

Pivot turns a plain-English goal (or a GitHub issue) into a validated dependency
graph of CLI tools and AI agents, then executes it in parallel — with live TUI,
cost tracking, checkpoints, hooks, and full SQLite session history.

```
pivot run "find all TODO comments, summarise with Claude, save to todos.md"
```

Pivot:
1. **Plans** — LLM designs the task graph (or load from a YAML file)
2. **Validates** — checks types, tools, deps before touching anything
3. **Executes** — parallel wave scheduler; independent tasks run concurrently
4. **Tracks** — per-task cost, per-model pricing, full journal in SQLite
5. **Recovers** — `pivot resume <session>` replays only the failed tasks

## Install

```bash
# Requires Go 1.23+ and gcc (for SQLite)
git clone https://github.com/Marwanmorsy999/pivot
cd pivot
make install          # installs to $GOPATH/bin

# Or run directly
export ANTHROPIC_API_KEY=sk-ant-...
make build && ./pivot setup
```

Pre-built binaries (Linux amd64, macOS amd64/arm64, Windows) are attached to
each [GitHub Release](https://github.com/Marwanmorsy999/pivot/releases).

## Quick Start

```bash
# 1. Auto-detect providers and configure interactively
pivot setup

# 2. Run a goal (LLM plans it)
pivot run "list all Go files and count lines of code"

# 3. Load a workflow file instead (no LLM needed)
pivot scaffold my-deploy     # generates my-deploy.yaml
pivot run --file my-deploy.yaml

# 4. Fetch goal from a GitHub issue
pivot run --issue 42         # reads GITHUB_TOKEN from env

# 5. Preview without executing
pivot run --dry-run "deploy my app to k8s"

# 6. Export session report
pivot export sess_1234567890_abcd --out report.md
```

## Commands

| Command | Description |
|---------|-------------|
| `pivot setup` | Interactive wizard: detect providers, pick model, write config |
| `pivot detect` | Auto-detect AI providers and local tools (non-interactive) |
| `pivot init` | Initialise config + state DB |
| `pivot models` | List all detected local models with sizes and usage commands |
| `pivot run "goal"` | Plan and execute a goal |
| `pivot run --file plan.yaml` | Load task graph from YAML (no LLM needed) |
| `pivot run --issue N` | Fetch goal from GitHub issue #N |
| `pivot run --dry-run` | Show task plan without executing |
| `pivot run --parallel N` | Max concurrent tasks (default 4) |
| `pivot run --model M --provider P` | Override model/provider for this run |
| `pivot resume <id>` | Resume failed session from saved plan |
| `pivot status` | List recent sessions with status icons |
| `pivot export <id>` | Export session as Markdown report |
| `pivot export <id> --out file.md` | Write report to file |
| `pivot scaffold [name]` | Generate example workflow YAML |

## Workflow Files

Skip the LLM entirely and define your task graph in YAML:

```yaml
# my-workflow.yaml
goal: Audit and report on the codebase
tasks:
  - id: count-lines
    type: tool
    tool: sh
    args: ["-c", "find . -name '*.go' | xargs wc -l | tail -1"]
    description: Count total lines of Go code

  - id: check-env
    type: checkpoint
    prompt: "Line count looks reasonable — continue with analysis?"
    depends_on: [count-lines]

  - id: analyse
    type: agent
    tool: claude-code
    args: ["summarise the architecture based on $OUTPUT[count-lines]"]
    depends_on: [check-env]
    before: "echo starting analysis"
    after: "echo done"
```

```bash
pivot run --file my-workflow.yaml
```

### Task Types

| Type | Description |
|------|-------------|
| `tool` | Runs a validated CLI binary deterministically |
| `agent` | Runs an AI agent (claude-code, ollama, gemini-cli) |
| `checkpoint` | Pauses and waits for human y/n confirmation |

### Task Fields

```yaml
- id: my-task          # required, unique slug
  type: tool           # tool | agent | checkpoint
  tool: sh             # executable (tool/agent only)
  args: ["-c", "..."]  # argument array
  depends_on: [other]  # task IDs this depends on
  description: "..."   # shown in TUI and reports
  timeout_sec: 300     # per-task deadline (default 300)
  retries: 2           # retry on failure with exponential backoff
  before: "echo start" # shell hook before execution
  after: "echo done"   # shell hook after success
  prompt: "Deploy?"    # message for checkpoint tasks
```

### Output Piping

```yaml
args: ["$OUTPUT[fetch]"]    # named: use output of task "fetch"
args: ["$OUTPUT"]           # legacy: use output of DependsOn[0]
```

## GitHub Integration

```bash
# Run goal from a GitHub issue
export GITHUB_TOKEN=ghp_...
pivot run --issue 42

# Specify repo explicitly (auto-detected from .git/config)
pivot run --issue 42 --github-repo owner/repo
```

## Supported Providers

| Provider | Auto-detected from | Default model |
|----------|--------------------|---------------|
| Anthropic | `ANTHROPIC_API_KEY` | claude-sonnet-4-5 |
| OpenAI | `OPENAI_API_KEY` | gpt-4o-mini |
| Groq | `GROQ_API_KEY` | llama-3.1-8b-instant |
| Gemini | `GEMINI_API_KEY` | gemini-1.5-flash |
| Mistral | `MISTRAL_API_KEY` | mistral-small-latest |
| Together AI | `TOGETHER_API_KEY` | llama-3.1-8b-instruct |
| OpenRouter | `OPENROUTER_API_KEY` | (your choice) |
| Ollama | running on :11434 | llama3.2:3b (free) |
| LM Studio | running on :1234 | (active model) |
| Jan | running on :1337 | (active model) |
| LocalAI | running on :8080 | (active model) |
| GGUF files | `~/.cache`, `~/models`, etc. | (loaded via llamafile) |

API keys are **never written to disk** — always read from environment variables.

## Cost Tracking

Per-task cost shown live in the TUI, stored in SQLite, included in exports.

| Model | Input / M tokens | Output / M tokens |
|-------|-----------------|-------------------|
| claude-opus-4-5 | $15.00 | $75.00 |
| claude-sonnet-4-5 | $3.00 | $15.00 |
| claude-3-5-haiku | $0.80 | $4.00 |
| gpt-4o | $2.50 | $10.00 |
| gpt-4o-mini | $0.15 | $0.60 |
| gemini-2.5-flash | $0.075 | $0.30 |
| groq llama-3.1-8b | $0.05 | $0.08 |
| mistral-small | $0.20 | $0.60 |
| mistral-large | $2.00 | $6.00 |
| together llama-3.1-8b | $0.18 | $0.18 |
| ollama / gguf (local) | $0 | $0 |

## Tool Allowlist (50+)

Every tool is validated against an allowlist before execution. Unknown executables are rejected immediately.

**Unix core:** `find grep awk sed cat echo wc sort uniq head tail xargs tar zip unzip cut tr tee diff patch ls cp mv rm mkdir chmod chown touch stat file env printenv which date sleep`  
**Shell:** `sh bash zsh fish`  
**Network / data:** `jq curl wget ssh rsync`  
**Dev:** `git docker podman kubectl make cmake python3 python node go npm npx pnpm yarn pip pip3 cargo rustc terraform helm`  
**Cloud CLIs:** `aws gcloud az gh`  
**Data tools:** `rg fd bat delta sqlite3 psql mysql ffmpeg`  
**AI / automation:** `ollama claude-code gemini-cli aider mistral uv`  
**Package managers:** `pip3 cargo` (already listed above under Dev)

## Session Management

```bash
pivot status              # list recent sessions with ✅/❌/▶ icons
pivot resume sess_abc123  # re-run only failed tasks
pivot export sess_abc123  # Markdown report: summary table + outputs + cost
```

## Configuration

`~/.pivot/config.yaml` (written by `pivot setup` or `pivot init`, API keys never stored here):

```yaml
planner:
  provider: anthropic
  model: claude-sonnet-4-5
  endpoint: ""          # leave empty for default
worktree:
  enabled: false
  base_dir: /tmp/pivot-worktrees
cost:
  enabled: true
```

Set `PIVOT_HOME` to override the `~/.pivot` directory.

## Architecture

See [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) for the full package map.

```
cmd/pivot/          CLI entry point (cobra commands)
internal/
  planner/          LLM planners + YAML loader + validator
  core/             Orchestrator, executor, task graph (wave scheduler)
  state/            SQLite session + journal persistence
  tui/              Bubble Tea TUI
  github/           GitHub API client
  export/           Markdown session reports
  cost/             Per-model pricing
  config/           Auto-detection + config file
  paths/            XDG-style path resolution
  worktree/         Git worktree isolation for agents
```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md). PRs welcome — tests required for new features.
