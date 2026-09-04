# Roadmap

## Current: v3.2 (released)

- [x] Parallel DAG execution (wave scheduler, semaphore, sync.WaitGroup)
- [x] Named output piping ($OUTPUT[task-id], legacy $OUTPUT)
- [x] Per-task timeout + exponential backoff retry
- [x] Pre-execution validation (types, tool allowlist, dep graph)
- [x] --dry-run flag
- [x] Deterministic resume (persisted task plan JSON)
- [x] Model-aware cost table (Anthropic, OpenAI, Groq, Gemini, Ollama)
- [x] SQLite session + journal with indices
- [x] 40+ tool allowlist
- [x] Bubble Tea TUI (pointer receivers, viewport, alt-screen)
- [x] Workflow files (pivot run --file plan.yaml)
- [x] checkpoint task type (tea.Suspend/ResumeMsg — TUI owns stdin)
- [x] Per-task hooks (before:/after: shell strings)
- [x] pivot export — Markdown session reports
- [x] pivot scaffold — generate example workflow YAML
- [x] GitHub integration (--issue N, GetIssue, CreateComment, ListIssues)
- [x] API key never persisted to disk
- [x] Session status tracking (active/completed/failed)
- [x] Deduplication in journal export (latest attempt per task)
- [x] Auto-detect prefers claude-sonnet-4-5 over Opus
- [x] Correct model pricing (Opus $15/$75, Gemini 2.5, o1, more Ollama)
- [x] 60+ tests across 10 files

## Next: v3.3

- [ ] **--close-on-success** — comment + close GitHub issue when session completes
- [ ] **pivot watch --label pivot** — poll for labelled issues and auto-dispatch
- [ ] **Linux arm64 release binary** — cross-compile in CI
- [ ] **pivot delete <session-id>** — remove session + journal from state
- [ ] **pivot status --failed** — filter to failed sessions only

## v3.4

- [ ] **Smart caching** — skip tasks whose inputs + tool haven't changed
- [ ] **Cost budget flag** — --max-cost 0.50 aborts if projected cost exceeds limit
- [ ] **JSON output mode** — --json for scripting pivot status/export
- [ ] **Env var interpolation** — ${ENV_VAR} expansion in task args
- [ ] **Parallel limit per tool** — max-parallel per tool type (e.g. max 2 claude-code)

## Future

- [ ] **Remote execution** — run tasks via SSH on remote machines
- [ ] **API server** — REST API for programmatic session control
- [ ] **Multi-agent coordination** — agent tasks that spawn sub-plans
- [ ] **Natural language session query** — "what failed last time I deployed?"
- [ ] **Homebrew formula** — easy macOS/Linux install
