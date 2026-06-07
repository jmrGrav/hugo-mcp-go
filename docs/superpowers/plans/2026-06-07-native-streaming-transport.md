# Native Streaming Transport Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a native HTTP transport to `hugo-mcp-go` that can replace `hugo-mcp-shim` on the VM while keeping `stdio` intact and adding optional backend-only streaming plus chunked large-result handling.

**Architecture:** Keep `cmd/hugo-mcp-go` as the single binary entrypoint and add explicit transport selection: `stdio` remains the default, while `http` becomes an opt-in mode that serves the same `POST /mcp` JSON-RPC contract currently provided by the shim. Large pages/assets/results are handled at the tool layer through pagination and chunking, and optional backend-only streaming is layered on top as a separate capability for direct tests and long-running operations, not as a dependency of `mcp-runtime-go`.

**Tech Stack:** Go 1.24, `net/http`, `log/slog`, `github.com/modelcontextprotocol/go-sdk/mcp`, systemd unit examples, Bash smoke scripts.

---

### Task 1: Add transport mode config and entrypoint dispatch

**Files:**
- Create: `internal/transport/config.go`
- Create: `internal/transport/config_test.go`
- Modify: `cmd/hugo-mcp-go/main.go`
- Modify: `internal/server/server.go`

- [ ] **Step 1: Write the failing tests**

Create tests that assert:

- `stdio` remains the default when `HUGO_MCP_TRANSPORT` is unset.
- `HUGO_MCP_TRANSPORT=http` requires `HUGO_MCP_HTTP_BIND_ADDR`, `HUGO_MCP_HTTP_BIND_PORT`, and a token source.
- token can be loaded from `HUGO_MCP_HTTP_TOKEN_FILE`.
- missing token in `http` mode fails closed with a clear error.
- bind defaults are loopback-safe when no explicit public bind is provided.

Use code like:

```go
func TestLoadFromEnv_DefaultsToStdio(t *testing.T) {
	t.Setenv("HUGO_MCP_TRANSPORT", "")
	cfg, err := LoadFromEnv()
	require.NoError(t, err)
	assert.Equal(t, "stdio", cfg.Transport)
}
```

- [ ] **Step 2: Run the tests and verify they fail**

Run:

```bash
go test ./internal/transport -run TestLoadFromEnv -v
```

Expected:

- package does not exist yet, or tests fail because transport config is missing

- [ ] **Step 3: Implement the minimal config and dispatch**

Implement:

- `internal/transport.Config` with transport mode, HTTP bind address/port, token/token-file, streaming flags, chunk and response limits.
- env parsing with fail-closed validation.
- `cmd/hugo-mcp-go/main.go` dispatch:
  - `stdio` calls `RunStdio(ctx)`
  - `http` constructs HTTP transport config and calls `RunHTTP(ctx, transportCfg)`
- keep `RunStdio(ctx)` unchanged.

- [ ] **Step 4: Run the tests and verify they pass**

Run:

```bash
go test ./internal/transport -run TestLoadFromEnv -v
go test ./... -run TestLoadFromEnv -v
```

Expected:

- transport config tests pass
- no other package regressions from the entrypoint dispatch

- [ ] **Step 5: Commit**

```bash
git add cmd/hugo-mcp-go/main.go internal/server/server.go internal/transport
git commit -m "feat: add native http transport config"
```

### Task 2: Implement native HTTP `POST /mcp` compatibility mode

**Files:**
- Create: `internal/server/http_transport.go`
- Create: `internal/server/http_transport_test.go`
- Modify: `internal/server/server.go`
- Modify: `internal/observability/observability.go` only if a small redaction helper is needed
- Reuse: `internal/shim/server.go`, `internal/shim/jsonrpc.go` as behavioral references only

- [ ] **Step 1: Write the failing tests**

Write HTTP tests that prove the native mode reproduces the shim contract:

- `GET /mcp` -> `405`
- non-POST methods -> `405`
- missing bearer -> `401`
- wrong bearer -> `401`
- malformed JSON -> `400`
- oversized body -> `413`
- `initialize` -> `200`
- `notifications/initialized` notification -> `202`
- `tools/list` -> `200`
- `tools/call` for `build_site` preserves `result.structuredContent.status == "built"`
- request logs do not leak bearer tokens or absolute paths

Use a small fake MCP server or fake tool bridge so the tests are deterministic.

- [ ] **Step 2: Run the tests and verify they fail**

Run:

```bash
go test ./internal/server -run TestHTTP -v
```

Expected:

- failures because the HTTP transport does not exist yet

- [ ] **Step 3: Implement the minimal HTTP adapter**

Implement `RunHTTP(ctx, cfg)` or equivalent in `internal/server` with:

- `POST /mcp`
- exact auth check: `Authorization: Bearer <token>`
- strict `Content-Type: application/json`
- request body size limit before parsing
- redacted request logging
- same JSON-RPC handling as the shim for:
  - `initialize`
  - `notifications/initialized`
  - `resources/list`
  - `prompts/list`
  - `tools/list`
  - `tools/call`
