# Gateway Attachment Analysis

## Scope

Read-only analysis of how the NUC gateway can be attached to `hugo-mcp-go` later, without changing any live service now.

## Observed Current State

### NUC

- `mcp-runtime.service` is active.
- `mcp-runtime.service` listens on `127.0.0.1:8086`.
- `mcp-runtime.service` is configured with:
  - `HUGO_MCP_URL=https://192.168.122.69:8000/mcp`
  - `PROXY_BASE_URL=https://mcp-hugo.arleo.eu`
  - `HUGO_HOST=192.168.122.69`
  - `MCP_CA_CERT=/etc/hugo-mcp/vm-ca.crt`
  - OAuth/token-related values in `/etc/mcp-runtime-go/mcp-runtime.env` are present but redacted here.
- NUC OpenResty is active and proxies `mcp-hugo.arleo.eu` to `127.0.0.1:8086`.
- The gateway path today is HTTP(S) from NUC to the VM backend.

### VM

- `hugo-mcp.service` is active and remains the live backend.
- `hugo-mcp.service` runs `uvicorn main:app --host 0.0.0.0 --port 8000 --ssl-keyfile ... --ssl-certfile ...`.
- `hugo-mcp-go.service` is installed but inactive and disabled.
- `hugo-mcp-go.service` is stdio-based:
  - `ExecStart=/srv/hugo-mcp-go/bin/hugo-mcp-go`
  - it does not expose HTTP itself.
- The Go RC layout exists on the VM and preflight passes.

## Transport Comparison

### A. Current Python HTTP/VM model

**Shape**

- Gateway on NUC calls VM HTTPS endpoint `/mcp`.
- Python service on VM serves HTTP(S) directly.

**Pros**

- already working today
- already attached to the gateway contract
- no gateway code changes required

**Cons**

- legacy Python backend remains in the path
- not the Go RC
- separate process and transport contract from the new repo

**Security**

- TLS is already in the path
- attack surface is larger than stdio because it is networked
- the gateway and VM both expose HTTP surfaces that must be hardened

**Observability**

- good enough today
- HTTP and journald logs exist

**Maintenance**

- operationally stable, but it preserves the legacy implementation

### B. Go RC stdio model

**Shape**

- `hugo-mcp-go` runs as a stdio process.

**Pros**

- smallest backend transport surface
- simplest executable shape
- already implemented in the RC

**Cons**

- not directly attachable to the current NUC gateway contract
- no HTTP listener for the existing gateway path
- cannot replace the live Python backend without an extra bridge

**Security**

- smallest surface once attached locally
- but unusable by the current gateway as-is

**Observability**

- process-level only unless wrapped

### C. Go HTTP loopback model

**Shape**

- `hugo-mcp-go` exposes local HTTP, typically on `127.0.0.1`.

**Pros**

- aligns cleanly with the current NUC gateway model
- can keep the gateway unchanged
- enables health/readiness and HTTP logging
- good long-term steady-state if the team wants direct attachment

**Cons**

- requires a code change in `hugo-mcp-go`
- adds an HTTP server transport layer
- increases implementation scope relative to the current stdio RC

**Security**

- larger surface than stdio
- but can be constrained to loopback and proxied

**Maintenance**

- cleaner than an external shim in the long term

### D. Gateway-managed subprocess

**Shape**

- `mcp-runtime-go` would own the backend child process.

**Pros**

- one supervisor owns the lifecycle
- no extra local adapter process

**Cons**

- `mcp-runtime-go` is on the NUC, while the Hugo site and RC currently live on `hugo-vm`
- current gateway contract is HTTP(S) to a remote VM backend, not local stdio child management
- would require changing the gateway design or moving the backend host

**Security**

- potentially compact if local, but not compatible with the current host topology

**Conclusion**

- not a direct fit for the current infrastructure

### E. Local adapter/shim on the VM

**Shape**

- a small VM-local service exposes the existing HTTP `/mcp` contract
- it forwards to the Go stdio backend or manages the stdio child locally
- the NUC gateway remains unchanged

**Pros**

- preserves the NUC gateway contract
- keeps Python live until the Go path is proven
- avoids moving `mcp-runtime-go`
- can be introduced independently of the live Python service

**Cons**

- adds another process on the VM
- adds one more failure mode
- requires an adapter implementation or a small bridge service

**Security**

- still networked, but only locally or within the private VM path
- can be constrained to loopback or the existing private address

**Maintenance**

- more moving parts than native HTTP in `hugo-mcp-go`
- less invasive than changing the NUC gateway

## Recommendation

### Direct answer

- Can `mcp-runtime-go` attach directly to the current Go RC? **No.**
- Why not? Because the current Go RC is stdio-only, while the gateway contract today is HTTP(S) to the VM backend, and the gateway is on a different host.
- Should `hugo-mcp-go` gain an HTTP mode? **Yes, if the goal is to remove the bridge and make direct attachment simple.**
- Should Go run on the NUC or the VM? **On the VM.**
- Is a wrapper/adapter needed before any attachment? **Yes, unless native HTTP is added to `hugo-mcp-go` first.**

### Lowest-risk path

For the current infrastructure, the lowest-risk path is:

1. keep the live Python service untouched
2. keep `mcp-runtime-go` untouched
3. keep `hugo-mcp-go` on the VM
4. introduce a VM-local adapter/shim or add native HTTP to `hugo-mcp-go`
5. validate in staging only
6. attach the gateway only after parity and rollback prove out

### Practical preference

- **Short-term attachment path:** VM-local adapter/shim
- **Long-term cleaner endpoint:** native HTTP mode in `hugo-mcp-go`

The shim is the lower-risk way to prove the Go backend without disturbing the current gateway or live Python service.

