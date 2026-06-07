# Backend Auth Contract

This document records the live backend authentication contract for the Hugo MCP stack without exposing secret values.

## Contract Summary

- `mcp-runtime.service` forwards inbound MCP requests to the backend with `Authorization: Bearer <HUGO_TOKEN>`.
- The live Python backend accepts the same shared secret as `MCP_TOKEN` fallback auth.
- `hugo-mcp-shim.service` currently expects its own backend secret in `Authorization: Bearer <HUGO_MCP_SHIM_BACKEND_TOKEN>`.
- The shim then forwards to `hugo-mcp-go`; the child process itself is not part of the external auth boundary.

## Redacted Mapping

| Component | Source env | Variable | Usage | Fingerprint |
| --- | --- | --- | --- | --- |
| `mcp-runtime.service` | `/etc/mcp-runtime-go/mcp-runtime.env` | `HUGO_TOKEN` | Bearer token injected by the runtime into backend requests | `sha256:9e53cdfa9592` |
| Python backend | `/home/jm/hugo-mcp/.env` | `MCP_TOKEN` | Fallback bearer token accepted by the live Python service | `sha256:9e53cdfa9592` |
| Shim | `/etc/hugo-mcp-go/hugo-mcp-shim.env` | `HUGO_MCP_SHIM_BACKEND_TOKEN` | Bearer token required by the shim on incoming runtime requests | `sha256:8eeb154baf01` |

## Observed State

- The runtime token and the Python fallback token are the same shared secret.
- The shim backend token is different.
- That mismatch is the confirmed blocker for the second gateway canary.

## Implication

For the next canary, the shim must accept the shared runtime/Python secret already used by `mcp-runtime.service`, unless the runtime token injection is changed instead.

