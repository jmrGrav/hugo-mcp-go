# Security Invariants

This document is the internal companion to the public [root SECURITY policy](../SECURITY.md).

`hugo-mcp-go` is operator-controlled, single-tenant, and fail-closed on filesystem ambiguity.

## Mandatory rules

- path traversal is rejected
- symlink escape is rejected
- writes are not allowed unless explicitly enabled by the mutation tools
- shell execution is not allowed
- `sh -c` and `bash -lc` are forbidden
- any future long-running execution must use `exec.CommandContext`
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
- residual read-before-write races are accepted only where explicitly documented

## Fixture policy

- no tokens
- no bearer headers
- no secrets
- no production-only environment data
- no unnecessary absolute paths
- no fixtures that depend on symlink escape behavior

## Transport scope

The native backend now supports both stdio and native HTTP.

- stdio remains the default transport
- HTTP mode requires explicit configuration
- `/mcp` is the compatibility endpoint
- `/mcp/events` is optional and backend-only
- OAuth public routing remains the responsibility of `mcp-runtime-go`

## Release-candidate scope

The release candidate exposes:

- `list_pages`
- `get_page`
- `get_page_chunk`
- `list_assets`
- `get_asset_chunk`
- `create_page`
- `update_page`
- `delete_page`
- `upload_asset`
- `build_site`
- `check_sri_versions`
- `generate_featured_image`

MCP tool metadata must be stable enough for client refresh and approval state:

- `title`
- `annotations.readOnlyHint`
- `annotations.destructiveHint`
