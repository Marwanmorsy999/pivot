# PIVOT Basic Usage Examples

This directory contains example workflows and usage patterns for PIVOT.

## Quick Start Examples

### 1. Initialize PIVOT

```bash
# Auto-detect all providers and local tools
pivot detect

# Initialize configuration with detected settings
pivot init
```

### 2. Run a Simple Goal

```bash
# Find and summarize TODOs in your codebase
pivot run "Find all TODO comments in the codebase and create a summary"

# Refactor a function
pivot run "Refactor the getUserById function to use async/await pattern"

# Generate documentation
pivot run "Generate API documentation for all exported functions in the internal/core package"
```

### 3. Check Session Status

```bash
# View recent sessions
pivot status

# Resume a failed session
pivot resume sess_1234567890
```

## Common Workflows

### Code Review Workflow

```bash
pivot run "Review the last commit for potential bugs, security issues, and code style violations. Create a detailed report."
```

### Bug Fix Workflow

```bash
pivot run "Find the root cause of the null pointer exception in the user authentication module and suggest a fix."
```

### Feature Implementation Workflow

```bash
pivot run "Implement a rate limiter middleware for the HTTP server with configurable limits per endpoint."
```

### Documentation Workflow

```bash
pivot run "Update README.md with installation instructions for Windows, macOS, and Linux. Include Docker setup."
```

### Testing Workflow

```bash
pivot run "Write comprehensive unit tests for the graph.go file covering edge cases and error conditions."
```

## Advanced Patterns

### Multi-Step Research Task

```bash
pivot run "Research best practices for Go error handling, compare with our current implementation, and propose improvements with code examples."
```

### Automated Refactoring

```bash
pivot run "Replace all instances of fmt.Sprintf with strings.Builder for better performance in high-throughput code paths."
```

### Security Audit

```bash
pivot run "Audit the codebase for common security vulnerabilities: SQL injection, XSS, CSRF, and insecure deserialization."
```

## Tips for Better Results

1. **Be Specific**: The more specific your goal, the better the results
2. **Break Down Complex Tasks**: Split large goals into smaller, manageable tasks
3. **Provide Context**: Include relevant file paths or function names when applicable
4. **Iterate**: Use resume to continue from where a session left off

## See Also

- [README.md](../README.md) - Full documentation
- [CONTRIBUTING.md](../CONTRIBUTING.md) - Contribution guidelines
- [docs/](../docs/) - Additional documentation
