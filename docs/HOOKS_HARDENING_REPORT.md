# Hooks Hardening Report

## Scope

This report records the final coverage and robustness pass for `internal/hooks`, plus the small shim-side testability change used to run protocol smoke checks locally without touching the live runtime.

## Changes Made

- tightened secret-file permissions handling to accept only `0600` and `0640`
- added injectable test hooks for schema creation and ID generation fallback
- expanded provider, store, and pipeline coverage with real failure and success paths
- added optional `HUGO_MCP_CHILD_PATH` support in the shim so a local `hugo` binary can be injected for smoke validation
- aligned `scripts/tool-parity-smoke.sh` with the current branch protocol version and server identity

## Validation

- `go test ./internal/hooks -cover` passes
- `go test ./...` passes
- `go test -race ./...` passes
- `go vet ./...` passes
- `scripts/hooks-smoke.sh` passes
- `scripts/tool-parity-smoke.sh` passes against the local branch shim using:
  - `MCP_URL=http://127.0.0.1:18182/mcp`
  - `MCP_TOKEN=local-token`

## Coverage Snapshot

- `internal/hooks`: `90.0%`
- `internal/shim`: `90.5%`
- global: `80.5%`

## Remaining Notes

- the current branch is still not the production live runtime
- local protocol smoke uses a branch-local shim and a temporary fake `hugo` binary for deterministic build validation
- no secret material is stored in SQLite or emitted in logs/docs/MCP responses

