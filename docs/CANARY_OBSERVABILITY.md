# Canary Observability Plan

## Scope

Monitor the routing canary from the NUC gateway to the VM shim and back to Python. The canary is only valid if the logs stay redacted, the child stays stable, and no write leaves the staging tree. The second canary is forbidden until the runtime token and shim backend token are aligned.

## Sources to Watch

### NUC

- `journalctl -fu mcp-runtime.service`
- `journalctl -u mcp-runtime.service -n 200 --no-pager`
- `systemctl status --no-pager mcp-runtime.service`

### VM

- `journalctl -fu hugo-mcp-shim.service`
- `journalctl -u hugo-mcp-shim.service -n 200 --no-pager`
- `journalctl -u hugo-mcp.service -n 200 --no-pager`
- `journalctl -u hugo-mcp-go.service -n 200 --no-pager` if the standalone backend unit is ever started during a later trial
- `systemctl status --no-pager hugo-mcp.service hugo-mcp-shim.service`
- `systemctl status --no-pager hugo-mcp-go.service` if the standalone backend unit is ever started during a later trial

## Metrics and Signals

### Journald health

Watch for:

- startup and shutdown events
- request status codes
- restart reasons
- child generation changes
- backend unavailable or child stopped errors

Stop condition:

- repeated restart messages in a short window
- unexpected 5xx bursts
- any log line that reveals a token, path, or payload that should have been redacted

### OAuth and MCP errors

Treat the following as canary failures unless they are the expected result of a deliberate negative test:

- `401 Unauthorized`
- `403 Forbidden`
- `502 Bad Gateway`
- `503 Service Unavailable`
- `504 Gateway Timeout`
- malformed JSON-RPC
- missing `initialize`
- invalid or missing bearer token
- direct NUC -> shim auth rejection when using the runtime token

### Latency

Use the direct NUC -> shim baseline already measured:

- `initialize`: about 2.87 ms direct to shim
- `tools/list`: about 5.84 ms direct to shim

For the gateway canary:

- track first-byte and total response time for each request class
- treat a sustained 3x regression over the direct shim baseline as a stop trigger
- treat any timeout as a stop trigger unless it was intentionally induced during a negative test

### Redaction

Verify the following never appear in logs:

- bearer tokens
- raw request bodies
- file contents
- absolute paths from user input

Allowed log content:

- method
- request id type
- status
- latency
- child generation
- bytes in and out
- restart reason

### Child process stability

The shim owns the child process, so watch for:

- child generation increments
- unexpected exits
- repeated backoff delays
- inability to reinitialize the child after a restart

If the child restarts once, reset the observation timer unless the restart was explicitly induced for testing and fully recovered.

### Filesystem safety

Verify the canary does not write outside the staging tree:

- `/var/lib/hugo-mcp-go`

Checks to run:

```bash
ssh hugo-vm 'sudo -n find /home/jm/hugo-site -xdev -mmin -30 -printf "%TY-%Tm-%Td %TT %p\n" | sort'
ssh hugo-vm 'sudo -n find /var/lib/hugo-mcp-go -xdev -mmin -30 -printf "%TY-%Tm-%Td %TT %p\n" | sort'
```

If available, also compare against the known live tree state before the canary window and after it.

## Observation Windows

- Start the timer at the first successful direct shim request or the first successful gateway request, whichever happens first.
- Before any gateway switch, require a direct NUC -> shim success with the runtime bearer token.
- Keep the NUC and VM journals visible for the full window.
- Absolute minimum window: 30 minutes.
- Preferred window: 60 minutes if any mutation is exercised.

## Escalation Triggers

Escalate and roll back immediately if any of these occur:

- child restart loop
- auth failures from the same caller after the route switch
- latency spikes beyond the stop threshold
- log redaction regression
- unexpected writes in the live site tree
- any need to touch OpenResty, Nginx, or Cloudflare to keep the canary alive

### Auth alignment check

Watch for:

- `mcp-runtime.service` forwarding the runtime bearer to the shim
- the shim accepting that runtime bearer without `401`
- the Python backend continuing to accept the same shared token during rollback

If the shim still rejects the runtime token, stop before any gateway switch.

## Success Definition

The canary is healthy only if all of the following remain true throughout the window:

- NUC gateway stays on the temporary shim route only for the planned window
- Python remains active and ready for rollback
- shim stays up and keeps the child stable
- logs stay redacted
- the staging tree stays isolated
- rollback remains one env restore plus one restart
