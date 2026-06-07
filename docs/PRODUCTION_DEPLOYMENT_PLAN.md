# Production Deployment Plan

## Goal

Define the production operating model for `hugo-mcp-go` without changing production systems in this repo.

## Current Constraint

The repository currently exposes a stdio MCP server. Production deployment must therefore first choose one of these transport models:

1. gateway-managed stdio subprocess
2. loopback HTTP backend in front of the gateway

The repo currently does not implement the loopback HTTP backend.

## Recommended Production Layout

- dedicated Unix user: `hugo-mcp`
- dedicated Unix group: `hugo-mcp`
- app root: `/srv/hugo-mcp-go`
- config dir: `/etc/hugo-mcp-go`
- state dir: `/var/lib/hugo-mcp-go`
- logs: journald only

Suggested on-disk layout:

- `/srv/hugo-mcp-go/bin/hugo-mcp-go`
- `/var/lib/hugo-mcp-go/content`
- `/var/lib/hugo-mcp-go/static`
- `/var/lib/hugo-mcp-go/public`
- `/var/lib/hugo-mcp-go/work`
- `/etc/hugo-mcp-go/hugo-mcp-go.env`

## Required Environment

Required:

- `HUGO_ROOT`
- `HUGO_CONTENT_ROOT`
- `HUGO_STATIC_ROOT`

Optional limits:

- `HUGO_MAX_REQUEST_BYTES`
- `HUGO_MAX_TOOL_ARGS_BYTES`
- `HUGO_MAX_PAGE_BYTES`
- `HUGO_MAX_ASSET_BYTES`
- `HUGO_MAX_LIST_PAGES`
- `HUGO_MAX_LIST_ASSETS`

Deployment requirement until the binary path is made explicit in code:

- pin `PATH` to the exact Hugo installation directory in the service environment

## Example systemd Unit

This is an example only. It is not active and must not be deployed without review.

```ini
[Unit]
Description=hugo-mcp-go
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=hugo-mcp
Group=hugo-mcp
WorkingDirectory=/var/lib/hugo-mcp-go
EnvironmentFile=/etc/hugo-mcp-go/hugo-mcp-go.env
Environment=PATH=/usr/local/bin:/usr/bin:/bin
ExecStart=/srv/hugo-mcp-go/bin/hugo-mcp-go
Restart=on-failure
RestartSec=2
NoNewPrivileges=yes
PrivateTmp=yes
ProtectSystem=strict
ProtectHome=yes
ReadWritePaths=/var/lib/hugo-mcp-go
CapabilityBoundingSet=
LockPersonality=yes
MemoryMax=512M
CPUQuota=50%
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
```

## Deployment Steps

1. build a reproducible binary
2. install the binary under `/srv/hugo-mcp-go/bin/`
3. create the state directories under `/var/lib/hugo-mcp-go`
4. create the environment file under `/etc/hugo-mcp-go`
5. validate the service starts under the dedicated user
6. validate the staging paths are canonical and non-symlinked
7. validate the build runner resolves the expected `hugo` binary
8. validate the gateway attachment mode chosen for production

## Production Acceptance Conditions

- deployment contract is explicit and repeatable
- service runs under a dedicated user and group
- file system writes are limited to the intended state roots
- binary release is versioned and checksum-verified
- gateway attachment mode is documented and tested
- rollback can restore the prior binary and prior config without touching production data trees