- same HTTP status mapping as the shim
- shutdown handling compatible with systemd

Prefer reusing the already-tested JSON-RPC helpers from `internal/shim` rather than duplicating edge cases.

- [ ] **Step 4: Run the tests and verify they pass**

Run:

```bash
go test ./internal/server -run TestHTTP -v
go test ./... -run TestHTTP -v
```

Expected:

- native HTTP transport tests pass
- `stdio` tests remain green

- [ ] **Step 5: Commit**

```bash
git add internal/server
git commit -m "feat: add native http mcp compatibility mode"
```

### Task 3: Add chunking and pagination for large pages and assets

**Files:**
- Modify: `internal/hugo/pages/pages.go`
- Modify: `internal/hugo/pages/pages_test.go`
- Modify: `internal/hugo/assets/assets.go`
- Modify: `internal/hugo/assets/assets_test.go`
- Modify: `internal/tools/tools.go`
- Modify: `internal/tools/tools_test.go`
- Create: `internal/tools/chunking.go`
- Create: `internal/tools/chunking_test.go`

- [ ] **Step 1: Write the failing tests**

Add tests proving:

- `list_pages` accepts pagination or cursor arguments and returns bounded slices.
- `list_assets` accepts pagination or cursor arguments and returns bounded slices.
- `get_page` on a file larger than the configured limit returns a structured continuation/error response instead of failing hard.
- oversized tool responses return a clear instruction to continue by chunk.
- chunk helpers preserve stable ordering and deterministic cursors.

Include at least one test with a deliberately large fixture or generated content.

- [ ] **Step 2: Run the tests and verify they fail**

Run:

```bash
go test ./internal/hugo/pages -run Test.*Chunk|Test.*Page -v
go test ./internal/hugo/assets -run Test.*Chunk|Test.*Asset -v
go test ./internal/tools -run Test.*Chunk|Test.*Pagination|Test.*Large -v
```

Expected:

- pagination/chunk tests fail because the helpers and arguments are not present yet

- [ ] **Step 3: Implement the minimal chunking model**

Implement:

- bounded page reads with explicit chunk metadata
- bounded asset reads with explicit chunk metadata
- pagination fields on list tools, using cursor or offset semantics that are stable and easy to smoke-test
- helper functions that:
  - serialize continuation state
  - cap response size
  - convert oversized results into clean continuation errors

Prefer small, explicit tool arguments rather than hidden behavior.

- [ ] **Step 4: Run the tests and verify they pass**

Run:

```bash
go test ./internal/hugo/pages -run Test.*Chunk|Test.*Page -v
go test ./internal/hugo/assets -run Test.*Chunk|Test.*Asset -v
go test ./internal/tools -run Test.*Chunk|Test.*Pagination|Test.*Large -v
go test ./... -run Test.*Chunk|Test.*Pagination|Test.*Large -v
```

Expected:

- chunking and pagination tests pass
- existing tool parity behavior remains intact for current fixtures

- [ ] **Step 5: Commit**

```bash
git add internal/hugo/pages internal/hugo/assets internal/tools
git commit -m "feat: add chunked large-result support"
```

### Task 4: Add optional backend-only streaming for long-running operations

**Files:**
- Create: `internal/server/streaming.go`
- Create: `internal/server/streaming_test.go`
- Modify: `internal/server/server.go`
- Modify: `internal/tools/tools.go`
- Modify: `internal/hooks/pipeline.go` or a focused helper if progress events need an emission hook
- Create if needed: `internal/streaming/*`

- [ ] **Step 1: Write the failing tests**

Write tests that prove:

- streaming is disabled by default
- enabling streaming does not change stdio behavior
- a direct backend streaming session can receive:
  - `started`
  - `progress`
  - `warning`
  - `completed`
  - `failed`
- progress events never include bearer tokens or raw secrets
- long-running tools can emit progress while still completing a normal JSON-RPC response

If the SDK’s streamable HTTP primitives are used, test them directly in-process.

- [ ] **Step 2: Run the tests and verify they fail**

Run:

```bash
go test ./internal/server -run Test.*Streaming -v
go test ./internal/tools -run Test.*Progress|Test.*Streaming -v
```

Expected:

- failures because streaming hooks and handlers are not wired yet

- [ ] **Step 3: Implement the minimal streaming path**

Implement optional backend-only streaming using the smallest viable mechanism:

- if the SDK’s streamable HTTP handler fits, use it behind a separate direct-backend route
- otherwise add a tiny SSE handler with explicit event framing
- gate the feature behind `HUGO_MCP_STREAMING_ENABLED`
- keep the public compatibility route untouched
- emit progress events from long-running operations only when streaming is enabled

Do not make streaming required for the production cutover.

- [ ] **Step 4: Run the tests and verify they pass**

