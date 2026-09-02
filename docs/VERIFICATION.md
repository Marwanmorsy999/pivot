# Repository Verification Record

> This is an evidence record, not a promise that future commits will remain green. Re-run the linked workflows after substantive changes.

## Baseline Verified

**Verification date:** 2026-09-02

**Verified `main` commit:** `d874832e82de8197acec50550ef5bda694ad37d7`

**Change validated:** Cobra `v1.9.1` → `v1.10.2`, with the required pflag update. The dependency metadata was regenerated with Go tooling rather than hand-editing checksums.

## Production CI Evidence

The following push-triggered workflows were executed against the verified `main` merge commit:

| Control | Run | Result |
| --- | --- | --- |
| CI | [run #85](https://github.com/Marwanmorsy999/pivot/actions/runs/33574168478) | ✅ Success |
| CodeQL | [run #76](https://github.com/Marwanmorsy999/pivot/actions/runs/33574168445) | ✅ Success |
| Docker | [workflow history](https://github.com/Marwanmorsy999/pivot/actions/workflows/docker.yml) | ✅ Pre-merge validation passed |

### CI jobs verified

Run #85 completed all six jobs successfully:

- Test (Go 1.26) — ✅
- Test (Go 1.27) — ✅
- Lint — ✅
- Build — ✅
- Security Scan (`gosec`) — ✅
- Go Vulnerability Check (`govulncheck`) — ✅

The test jobs included the Go race detector, and the build job also executed the compiled `pivot --help` smoke test.

### CodeQL

CodeQL run #76 completed its Analyze job successfully with no workflow failure.

## Release Pipeline Evidence

The repository's release workflow is configured to run for `v*` tags and validates release artifacts before publication. It currently:

1. Runs the race-enabled test suite.
2. Builds Linux amd64, macOS amd64, macOS arm64, and Windows amd64 binaries.
3. Smoke-tests the produced binaries.
4. Generates an SPDX SBOM.
5. Validates expected release files.
6. Generates SHA-256 checksums.
7. Publishes the release artifacts.

The latest published release candidate is [`v0.1.0-rc.1`](https://github.com/Marwanmorsy999/pivot/releases/tag/v0.1.0-rc.1).

## Repository Quality Controls

| Area | Evidence |
| --- | --- |
| Automated tests | GitHub Actions CI, Go 1.26 + 1.27 |
| Race detection | `go test -race` in CI |
| Static analysis | `golangci-lint` |
| Security scanning | `gosec` + `govulncheck` |
| SAST | CodeQL |
| Build verification | Go build + CLI smoke test |
| Container validation | Docker workflow |
| Release integrity | SBOM + SHA-256 checksums |
| Dependency management | Dependabot configuration and normal module hygiene |

## How to Reproduce Locally

From a clean checkout:

```bash
go mod download
go test -race ./...
go vet ./...
go build ./...
```

When `golangci-lint` is installed:

```bash
golangci-lint run ./...
```

For a release-like build, use the repository's release workflow as the source of truth for target platforms and artifact validation.

## Interpretation

A green verification record means the referenced commit passed the listed controls at the recorded time. It does **not** replace code review, threat modeling, or validation of behavior introduced by later commits.
