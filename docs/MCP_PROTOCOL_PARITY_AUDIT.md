# MCP Protocol Parity Audit

Scope:

- live Python backend: `https://192.168.122.69:8000/mcp`
- local Go shim: `http://127.0.0.1:18181/mcp`
- same JSON-RPC request set on both sides
- token values redacted

Status note:

- This audit is historical.
- The blocker fixes are recorded in [`MCP_PROTOCOL_PARITY_FIX_REPORT.md`](/home/jm/Documents/hugo-mcp-go/docs/MCP_PROTOCOL_PARITY_FIX_REPORT.md).
- The verdicts below describe the pre-fix state.

Note:

- I used the live backend directly because the public OAuth bearer for `https://mcp-hugo.arleo.eu/mcp` was not available in this session.
- That is still the current Python source of truth for the backend contract.

## Executive Summary

The Go path is not protocol-parity with the live Python backend.

The biggest differences are:

- `initialize` returns a different `protocolVersion`, different `serverInfo`, and different `capabilities`
- `tools/list` exposes a different catalog, different ordering, different descriptions, and different schemas
- `resources/list` and `prompts/list` are unsupported on Python but return empty lists on Go
- error semantics differ for unknown methods, invalid params, and notification handling
- `id: null` is a hard failure on Go and a normal success on Python

The most plausible Claude reload trigger is the Go session lifecycle gate:

- Python answers `tools/list` even before `initialize`
- Go rejects `tools/list` during session initialization with a session-state error

That is a real compatibility risk if Claude or the gateway can issue discovery requests before the Go child has fully transitioned to initialized.

## Verdict Claude Reload

Claude reload compatible: `no`

Reason:

- the discovery sequence is not equivalent between Python and Go
- the Go path is stricter about session state
- `id: null` is not safe on Go

## Initialize Parity

| Field | Python | Go | Delta |
| --- | --- | --- | --- |
| `protocolVersion` | `2025-03-26` | `2025-11-25` | different protocol version |
| `serverInfo.name` | `hugo-mcp` | `hugo-mcp-go` | different server identity |
| `serverInfo.version` | `1.0.0` | `0.1.0` | different versioning |
| `capabilities` | `{"tools":{}}` | `{"logging":{},"tools":{"listChanged":true}}` | Go advertises more capabilities |

Python initialize body:

```json
{"protocolVersion":"2025-03-26","capabilities":{"tools":{}},"serverInfo":{"name":"hugo-mcp","version":"1.0.0"}}
```

Go initialize body:

```json
{"capabilities":{"logging":{},"tools":{"listChanged":true}},"protocolVersion":"2025-11-25","serverInfo":{"name":"hugo-mcp-go","version":"0.1.0"}}
```

## Tools/List Parity

### Tool count and order

- Python returns 10 tools
- Go returns 8 tools
- Python order is not the same as Go order

Python tool names:

1. `list_pages`
2. `get_page`
3. `create_page`
4. `update_page`
5. `delete_page`
6. `build_site`
7. `upload_asset`
8. `list_assets`
9. `check_sri_versions`
10. `generate_featured_image`

Go tool names:

1. `build_site`
2. `create_page`
3. `delete_page`
4. `get_page`
5. `list_assets`
6. `list_pages`
7. `update_page`
8. `upload_asset`

### Missing and extra tools

- Missing from Go, present in Python:
  - `check_sri_versions`
  - `generate_featured_image`
- Present in Go, absent from Python:
  - none

### Schema and metadata deltas

- Python tool entries have `name`, `description`, and `inputSchema`
- Go tool entries also include `outputSchema`
- Python tool descriptions are French and much more specific
- Go descriptions are short English summaries
- Python schemas include defaults, enums, and richer field docs
- Go schemas are stricter in some places, but also less expressive in others

Concrete example:

- Python `list_assets` input includes `type` enum, `path_prefix`, and `max_results` with defaults and descriptions
- Go `list_assets` input exposes the same fields, but without the Python enum/default metadata

## Resources/List Parity

Python direct backend:

```json
{"jsonrpc":"2.0","id":3,"error":{"code":-32601,"message":"Method not found: resources/list"}}
```

Go shim:

```json
{"jsonrpc":"2.0","id":3,"result":{"resources":[]}}
```

