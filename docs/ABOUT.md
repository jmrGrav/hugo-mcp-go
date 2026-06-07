# About

`hugo-mcp-go` exists to provide a dedicated Hugo MCP backend in Go with a production-safe transport model.

## Why Go

- predictable deployment and runtime behavior
- explicit binaries and systemd packaging
- native HTTP support for the backend path
- easier fail-closed handling of file access and transport boundaries

## Why MCP

- the project is meant to be used by MCP clients, not through a custom UI
- MCP gives a standard tool catalog, structured arguments, and permission metadata
- the same contract works across Claude, ChatGPT, Gemini, and other MCP-capable clients

## Why Hugo

- the site is already Hugo-based
- the backend should operate on the existing Hugo content and asset model
- page creation, asset handling, site builds, and SRI checks need first-class support

## Security model

- single-tenant, operator-controlled deployment
- OAuth remains in `mcp-runtime-go`
- the native backend uses bearer auth for backend-to-backend traffic
- secrets are file-backed and not logged
- path traversal and symlink escape are rejected

## Current status

- production validated
- rollback preserved
- release candidate prepared
- native HTTP backend active

## Known limits

- Claude refresh can still show `Impossible de recharger les outils depuis le serveur` even when the tools remain usable
- backend SSE exists only as an optional backend capability for now
- `mcp-runtime-go` is still required for public OAuth and gateway behavior
