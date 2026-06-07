# Canary Migration Plan: Python to hugo-mcp-go

## Goal

Validate `hugo-mcp-go` through the existing gateway path with a temporary, fully reversible canary. Keep the live Python backend active until the canary is explicitly approved or rolled back.

## Current State Snapshot

Observed on 2026-06-06 and updated after the first gateway attempt:

- `mcp-runtime.service` on the NUC is active and listens on `127.0.0.1:8086`.
- `mcp-runtime.service` currently points to `HUGO_MCP_URL=https://192.168.122.69:8000/mcp`.
- `hugo-mcp.service` on the VM is active and serves the live Python backend on port `8000`.
- `hugo-mcp-shim.service` is installed on the VM but inactive.
- The shim bind target is `192.168.122.69:18180`.
- The shim backend target is `hugo-mcp-go` in `/srv/hugo-mcp-go/bin/hugo-mcp-go`.
- VM firewall rules currently allow NUC -> Python on port `8000`; the canary needs a temporary allow rule for the real NUC egress source `192.168.122.1` -> shim on port `18180`.
- OpenResty, Nginx, and Cloudflare are out of scope and stay unchanged.

## Preconditions

Before any gateway canary:

1. Keep Python live.
2. Keep `mcp-runtime.service` unchanged until the temporary gateway switch step.
3. Do not enable the shim permanently.
4. Confirm the shim binary and env file still exist on the VM.
5. Confirm the staging tree is `/var/lib/hugo-mcp-go` and not the live Hugo tree.
6. Create a backup of `/etc/mcp-runtime-go/mcp-runtime.env` before changing any gateway routing.
7. Add a temporary VM firewall allow rule for `192.168.122.1 -> 192.168.122.69:18180/tcp`.
8. Do not proceed to the gateway switch until the shim backend token matches the runtime token used by `mcp-runtime.service`.

## Canary Sequence

### 1. Bring up the shim on the VM

Start `hugo-mcp-shim.service` on the VM without enabling it.

Expected result:

- the shim binds `192.168.122.69:18180`
- the child `hugo-mcp-go` process starts under the shim cgroup
- the shim reports healthy only after the MCP child handshake succeeds

### 2. Direct NUC -> shim validation

From the NUC, call the shim directly before touching the gateway.

Validate:

- `POST /mcp`
- `initialize`
- `tools/list`
- one deterministic read-only tool call, if the first two succeed
- use the runtime token from `/etc/mcp-runtime-go/mcp-runtime.env` as the bearer

Stop immediately if:

- the shim returns `401`, `502`, `503`, or `504` unexpectedly
- the child restarts repeatedly
- logs are not redacted
- the shim touches the live site tree

### 3. Temporary gateway switch

If direct NUC -> shim validation is clean, temporarily point `mcp-runtime.service` at the shim by changing only `HUGO_MCP_URL` in `/etc/mcp-runtime-go/mcp-runtime.env`.

Use the shim URL:

- `http://192.168.122.69:18180/mcp`

Then:

1. reload systemd
2. restart `mcp-runtime.service`
3. confirm the NUC gateway still listens on `127.0.0.1:8086`

### 4. Gateway canary traffic

Run the gateway canary through the existing front door, with Python still available as the rollback target.

Recommended traffic shape:

- a small number of read-only calls first
- one staging-only mutation only if the read-only calls stay clean
- no batch load
- no long idle gap before rollback readiness is confirmed

### 5. Rollback immediately if needed

Rollback is routing-only:

1. restore the saved `/etc/mcp-runtime-go/mcp-runtime.env`
2. set `HUGO_MCP_URL` back to `https://192.168.122.69:8000/mcp`
3. reload systemd
4. restart `mcp-runtime.service`
5. leave Python running
6. stop the shim only after the gateway has returned to Python

## Stop Criteria

Stop the canary and roll back immediately if any of the following occurs:

- auth errors from the gateway path that did not exist before the switch
- repeated `502`, `503`, or `504` responses
- child crash loop or restart storm
- log redaction failure
- any write outside `/var/lib/hugo-mcp-go`
- any mismatch in deterministic read-only responses that cannot be explained and accepted in advance
- latency that more than triples the direct shim baseline for the same request class
- any unexpected interaction with OpenResty, Nginx, or Cloudflare

## Minimum Observation Window

- Absolute minimum: 30 minutes after the first successful gateway request
- Recommended: 60 minutes if at least one staging mutation is exercised
- Reset the timer if the shim restarts or the gateway is rolled back once during the window

## Verdict

- Ready for manual canary now: no
- Ready for permanent cutover: no
- Exact blockers:
  - shim is installed but not started
  - the temporary `18180/tcp` VM ingress rule is not yet in place
  - `HUGO_MCP_URL` still points to Python
  - the shim backend token is not yet aligned with the runtime token
  - no gateway soak has been run yet
- Rollback guaranteed: yes, for the routing-only canary described here, provided the env backup is created before the first restart
