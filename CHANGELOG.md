# Changelog

## [Unreleased] — v3.0.0-dev

### Breaking Changes
- `NewOrchestrator` signature now takes `OrchestratorOptions` as 5th argument
- `executor.Execute` now takes `context.Context` instead of `<-chan struct{}`
- `state.Log` stores args as JSON array (was space-joined string)
- `cost.EstimateCost` now takes `provider, model string` as first two args

### Added
- **Parallel DAG execution**: tasks in the same topological wave run concurrently
  via goroutines + semaphore (`--parallel N` flag, default 4)
- **Named output piping**: `$OUTPUT[task-id]` references support multi-dep tasks
  without silent data loss; legacy bare `$OUTPUT` still works
- **Per-task timeout**: `timeout_sec` field on Task; wraps execution in
  `context.WithTimeout` (default 300s)
- **Retry with exponential backoff**: `retries` field on Task; backs off 1s→2s→4s
- **Pre-execution validation**: `planner.Validate()` checks unique IDs, valid
  types, tool allowlist, and dependency graph integrity before any task runs
- **`--dry-run` flag**: prints task plan without executing
- **`--parallel N` flag**: configures max concurrent tasks
- **Deterministic resume**: `pivot run` persists task JSON to SQLite;
  `pivot resume` loads it instead of re-planning with the LLM
- **Model-aware cost tracking**: pricing table for Anthropic, OpenAI, Groq,
  Gemini, Ollama; `RateFor()` with prefix matching for dated model names
- **Expanded tool allowlist**: 15 → 40+ tools including `sh/bash`, `ls/cp/mv/rm`,
  `npm/npx/pip/cargo`, `aws/gcloud/az`, `terraform/helm`
- **`internal/paths` package**: shared `Home()`, `ConfigFile()`, `StateFile()`
  (eliminates duplicated `pivotHome()` in config and state packages)
- **SQLite indices**: `idx_journal_session_task` and `idx_journal_session_status`
  on the journal table (eliminates full table scans)
- **Comprehensive tests**: executor, orchestrator, graph, validate, cost, paths
  (6 new test files, 45+ new test cases including race detector tests)

### Fixed
- `config.Detect()` was setting wrong Anthropic endpoint (`/v1/chat/completions`
  instead of `/v1/messages`)
- Orchestrator was calling `return err` on dependency failure, killing all
  independent branches; now continues so unrelated tasks still execute
- CI was referencing Go 1.26 and 1.27 (don't exist); fixed to 1.23 and 1.24
- Dockerfile was using `golang:1.27.0-alpine3.24` (nonexistent tag)
- `run.bat`/`run.ps1` were missing `CGO_ENABLED=1` (required for go-sqlite3)
- `git worktree add` used hardcoded branch `pivot-temp`; concurrent sessions
  collided. Now uses unique branch `pivot-wt-<random>` per worktree
- TUI `Model` methods were mixed pointer/value receivers causing mutation to
  be lost across Bubble Tea frames; all methods now use pointer receivers
- `state.Log` was joining args with spaces (lost structure for args with spaces)
- `fmt.Sprintf` format string warnings in `model.go` (use `fmt.Fprintf`)

### Changed
- System prompt unified across all three planners (Anthropic, OpenAI, Ollama)
  so the LLM gets identical instructions regardless of provider
- System prompt updated to document `$OUTPUT[task-id]` syntax and full tool list
- `max_tokens` increased from 1024 to 2048 (complex plans were being truncated)
- Anthropic default model updated to `claude-opus-4-5`
- `graph.Order()` now sorts queue/neighbours for deterministic output
- `graph.Order()` error messages improved (include task ID and dep ID)
- TUI uses `tea.WithAltScreen()` for cleaner full-screen rendering
- TUI viewport wired for scrollable log output

All notable changes to PIVOT are documented here.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and versions follow [Semantic Versioning](https://semver.org/spec/v2.0.0.html) where applicable.

## [Unreleased]

### Changed
- Professionalized the README and corrected release-status documentation.
- Added a dated repository verification record with direct CI and security evidence.
- Strengthened contributor and security guidance.

## [0.1.0-rc.1] - 2026-09-01

### Added
- Initial public release candidate of PIVOT.
- Hybrid orchestration of AI agents and traditional CLI tools.
- Dependency-graph task execution.
- Real-time Bubble Tea terminal UI.
- Git worktree isolation for agent-driven work.
- SQLite-backed session persistence and resume support.
- Multi-provider planner support.
- Token and estimated cost tracking.
- Multi-platform release artifacts, SPDX SBOM generation, and SHA-256 checksums.

### Quality and Security
- CI validation against Go 1.26 and Go 1.27.
- Race-detector test execution.
- `golangci-lint` validation.
- `gosec` security scanning.
- `govulncheck` dependency vulnerability scanning.
- CodeQL analysis.
- Release binary smoke tests.

## Historical Development Notes

The entries below record earlier internal milestones that preceded the current `0.x` release line. They are retained for historical context.

### 2.1.0 — 2025-09-01

- Added unit tests for core, config, cost, and state packages.
- Added `PIVOT_HOME` for testing and custom configuration locations.
- Added Makefile targets, CI, issue templates, contribution/security policies, editor configuration, example configuration, and Docker support.
- Improved error propagation and HTTP timeout handling.
- Resolved linting and error-handling issues across the codebase.

### 2.0.0 — 2025-08-31

- Initial internal PIVOT milestone.
- Hybrid CLI orchestration combining AI agents and traditional CLI tools.
- Real-time TUI with Bubble Tea.
- Worktree isolation for AI agents.
- Cost tracking and token usage.
- Session management with SQLite persistence.
- Multi-provider support.
- Dependency graph with topological-sort execution.
