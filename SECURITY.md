# Security Policy

PIVOT executes local commands and can run task plans produced by AI providers. Security therefore covers both the application itself and the way it is deployed and operated.

## Supported Versions

PIVOT is currently in pre-release development. Until a stable `1.0.0` release is published, security support is provided for the latest published release candidate and the current `main` branch.

| Version | Support |
| --- | --- |
| `v0.1.0-rc.1` | Supported |
| `main` | Supported during active development |
| Older pre-release builds | Best effort only |

The authoritative list of published versions is the [GitHub Releases page](https://github.com/Marwanmorsy999/pivot/releases).

## Reporting a Vulnerability

**Please do not open a public GitHub issue for an undisclosed security vulnerability.**

Use GitHub's [private security advisory](https://github.com/Marwanmorsy999/pivot/security/advisories/new) mechanism whenever possible. Include:

- A clear description of the issue and affected component.
- Reproduction steps or a minimal proof of concept.
- Expected and observed behavior.
- Potential impact, including data exposure, arbitrary command execution, or privilege escalation where applicable.
- The affected PIVOT version or commit.
- Any suggested mitigation or fix.

Please avoid submitting live credentials, API keys, personal data, or other secrets in a report.

## Response Targets

These are engineering targets rather than guarantees:

| Stage | Target |
| --- | --- |
| Initial acknowledgment | Within 48 hours |
| Initial assessment | Within 7 days |
| Fix or mitigation | As soon as practical based on severity and complexity |
| Coordinated disclosure | Agreed with the reporter |

## Operational Security

### Credentials

PIVOT configuration may contain provider credentials. Protect the local configuration and never commit secrets:

```bash
chmod 600 ~/.pivot/config.yaml
chmod 600 ~/.pivot/state.db
```

Prefer environment-specific secret stores or provider mechanisms when available. Rotate any credential that may have been exposed.

### AI-Generated Commands

AI-generated task plans can result in local command execution. Treat generated plans as untrusted input:

- Review task graphs before running high-impact operations.
- Use worktree isolation when working on important repositories.
- Avoid running PIVOT with unnecessary operating-system privileges.
- Do not point an AI provider at sensitive data unless you understand where that data will be sent.

### Network Behavior

PIVOT communicates with configured AI/provider endpoints. HTTP clients use explicit request timeouts to reduce the risk of indefinitely hanging requests.

PIVOT does not intentionally collect telemetry or analytics data as part of its normal CLI operation.

### Dependencies

Dependency health is checked in CI with `govulncheck` and security scanning. Dependabot is used where configured by the repository.

For local review, use the same classes of checks used by CI rather than relying on a single package-audit command.

## Security Controls in CI

The main branch is continuously validated with:

- Go test suites on Go `1.26` and `1.27`.
- Race-detector test execution.
- `golangci-lint`.
- `gosec` with high severity/high confidence findings enforced.
- `govulncheck`.
- CodeQL analysis.
- Build and binary smoke tests.

Release builds additionally validate platform binaries, generate an SPDX SBOM, and publish SHA-256 checksums.

See [`docs/VERIFICATION.md`](docs/VERIFICATION.md) for the current dated evidence record.

## Contributor Security Checklist

Before submitting a security-sensitive change:

- [ ] No hardcoded secrets or credentials.
- [ ] External input is validated and handled safely.
- [ ] Database access uses parameterized queries.
- [ ] Command execution does not introduce avoidable shell injection.
- [ ] Errors are handled without leaking sensitive data.
- [ ] Tests cover security-relevant behavior.
- [ ] CI, CodeQL, `gosec`, and `govulncheck` remain clean.
