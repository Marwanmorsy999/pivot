# PIVOT Architecture

## Overview

PIVOT is a hybrid CLI orchestrator that combines AI agents with traditional command-line tools. The system follows a pipeline architecture: **Plan → Graph → Execute → Display**.

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   Planner   │────▶│    Graph    │────▶│  Executor   │────▶│     TUI     │
│  (AI/LLM)   │     │ (Dependency)│     │  (Tools)    │     │  (Display)  │
└─────────────┘     └─────────────┘     └─────────────┘     └─────────────┘
```

## Core Components

### 1. Planner (`internal/planner/`)

The planner converts a natural-language goal into a structured task graph using an LLM.

**Interface:**
```go
type Planner interface {
    Plan(goal string) ([]Task, error)
}
```

**Providers:**
- **OllamaPlanner**: Uses local Ollama instance for planning
- **OpenAPlanner**: Uses OpenAI-compatible APIs (OpenAI, Groq, Gemini, Anthropic)

**Task Types:**
- `tool`: Traditional CLI commands (grep, jq, curl, etc.)
- `agent`: AI-powered tasks (ollama, claude-code, gemini-cli)

### 2. Dependency Graph (`internal/core/graph.go`)

The graph module manages task dependencies and determines execution order.

**Algorithm:** Kahn's algorithm for topological sorting with cycle detection.

```go
func (g *Graph) Order() ([]string, error)
```

**Features:**
- Detects circular dependencies
- Validates all dependencies exist
- Returns tasks in execution order

### 3. Executor (`internal/core/executor.go`)

Executes individual tasks and manages output piping between them.

**Key behaviors:**
- Resolves `$OUTPUT` references from dependency outputs
- Supports worktree isolation for agent tasks
- Tracks token usage and cost
- Logs all operations to SQLite

### 4. Orchestrator (`internal/core/orchestrator.go`)

The main execution loop that coordinates the entire workflow.

**Flow:**
1. Convert planner tasks to core tasks
2. Build dependency graph
3. Compute execution order
4. Execute tasks sequentially (respecting dependencies)
5. Emit events for TUI updates
6. Handle context cancellation (SIGINT/SIGTERM)

### 5. Terminal UI (`internal/tui/model.go`)

Real-time terminal display using the Bubble Tea framework.

**Features:**
- Live task table with status updates
- Cost and token tracking
- Event log with timestamps
- Spinner for active tasks
- Keyboard controls (q to quit)

### 6. State Management (`internal/state/state.go`)

SQLite-backed persistence for sessions and journal entries.

**Tables:**
- `sessions`: Session metadata (id, goal, created_at, status)
- `journal`: Task execution log (session_id, task_id, tool, args, output, error, status, cost, tokens)

**Features:**
- Resume failed sessions
- Query execution history
- Track costs across sessions

### 7. Worktree Isolation (`internal/worktree/worktree.go`)

Git worktree management for safe agent execution.

**Flow:**
1. Create temporary worktree with `git worktree add`
2. Execute agent task in isolated directory
3. Clean up with `git worktree remove`

### 8. Cost Tracking (`internal/cost/cost.go`)

Token usage estimation and cost calculation.

```go
func EstimateTokens(text string) int
func EstimateCost(tokens int) float64
```

## Data Flow

```
User Goal
    │
    ▼
┌──────────┐    JSON Task Graph
│  Planner │────────────────────┐
└──────────┘                    │
                                ▼
                         ┌────────────┐
                         │    Graph   │
                         │ (topo sort)│
                         └────────────┘
                                │
                                ▼
                         ┌────────────┐
                         │ Orchestrator│
                         └────────────┘
                                │
              ┌─────────────────┼─────────────────┐
              ▼                 ▼                 ▼
        ┌──────────┐     ┌──────────┐     ┌──────────┐
        │ Executor │     │  State   │     │   TUI    │
        │ (tools)  │     │  (SQLite)│     │ (display)│
        └──────────┘     └──────────┘     └──────────┘
```

## Configuration

Configuration is loaded from `~/.pivot/config.yaml` with the following structure:

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

**Environment Variables:**
- `PIVOT_HOME`: Override config/data directory (default: `~/.pivot`)

## Error Handling

PIVOT follows these error handling principles:

1. **No silent failures**: All errors are checked and reported
2. **Graceful degradation**: Non-fatal errors don't crash the application
3. **Context cancellation**: SIGINT/SIGTERM properly cancel ongoing operations
4. **Timeout protection**: HTTP requests have 60-second timeouts
5. **Worktree cleanup**: Deferred cleanup ensures no orphaned worktrees

## Testing

```bash
# Run all tests
go test ./...

# Run with race detector
go test -race ./...

# Run with coverage
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
```

## Extending PIVOT

### Adding a New Planner

1. Implement the `Planner` interface
2. Add provider case in `main.go`
3. Add tests for the new planner

### Adding a New Task Type

1. Extend the `TaskType` constants
2. Add handling in `executor.go`
3. Update planner prompts to include the new type
