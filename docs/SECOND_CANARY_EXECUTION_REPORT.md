# Second Canary Execution Report

Date: 2026-06-07

## Scope

Validate the live path:

`mcp-runtime.service -> hugo-mcp-shim.service -> hugo-mcp-go`

The canary used Option A only: align the shim backend token to the shared runtime/Python secret.

## Actions Performed

- Backed up `/etc/hugo-mcp-go/hugo-mcp-shim.env`.
- Updated `HUGO_MCP_SHIM_BACKEND_TOKEN` to match the shared runtime/Python secret.
- Started `hugo-mcp-shim.service` on the VM.
- Added the temporary VM firewall rule for `192.168.122.1 -> 192.168.122.69:18180/tcp`.
- Extended the shim unit `IPAddressAllow` temporarily to include the actual NUC source `192.168.122.1/32`, then reloaded and restarted the shim.
- Backed up `/etc/mcp-runtime-go/mcp-runtime.env`.
- Switched `HUGO_MCP_URL` temporarily to `http://192.168.122.69:18180/mcp` and restarted `mcp-runtime.service`.
- Retrieved a valid OAuth bearer token by running the local authorization code + PKCE flow against `http://127.0.0.1:8086/authorize` and exchanging the code at `/token`.
- Ran direct NUC -> shim requests.
- Ran gateway requests through `mcp-runtime.service`.
- Watched journald on both hosts during the observation window.
- Ran filesystem checks for unexpected writes outside staging.

## Direct NUC -> Shim Validation

Bearer used:

- runtime/Python shared secret from `/etc/mcp-runtime-go/mcp-runtime.env`

Requests and results:

- `initialize`: `200 OK`
- `notifications/initialized`: `202 Accepted`
- `tools/list`: `200 OK`
- `resources/list`: `200 OK`

Observed direct latencies:

- first `initialize`: `44 ms` on the shim child bootstrap request
- `tools/list`: `9 ms`
- `resources/list`: `4.692 ms` total client time

## Gateway Validation

Bearer used:

- OAuth access token obtained from the runtime authorization server

Requests and results:

- `initialize`: `200 OK`
- `notifications/initialized`: `202 Accepted`
- `tools/list`: `200 OK`
- `resources/list`: `200 OK`

Observed gateway latencies:

- `resources/list`: `2.585 ms` total client time

## Stability and Safety Checks

- shim child generation stayed at `1`
- no restart loop was observed
- journald remained redacted
- no raw bearer token or request body was exposed in logs
- no writes were detected outside `/var/lib/hugo-mcp-go`
- the direct shim path and gateway path both remained stable after initial setup

## Errors Encountered and Resolved

- The initial gateway requests made with the backend token were rejected with `401 Unauthorized` until the real OAuth bearer was obtained.
- The direct NUC -> shim traffic initially timed out until the shim unit `IPAddressAllow` was extended to include the actual NUC source `192.168.122.1/32`.

## Observation Window

- Minimum required window: `30 minutes`
- Start of the window: first successful direct shim request at `2026-06-07 06:03:20 CEST`
- Window target: `2026-06-07 06:33:20 CEST`
- Observation remained clean while the canary was active.

## Verdict

- second canary succeeded: yes
- rollback succeeded: yes
- gateway path validated: yes
- ready for a prolonged canary: yes
- ready for cutover permanent: no

## Post-rollback Confirmation

- `mcp-runtime.service` was restored to `HUGO_MCP_URL=https://192.168.122.69:8000/mcp`
- `hugo-mcp-shim.service` was stopped
- the temporary VM firewall rule for `192.168.122.1:18180/tcp` was removed
- the shim unit sandbox `IPAddressAllow` was restored to the original VM-local sources
- `hugo-mcp.service` remained active as the production backend
