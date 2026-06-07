# MCP Runtime Refresh Diagnostic

Date: 2026-06-07

## Question

Does the Claude.ai refresh failure originate in `mcp-runtime-go`, before the Hugo backend, or in the Claude client/cache layer?

## Observed State

- Public URL: `https://mcp-hugo.arleo.eu`
- Gateway host listener: `127.0.0.1:8086`
- Active native backend: `http://192.168.122.69:18181/mcp`
- Rollback shim backend: `http://192.168.122.69:18180/mcp`
- Rollback Python backend: `http://192.168.122.69:8000/mcp`

## Discovery Checks

Live discovery endpoints return healthy responses:

- `GET /.well-known/oauth-authorization-server` -> `200`
- `GET /.well-known/oauth-protected-resource` -> `200`
- `POST /mcp` without bearer -> `401` with `WWW-Authenticate: Bearer realm="https://mcp-hugo.arleo.eu", resource_metadata="https://mcp-hugo.arleo.eu/.well-known/oauth-protected-resource"`

Observed response content is consistent with the runtime OAuth contract:

- issuer: `https://mcp-hugo.arleo.eu`
- authorization endpoint: `https://mcp-hugo.arleo.eu/authorize`
- token endpoint: `https://mcp-hugo.arleo.eu/token`
- protected resource: `https://mcp-hugo.arleo.eu/mcp`
- scopes: `mcp`

## Audit Log Evidence

The gateway audit log at `/var/log/mcp-runtime-go/audit.jsonl` shows successful refresh sequences from Claude user traffic:

- `resource_metadata_served`
- `metadata_served`
- `client_registered`
- `authorize_approved`
- `token_issued`
- `proxy_hit` on `POST /mcp`

Relevant observed sequence:

- `GET /.well-known/oauth-protected-resource` -> `resource_metadata_served`
- `GET /.well-known/oauth-authorization-server` -> `metadata_served`
- `POST /register` -> `client_registered`
- `GET /authorize` -> `authorize_approved`
- `POST /token` -> `token_issued`
- `POST /mcp` -> `200`
- `POST /mcp` -> `202`
- `POST /mcp` -> `200`

That exact 200 / 202 / 200 pattern is consistent with:

- `initialize`
- `notifications/initialized`
- `tools/list`

The same pattern is present multiple times in the 19:45 to 19:47 window and again earlier in the day.

## Backend Evidence

The native Hugo backend at `192.168.122.69:18181` logs the same successful `POST /mcp` sequences in the matching windows.

No backend-side 5xx failure, protocol error, or auth rejection is present in the captured refresh windows.

## What Did Not Happen

- No `proxy_error` audit entry was observed for the refresh windows.
- No `authorize_forbidden` entry was observed for the refresh windows.
- No `token_rejected` entry was observed for the refresh windows.
- No `proxy_rejected` entry was observed for the refresh windows.
- No evidence suggests the refresh failed before discovery or token issuance.

## Classification

- Refresh reached nginx / public gateway: yes
- Refresh reached `mcp-runtime-go`: yes
- Refresh reached backend Hugo: yes
- Backend Hugo suspect: no
- `mcp-runtime-go` suspect: no
- Claude cache/UI suspect: probable

## Why Claude Still Shows an Error

The server-side handshake succeeds, and the backend tools are usable, but Claude still shows:

`Impossible de recharger les outils depuis le serveur`

Given the successful discovery + auth + proxy sequence, the remaining error is most likely in the Claude client state, refresh cache, or connector UI flow rather than in `mcp-runtime-go`.

## Recommended Next Step

- Reconnect or remove/re-add the Claude connector once.
- Retry refresh immediately after reconnect.
- If the UI error persists while the audit log still shows successful discovery + proxy hits, treat it as a Claude-side cache/refresh issue.

## Verdict

- backend Hugo: not suspect
- `mcp-runtime-go`: not suspect
- Claude cache/UI: probable
