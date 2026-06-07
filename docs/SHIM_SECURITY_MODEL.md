# VM-Local Shim Security Model

## Scope

This document defines the security posture for the VM-local HTTP shim that fronts `hugo-mcp-go` stdio.

## Primary Security Goal

Expose the minimum network surface needed for staging validation while keeping the live Python service untouched.

## Recommended Bind Model

### Bind address

Bind only to the VM's private RFC1918 address on the staging port.

Recommended form:

- `192.168.122.69:18180` if the address is static in the VM staging setup
- if the private address can move, inject the exact bind address through systemd environment rather than binding to `0.0.0.0`

Do not bind the shim to a public interface.

### Port

Use a staging port that is clearly separate from the live Python service.

Recommended port:

- `18180`

This is intentionally different from Python `8000` and from the NUC gateway port.

## TLS Decision

Recommended for staging:

- no TLS on the shim listener

Why:

- the listener is private to the VM network
- the port is not production-routed
- the validation path is simpler without cert management

If a future test path needs to cross a less trusted network, add TLS later as a separate concern. Do not reuse production certificates for this staging shim.

## Authentication

Require a backend token on every request.

Recommended model:

- `Authorization: Bearer <staging-token>`
- the token is injected through an environment variable or a systemd drop-in
- the token is not committed to the repository
- the token is redacted in logs

This token is for staging validation and manual calls. It is not a substitute for gateway authentication in production.

## Allowed Clients

Limit access to:

- localhost on the VM
- the NUC host used for validation

Practical enforcement:

- bind only to the VM private address
- restrict traffic at the host firewall layer
- require the bearer token

Do not rely on the application alone as the only access control.

## Request Size Limits

Recommended hard limits:

- total HTTP request body: 1 MiB
- JSON-RPC tool arguments: keep the repo default limit unless a specific staging case needs less

If the request exceeds the limit, reject it before the child process sees it.

## Rate Limiting

For the initial shim, a separate rate limiter is not required.

Reason:

- access is already constrained by bind address and token
- the child stdio session is the serialization point
- the bounded queue and timeouts already prevent unbounded load

If a later staging window needs extra protection, add a simple local token bucket at low rate, but do not make it part of the first design.

## Systemd Isolation

The shim should run under a dedicated service account with a strict unit sandbox.

Recommended isolation profile:

- `User=hugo-mcp-shim`
- `Group=hugo-mcp-shim`
- `NoNewPrivileges=yes`
- `ProtectSystem=strict`
- `ProtectHome=yes`
- `PrivateTmp=yes`
- `PrivateDevices=yes`
- `LockPersonality=yes`
- `UMask=0027`
- `CapabilityBoundingSet=`
- `AmbientCapabilities=`
- `ReadWritePaths` only for the staging tree, runtime state, and logs

The shim does not need elevated privileges.

## Filesystem Isolation

The shim must not touch:

- `/home/jm/hugo-site`
- the live Python service directories
- production configuration paths

The shim should have:

- read access only to the `hugo-mcp-go` binary and its staging roots
- write access only to its own runtime state and journal output

## Network Isolation

The shim does not need outbound network access.

Recommended posture:

- no external egress
- no DNS dependence
- only the listening socket required for inbound validation traffic

## Env Minimum

Recommended environment variables:

- `HUGO_MCP_SHIM_BIND_ADDR`
- `HUGO_MCP_SHIM_BIND_PORT`
- `HUGO_MCP_SHIM_BACKEND_TOKEN`
- `HUGO_MCP_GO_BIN`
- `HUGO_MCP_GO_WORKDIR`
- `HUGO_MCP_REQUEST_TIMEOUT_MS`
- `HUGO_MCP_STARTUP_TIMEOUT_MS`
- `HUGO_MCP_MAX_REQUEST_BYTES`
- `HUGO_MCP_LOG_LEVEL`

No secret should be embedded in unit files or docs.

## Logging Rules

Allowed:

- status
- timings
- request id type
- restart count
- child generation

Blocked:

- bearer tokens
- raw request bodies
- file contents
- absolute paths supplied by callers

## Failure Posture

The shim must fail closed.

- missing token -> reject requests
- missing child binary -> service does not become ready
- child crash loop -> service stays up but returns an explicit unhealthy status to callers
- request timeout -> reject the request and log the event without leaking payloads

## Separation From Python

The security model depends on strict non-overlap:

- Python stays on `8000`
- shim stays on `18180`
- no shared unit name
- no shared log file
- no shared bind address
- no shared rollback path

Rollback is stopping the shim only.

