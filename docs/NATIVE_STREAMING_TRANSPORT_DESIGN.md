# Native Streaming Transport Design

**Status:** approved direction for implementation

## Goal

Add a native HTTP transport to `hugo-mcp-go` that can replace the active `hugo-mcp-shim` path without changing `mcp-runtime-go`, while keeping the existing `stdio` mode intact.

## Scope

This design covers:

- a native HTTP mode for `hugo-mcp-go`
- compatibility with the current shim contract on `POST /mcp`
- chunking and pagination for large pages, assets, and long-running results
- optional backend-only SSE support for progress and direct staging tests
- configuration, security, tests, and rollout on the VM

This design explicitly does not cover:

- any change to `mcp-runtime-go`
- any public SSE relay through `mcp-runtime-go`
- removal of Python rollback services
- removal of `hugo-mcp-shim` before validation
- Cloudflare or OpenResty changes unless a routing defect is proven

## Current State

`hugo-mcp-go` is currently a stdio MCP server:

- `cmd/hugo-mcp-go/main.go` calls `svc.RunStdio(ctx)`
- `internal/server.Service` currently exposes only `RunStdio`
- the active production path still uses `hugo-mcp-shim` as the HTTP adapter

The shim already proves the HTTP contract we need to preserve:

- `POST /mcp`
- bearer auth on the backend token
- `initialize` returns `200`
- `notifications/initialized` returns `202`
- `tools/list` returns `200`
- errors are redacted
- request bodies are size-limited

## Proposed Architecture

### Transport modes

`hugo-mcp-go` will support two explicit modes:

- `stdio` default mode, unchanged
- `http` opt-in mode, enabled by config or environment

The binary remains the same. Only the transport changes.

### HTTP mode

The native HTTP mode will expose:

- `POST /mcp` for JSON-RPC requests
- optional backend-only SSE endpoints for direct progress streaming

The HTTP mode will reproduce the shim contract for the public path:

- same auth behavior
- same status codes
- same JSON-RPC request and response shapes
- same log redaction behavior
- same fail-closed semantics on missing token or malformed auth

### Large payload strategy

Large content must not force SSH or a manual fallback.

The backend will support chunking and pagination through MCP-classic requests, not by requiring SSE:

- `get_page` may return a bounded slice or a pointer to chunks when the result exceeds configured limits
- `get_asset` or asset-equivalent reads follow the same pattern
- list operations can paginate via cursor or offset-style arguments
- oversized responses fail with a structured error that tells the client how to continue

SSE is additive and optional:

- useful for progress updates and direct backend testing
- not required for successful production cutover
- not a dependency of `mcp-runtime-go`

## Configuration

HTTP mode uses explicit environment variables:

- `HUGO_MCP_TRANSPORT=stdio|http`
- `HUGO_MCP_HTTP_BIND_ADDR`
- `HUGO_MCP_HTTP_BIND_PORT`
- `HUGO_MCP_HTTP_TOKEN`
- `HUGO_MCP_HTTP_TOKEN_FILE`
- `HUGO_MCP_STREAMING_ENABLED`
- `HUGO_MCP_MAX_CHUNK_BYTES`
- `HUGO_MCP_MAX_RESPONSE_BYTES`

Rules:

- default transport is `stdio`
- HTTP mode is disabled unless explicitly requested
- HTTP mode fails closed if no backend token is available
- bind defaults to loopback if not specified
- log output must redact tokens and absolute paths

## HTTP Contract

### `/mcp`

`POST /mcp` remains the compatibility endpoint.

Expected behavior:

- `GET /mcp` and other non-POST methods are rejected cleanly
- missing bearer returns `401`
- wrong bearer returns `401`
- malformed JSON returns `400`
- oversized request body returns `413`
- `initialize` returns `200` with the same MCP envelope used today
- `notifications/initialized` returns `202` when sent as a notification
- `tools/list` returns `200`
- normal tool calls return the same JSON-RPC envelope that the shim already preserves

### Optional streaming endpoints

