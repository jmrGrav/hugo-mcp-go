# hugo-mcp-go

`hugo-mcp-go` is the dedicated Go MCP server for Hugo content access behind `mcp-runtime-go`.

Current scope:

- read-only MCP tools plus staging-only mutation tools
- explicit tool registration only
- no shell execution
- no production cutover
- no integration into `mcp-runtime-go`
- file-backed post-build hooks with SQLite operational state
- native `check_sri_versions` parity with the historical SRI plugin

Current status and documentation order:

- [`docs/README.md`](/home/jm/Documents/hugo-mcp-go/docs/README.md)

Implemented tools:

- `list_pages`
- `get_page`
- `list_assets`
- `create_page`
- `update_page`
- `delete_page`
- `upload_asset`
- `build_site`
- `check_sri_versions`
- `generate_featured_image`

Hook subsystem:

- Cloudflare purge after rebuild/update
- Google Indexing notifications
- IndexNow submission
- admin hooks tools are opt-in by config

## Layout

- `cmd/hugo-mcp-go/` entrypoint
- `internal/config/` env parsing and fail-closed validation
- `internal/security/pathguard/` path traversal and symlink guards
- `internal/hugo/frontmatter/` YAML frontmatter parsing/rendering
- `internal/hugo/pages/` page discovery and page loading
- `internal/hugo/assets/` asset discovery
- `internal/tools/` MCP tool registration
- `internal/server/` MCP server wiring
- `internal/hooks/` hook pipeline, providers, and SQLite store
- `testdata/fixtures/` oracle and minimal Hugo fixtures

## Configuration

Required environment variables:

- `HUGO_ROOT`
- `HUGO_CONTENT_ROOT`
- `HUGO_STATIC_ROOT`

Optional limits:

- `HUGO_MAX_REQUEST_BYTES`
- `HUGO_MAX_TOOL_ARGS_BYTES`
- `HUGO_MAX_PAGE_BYTES`
- `HUGO_MAX_ASSET_BYTES`
- `HUGO_MAX_LIST_PAGES`
- `HUGO_MAX_LIST_ASSETS`

## Development

Run tests:

```bash
go test ./...
```

The current implementation is Phase 2 staging-only for mutations and Phase 1 read-only for discovery. Production cutover remains deferred.
