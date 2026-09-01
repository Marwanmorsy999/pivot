# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Project logo and banner (`assets/`)
- CODE_OF_CONDUCT.md for community guidelines
- `.github/settings.yml` for repository configuration
- FUNDING.yml for sponsorship support
- Architecture documentation (`docs/ARCHITECTURE.md`)
- Roadmap (`docs/ROADMAP.md`)
- Enhanced SECURITY.md with detailed reporting and best practices

### Changed
- Enhanced README with professional banner, trust badges, and better structure

## [2.1.0] - 2025-09-01

### Added
- LICENSE file (MIT)
- Unit tests for `core`, `config`, `cost`, and `state` packages
- `PIVOT_HOME` environment variable for testing and custom config locations
- Makefile with build, test, lint, and clean targets
- GitHub Actions CI workflow (test, lint, build)
- GitHub issue templates (bug report, feature request)
- Pull request template
- CONTRIBUTING.md guide
- SECURITY.md policy
- `.editorconfig` for consistent coding styles
- Example configuration file (`config.example.yaml`)
- Dockerfile for container builds

### Fixed
- Variable shadowing bug in `executor.go` where `err` was redeclared in worktree block
- Ignored errors in `main.go` (config load, state init, session creation)
- Ignored errors in `state.go` migrate function
- Missing error handling in HTTP clients (Ollama and OpenAI planners)
- HTTP clients now have a 60-second timeout to prevent hanging
- All golangci-lint issues resolved (errcheck, unused)

### Changed
- Improved error messages with proper wrapping (`fmt.Errorf` with `%w`)
- All error paths now properly propagate errors instead of silently failing

## [2.0.0] - 2025-08-31

### Added
- Initial release of PIVOT v2.0
- Hybrid CLI orchestration combining AI agents and traditional CLI tools
- Real-time TUI with Bubble Tea
- Worktree isolation for AI agents
- Cost tracking and token usage
- Session management with SQLite persistence
- Multi-provider support (Ollama, OpenAI, Groq, Gemini, Anthropic)
- Dependency graph with topological sort execution
