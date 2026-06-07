# Tree and Docs Cleanup and Validation Report

Scope:

- repository: `hugo-mcp-go`
- focus: tree cleanup, docs rationalization, security-doc scan, and pre-canary validation
- no production change, no Python change, no gateway change, no `mcp-runtime.service` change

## Executive Summary

The repository was cleaned of obvious temporary root artifacts and the documentation set was rationalized around a current-status index.

Current operational status:

- Claude reload compatible: `yes`
- protocol blockers: `fixed`
- canary: `possible, observation only`
- cutover: `no`

What changed:

- added a docs index with a recommended reading order
- marked the parity audit as historical and pointed readers to the fix report
- added a non-destructive parity smoke script for local shim validation
- removed obvious temporary root artifacts

Validation was strong but not perfect:

- `go test ./...` passed
- `go test -race ./...` passed
- `go vet ./...` passed
- `go test -cover ./internal/...` passed
- local parity smoke against the shim passed
- `go test -cover ./...` failed because this Go toolchain does not have the `covdata` tool needed by the new coverage flow for `cmd/...`
- `govulncheck ./...` reported standard-library vulnerabilities in the local Go 1.25 toolchain

## Tree Audit

### Keep

- source code under [`cmd/`](/home/jm/Documents/hugo-mcp-go/cmd)
- source code under [`internal/`](/home/jm/Documents/hugo-mcp-go/internal)
- fixtures under [`testdata/`](/home/jm/Documents/hugo-mcp-go/testdata)
- release scripts under [`scripts/`](/home/jm/Documents/hugo-mcp-go/scripts)
- deployment artifacts under [`deploy/`](/home/jm/Documents/hugo-mcp-go/deploy)
- current status reports:
  - [`docs/MCP_PROTOCOL_PARITY_FIX_REPORT.md`](/home/jm/Documents/hugo-mcp-go/docs/MCP_PROTOCOL_PARITY_FIX_REPORT.md)
  - [`docs/MCP_PROTOCOL_PARITY_AUDIT.md`](/home/jm/Documents/hugo-mcp-go/docs/MCP_PROTOCOL_PARITY_AUDIT.md)

### Archive or Retain as Historical

These are kept in-tree because they document prior work, but they are historical rather than current authority:

- canary execution and rollback reports
- phase reports and hardening reports
- readiness reviews and runbooks
- shim implementation and validation reports

### Delete

- [`coverage.out`](/home/jm/Documents/hugo-mcp-go/coverage.out) removed because it was a transient local coverage artifact
- [`hugo-mcp-shim`](/home/jm/Documents/hugo-mcp-go/hugo-mcp-shim) removed because it was a transient locally built binary at the repository root

### Merge

- no physical file merge was required
- instead, [`docs/README.md`](/home/jm/Documents/hugo-mcp-go/docs/README.md) now serves as the index and status entrypoint

## Docs Cleanup

### New Index

Added [`docs/README.md`](/home/jm/Documents/hugo-mcp-go/docs/README.md) with:

- current status banner
- recommended reading order
- explicit note that the fix report is the source of truth for current Claude reload / parity status
- explicit note that the parity audit is historical

### Contradictions Corrected

The following historical/current tension was normalized:

- `docs/MCP_PROTOCOL_PARITY_AUDIT.md` previously described the pre-fix state and still said Claude reload was not compatible
- `docs/MCP_PROTOCOL_PARITY_FIX_REPORT.md` documents the corrected state
- `docs/README.md` now separates those two timeframes so readers do not treat the historical audit as the live verdict

No production-readiness, transport, or systemd document was rewritten in place because those documents are now readable as time-stamped operational records, not the current source of truth.

### Status Updates

Updated documentation status to:

- Claude reload compatible: `yes`
- protocol blockers: `fixed`
- canary: `possible, observation only`
- cutover: `no`

## Security Doc Review

### Commands Run

- targeted `rg` search across docs and repo for bearer/token/secret-style strings
- `gitleaks detect --source .`
- `trufflehog filesystem --no-update --json .`

### Findings

- no real secret was found in the documentation tree
- all token references reviewed were either placeholders, fingerprints, or operational descriptions
- no replacement was required

### Tool Notes

- `gitleaks` reported `0 commits scanned` because this checkout does not expose a Git repository at the current path
- `trufflehog` completed with `verified_secrets=0` and `unverified_secrets=0`

## Validation Results

### Passed

- `gofmt -l` on all `*.go` files returned no output
- `go test ./...`
- `go test -race ./...`
- `go vet ./...`
- `go test -cover ./internal/...`
- [`scripts/parity-smoke.sh`](/home/jm/Documents/hugo-mcp-go/scripts/parity-smoke.sh) against the local shim

### Failed or Limited

- `go test -cover ./...`
  - failure: `go: no such tool "covdata"` for `cmd/hugo-mcp-go` and `cmd/hugo-mcp-shim`
  - interpretation: local toolchain limitation, not a repo regression
- `govulncheck ./...`
  - reported 20 standard-library vulnerabilities from the local Go 1.25 toolchain
  - interpretation: environment/toolchain issue requiring a separate Go update decision

## Remaining Gaps

- `tools/list` is still not a full catalog parity match with Python
- `check_sri_versions` remains out of scope
- `generate_featured_image` remains out of scope
- tool ordering and schema metadata still differ from the Python oracle
- full `go test -cover ./...` is blocked by the local toolchain missing `covdata`
- `govulncheck` findings remain pending against the current Go toolchain version

## Verdict

- tree clean: `yes`
- docs coherent: `yes`
- tests max passed: `no`
- canary observation possible: `yes`
- cutover possible: `no`

## Files Modified

- [`docs/README.md`](/home/jm/Documents/hugo-mcp-go/docs/README.md)
- [`docs/MCP_PROTOCOL_PARITY_AUDIT.md`](/home/jm/Documents/hugo-mcp-go/docs/MCP_PROTOCOL_PARITY_AUDIT.md)
- [`docs/TREE_DOCS_CLEANUP_AND_VALIDATION_REPORT.md`](/home/jm/Documents/hugo-mcp-go/docs/TREE_DOCS_CLEANUP_AND_VALIDATION_REPORT.md)
- [`scripts/parity-smoke.sh`](/home/jm/Documents/hugo-mcp-go/scripts/parity-smoke.sh)
- [`README.md`](/home/jm/Documents/hugo-mcp-go/README.md)

## Files Deleted

- [`coverage.out`](/home/jm/Documents/hugo-mcp-go/coverage.out)
- [`hugo-mcp-shim`](/home/jm/Documents/hugo-mcp-go/hugo-mcp-shim)

## Notes

- No commit was created.
- No push was performed.
- No production service or gateway was modified.
