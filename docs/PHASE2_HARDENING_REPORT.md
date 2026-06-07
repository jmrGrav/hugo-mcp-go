# Phase 2 Hardening Report

## Scope

This sprint hardened the existing Phase 2 staging-only implementation without changing the functional scope:

- no production cutover
- no new product features
- no changes to `mcp-runtime-go`
- no changes to the Python oracle

## Changes made

### Filesystem hardening

- added `openat`-style helpers in `internal/security/pathguard/openat.go`
- replaced path-based final writes in page mutations with parent-dir anchored writes
- replaced path-based asset uploads with anchored parent-dir writes
- replaced delete rollback backup and path removal with anchored `work/` temp files plus `unlinkat`
- kept staging validation and existing route lookup logic intact

### Validation hardening

- added explicit fail-closed checks for missing required fields in mutation services
- added fail-closed checks for missing staging wiring in page, asset, and build services
- bounded `upload_asset` payload size before base64 decode
- preserved redacted build stderr in failure errors

### Test hardening

- added `openat` helper tests in `internal/security/pathguard/pathguard_test.go`
- added unreadable-file skip coverage for `list_pages`
- added MCP error-path tests for missing args, traversal, invalid base64, symlink escape, nil deps, missing staging, timeout, and runner failure
- added build wiring negative-path tests

## Tests added or expanded

- `internal/security/pathguard/pathguard_test.go`
- `internal/hugo/pages/pages_test.go`
- `internal/config/config_test.go`
- `internal/hugo/staging/staging_test.go`
- `internal/hugo/mutations/build_test.go`
- `internal/tools/tools_test.go`

## Coverage before/after

### Before this sprint

- `internal/config` 59.4%
- `internal/hugo/mutations` 71.5%
- `internal/hugo/pages` 70.7%
- `internal/hugo/staging` 62.2%
- `internal/security/pathguard` 66.4%
- `internal/tools` 48.3%

### After this sprint

- `internal/config` 60.9%
- `internal/hugo/mutations` 70.4%
- `internal/hugo/pages` 72.4%
- `internal/hugo/staging` 67.6%
- `internal/security/pathguard` 66.3%
- `internal/tools` 69.0%

## TOCTOU status

- improved: the final write/delete operations now use dirfd-anchored `openat`/`renameat2`/`unlinkat` paths instead of only path-string I/O
- still open: read-before-write target selection and some rollback orchestration still happen before the final anchored hop, and a full tree-wide dirfd refactor would be a larger Linux-specific rewrite
- current recommendation: keep the anchored final I/O hardening, and defer the tree-wide refactor unless we decide the remaining race window is unacceptable

## Findings from Superpower/Brooks

### Security / Filesystem

- confirmed that the remaining risk is mostly around intermediate directory creation and rollback staging, not the final file mutation step
- confirmed that a full tree-wide dirfd rewrite is feasible but materially more complex than the current change set

### MCP / Error / Testability

- confirmed the previous tests were too happy-path heavy
- confirmed that MCP tool adapters now have direct negative-path coverage for invalid args and missing wiring
- confirmed that build service negative paths are now exercised through MCP as well as directly

## Remaining risks

- read-before-write target selection still happens before the final dirfd-anchored mutation step
- read-only listing still skips some file-level errors by design to preserve oracle parity
- build/deploy promotion, featured image, SRI, Cloudflare hooks, IndexNow, and Google hooks remain deferred

## Go / no-go

- Go for staging-only shadow/write testing
- Go for further Phase 2 validation in the current repo
- No-go for production cutover
- No-go for expanding scope beyond the current mutation set

## Final audit follow-up

- the MCP layer rejects unknown tools, unknown fields, and invalid argument types before domain code runs
- oversized upload payloads are rejected before base64 decode
- the remaining race window is still limited to read-before-write orchestration and rollback staging, which is acceptable for staging-only use

## Next prompt recommended

- If we continue hardening, the next useful step is a focused audit of intermediate directory creation and rollback staging to decide whether a tree-wide dirfd refactor is worth the complexity.
