# Phase 2 Report

## Status

Phase 2 staging-only implementation is complete for the five mutation tools, and the current hardening sprint has added dirfd/openat-style final I/O anchoring plus broader negative-path coverage. The repo now exposes the following MCP tools:

- `list_pages`
- `get_page`
- `list_assets`
- `create_page`
- `update_page`
- `delete_page`
- `upload_asset`
- `build_site`

Production cutover remains out of scope. `mcp-runtime-go` is unchanged.

## Files created or modified

### Domain and staging

- `internal/hugo/staging/staging.go`
- `internal/hugo/staging/paths.go`
- `internal/security/pathguard/write.go`
- `internal/security/pathguard/openat.go`
- `internal/hugo/mutations/frontmatter.go`
- `internal/hugo/mutations/pages.go`
- `internal/hugo/mutations/assets.go`
- `internal/hugo/mutations/build.go`
- `internal/runner/runner.go`

### MCP wiring

- `internal/tools/tools.go`
- `internal/server/server.go`

### Tests

- `internal/config/config_test.go`
- `internal/security/pathguard/write_test.go`
- `internal/security/pathguard/pathguard_test.go`
- `internal/hugo/staging/staging_test.go`
- `internal/hugo/pages/pages_test.go`
- `internal/hugo/mutations/pages_test.go`
- `internal/hugo/mutations/assets_test.go`
- `internal/hugo/mutations/build_test.go`
- `internal/runner/runner_test.go`
- `internal/tools/tools_test.go`

### Documentation

- `README.md`
- `docs/SECURITY.md`
- `docs/PARITY.md`
- `docs/PHASE2_HARDENING_REPORT.md`
- `docs/PHASE2_REPORT.md`

## Implemented behavior

### `create_page`

- staging-only write into the allowlisted content root
- path traversal rejected
- symlink escape rejected
- frontmatter conflict rejected
- auto-populates `date` and `lastmod` when absent
- returns the oracle shape with deployment normalized to `DEPLOY_SKIPPED`

### `update_page`

- reads the existing page first
- deep-merges frontmatter
- supports `null` deletion
- rejects immutable `date`
- preserves the oracle response shape

### `delete_page`

- resolves the existing page through the read-only reader first
- writes a rollback breadcrumb into staging `work/`
- removes the content file atomically from the staging tree
- rejects traversal and symlink escape

### `upload_asset`

- staging-only writes into `static/`
- extension allowlist enforced
- oversized base64 payloads are rejected before decode
- base64 decode validated
- size limit enforced
- traversal and symlink escape rejected

### `build_site`

- executed through an injected runner
- uses `exec.CommandContext` only in the runner wrapper
- no shell invocation
- timeout enforced
- stderr from a failed build is preserved in the returned error message
- result normalized to the oracle response shape

## Tests passed

- `go test ./...`
- `go test -cover ./internal/...`
- targeted package tests for staging, runner, mutations, tools, and server

## Coverage snapshot

- `internal/config` 60.9%
- `internal/hugo/assets` 67.7%
- `internal/hugo/frontmatter` 68.0%
- `internal/hugo/mutations` 70.4%
- `internal/hugo/pages` 72.4%
- `internal/hugo/staging` 67.6%
- `internal/observability` 75.0%
- `internal/runner` 95.5%
- `internal/security/pathguard` 66.3%
- `internal/server` 70.8%
- `internal/tools` 69.0%

## Oracle gaps and known normalization

- `deploy` remains normalized to `DEPLOY_SKIPPED` in the staging harness.
- `build_site` does not perform Cloudflare purge, featured image generation, SRI, IndexNow, or Google hooks.
- `list_assets` still normalizes non-deterministic `modified` values in golden comparisons.
- Phase 2 mutation parity is based on captured Python fixtures; anything not captured remains unknown and is not invented.

## Security review findings

Confirmed:

- path traversal rejection is enforced for existing and new targets
- symlink escape rejection is enforced for existing and new targets
- writes are confined to staging roots
- write, delete, and rollback backup I/O use dirfd/openat-style anchoring for their last hop
- build execution is runner-injected and shell-free
- direct `exec.Command` usage does not exist outside the runner wrapper

Risks kept open by design:

- there is still a residual race window around read-before-write target selection and rollback staging
- read-only list services still skip some file-level errors instead of surfacing them directly
- production cutover remains deferred
- full tree-wide dirfd refactor remains deferred
- featured image and SRI remain deferred
- future deploy integration must stay behind injected wrappers and explicit allowlists

## Go / no-go

- Go for staging-only read/write testing and internal validation.
- No-go for production cutover.
- No-go for any feature expansion beyond the five mutation tools and existing read-only tools.

## Next steps

1. Keep mutation validation staging-only until a separate cutover decision is made.
2. Preserve the oracle fixtures as the source of truth for future parity checks.
3. Add any future deploy integration only through injected wrappers and explicit safety checks.
