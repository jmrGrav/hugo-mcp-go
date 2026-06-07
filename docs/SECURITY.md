# Security Invariants

`hugo-mcp-go` is read-only in Phase 1 and must fail closed on filesystem ambiguity.

## Mandatory rules

- path traversal is rejected
- symlink escape is rejected
- writes are not allowed in Phase 1
- shell execution is not allowed
- `sh -c` and `bash -lc` are forbidden
- all process execution, when added later, must use `exec.CommandContext`
- timeouts are required for any future long-running operation
- payload sizes are bounded
- logs must be redacted
- final file mutation is anchored through dirfd/openat-style helpers
- mutation error paths are tested through MCP for invalid args, unknown tools, unknown fields, oversized uploads, and build failures
- fixtures must remain secret-free
- no hardcoded production paths

## Path handling

Allowed paths must be relative to configured allowlisted roots.

The implementation rejects:

- `..`
- absolute user-supplied paths
- dangerous backslashes
- roots that do not exist
- symlink escapes
- resolved paths outside the allowlisted root

## Repository boundaries

- `hugo-mcp-go` is a dedicated repo
- `mcp-runtime-go` remains the OAuth/gateway layer
- `hugo-mcp-go` does not import internal packages from `mcp-runtime-go`
- no plugin discovery is used
- residual read-before-write races are accepted only for staging-only use and are documented in the Phase 2 audit

## Fixture policy

- no tokens
- no bearer headers
- no secrets
- no prod-only environment data
- no unnecessary absolute paths
- no positive fixtures that depend on symlink escape behavior

## Phase 1 scope

Phase 1 exposes only:

- `list_pages`
- `get_page`
- `list_assets`

Phase 2 adds staging-only writes and build execution behind injected wrappers. Production cutover remains deferred.
