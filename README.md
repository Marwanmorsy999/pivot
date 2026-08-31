# PIVOT v2.0 — The Hybrid CLI Orchestrator

A universal hybrid CLI orchestrator that combines AI agents and traditional CLI tools in a single workflow, with a real-time TUI, worktree isolation, cost tracking, and a plugin-ready architecture.

## Features

- **Hybrid Execution**: Seamlessly mix AI agents (Ollama, Claude Code, Gemini CLI) with traditional CLI tools (grep, jq, curl, etc.)
- **Real-time TUI**: Beautiful terminal UI with live task tracking, cost monitoring, and event logging
- **Worktree Isolation**: Run AI agents in isolated git worktrees to prevent unwanted changes
- **Cost Tracking**: Track token usage and estimated costs across all AI operations
- **Session Management**: Resume failed or paused sessions with full state persistence
- **Multi-Provider**: Support for Ollama, OpenAI, Groq, Gemini, and Anthropic
- **Dependency Graph**: Tasks declare dependencies and execute in topological order

## Installation

```bash
cd pivot
go mod tidy
go build -o pivot.exe
```

## Quick Start

```bash
# Initialize pivot (creates config and state DB)
./pivot init

# Run a goal
./pivot run "Find all TODO comments in the codebase and create GitHub issues for them"

# Check status of recent sessions
./pivot status

# Resume a failed session
./pivot resume sess_1234567890
```

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

## Architecture

```
pivot/
├── main.go                 # CLI entry point (cobra)
├── internal/
│   ├── config/             # Configuration management
│   ├── state/              # SQLite-backed session & journal storage
│   ├── planner/            # AI-powered task planning (Ollama, OpenAI-compatible)
│   ├── core/
│   │   ├── task.go         # Task type definition
│   │   ├── graph.go        # Dependency graph with topological sort
│   │   ├── executor.go     # Task execution (tool + agent)
│   │   └── orchestrator.go # Main orchestration loop
│   ├── tui/                # Bubble Tea terminal UI
│   ├── worktree/           # Git worktree isolation
│   └── cost/               # Token/cost estimation
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

## License

MIT