Run:

```bash
go test ./internal/server -run Test.*Streaming -v
go test ./internal/tools -run Test.*Progress|Test.*Streaming -v
go test ./... -run Test.*Streaming -v
```

Expected:

- streaming tests pass
- stdio tests remain unchanged

- [ ] **Step 5: Commit**

```bash
git add internal/server internal/tools internal/hooks internal/streaming
git commit -m "feat: add optional native streaming support"
```

### Task 5: Add systemd examples, native smoke, and operational docs

**Files:**
- Create: `deploy/systemd/hugo-mcp-go-http.service`
- Create: `deploy/systemd/hugo-mcp-go-http.env.example`
- Create: `scripts/native-http-smoke.sh`
- Create: `docs/NATIVE_STREAMING_TRANSPORT_REPORT.md`
- Modify: `docs/FAST_PROD_CUTOVER_REPORT.md`
- Modify: `docs/README.md` only if a docs index needs the new design/report links

- [ ] **Step 1: Write the failing smoke expectations**

Create a native HTTP smoke that checks:

- `initialize`
- `tools/list`
- `list_pages`
- `get_page`
- `build_site` returns `structuredContent.status == "built"`
- `check_sri_versions` in safe/dry-run mode
- optional chunked fetch if the new chunk tool exists

The script should fail clearly if the backend token is missing or if the direct HTTP backend does not return the expected shape.

- [ ] **Step 2: Run the smoke and verify it fails**

Run:

```bash
bash scripts/native-http-smoke.sh
```

Expected:

- failure until the native HTTP service is actually deployed

- [ ] **Step 3: Add the service example and docs**

Add the unit example with:

- `HUGO_MCP_TRANSPORT=http`
- `HUGO_MCP_HTTP_BIND_ADDR=192.168.122.69`
- `HUGO_MCP_HTTP_BIND_PORT=18181`
- token from file
- PATH pinned to the real Hugo binary
- the same sandbox posture as the existing example service

Update the reports with:

- transport design summary
- differences versus the shim
- security decisions
- smoke commands
- rollback posture

- [ ] **Step 4: Run the smoke and doc checks**

Run:

```bash
bash scripts/native-http-smoke.sh
go test ./... -run Test -v
```

Expected:

- smoke passes against the staged native backend
- docs contain the current source of truth

- [ ] **Step 5: Commit**

```bash
git add deploy/systemd scripts docs
git commit -m "docs: add native streaming transport rollout material"
```

### Task 6: VM staging deployment and production cutover

**Files:**
- No new code files unless staging packaging needs a small helper script
- Modify: `/etc/mcp-runtime-go/mcp-runtime.env` only on the VM during deployment
- Possibly modify: `docs/FAST_PROD_CUTOVER_REPORT.md`

- [ ] **Step 1: Validate the native HTTP service on the VM**

Install the native HTTP binary on `hugo-vm`, start it on `192.168.122.69:18181`, and verify:

- the service is active
- `POST /mcp` responds correctly
- the direct smoke passes
- `hugo-mcp-shim.service` remains installed and untouched
- Python rollback remains installed

- [ ] **Step 2: Run the direct backend smoke**

From the host:

```bash
MCP_URL=http://192.168.122.69:18181/mcp MCP_TOKEN=<redacted-source> scripts/native-http-smoke.sh
```

Expected:

- direct backend parity passes
- chunked large-result cases are usable without SSH

- [ ] **Step 3: Cut over the runtime URL**

Back up `/etc/mcp-runtime-go/mcp-runtime.env`, change only `HUGO_MCP_URL` to the new native HTTP port, restart `mcp-runtime.service`, and verify:

- the public path still works
- Claude tools refresh successfully
- the native Go backend receives the traffic
- the shim stops receiving normal calls

- [ ] **Step 4: Produce the final reports**

Update:

- `docs/NATIVE_STREAMING_TRANSPORT_REPORT.md`
- `docs/FAST_PROD_CUTOVER_REPORT.md`

Include:

- native HTTP URL
- old shim URL
- streaming state
- chunking state
- smoke results
- rollback target
- verdict

- [ ] **Step 5: Commit**

```bash
git add docs deploy/systemd scripts
git commit -m "docs: record native http transport cutover"
```

## Coverage Checklist

- HTTP native transport: Task 1, Task 2
- `POST /mcp` compatibility: Task 2
- chunking/pagination for large results: Task 3
- optional backend-only streaming: Task 4
- stdio preserved: Task 1, Task 2, Task 6
- systemd packaging examples: Task 5
- smoke and report updates: Task 5, Task 6

## Notes for the Implementer

- Keep `stdio` as the default transport.
- Do not change `mcp-runtime-go` in this mission.
- Do not make public SSE a requirement for cutover.
- Prefer reusing existing shim behavior and existing MCP SDK primitives.
- If a requirement needs more than one file touched, keep the change grouped tightly and commit after each task.
