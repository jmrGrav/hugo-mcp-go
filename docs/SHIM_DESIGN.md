# VM-Local MCP Shim Design

## Scope

This document defines a VM-local shim that exposes the HTTP `/mcp` contract expected by the gateway while speaking stdio to `hugo-mcp-go`.

## Non-Goals

- no production cutover
- no changes to the NUC gateway
- no changes to `mcp-runtime.service`
- no changes to OpenResty, Nginx, or Cloudflare
- no changes to `/home/jm/hugo-site`
- no changes to the live Python service

## Recommendation

Use a small Go shim process on the VM.

### Why Go

- the repo is already in Go
- the shim needs strict control over JSON-RPC, child process lifecycle, and timeouts
- Go gives a small static binary, easy systemd integration, and simple concurrency primitives
- the repo already has redaction, size limit, and staging conventions that the shim can reuse

## Process Model

The shim is a long-running daemon supervised by systemd.

- one shim process listens on the staging HTTP port
- one `hugo-mcp-go` child process is owned by the shim
- the shim keeps a single stdio session open to the child
- the shim translates each inbound HTTP `/mcp` request into a JSON-RPC message over the child stdio transport

The shim does not spawn a new `hugo-mcp-go` process per request. That would make initialization, session state, and error handling too fragile.

## Child Lifecycle

### Startup

1. systemd starts the shim
2. the shim validates environment, bind address, port, and child binary path
3. the shim spawns `hugo-mcp-go` as a child process with stdio pipes
4. the shim performs the MCP initialization handshake with the child
5. the shim only starts serving HTTP after the child is ready

### Normal operation

- the child remains alive for the life of the shim
- the shim tracks the child generation number
- all requests in flight are tied to the active child generation

### Child crash or exit

1. the shim marks the current child generation unhealthy
2. new requests receive `502 Bad Gateway` or `503 Service Unavailable` until a replacement child is ready
3. the shim restarts the child with backoff
4. after restart, the shim repeats the MCP initialization handshake

### Restart policy

- restart the child on unexpected exit
- use exponential backoff with a cap
- stop restarting after a small fixed number of rapid failures, then fail closed and keep the shim process alive only long enough to report the condition

This avoids a crash loop that would hide a bad child binary or a broken staging tree.

## HTTP to stdio Mapping

### Inbound contract

- the shim exposes `POST /mcp`
- the request body is JSON
- the body is treated as a JSON-RPC 2.0 message or batch only if batch support is explicitly enabled later
- unsupported methods and malformed payloads are rejected before they reach the child

### Outbound contract

- the shim writes one JSON-RPC message to the child stdio stream
- the shim reads JSON-RPC responses from the child stdout stream
- stderr is reserved for child diagnostics and is never parsed as protocol data

### Request flow

1. HTTP request arrives
2. shim validates content type, auth token, and size
3. shim parses JSON-RPC
4. shim assigns or preserves the request id
5. shim serializes the request onto the child stdin stream
6. shim waits for the matching child response
7. shim returns the matching HTTP response

## JSON-RPC Handling

The shim must preserve JSON-RPC semantics.

### Supported cases

- standard request/response pairs with `id`
- notifications without `id`
- JSON-RPC error objects from the child

### Request ids

- if the client sends an `id`, the shim preserves it exactly
- ids may be string, number, or null according to JSON-RPC
- the shim may use an internal monotonic sequence only for bookkeeping
- internal sequence values must never leak into the external response

### Error mapping

- malformed JSON body -> HTTP `400`
- missing or invalid token -> HTTP `401` or `403`
- request too large -> HTTP `413`
- unsupported protocol shape -> HTTP `400`
- child unavailable -> HTTP `502`
- child timeout -> HTTP `504`
- local overload -> HTTP `429`

The JSON-RPC body should still carry the structured error object when possible, even if the HTTP status is non-200.

## Concurrency Model

The stdio child is a single transport stream, so the shim should treat it as the serialization point.

Recommended model:

- HTTP requests may arrive concurrently
- the shim accepts them into a bounded queue
- one writer goroutine serializes JSON-RPC messages to child stdin
- one reader goroutine demultiplexes child stdout responses back to pending HTTP requests
- notifications bypass the response wait path

Practical limit:

- keep the in-flight request count small and bounded
- prefer correctness over throughput
- do not allow unbounded queue growth

This keeps the shim predictable and prevents a client burst from turning into a memory or latency problem.

## Timeouts

Recommended defaults:

- HTTP read timeout: 15s
- child response timeout: 30s for read-only calls, 60s for mutations and build calls
- child startup handshake timeout: 20s
- child shutdown grace period: 5s

Timeout behavior:

- timeout at the HTTP layer returns `504`
- timeout at the child layer also triggers a child health mark
- repeated timeouts should cause a child restart, not a silent retry loop

## Logging

Logs must be redacted and low-noise.

### Allow

- request method
- request id type, not raw payload
- response status
- elapsed time
- child generation number
- bytes in and out
- restart reason

### Redact

- bearer tokens
- absolute paths from user input
- file contents
- request params
- raw JSON-RPC payloads

The shim should reuse the repository redaction style where possible.

## Operational Shape

The shim should be a separate service unit, separate from the existing Python service.

- Python stays on port 8000
- the shim uses a different staging port
- the shim does not reuse the Python service name
- rollback is stop-the-shim only

## Alternatives

### 1. Minimal Python shim

What it is:

- a tiny Python HTTP wrapper that proxies `/mcp` to the stdio child

Why not choose it now:

- it adds a second Python runtime beside the live Python service
- it creates ambiguity around which Python stack is authoritative
- it is less aligned with the Go RC work and the existing Go deployment path
- it does not reduce the long-term desire to validate the Go binary in its own operational shape

### 2. Minimal Go shim

What it is:

- a dedicated Go HTTP shim that owns the stdio child

Why choose it:

- best match for the repository
- best fit for strict process control and redaction
- easiest to package as a small systemd service
- least surprising operationally for a Go backend

### 3. Native HTTP in `hugo-mcp-go`

What it is:

- add a loopback HTTP listener directly to the Go backend

Why not choose it now:

- it changes the backend binary instead of isolating the transport problem
- it expands the scope from adapter validation into product behavior
- it is the right long-term cleanup after the shim proves the contract, not the first validation step

### 4. Modify `mcp-runtime-go`

What it is:

- change the NUC gateway to spawn or talk stdio to the backend directly

Why not choose it now:

- it violates the current host split
- it moves validation risk onto the gateway path
- it requires NUC-side changes that are explicitly out of scope

## Final Verdict

- shim recommended: yes
- recommended language: Go
- recommended staging port: `18180`
- ready to implement now: yes for documentation and staging-only implementation planning, no for cutover
- code changes required: yes, but only for the future shim and not for the live Python service or the NUC gateway
- gateway attachable after shim validation: yes
- cutover possible now: no

## Suggested Future Files

- `cmd/hugo-mcp-shim/main.go`
- `internal/shim/server.go`
- `internal/shim/jsonrpc.go`
- `internal/shim/child.go`
- `deploy/systemd/hugo-mcp-shim.service`
- `deploy/systemd/hugo-mcp-shim.env.example`