For direct backend usage only, the HTTP transport may expose an optional SSE endpoint such as `/mcp/events`.

The intent is:

- stream operation progress
- stream warnings
- stream completion or failure
- support long-running tasks without polling when the backend is called directly

This endpoint is not part of the public gateway contract in this mission.

## Chunking and Pagination

Large result support should be implemented at the tool layer and not rely on transport tricks.

### Required behaviors

- bounded responses for `get_page`, `list_pages`, `list_assets`, and asset/page-related operations
- explicit continuation fields when a result is truncated
- deterministic chunk IDs or cursors
- clear error messages for requests that exceed maximum response sizes

### Result shape

When a result is too large to return in one response, the backend should return one of:

- a partial result with `nextCursor` or `chunkId`
- a result envelope that indicates truncation and how to continue
- a structured error that tells the client which follow-up tool call to use

The exact shape must remain MCP-friendly and JSON-RPC-compatible.

### Tool-level extensions

If necessary, add focused follow-up tools rather than overloading one tool with hidden behavior:

- `get_page_chunk`
- `get_asset_chunk`
- `get_operation_status`
- `stream_operation_events`

These should only be added if the existing tool surface cannot express chunking clearly enough.

## Long-Running Operations

Operations such as these may emit progress:

- `build_site`
- `check_sri_versions`
- `generate_featured_image`
- hook-driven post-build work

For direct backend HTTP use, progress events may be emitted as:

- `started`
- `progress`
- `warning`
- `completed`
- `failed`

Progress events must:

- avoid secrets
- avoid raw filesystem paths when redaction is required
- remain optional and non-blocking for the caller

## Security Model

### Auth

- HTTP mode requires a bearer token
- the token is read from a file or explicit env, never from logs
- missing or malformed bearer headers are rejected
- multiple auth schemes or ambiguous headers are rejected

### Network exposure

- default bind is loopback unless a private VM address is explicitly configured
- the backend must not listen publicly by default
- no permissive CORS unless a specific consumer requires it and it is reviewed

### Resource limits

- strict request body size limits
- strict response size limits
- strict chunk size limits
- explicit timeouts for long operations
- simple overload protection if necessary

### Logging

- redact bearer tokens
- redact absolute paths where they may leak private context
- never log raw request bodies containing secrets

## Implementation Boundary

The implementation should be contained to the Hugo MCP repo only.

Expected areas of change:

- transport selection in `cmd/hugo-mcp-go`
- HTTP transport implementation under `internal/server` or a new focused package
- config loading for transport, bind, token, and size limits
- tool behavior for pagination and chunked results
- unit and integration tests
- systemd examples under `deploy/systemd`
- operational documentation under `docs/`

## Validation Strategy

### Local validation

The following must continue to pass:

- `go test ./...`
- `go test -race ./...`
- `go vet ./...`
- `go test -coverprofile=coverage.out ./...`
- `go tool cover -func=coverage.out | tail -5`

### HTTP compatibility tests

Tests must prove:

- stdio mode still works unchanged
- HTTP auth failure returns `401`
- `POST /mcp` initialize works
- `notifications/initialized` returns `202`
- `tools/list` works
- oversized responses become structured continuation responses or explicit continuation errors
- logging remains redacted

### Staging validation

After code passes locally:

- deploy native HTTP mode on the VM on a distinct port such as `18181`
- smoke test direct backend HTTP
- switch `HUGO_MCP_URL` in `mcp-runtime-go` only after native HTTP smoke passes
- verify the shim stops receiving traffic after cutover

## Rollback

Rollback remains unchanged:

- restore the previous `HUGO_MCP_URL`
- restart `mcp-runtime.service`
- keep Python rollback intact
- keep the shim installed until native HTTP is validated

## Decision Summary

- keep `stdio` as the default binary mode
- add native HTTP as an explicit opt-in mode
- preserve the shim contract on `POST /mcp`
- add chunking/pagination as the primary answer to large results
- make SSE optional and backend-only for now
- defer any `mcp-runtime-go` SSE relay to a later mission
