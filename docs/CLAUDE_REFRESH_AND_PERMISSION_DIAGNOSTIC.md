# Claude Refresh and Permission Diagnostic

Date: 2026-06-07

## Symptoms

- Claude.ai connects to the Hugo MCP connector.
- The UI sometimes shows 12 tools and sometimes falls back to 10 tools.
- The refresh button can still show `Impossible de recharger les outils depuis le serveur`.
- Claude can still prompt for approval even when the connector was previously marked `Always allow`.

## Live Catalog Comparison

Captured `tools/list` responses on the live services show:

- Native Go backend on `192.168.122.69:18181`: 12 tools
- VM shim on `192.168.122.69:18180`: 12 tools
- Legacy Python backend on `192.168.122.69:8000`: 10 tools

Tool names observed on the native Go path:

- `build_site`
- `check_sri_versions`
- `create_page`
- `delete_page`
- `generate_featured_image`
- `get_asset_chunk`
- `get_page`
- `get_page_chunk`
- `list_assets`
- `list_pages`
- `update_page`
- `upload_asset`

Tool names observed on the Python path:

- `list_pages`
- `get_page`
- `create_page`
- `update_page`
- `delete_page`
- `build_site`
- `upload_asset`
- `list_assets`
- `check_sri_versions`
- `generate_featured_image`

The two missing Python-era tools are the chunk helpers:

- `get_page_chunk`
- `get_asset_chunk`

After redeploying the corrected binary, the live native backend now exposes the MCP metadata hints in `tools/list`:

- `title`
- `annotations.readOnlyHint`
- `annotations.destructiveHint`

Examples from the live native backend after redeploy:

- `list_pages`
  - `title`: `List pages`
  - `annotations.readOnlyHint`: `true`
- `create_page`
  - `title`: `Create page`
  - `annotations.destructiveHint`: `true`

## Python vs Go Tool Schema

Current live captures show:

- Python tools expose `name`, `description`, and `inputSchema`.
- Native Go and shim tools expose `name`, `description`, `inputSchema`, `outputSchema`, `title`, and `annotations`.

That means the live catalog now exposes the MCP metadata Claude’s connector docs recommend for tool classification:

- `title`
- `readOnlyHint` for read-only tools
- `destructiveHint` for tools that modify or delete data

The repository adds those fields in code and the native backend now exposes them after redeploy.

## Tool Hints Audit

Per Claude’s MCP connector docs, every tool should include a title and the applicable hint.

Read-only tools in this project should be classified as read-only:

- `list_pages`
- `get_page`
- `get_page_chunk`
- `list_assets`
- `get_asset_chunk`

Mutating or destructive tools should be classified as destructive:

- `create_page`
- `update_page`
- `delete_page`
- `build_site`
- `upload_asset`
- `check_sri_versions`
- `generate_featured_image`

Earlier captures in this session did not expose these hints, which is a plausible contributor to the earlier approval churn. The current live native backend does expose them after redeploy.

## Stability

Repeated `tools/list` sampling showed:

- Native Go backend: 100 / 100 identical hashes
- VM shim: 100 / 100 identical hashes

The legacy Python backend was not stable under burst sampling:

- 71 samples returned the 10-tool catalog
- 29 samples returned `{"error":"Rate limit exceeded: 60 per 1 minute"}`

So the 12 -> 10 symptom is not backend instability on Go.
It is consistent with:

- Claude reusing a cached 10-tool snapshot from the Python era, or
- Claude failing a refresh and falling back to the older cached catalog

## `list_assets.path_prefix`

Observed semantics on the live Go backend:

- `path_prefix` is root-relative under the Hugo content/static roots
- exact prefixes match
- partial prefixes do not substring-match
- traversal attempts are rejected

Examples:

- `posts/debug-seo-404-broken-links` -> matches assets under that bundle
- `posts/debug-seo-404` -> no results
- `content/posts/debug-seo-404-broken-links` -> no results
- `../escape` -> rejected

So `path_prefix` is not a bug; it is exact-prefix behavior and should be documented that way.

## `generate_featured_image.accent`

The current validation is strict:

- accepted format: 6-digit hex color like `#7aa2f7`
- rejected: non-hex strings and unsupported names

That strictness is intentional in the current implementation and should be documented rather than loosened unless the repo explicitly decides to support a named-color allowlist.

## Refresh Logs

During the observed Claude refresh on the native backend:

- the native backend received successful `POST /mcp` requests
- the shim did not receive new traffic in the same window
- Python did not receive new traffic in the same window
- the native backend logs did not show a server-side failure

That means the refresh problem is not proven to be a Go runtime failure.

## Most Likely Cause

Most likely explanation:

- Claude refresh/permission state is stale or client-side
- the old 10-tool Python catalog is still cached somewhere in the connector state
- the earlier lack of live metadata hints likely contributed to approval churn, but that gap is now fixed on the native backend

No evidence in the logs proves a runtime failure in `hugo-mcp-go`.

## Recommendation

1. Reconnect or remove/re-add the Claude connector once after the schema change.
2. Re-run the refresh and confirm whether Claude keeps the 12-tool catalog.
3. If the refresh error persists after the metadata fix, treat it as a Claude-side refresh/cache bug.

## Verdict

- bug backend Go: no runtime failure proven; the metadata gap has been fixed and is now visible live
- bug mcp-runtime-go: no evidence
- bug/limitation Claude.ai: probable
- correction necessary before release: only if Claude refresh/Always Allow still fails after reconnecting to the updated catalog
