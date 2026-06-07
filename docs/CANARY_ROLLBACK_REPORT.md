# Canary Rollback Report

Date: 2026-06-07

## Rollback Objective

Restore the environment to the original Python-backed state and remove every temporary canary change.

## Actions Performed

- Restored `/etc/mcp-runtime-go/mcp-runtime.env` from the timestamped backup.
- Restarted `mcp-runtime.service`.
- Removed the temporary VM firewall allow rules for `18180/tcp`.
- Removed the temporary systemd drop-in used for shim source allowance.
- Stopped `hugo-mcp-shim.service`.
- Confirmed `hugo-mcp.service` remained active on the VM.

## Confirmation

- `HUGO_MCP_URL` is back to `https://192.168.122.69:8000/mcp`.
- `mcp-runtime.service` is active again after restart.
- Temporary `18180/tcp` firewall rules were deleted.
- Shim service is stopped.
- Python backend is still active.

## Rollback Quality

- Rollback executed cleanly: yes
- Python restored: yes
- Gateway left on Go: no
- Temporary canary firewall left behind: no
- Temporary shim source allowance left behind: no

## Residual State

- The canary did not complete successfully, so no production routing change remains.
- The unresolved blocker is the backend-token mismatch between the NUC runtime and the shim VM.

