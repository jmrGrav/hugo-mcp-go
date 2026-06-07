# Native Streaming Transport Report

Date: 2026-06-07

## Summary

`hugo-mcp-go` now supports an explicit native HTTP transport in addition to the existing stdio mode.
The HTTP path is compatible with the current shim contract on `POST /mcp`, while large content is handled through chunking and pagination at the tool layer.

SSE support exists as an optional backend-only capability for direct backend tests and progress streaming, but it is not required for the production cutover path.

## Implemented Transport

- stdio remains the default transport
- `HUGO_MCP_TRANSPORT=http` enables HTTP mode explicitly
- `POST /mcp` is the compatibility endpoint
- `/mcp/events` is available only when streaming is enabled
- bearer auth is required in HTTP mode
- token sources: `HUGO_MCP_HTTP_TOKEN` or `HUGO_MCP_HTTP_TOKEN_FILE`
- loopback bind remains the default unless a VM bind address is configured

## Chunking and Pagination

The backend now supports chunk-oriented follow-up tools and paginated listings:

- `list_pages` supports cursor and limit parameters
- `list_assets` supports cursor-based pagination
- `get_page_chunk` returns bounded page slices
- `get_asset_chunk` returns bounded asset chunks
- oversized page reads fail with a structured suggestion to use chunking

## Progress and Streaming

Long-running operations can emit progress notifications when a progress token is present:

- `build_site`
- `check_sri_versions`
- `generate_featured_image`

Streaming is optional and backend-only in this mission. It does not require any change to `mcp-runtime-go`.

## Tests Added

Verification coverage now includes:

- transport config parsing and defaults
- HTTP auth failures and method guards
- initialize, tools/list, and tool-call compatibility on `/mcp`
- request body size rejection
- chunking and pagination for pages and assets
- redacted logs
- optional streaming route behavior and progress notifications

## Packaging

Added deployment artifacts:

- `deploy/systemd/hugo-mcp-go-http.service`
- `deploy/systemd/hugo-mcp-go-http.env.example`

These are staging/rollout artifacts only. They do not replace the existing shim automatically.

## Smoke

Added a dedicated smoke script:

- `scripts/native-http-smoke.sh`

This script exercises:

- initialize
- tools/list
- `list_pages`
- `get_page`
- `get_page_chunk`
- `list_assets`
- `get_asset_chunk`
- `build_site`
- `check_sri_versions`

## VM Validation Notes

The native service required a few VM-side fixes before the smoke could pass:

- open `18181/tcp` from `192.168.122.1` in UFW so the host can reach the backend
- add `SupplementaryGroups=hugo-mcp-shim` to the service so it can read existing content and static files owned by that group
- change ownership of `/var/lib/hugo-mcp-go/public` to `hugo-mcp:hugo-mcp` so Hugo can build into the destination directory

After those corrections:

- `GET /mcp` returned `405 Method Not Allowed`
- `initialize` returned the fixed shim-compatible `200` JSON-RPC response
- `tools/list` returned the native tool catalog
- `get_page` and `get_page_chunk` worked for a real page and language pair
- `get_asset_chunk` worked for a real static asset
- `build_site` returned `structuredContent.status == "built"`
- `check_sri_versions` returned the expected dry-run result

## Catalog Notes

- the native backend now exposes 12 tools, including the chunk helpers `get_page_chunk` and `get_asset_chunk`
- `list_assets.path_prefix` is exact and root-relative under the Hugo content/static roots; partial substrings do not match
- `generate_featured_image.accent` remains strict hex `#RRGGBB` in the current implementation
- the repo now adds explicit MCP `title` and read-only/destructive hints for the tool catalog so Claude can classify tools consistently after redeploy

## Production Cutover Notes

- gateway URL updated from `http://192.168.122.69:18180/mcp` to `http://192.168.122.69:18181/mcp`
- cutover time: 2026-06-07 18:05 CEST
- public `POST /mcp` probe without bearer: `401 Unauthorized` with `Bearer token required`
- public `GET /mcp` probe: `405 Not Allowed`
- host gateway remained on `127.0.0.1:8086`
- native backend remained active on `192.168.122.69:18181`
- shim service remained installed for rollback
- Python backend remained installed for rollback
- live native `tools/list` after redeploy includes `title`, `annotations.readOnlyHint`, and `annotations.destructiveHint`

## Rollback

Rollback remains unchanged:

- restore `HUGO_MCP_URL` to the last known good backend URL
- restart `mcp-runtime.service`
- keep Python and the shim available until native HTTP is fully validated

## Verdict

- HTTP native transport implemented: yes
- SSE/streaming backend capability: yes, optional
- chunking/pagination for large content: yes
- stdio preserved: yes
- shim still on active path: no, the shim is rollback-only and the active path is native HTTP
- compatible with current `mcp-runtime-go` contract: yes for `/mcp`
- rollback ready: yes
- direct native smoke on VM/host path: yes
