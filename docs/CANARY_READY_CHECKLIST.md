# Canary Ready Checklist
## Before
- [ ] `mcp-runtime.service` is active.
- [ ] Python `hugo-mcp.service` is still active.
- [ ] `hugo-mcp-shim.service` is installed but stopped.
- [ ] `HUGO_MCP_URL` still points to Python.
- [ ] The shim backend token is aligned to the runtime token fingerprint.
- [ ] The VM firewall allow rule for `192.168.122.1 -> 192.168.122.69:18180/tcp` is absent.
- [ ] You have the exact `initialize` payload validated in the previous session.
- [ ] You will use the runtime bearer token, not the shim-specific token, for the direct NUC -> shim test.
## During
- [ ] Start `hugo-mcp-shim.service`.
- [ ] Add the temporary firewall rule for source `192.168.122.1`.
- [ ] Run direct NUC -> shim `initialize` and `tools/list` with the runtime token.
- [ ] If the exact `initialize` payload is missing, stop and recover it before proceeding.
- [ ] Change only `HUGO_MCP_URL` to `http://192.168.122.69:18180/mcp`.
- [ ] Reload and restart `mcp-runtime.service`.
- [ ] Send only minimal canary traffic.
- [ ] Do not continue if the shim still rejects the runtime token.
## Rollback
- [ ] Restore `/etc/mcp-runtime-go/mcp-runtime.env` from backup.
- [ ] Set `HUGO_MCP_URL` back to `https://192.168.122.69:8000/mcp`.
- [ ] Reload and restart `mcp-runtime.service`.
- [ ] Remove the temporary `18180/tcp` rule for source `192.168.122.1`.
- [ ] Stop `hugo-mcp-shim.service`.
- [ ] Restore the previous shim env fingerprint if it was changed.
## Confirm Rollback
- [ ] `mcp-runtime.service` is running.
- [ ] `HUGO_MCP_URL` points to Python again.
- [ ] Python `hugo-mcp.service` is still active.
- [ ] `hugo-mcp-shim.service` is stopped.
## Go / No-Go
- [ ] Go only if direct shim tests pass, gateway tests pass, logs are redacted, and no writes escape `/var/lib/hugo-mcp-go`.
- [ ] No-go if auth, latency, child stability, rollback checks fail, or the shim token is still misaligned.
