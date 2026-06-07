# Canary Execution Report

Date: 2026-06-06 to 2026-06-07

## Scope

Real gateway canary attempt for the path:

`mcp-runtime.service -> shim -> hugo-mcp-go`

No cutover was intended or left in place.

## What Was Executed

- Temporary shim service start on the VM.
- Temporary VM firewall exposure for `18180/tcp`.
- Temporary NUC gateway switch to `HUGO_MCP_URL=http://192.168.122.69:18180/mcp`.
- OAuth authorization code flow against the local gateway runtime to obtain a valid bearer token.
- Gateway requests attempted:
  - `initialize`
  - `notifications/initialized`
  - `tools/list`
  - one read-only `tools/call` for `list_pages`

## Result

- `initialize`: no
- `notifications/initialized`: no
- `tools/list`: no
- read-only extra call: no

All gateway requests were rejected with `401 Unauthorized`.

## Observed Latency

- Rejected gateway requests returned in roughly 20-30 ms.
- No successful MCP round-trip latency could be measured through the gateway path.

## Errors Observed

- Gateway path rejected the MCP calls with `401 Unauthorized`.
- The local shim/backend chain did not accept the backend authorization used by the NUC runtime during the canary.
- The blocker is a backend-token mismatch between the NUC runtime injection and the shim expectation.

## Child Process Stability

- No child restart was observed from the canary traffic.
- Direct shim validation had already shown the child bridge was stable earlier, but the real gateway canary never reached the Go backend because the request chain failed before that point.

## Duration

- Actual gateway canary attempt: approximately 2-3 minutes.
- Minimum 30-minute observation window was not reached because the gateway path was blocked.

## Verdict

- Canary succeeded: no
- Ready for a longer second canary: no
- Ready for permanent cutover: no

## Notes

- The runtime bearer token was obtained correctly.
- The failure was not at OAuth token acquisition time.
- The failure occurred after the runtime forwarded the request toward the shim/backend path.

