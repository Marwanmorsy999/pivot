# Roadmap

This document outlines the planned features and improvements for PIVOT.

## Current Release: v2.1.0

- [x] Professional repository structure
- [x] Comprehensive unit tests
- [x] CI/CD with GitHub Actions
- [x] Linting with golangci-lint
- [x] Security policy and vulnerability reporting
- [x] Contribution guidelines
- [x] Code of conduct
- [x] Project branding (logo, banner)

## Upcoming: v2.2.0

### Features
- [ ] **Parallel Execution**: Execute independent tasks concurrently
- [ ] **Task Validation**: Validate LLM-generated tasks before execution
- [ ] **Retry Logic**: Automatic retry for failed tasks with exponential backoff
- [ ] **Plugin System**: Support for custom task types via plugins
- [ ] **Web Dashboard**: Browser-based UI for session monitoring

### Improvements
- [ ] **Better Token Counting**: Use tiktoken for accurate token estimation
- [ ] **Model-Specific Costs**: Different cost rates per provider/model
- [ ] **Session Export**: Export sessions as JSON/Markdown reports
- [ ] **Interactive Mode**: Step-through execution with user confirmation
- [ ] **Task Templates**: Reusable task graph templates

### Infrastructure
- [ ] **Release Automation**: Automated releases via GitHub Actions
- [ ] **Homebrew Formula**: Easy installation on macOS/Linux
- [ ] **Signed Binaries**: Code signing for released binaries
- [ ] **SBOM Generation**: Software Bill of Materials for releases

## Future: v3.0.0

### Major Features
- [ ] **Multi-Agent Coordination**: Multiple AI agents collaborating on tasks
- [ ] **Remote Execution**: Execute tasks on remote machines via SSH
- [ ] **Workflow Sharing**: Share and import workflow definitions
- [ ] **API Server**: REST API for programmatic access
- [ ] **Team Features**: Shared sessions and collaboration tools

### Research
- [ ] **Smart Caching**: Cache LLM responses for similar goals
- [ ] **Cost Optimization**: Automatically select cheapest provider for task
- [ ] **Self-Healing**: Automatically fix common task failures
- [ ] **Natural Language Queries**: Query session history in plain English

## How to Contribute

We welcome contributions! See [CONTRIBUTING.md](../CONTRIBUTING.md) for guidelines.

To pick up a task from this roadmap:
1. Open an issue referencing the roadmap item
2. Discuss approach with maintainers
3. Submit a pull request

## Completed Milestones

### v2.0.0 (2025-08-31)
- Initial release
- Hybrid CLI orchestration
- Real-time TUI
- Worktree isolation
- Cost tracking
- Session management
- Multi-provider support
- Dependency graph execution

### v2.1.0 (2025-09-01)
- Professional repository structure
- Unit tests with coverage
- CI/CD pipeline
- Security and contribution policies
- Project branding
