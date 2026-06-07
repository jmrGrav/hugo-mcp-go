# Security Policy

`hugo-mcp-go` is an operator-controlled single-tenant MCP backend. It is not a public multi-tenant SaaS.

## Supported versions

- `v0.1.0` release candidate and matching `feature/native-sri-port` state
- the current native HTTP production line
- the preserved rollback paths remain available until the operator removes them

## Threat model

The service must assume:

- untrusted MCP clients
- hostile or malformed tool arguments
- untrusted filesystem input under configured Hugo roots
- stale or incorrect connector state in client UIs

## Secret handling

- secrets are file-backed or environment-backed only for runtime wiring
- secrets are never committed
- secrets are never stored in SQLite
- secrets are not written to logs
- token files must be readable only by the service user or its intended group

## Transport security

- HTTP mode requires bearer authentication
- OAuth discovery and client flow remain delegated to `mcp-runtime-go`
- the native backend binds only to the configured interface and port
- there is no permissive CORS policy
- request and response sizes are bounded
- timeouts are required for long-running operations

## Path security

- paths are root-relative under configured Hugo roots
- traversal attempts are rejected
- symlink escapes are rejected
- absolute user-supplied paths are rejected
- dangerous backslashes are rejected

## Tool classification

The tool catalog uses MCP metadata so clients can distinguish read-only tools from mutating ones.

- read-only tools should carry `annotations.readOnlyHint: true`
- destructive or mutating tools should carry `annotations.destructiveHint: true`
- the tool `title` must be stable and descriptive

## Reporting vulnerabilities

If this repository is public, report issues through GitHub Security Advisories.
Otherwise, contact the maintainer directly through the repository’s agreed operator channel.

## Known issues

- [`docs/known-issues/CLAUDE_REFRESH_001.md`](docs/known-issues/CLAUDE_REFRESH_001.md)
