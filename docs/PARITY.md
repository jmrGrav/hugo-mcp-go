# Parity Strategy

This repository uses the Python `hugo-mcp` service as the behavioral oracle for both Phase 1 read-only tools and Phase 2 mutation tools.

## Captured tools

Phase 1:

- `list_pages`
- `get_page`
- `list_assets`

Phase 2:

- `create_page`
- `update_page`
- `delete_page`
- `upload_asset`
- `build_site`

## Fixture model

The parity fixtures live under `testdata/fixtures/oracle/` for Phase 1 and `testdata/fixtures/oracle_phase2/` for Phase 2. They are split into:

- raw JSON-RPC/MCP inputs
- raw Python responses
- normalized responses used for golden comparisons
- notes for any normalization step

## Normalization rules

- compare only deterministic fields in golden tests
- do not compare `modified` timestamps for asset parity
- use stable ordering in tests when filesystem ordering is not guaranteed
- keep path traversal and symlink failures as explicit negative fixtures

## Deterministic fields

Typical deterministic fields include:

- page route
- page language
- page section
- page title
- page kind
- page bundle metadata when it is derived from a stable fixture
- asset path
- asset size
- asset MIME type

## Non-deterministic or environment-dependent fields

Typical non-deterministic fields include:

- filesystem mtimes
- any time-based metadata not explicitly fixed in fixtures
- any ordering that depends on the host filesystem unless normalized in tests

## Open parity points

- Phase 1 intentionally does not invent a draft filter for `list_pages`
- Phase 1 reproduces the Python language fallback behavior in `get_page`
- Phase 1 does not add page-to-asset association for `list_assets`
- Phase 2 keeps `deploy` normalized in the staging harness
- Phase 2 keeps build, featured image, SRI, Cloudflare, IndexNow, and Google hooks deferred
- Phase 2 hardening now uses dirfd/openat-style final I/O anchoring for writes and deletes
- Phase 2 still keeps read-only listing skip behavior where needed to preserve oracle parity
- `list_pages` reports skipped unreadable entries through its `skipped` count
- `list_assets` keeps file-level stat failures skip-only rather than surfacing them
- mutation tool inputs reject unknown fields and invalid types through MCP schema validation before domain logic runs
- oversized upload payloads fail closed before base64 decode

## Practical rule

If a Python field is not confirmed by fixture capture, it remains unknown and is not invented in Go.
