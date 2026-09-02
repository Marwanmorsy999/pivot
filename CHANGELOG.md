# Changelog

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
