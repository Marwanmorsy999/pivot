# Pivot Architecture

## Overview

Pivot is a **Hybrid CLI Orchestrator** — it combines AI agents and CLI tools into
a single dependency graph, executes them with full parallelism, and presents live
status in a Bubble Tea TUI.

```
pivot run "goal"
      │
      ▼
  Planner (LLM)
      │  returns []Task JSON
      ▼
  Validate()        ← pre-execution sanity check
      │
      ▼
  Orchestrator
  ┌───────────────────────────┐
  │  Graph.Waves()            │  topological sort → wave groups
  │                           │
  │  Wave 0: [A]              │  run sequentially across waves
  │  Wave 1: [B, C, D] ──────┼─ goroutines within a wave
  │  Wave 2: [E]              │  semaphore(MaxParallel) limits concurrency
  └───────────────────────────┘
      │
      ▼
  Executor (per task)
  ├── resolveArgs()   $OUTPUT[id] substitution
  ├── context.WithTimeout(TimeoutSec)
  ├── retry loop (exponential backoff)
  ├── commandForTool()  validated allowlist
  └── state.Log()  → SQLite journal
      │
      ▼
  Event channel → TUI (Bubble Tea)
```

## Packages

| Package | Responsibility |
|---------|---------------|
| `cmd/pivot` | Cobra CLI entry point; wires planner → orchestrator → TUI |
| `internal/planner` | LLM providers (Anthropic, OpenAI-compat, Ollama); shared system prompt; `Validate()` |
| `internal/core` | `Graph` (DAG + Waves), `Orchestrator` (parallel scheduler), `Executor` (run + retry + cost) |
| `internal/state` | SQLite persistence: sessions, journal, task JSON; `IsTaskCompleted` |
| `internal/cost` | Model-aware pricing table; `EstimateCost(provider, model, in, out)` |
| `internal/config` | YAML config load/save; auto-detection of providers and local tools |
| `internal/paths` | `Home()`, `ConfigFile()`, `StateFile()` — shared path resolution |
| `internal/tui` | Bubble Tea model: task table, log viewport, cost display |
| `internal/worktree` | Ephemeral git worktrees for isolated agent execution |

## Key Design Decisions

### Parallel execution via waves
Kahn's topological sort naturally groups tasks into "waves" — all tasks at
depth N can run once depth N-1 is complete. We run each wave with a
`sync.WaitGroup` and a configurable semaphore (`--parallel N`, default 4).

### Deterministic resume
After planning, the full task JSON is persisted in `sessions.tasks_json`.
`pivot resume <id>` loads this instead of re-calling the LLM, guaranteeing
the same task graph even if the model's output changes.

### Named output piping
`$OUTPUT[task-id]` references are resolved by `Executor.resolveArgs()` before
execution. This supports multi-dependency tasks without data loss. The legacy
bare `$OUTPUT` still works for single-dep chains.

### Pre-execution validation
`planner.Validate()` runs after planning and before execution to catch:
duplicate IDs, invalid types, unsupported tools, and broken dependency refs.
Errors surface immediately with a clear message rather than failing mid-run.

### Model-aware cost tracking
`internal/cost` maintains a pricing table for every major provider/model.
`RateFor()` does prefix matching so dated model variants (e.g.
`claude-3-5-sonnet-20241022`) match base names. Ollama local runs cost $0.

## Data Flow: `pivot run "goal"`

1. `config.Load()` → provider/model/key
2. `planner.Plan(goal)` → `[]Task` (LLM call)
3. `planner.Validate(tasks)` → error or nil
4. `state.SaveSessionTasks()` → persist JSON
5. `core.NewOrchestrator(...).Run(ctx)` → goroutine
6. `graph.Waves()` → `[][]string`
7. Per wave: `wg.Add` + goroutine per task → `executor.Execute()`
8. `executor.Execute()`:
   a. `resolveArgs()` — substitute `$OUTPUT[id]`
   b. `context.WithTimeout(TimeoutSec)`
   c. `commandForTool()` — validate + `exec.CommandContext`
   d. retry loop with exponential backoff
   e. `state.Log()` — write journal entry
   f. emit `Event` to channel
9. TUI receives events → updates table + log
10. `EventComplete` → TUI shows cost/token totals
