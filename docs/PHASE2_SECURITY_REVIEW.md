# Phase 2 Security Review

## Scope

This review covers the future mutation surface only:

- `create_page`
- `update_page`
- `delete_page`
- `upload_asset`
- `build_site`

## Confirmed findings

### 1. Staging model was missing

The repository did not previously define an in-repo staging contract for mutations.

Status:

- addressed by [`docs/PHASE2_STAGING_MODEL.md`](/home/jm/Documents/hugo-mcp-go/docs/PHASE2_STAGING_MODEL.md)
- still a gate for implementation, not for planning

### 2. Read-only path guard is not sufficient for new targets

`ResolveExistingPath` only works for existing files, so it cannot be reused directly for:

- `create_page`
- `update_page`
- `delete_page`
- `upload_asset`

Required Phase 2 response:

- separate validation for existing targets and new targets
- parent-directory allowlist checks
- symlink-aware resolution for parents

### 3. Deployment output is external and must be wrapped

`build_site` and the write tools currently call an external deploy script in Python.

Phase 2 must ensure:

- `exec.CommandContext` only
- no shell
- bounded stdout/stderr
- explicit timeout
- injected wrapper for build/deploy

### 4. Mutation comparisons must ignore non-semantic file noise

Page and asset comparisons must not depend on:

- raw YAML formatting
- key ordering
- filesystem mtimes
- host-specific line endings

## Resolved hardening items

These are already addressed in the Go repo before Phase 2 implementation begins:

- startup error redaction is wired through `main.go`
- invalid numeric env vars now fail closed
- read-only scans honor context cancellation
- Phase 1 oracle fixtures are updated to include the Unicode page and uppercase asset

## Remaining risks

These are still open by design and must be handled during Phase 2 implementation:

- path validation for new files and directories
- atomic write and rollback semantics
- asset upload byte-for-byte comparison
- build promotion and rollback
- log redaction for any future structured output beyond the current startup path

## Recommendation

Proceed with Phase 2 planning and implementation only after the staging model is treated as a hard contract.

Do not start mutation code until:

- the staging model is approved
- the oracle_phase2 capture is accepted
- the implementation plan is reviewed for file-by-file scope
