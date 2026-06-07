# Gateway Attachment Readiness

## Status

The NUC-to-shim path is now validated end to end in staging.

## What Is Proven

- the NUC can reach the VM shim directly on `18180`
- the shim answers the expected MCP handshake
- the shim preserves request ids and returns the Go tool catalog
- the live Python backend on `8000` remained untouched
- the NUC gateway was not modified

## What Is Not Done

- no gateway attachment has been applied
- `mcp-runtime.service` remains unchanged
- no cutover has been performed

## Readiness Verdict

- gateway attachment technically possible: yes
- gateway attachment applied: no
- cutover possible now: no
