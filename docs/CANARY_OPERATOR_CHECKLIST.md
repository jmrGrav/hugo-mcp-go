# Canary Operator Checklist

## Before

- [ ] Confirm `mcp-runtime.service` is active on the NUC.
- [ ] Confirm Python `hugo-mcp.service` is still active on the VM.
- [ ] Confirm `hugo-mcp-shim.service` is installed but not yet enabled.
- [ ] Confirm the live gateway still points to `https://192.168.122.69:8000/mcp`.
- [ ] Confirm the shim bind target is `192.168.122.69:18180`.
- [ ] Confirm the shim backend token is not yet aligned with the runtime token.
- [ ] Confirm the staging tree is `/var/lib/hugo-mcp-go`.
- [ ] Back up `/etc/mcp-runtime-go/mcp-runtime.env`.
- [ ] Back up `/home/jm/hugo-mcp/.env`.
- [ ] Back up `/etc/hugo-mcp-go/hugo-mcp-shim.env`.
- [ ] Confirm no OpenResty, Nginx, or Cloudflare changes are pending.
- [ ] Confirm the temporary VM firewall rule for `192.168.122.1 -> 192.168.122.69:18180/tcp` is not already present.
- [ ] Confirm the runtime token and Python fallback token match by fingerprint.

## During

- [ ] Start `hugo-mcp-shim.service` on the VM.
- [ ] Verify the shim journal shows a successful child handshake.
- [ ] Run direct NUC -> shim tests for `initialize` and `tools/list` with the runtime token as bearer.
- [ ] Confirm `401` is returned for missing or invalid token, not `200`.
- [ ] Confirm the shim logs are redacted.
- [ ] Add the temporary `18180/tcp` firewall allow rule on the VM for source `192.168.122.1`.
- [ ] Change only `HUGO_MCP_URL` in `/etc/mcp-runtime-go/mcp-runtime.env`.
- [ ] Run `systemctl daemon-reload`.
- [ ] Restart `mcp-runtime.service`.
- [ ] Verify the gateway still binds `127.0.0.1:8086`.
- [ ] Send a small amount of real gateway traffic.
- [ ] If the read-only calls are clean, send one staging-only mutation.
- [ ] Watch `journalctl` on both hosts while the canary is live.

## Rollback

- [ ] Restore `/etc/mcp-runtime-go/mcp-runtime.env` from the backup.
- [ ] Change `HUGO_MCP_URL` back to `https://192.168.122.69:8000/mcp`.
- [ ] Run `systemctl daemon-reload`.
- [ ] Restart `mcp-runtime.service`.
- [ ] Remove the temporary `18180/tcp` firewall allow rule.
- [ ] Stop `hugo-mcp-shim.service`.
- [ ] Confirm Python is still active and unchanged.
- [ ] Confirm the gateway path is back on Python before closing the window.
- [ ] Confirm the shim backend token rollback restored the previous fingerprint.

## After

- [ ] Save the canary start time, end time, and observed latency.
- [ ] Save the journal excerpts for the NUC, shim, and Python service.
- [ ] Record any accepted drift explicitly.
- [ ] Record whether the staging tree stayed isolated.
- [ ] Record whether rollback was immediate and clean.

## Go / No-Go

- [ ] Go for manual canary if:
  - [ ] direct shim calls succeed
  - [ ] direct shim calls succeed with the runtime token
  - [ ] gateway calls succeed after the temporary route switch
  - [ ] no redaction leak appears
  - [ ] no write escapes `/var/lib/hugo-mcp-go`
  - [ ] no child restart loop appears
- [ ] No-go for permanent cutover if:
  - [ ] Python remains the authoritative backend
  - [ ] no long soak has been completed
  - [ ] the gateway switch is still temporary only
  - [ ] rollback has not been proven in the live window
  - [ ] the shim backend token has not been aligned with the runtime token
