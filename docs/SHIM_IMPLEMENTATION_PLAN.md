# VM-Local MCP Shim Implementation Plan

> **For agentic workers:** implement this plan task-by-task only after the design and security docs have been reviewed. Keep the live Python service untouched and do not modify the NUC gateway.

**Goal:** build a VM-local HTTP shim that exposes `/mcp` and forwards to the stdio `hugo-mcp-go` child process for staging validation.

**Architecture:** a small Go daemon listens on a private VM address, authenticates requests with a staging bearer token, translates HTTP `/mcp` into JSON-RPC over a persistent stdio child, and restarts the child on failure with bounded backoff.

**Tech Stack:** Go, `net/http`, JSON parsing, `os/exec`, `systemd`, journal logging

---

## Task 1: Define the shim package layout

**Files:**
- Create: `cmd/hugo-mcp-shim/main.go`
- Create: `internal/shim/server.go`
- Create: `internal/shim/child.go`
- Create: `internal/shim/jsonrpc.go`
- Create: `internal/shim/config.go`
- Create: `internal/shim/server_test.go`

- [ ] **Step 1: Write the failing tests**

Cover:

- config load failure when bind address or token is missing
- child spawn failure when the Go binary path is invalid
- HTTP reject on missing bearer token

- [ ] **Step 2: Run the tests to confirm they fail**

Run:

```bash
go test ./internal/shim -run Test -v
```

Expected:

- package or symbols missing

- [ ] **Step 3: Add minimal package skeleton**

Define the server, config, and child process types with no protocol logic yet.

- [ ] **Step 4: Run the tests again**

Run:

```bash
go test ./internal/shim -run Test -v
```

Expected:

- the skeleton tests compile and fail only on the missing logic they are meant to cover

## Task 2: Implement config and startup validation

**Files:**
- Modify: `internal/shim/config.go`
- Modify: `cmd/hugo-mcp-shim/main.go`
- Modify: `internal/shim/server_test.go`

- [ ] **Step 1: Write tests for config validation**

Cover:

- missing bind address
- missing port
- missing token
- missing child binary path
- request timeout and startup timeout defaults

- [ ] **Step 2: Run the tests to confirm they fail**

Run:

```bash
go test ./internal/shim -run TestConfig -v
```

- [ ] **Step 3: Implement config parsing and validation**

Use environment variables only. Fail closed on missing required values.

- [ ] **Step 4: Run the tests and confirm pass**

Run:

```bash
go test ./internal/shim -run TestConfig -v
```

## Task 3: Add child process management

**Files:**
- Modify: `internal/shim/child.go`
- Modify: `internal/shim/server_test.go`

- [ ] **Step 1: Write tests for child lifecycle**

Cover:

- spawn success
- spawn failure
- restart after exit
- initialization handshake timeout

- [ ] **Step 2: Run the tests to confirm they fail**

Run:

```bash
go test ./internal/shim -run TestChild -v
```

- [ ] **Step 3: Implement child start, stop, and restart loop**

Use `exec.CommandContext` and stdio pipes. Keep the child generation number in memory and never expose it over HTTP.

- [ ] **Step 4: Run the tests and confirm pass**

Run:

```bash
go test ./internal/shim -run TestChild -v
```

## Task 4: Implement JSON-RPC translation

**Files:**
- Modify: `internal/shim/jsonrpc.go`
- Modify: `internal/shim/server.go`
- Modify: `internal/shim/server_test.go`

- [ ] **Step 1: Write tests for JSON-RPC mapping**

Cover:

- valid request with numeric id
- valid request with string id
- notification without id
- malformed JSON body
- child error object passthrough
- timeout mapping to HTTP 504

- [ ] **Step 2: Run the tests to confirm they fail**

Run:

```bash
go test ./internal/shim -run TestJSONRPC -v
```

- [ ] **Step 3: Implement the request router and response matcher**

Preserve client ids, serialize writes to the child, and map child responses back to the correct HTTP request.

- [ ] **Step 4: Run the tests and confirm pass**

Run:

```bash
go test ./internal/shim -run TestJSONRPC -v
```

## Task 5: Add HTTP server, auth, and size limits

**Files:**
- Modify: `internal/shim/server.go`
- Modify: `internal/shim/server_test.go`
- Create: `deploy/systemd/hugo-mcp-shim.env.example`
- Create: `deploy/systemd/hugo-mcp-shim.service`

- [ ] **Step 1: Write tests for HTTP access control**

Cover:

- missing token -> 401 or 403
- oversized body -> 413
- correct token -> request accepted
- only `/mcp` is exposed

- [ ] **Step 2: Run the tests to confirm they fail**

Run:

```bash
go test ./internal/shim -run TestHTTP -v
```

- [ ] **Step 3: Implement the HTTP listener**

Bind only to the configured VM address and port. Reject bodies over the configured limit before parsing.

- [ ] **Step 4: Run the tests and confirm pass**

Run:

```bash
go test ./internal/shim -run TestHTTP -v
```

## Task 6: Add logging and redaction

**Files:**
- Modify: `internal/shim/server.go`
- Modify: `internal/shim/child.go`
- Modify: `internal/shim/server_test.go`

- [ ] **Step 1: Write tests for redacted logs**

Cover:

- bearer tokens are not present
- raw JSON bodies are not present
- absolute paths in errors are redacted

- [ ] **Step 2: Run the tests to confirm they fail**

Run:

```bash
go test ./internal/shim -run TestLog -v
```

- [ ] **Step 3: Implement structured redacted logging**

Log status, latency, child generation, and restart reason only.

- [ ] **Step 4: Run the tests and confirm pass**

Run:

```bash
go test ./internal/shim -run TestLog -v
```

## Task 7: Add unit files and operator docs

**Files:**
- Create: `deploy/systemd/hugo-mcp-shim.service`
- Create: `deploy/systemd/hugo-mcp-shim.env.example`
- Modify: `docs/SHIM_DESIGN.md`
- Modify: `docs/SHIM_SECURITY_MODEL.md`
- Modify: `docs/SHIM_VALIDATION_PLAN.md`

- [ ] **Step 1: Verify the unit file references the shim only**

Use a dedicated user, dedicated working directory, and strict sandboxing.

- [ ] **Step 2: Verify the environment example contains no secrets**

Use explicit sample values such as `127.0.0.1`, `18180`, and `REDACTED`, and do not place any secret material in the file.

- [ ] **Step 3: Confirm the docs match the unit defaults**

The bind address, port, and token names must be consistent across docs and the example unit.

## Task 8: End-to-end staging verification

**Files:**
- No new code files

- [ ] **Step 1: Build the shim binary**

Run:

```bash
go build ./cmd/hugo-mcp-shim
```

- [ ] **Step 2: Start the shim in staging only**

Use a non-routed staging port and keep Python live.

- [ ] **Step 3: Run direct HTTP checks**

Validate read-only requests, mutation requests on staging only, and log redaction.

- [ ] **Step 4: Stop the shim**

Confirm rollback is stop-only and that Python remains active.

## Self-Review Checklist

- every required doc file exists
- no task proposes a cutover
- no task modifies the NUC
- no task modifies the live Python service
- no task touches `/home/jm/hugo-site`
- the shim security model stays separate from production
- the implementation plan uses explicit file paths and test commands
