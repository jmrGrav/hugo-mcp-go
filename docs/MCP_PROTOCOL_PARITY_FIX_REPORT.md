# MCP Protocol Parity Fix Report

Scope:

- Python oracle: `https://192.168.122.69:8000/mcp`
- Go shim: `http://127.0.0.1:18181/mcp`
- Focus: blockers that could trigger Claude reload failures
- No prod mutation, no gateway cutover, no Cloudflare/OpenResty change, no Python change

## Executive Summary

The two hard blockers from the prior parity audit are fixed on the Go shim path:

- `id: null` now returns a deterministic JSON-RPC response and no longer hangs until HTTP timeout
- `tools/list` now succeeds before `initialize` in the shim path

The shim also now matches the Python oracle on:

- `initialize`
- `resources/list`
- `prompts/list`
- unknown methods
- `tools/list` with invalid params
- `notifications/initialized` handling

The remaining mismatch is the tool catalog itself:

- Python exposes 10 tools
- Go currently exposes 8 tools
- the two Python-only tools are `check_sri_versions` and `generate_featured_image`

Those two tools were not added in this fix. They are documented as out of scope for the blocker repair, so this report does not claim full `tools/list` catalog parity.

## Root Cause

The blocker was not one single bug.

### 1. Session gating in the Go child

The Go backend session layer rejected discovery requests before initialization completed. That made `tools/list` unsafe during Claude reload timing, because the client can race discovery before the child session is fully initialized.

### 2. Null request id handling

The Go transport path treated `id: null` as a special case that never completed cleanly, which surfaced as a timeout and HTTP 504 instead of a deterministic JSON-RPC response.

### 3. Discovery contract mismatch

The Go shim previously returned empty lists for `resources/list` and `prompts/list`, while the Python oracle returns JSON-RPC `-32601 Method not found`.

### 4. Handshake metadata drift

The Go shim advertised a different protocol version, server identity, and capability set from the Python oracle.

## Files Modified

- [`internal/shim/child.go`](/home/jm/Documents/hugo-mcp-go/internal/shim/child.go)
- [`internal/shim/server.go`](/home/jm/Documents/hugo-mcp-go/internal/shim/server.go)
- [`internal/shim/server_test.go`](/home/jm/Documents/hugo-mcp-go/internal/shim/server_test.go)
- [`docs/MCP_PROTOCOL_PARITY_FIX_REPORT.md`](/home/jm/Documents/hugo-mcp-go/docs/MCP_PROTOCOL_PARITY_FIX_REPORT.md)

## What Changed

### `initialize`

The shim now returns the Python oracle shape directly:

- `protocolVersion: "2025-03-26"`
- `serverInfo.name: "hugo-mcp"`
- `serverInfo.version: "1.0.0"`
- `capabilities: {"tools":{}}`

It no longer advertises `logging` or `tools.listChanged` in the client-facing initialize result.

### `tools/list`

The shim now accepts `tools/list` before `initialize` by bootstrapping the child internally before forwarding discovery traffic.

It also normalizes `tools/list` params to `{}` so invalid discovery params such as `123` do not break discovery.

### `id: null`

The Go transport now rewrites `id: null` to an internal synthetic id for the child request, then restores `null` on the response.

That prevents the timeout / 504 path while preserving the external JSON-RPC id semantics.

### `resources/list` and `prompts/list`

The shim now returns:

- `-32601 Method not found: resources/list`
- `-32601 Method not found: prompts/list`

This matches the Python oracle.

### `notifications/initialized`

The shim now treats the no-id notification as a normal notification and returns HTTP 202 with no body.

If an id is present, it returns a deterministic JSON-RPC `-32600` invalid-request error rather than hanging or misrouting the message.

### Unknown methods

The child bridge now rewrites unsupported-method responses to:

- `-32601 Method not found: <method>`

This matches the Python oracle and keeps Claude-facing errors stable.

## Tests Added

New and extended parity coverage in [`internal/shim/server_test.go`](/home/jm/Documents/hugo-mcp-go/internal/shim/server_test.go):

- `TestMCPProtocolParityInitializeAndDiscovery`
- extended `TestHTTPRequestIdPreserved` with `id: null`

Coverage now includes:

- `initialize`
- `tools/list` before `initialize`
- `tools/list` after `initialize`
- string request ids
- numeric request ids
- null request ids
- unknown methods
- invalid params for `tools/list`
- `resources/list`
- `prompts/list`
- `notifications/initialized` with and without id

Validation run:

- `go test ./...`
- local HTTP shim verification against `http://127.0.0.1:18181/mcp`

## Before / After

### Before

- `tools/list` before `initialize` could fail with a session-state error
- `id: null` could hang and end in HTTP 504
- `initialize` metadata diverged from the Python oracle
- `resources/list` and `prompts/list` returned empty lists instead of `-32601`
- unknown methods did not match Python error semantics

### After

- `tools/list` before `initialize` succeeds
- `id: null` succeeds and returns a deterministic JSON-RPC response
- `initialize` matches the Python oracle fields now used for Claude compatibility
- `resources/list` and `prompts/list` now match Python `-32601`
- unknown methods now match Python `-32601`
- `notifications/initialized` no longer causes an unexpected reload path

## Local Parity Evidence

The following local HTTP checks were run against the shim:

- `initialize` returned `protocolVersion: "2025-03-26"` and `serverInfo.name: "hugo-mcp"`
- `tools/list` before `initialize` returned HTTP 200 with a tools payload
- `tools/list` with `id: null` returned HTTP 200 with `"id": null`
- `resources/list` returned `-32601 Method not found: resources/list`
- `prompts/list` returned `-32601 Method not found: prompts/list`
- unknown method returned `-32601 Method not found: does/not_exist`
- `tools/list` with invalid params succeeded
- `notifications/initialized` without id returned HTTP 202
- `notifications/initialized` with id returned `-32600 invalid request`

## Remaining Gaps

### Non-blocking

- `tools/list` catalog is not a full parity match with Python
- Python-only tools still missing from Go:
  - `check_sri_versions`
  - `generate_featured_image`
- tool ordering and schema metadata still differ from the Python oracle

### Not changed in this fix

- Python backend
- `mcp-runtime.service`
- OpenResty / Nginx
- Cloudflare
- production gateway routing

## Verdict

- Claude reload compatible: `yes`
- gap bloquant: `no`
- code change requis: `yes`
- canary possible: `yes`, observation only
- cutover possible: `non`

## Notes

This report only claims parity for the blocker paths that can trigger Claude reload failures.

It does not claim full tool-catalog parity because `check_sri_versions` and `generate_featured_image` were intentionally left out of scope for this fix.
