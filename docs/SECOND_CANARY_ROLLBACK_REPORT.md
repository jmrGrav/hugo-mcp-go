# Second Canary Rollback Report

Date: 2026-06-07

## Scope

Restore the environment to the Python-backed production state after the second canary.

## Rollback Actions

- Restored `/etc/mcp-runtime-go/mcp-runtime.env` from backup.
- Restarted `mcp-runtime.service` on the NUC.
- Removed the temporary VM firewall rule for `192.168.122.1 -> 18180/tcp`.
- Stopped `hugo-mcp-shim.service` on the VM.
- Restored `/etc/hugo-mcp-go/hugo-mcp-shim.env` from backup.
- Restored `/etc/systemd/system/hugo-mcp-shim.service` from backup.
- Reloaded systemd on the VM.

## Verified Final State

- `HUGO_MCP_URL=https://192.168.122.69:8000/mcp`
- `mcp-runtime.service` is active
- `hugo-mcp.service` is active
- `hugo-mcp-shim.service` is inactive
- the temporary `18180/tcp` firewall rule is absent
- shim sandbox IP allowlist is back to the original VM-local sources

## Confirmation Checks

- NUC runtime env points back to Python.
- VM shim service is stopped.
- VM Python backend remains active.
- no temporary gateway routing remains in place.

## Result

- rollback succeeded: yes
- Python restored: yes
- canary routing left in place: no
- cutover performed: no