Assessment:

- Python does not support `resources/list`
- Go does support it, but only as an empty list
- this is a real protocol-contract divergence

## Prompts/List Parity

Python direct backend:

```json
{"jsonrpc":"2.0","id":4,"error":{"code":-32601,"message":"Method not found: prompts/list"}}
```

Go shim:

```json
{"jsonrpc":"2.0","id":4,"result":{"prompts":[]}}
```

Assessment:

- Python does not support `prompts/list`
- Go does support it, but only as an empty list
- this is a real protocol-contract divergence

## Error Semantics Parity

| Case | Python | Go | Assessment |
| --- | --- | --- | --- |
| unknown method | `-32601 Method not found: does/not_exist` | `code 0 JSON RPC not handled: "does/not_exist" unsupported` | different error code and wording |
| invalid params (`tools/list`, `params: 123`) | succeeds and returns tools list | `code 0 handling 'tools/list': unmarshaling "123" ...` | Python ignores, Go rejects |
| auth missing | `401` JSON `{"detail":"Unauthorized"}` | `401` plain text `unauthorized` | transport-level mismatch |
| session not initialized (`tools/list` before init) | succeeds | `code 0 method "tools/list" is invalid during session initialization` | major lifecycle mismatch |
| request id string | succeeds | succeeds | parity on acceptance |
| request id number | succeeds | succeeds | parity on acceptance |
| request id null | succeeds with `id: null` | `504 Gateway Timeout` after `context deadline exceeded` | blocking mismatch |
| notification without id | `-32601 Method not found: notifications/initialized` | `202 Accepted` | different contract |
| notification with id | `-32601 Method not found: notifications/initialized` | `-32600 invalid request: unexpected id for "notifications/initialized"` | different contract |

## Gaps Bloquants

1. Session lifecycle mismatch on discovery.
   - Python accepts `tools/list` before `initialize`.
   - Go rejects `tools/list` while the session is still initializing.
   - If Claude or the bridge emits discovery too early, the Go path can surface the reload error.

2. `id: null` is unsafe on Go.
   - Python returns a normal response.
   - Go hangs until timeout and returns `504`.
   - This is a protocol-level bug in the Go bridge or child handling path.

3. `notifications/initialized` is not aligned.
   - Python direct backend does not implement it.
   - Go shim does.
   - The adapter boundary must define which side owns this notification.

## Gaps Non Bloquants

1. `resources/list` and `prompts/list`.
   - Python says method not found.
   - Go returns empty lists.
   - This is a contract difference, but it is less likely than session state to explain the reload alert by itself.

2. Tool catalog drift.
   - Python currently exposes 10 tools.
   - Go currently exposes 8 tools.
   - The two Python-only tools are `check_sri_versions` and `generate_featured_image`.

3. Schema drift.
   - Python and Go expose different descriptions, defaults, and schema metadata.
   - Go also adds `outputSchema` that the Python capture does not expose.

## Corrections Recommended

1. Fix the Go bridge handling for `id: null`.
   - Bug class: Go/shim
   - Target behavior: never hang a JSON-RPC request with a null id
   - Minimal change: return a deterministic JSON-RPC error instead of waiting for child timeout

2. Decide and codify the session-init contract.
   - Bug class: shim/protocol boundary
   - Target behavior: either mirror Python's lenient discovery behavior, or ensure the gateway never dispatches discovery before initialized is complete
   - Minimal change: if Claude reload can race, buffer or gate the request sequence so `tools/list` never lands in the Go child before initialization is complete

3. Align discovery policy for `resources/list` and `prompts/list`.
   - Bug class: behavior mismatch
   - Target behavior: either mirror Python's `Method not found` response or document the Go empty-list contract as intentional
   - Minimal change: make the choice explicit in the transport contract

4. Decide whether the Python-only tools are in scope for parity.
   - Bug class: product/catalog delta
   - Target behavior: either add them to Go or explicitly document them as out of scope
   - Minimal change: if the tools are intentionally absent, filter them from the parity promise instead of leaving them implicit

## Final Conclusion

- Claude reload compatible: `no`
- gap bloquant: `yes`
- code change requis: `yes`
- canary prolonge possible: `yes`, as observation only
- cutover possible: `no`
