# Changelog

All notable changes to Pivot are documented here.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and versions follow [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [Unreleased]

### Added
- 12 new tests for config detection layer (`parseOllamaList`, `fetchOllamaModels`, `fetchOpenAIModels`) using `httptest` servers.

### Fixed
- TUI `WindowSizeMsg` handler was calling `waitForEvent()` spuriously, creating a second goroutine racing with the one launched by `Init()`. Removed the duplicate call.

---

## [0.3.0] — 2026-09-06

### Added
- **`pivot setup`** — interactive wizard: scans all providers, lists detected options, user picks by number. Replaces the two-step `pivot detect && pivot init` flow.
- **`pivot models`** — lists all detected local models with provider, size (GB), and copy-paste usage commands.
- **`--model / --provider / --endpoint` flags on `pivot run`** — per-run overrides without touching `~/.pivot/config.yaml`.
- **Extended local server detection** — scans 9 ports (Ollama :11434, LM Studio :1234, Jan :1337, KoboldCPP :5001, Llamafile :8080, text-gen-webui :5000, LocalAI :8081, plus two extras). Fetches actual model lists via `/api/tags` or `/v1/models`.
- **GGUF file detection** — walks 10+ directories (`~/.cache`, `~/models`, `~/Downloads`, etc.) for `.gguf` files; surfaces them with path and llamafile loading instructions.
- **New providers** — Mistral (Small, Large, Codestral), Together AI (Llama 3.1 8B/70B), OpenRouter (free tier). Added to cost table and auto-detection.
- **Expanded tool allowlist** — 40 → 50+ tools: `zsh fish uv yarn pnpm podman cmake gh rg fd bat delta aider mistral ffmpeg sqlite3 psql mysql` and more.
- **`ollama list` fallback** — if the Ollama server is not running but the binary is installed, `pivot detect` runs `ollama list` to discover available models.

### Fixed
- `golangci-lint` config schema corrected from `issues.exclude-rules` to `linters.exclusions.rules` (wrong v2 schema caused exit-code 3, masking real linter output).
- British spellings in comments and test files flagged by `misspell` with `locale: US`: `summarise`, `behaviour`, `initialise`, `cancelled` (5 occurrences).
- `#nosec G107` annotations added to all three planner HTTP calls (user-configured endpoints are legitimate).

---

## [0.2.0] — 2026-08-28

### Added
- **Parallel DAG execution** — `graph.Waves()` groups tasks by topological depth; orchestrator runs each wave with `sync.WaitGroup` + semaphore (`--parallel N`, default 4).
- **Named output piping** — `$OUTPUT[task-id]` resolved by `resolveArgs()` regex; legacy bare `$OUTPUT` still works.
- **Per-task timeout + retry** — `Task.TimeoutSec`, `Task.Retries` fields; `context.WithTimeout` wrapping; exponential backoff 1 s → 2 s → 4 s.
- **Pre-execution validation** — `planner.Validate()`: unique IDs, valid types, 40+ tool allowlist, dep graph integrity.
- **`--dry-run` flag** — shows task plan without executing.
- **Deterministic resume** — tasks JSON persisted to `sessions.tasks_json`; `pivot resume` loads it, never re-plans.
- **Model-aware cost table** — per-model rates for Anthropic, OpenAI, Groq, Gemini, Ollama; `RateFor()` with prefix matching.
- **`internal/paths` package** — shared `Home()`, `ConfigFile()`, `StateFile()`; eliminates duplicated `pivotHome()`.
- **SQLite indices** — `idx_journal_session_task` and `idx_journal_session_status`; no more full table scans.
- **45+ tests** across 6 files: executor, orchestrator, graph, validate, cost, paths (including race-detector tests).

### Fixed
- Orchestrator was `return`-ing on the first dependency failure, killing all independent branches; now continues so unrelated tasks still complete.
- TUI `Model` methods were mixed pointer/value receivers — mutations lost across Bubble Tea frames; all methods now use pointer receivers.
- `git worktree add` used hardcoded branch name `pivot-temp`; concurrent sessions collided. Now uses unique branch `pivot-wt-<random>`.
- `state.Log` was joining args with spaces (lost structure for args containing spaces); now stores as JSON array.
- `config.Detect()` was setting wrong Anthropic endpoint (`/v1/chat/completions` instead of `/v1/messages`).
- CI matrix referenced Go 1.26/1.27 (non-existent); fixed to 1.23 and 1.24.
- Dockerfile was using `golang:1.27.0-alpine3.24` (non-existent tag).
- `run.bat` / `run.ps1` missing `CGO_ENABLED=1` (required for go-sqlite3).

### Changed
- System prompt unified across all three planners so the LLM gets identical instructions regardless of provider.
- `max_tokens` increased from 1024 to 2048 (complex plans were being truncated).
- `graph.Order()` sorts queue and neighbours for fully deterministic output.
- TUI uses `tea.WithAltScreen()` for clean full-screen rendering; viewport wired for scrollable log.

---

## [0.1.0] — 2026-08-15

### Added
- Initial release of Pivot.
- Anthropic, OpenAI-compatible, and Ollama planners.
- Sequential task execution with SQLite session persistence.
- Bubble Tea TUI with task table and log viewport.
- `pivot run`, `pivot resume`, `pivot status`, `pivot export`, `pivot scaffold` commands.
- GitHub issue integration (`--issue N`).
- Git worktree isolation for agent tasks.
- Cost tracking per task and session.
