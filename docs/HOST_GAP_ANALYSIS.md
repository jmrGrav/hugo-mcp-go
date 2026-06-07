# Host Gap Analysis

## Scope

This review compares the release-candidate deployment contracts against the real Arleo infrastructure that exists today.

Observed environment:

- NUC host: OpenResty, CrowdSec, and `mcp-runtime.service`
- Hugo site VM: `hugo-vm`
- Existing Hugo MCP service on the VM: Python `hugo-mcp.service`
- Existing Hugo site tree on the VM: `/home/jm/hugo-site`

This document does not change production. It only records the gaps.

## Executive Summary

The infrastructure is now **ready for a staging-host installation**, but **not ready** for a clean RC cutover.

Main remaining blockers:

1. The gateway/backend topology is still split across hosts: `mcp-runtime` is on the NUC, while the real Hugo content tree and existing MCP service are on `hugo-vm`.
2. The current live Hugo MCP service on the VM is still Python/HTTP, not the Go RC transport model.
3. The RC service has not been started yet, so the host is staged but not live on the Go path.
4. The gateway/backend attachment for the Go RC is not installed yet.

## Inventory

### NUC host

Confirmed:

- `openresty.service` is active.
- `mcp-runtime.service` is active.
- `mcp-runtime.service` listens on `127.0.0.1:8086`.
- `mcp-runtime.service` runs as `mcp-runtime:mcp-runtime`.
- `mcp-runtime.service` uses a hardened systemd profile.

### Hugo VM

Confirmed:

- `hugo-mcp.service` is active.
- `nginx.service` is active.
- `hugo` exists at `/usr/local/bin/hugo`.
- `hugo version` reports `v0.147.0-7d0039b86ddd6397816cc3383cb0cfa481b15f32+extended`.
- `hugo-mcp` user and group exist.
- The site tree is rooted at `/home/jm/hugo-site`.
- `content/`, `static/`, `public/`, and `themes/` exist.
- A staging tree now exists at `/var/lib/hugo-mcp-go`.
- `work/` exists under the RC layout.
- No symlinks were found under `/home/jm/hugo-site`.

## Conformance Matrix

### Conformant

- Hugo binary exists on the VM at the expected canonical location.
- The Hugo site tree contains the expected top-level content/static/public structure.
- The site tree contains no symlink escapes.
- The VM has a dedicated `hugo-mcp` user and group available.
- The NUC already has the gateway service and reverse proxy in place for MCP traffic.

### Partially conformant

- The live VM service has journald/systemd supervision and restart policy, but it is still the Python service, not the Go RC.
- The site tree is writable and functional, but it is located in `/home/jm/hugo-site`, not in the RC package layout.
- The VM has a dedicated `hugo-mcp` user, and the RC staging tree is now owned by that identity, but the current live service still runs as `jm`.
- The Hugo binary contract is satisfied at the path level, and the RC preflight now passes against the staging layout, but the Go RC service has not been started yet.

### Non-conformant

- The current VM service transport is still Python HTTP/TLS on port `8000`, not the Go RC transport shape.
- The NUC gateway and the VM site service are split across hosts, and the RC attachment contract is not yet installed.
- The Go RC service is installed but not started, so there is no runtime proof of the backend attachment contract yet.
- The gateway attachment for the Go RC is not yet installed.

## Comparison to Repo Contracts

### `docs/TRANSPORT_CONTRACT.md`

- **Partially conformant**
- The repo recommends stdio subprocess behind `mcp-runtime-go`, but the real environment today still has a Python HTTP backend on the VM.
- The gateway is present on the NUC, but the backend attachment shape needed for the Go RC is not yet in place.

### `docs/HUGO_BINARY_CONTRACT.md`

- **Conformant**
- `/usr/local/bin/hugo` exists on the VM.
- The staging preflight now validates the binary path and version prefix.

### `docs/OPERATIONS_READINESS.md`

- **Conformant**
- Journald/systemd supervision exists.
- Stop/upgrade/rollback procedures are documented.
- The RC package layout and rollback breadcrumb directory now exist on the host.

### `docs/PRODUCTION_DEPLOYMENT_PLAN.md`

- **Partially conformant**
- The plan expects a dedicated `hugo-mcp` service, explicit package directories, and a controlled deployment tree.
- The RC staging layout now exists and the service identity is present, but the legacy Python service still remains live under `jm`.

### `docs/PRODUCTION_CUTOVER_CHECKLIST.md`

- **Partially conformant**
- The site-clone validation target is now materialized under the RC package layout.
- The transport contract is still not frozen in the real host attachment model.

### `docs/ROLLBACK_RUNBOOK.md`

- **Conformant**
- The host can stop and restart services.
- The RC-specific rollback artifacts and package versioning contract are now materialized in the staging layout.

## Final Gap Summary

| Area | Status | Why |
|---|---|---|
| Hugo binary | conformant | `/usr/local/bin/hugo` exists on the VM |
| Site tree | conformant | RC staging layout now exists and is owned by `hugo-mcp` |
| Service identity | partially conformant | RC identity exists, but the live Python service still runs as `jm` |
| Transport | non-conformant | current live backend is Python/HTTP, not the Go RC shape |
| Gateway attachment | non-conformant | NUC gateway and VM backend are not yet wired for the RC |
| Rollback layout | conformant | `work/` and RC package roots are present |
| Deployment package | conformant | host package now exists under `/srv`, `/etc`, and `/var/lib` |

## Evidence Notes

- NUC:
  - `mcp-runtime.service` active
  - `openresty.service` active
- VM:
  - `hugo-mcp.service` active
  - `nginx.service` active
  - `hugo` at `/usr/local/bin/hugo`
  - `hugo-site` rooted at `/home/jm/hugo-site`
  - RC staging layout rooted at `/var/lib/hugo-mcp-go`
  - `work/` present
  - no symlinks under the site tree

## Decision

- Ready to install the RC on the host: **yes**
- Ready to start the RC in staging on the host: **yes**
- Ready to cut over: **no**
