# Release Validation Pipeline

## Purpose

This pipeline validates a `hugo-mcp-go` release candidate locally before any release attempt.
It does not touch `mcp-runtime.service`, `hugo-mcp.service`, OpenResty, Nginx, Cloudflare, or the live canary path.

## Prerequisites

- Bash
- `go`
- `gofmt`
- `govulncheck`
- `gitleaks`
- `trufflehog`
- write access to the repository working tree

Recommended versions:

- Go toolchain `go 1.25.11` or newer
- current `gitleaks`
- current `trufflehog`
- current `govulncheck`

The release pipeline forces `GOTOOLCHAIN=go1.25.11+auto` and rejects older toolchains.

## Commands

Run the full pipeline:

```bash
make release-check
```

Run individual stages:

```bash
make test
make race
make vet
make vuln
make secrets
make build
make build-all
make clean
```

## Validation Order

The release check runs in this order:

1. `gofmt` check
2. `go test ./...`
3. `go test -race ./...`
4. `go vet ./...`
5. `govulncheck ./...`
6. `gitleaks detect`
7. `trufflehog filesystem`
8. local build
9. multi-arch build
10. `sha256sum` generation and verification

## Output

- local build artifacts are written under `dist/local/<goos>_<goarch>/`
- multi-arch build artifacts are written under:
  - `dist/linux_amd64/`
  - `dist/linux_arm64/`
- checksums are written to `dist/SHA256SUMS`
- secret scanning skips generated binaries with `trufflehog --force-skip-binaries` because those artifacts are build outputs, not source.

## Tool Failure Policy

If `govulncheck`, `gitleaks`, or `trufflehog` are missing, the pipeline fails immediately and prints the recommended install command.

Recommended installs:

```bash
go install golang.org/x/vuln/cmd/govulncheck@latest
go install github.com/gitleaks/gitleaks/v8@latest
go install github.com/trufflesecurity/trufflehog/v3@latest
```

No tool is installed automatically.

If the active Go toolchain is older than `go1.25.11`, the pipeline fails before running the scanners.

## Interpretation

- `go test ./...` green: unit and integration-level tests passed
- `go test -race ./...` green: race detector found no concurrency issues in the test suite
- `go vet ./...` green: static vet checks passed
- `govulncheck ./...` green: no known vulnerabilities reported for the module graph
- `gitleaks` green: no leaked secrets detected in the scanned source tree
- `trufflehog` green: no leaked secrets detected in the scanned source tree
- build steps green: release binaries compiled successfully for local and multi-arch targets
- `SHA256SUMS` green: generated checksums match the produced artifacts

## Release Green Criteria

Release validation is green only if:

- every stage above passes
- no secret scan finding is accepted without a documented note
- the generated artifacts exist in `dist/`
- checksums are generated for all produced binaries

## Failure Procedure

If any stage fails:

1. stop the pipeline at the first failing stage
2. inspect the command output and artifact path
3. fix the code or document the accepted false positive
4. rerun the pipeline from the start

If a secret scanner flags a false positive, document the rationale in `docs/SECRET_SCAN_NOTES.md` before rerunning.
