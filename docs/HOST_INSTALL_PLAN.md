# Host Install Plan

## Goal

Describe the minimum safe installation shape for `hugo-mcp-go` on the real Arleo infrastructure without performing the installation now.

This plan assumes the RC is installed on the Hugo VM where the real site tree exists.

## Target Layout

- service user: `hugo-mcp`
- service group: `hugo-mcp`
- binary: `/srv/hugo-mcp-go/bin/hugo-mcp-go`
- env file: `/etc/hugo-mcp-go/hugo-mcp-go.env`
- state root: `/var/lib/hugo-mcp-go`
- content root: `/var/lib/hugo-mcp-go/content`
- static root: `/var/lib/hugo-mcp-go/static`
- public root: `/var/lib/hugo-mcp-go/public`
- work root: `/var/lib/hugo-mcp-go/work`

## Source Tree

The existing live Hugo tree is currently at:

- `/home/jm/hugo-site`

That tree is the source of truth for the real site today.

## Installation Steps

1. Create the dedicated service user and group if they are not already present.
2. Create the package directories:
   - `/srv/hugo-mcp-go/bin`
   - `/etc/hugo-mcp-go`
   - `/var/lib/hugo-mcp-go`
   - `/var/lib/hugo-mcp-go/content`
   - `/var/lib/hugo-mcp-go/static`
   - `/var/lib/hugo-mcp-go/public`
   - `/var/lib/hugo-mcp-go/work`
3. Seed the RC staging tree from the real Hugo site tree using a non-destructive copy or clone process.
4. Install the `hugo-mcp-go` binary under `/srv/hugo-mcp-go/bin/`.
5. Install the environment file under `/etc/hugo-mcp-go/hugo-mcp-go.env`.
6. Install the systemd unit from `deploy/systemd/hugo-mcp-go.service` as a reviewed, non-active artifact.
7. Run `deploy/validation/preflight.sh` against the target root.
8. Start the RC only in staging mode on the host after the preflight passes.

## Required Environment

- `HUGO_ROOT=/var/lib/hugo-mcp-go`
- `HUGO_CONTENT_ROOT=/var/lib/hugo-mcp-go/content`
- `HUGO_STATIC_ROOT=/var/lib/hugo-mcp-go/static`
- `HUGO_EXPECTED_HUGO_BIN=/usr/local/bin/hugo`
- `HUGO_EXPECTED_HUGO_VERSION_PREFIX=` or an operator-approved prefix
- `HUGO_MAX_REQUEST_BYTES`
- `HUGO_MAX_TOOL_ARGS_BYTES`
- `HUGO_MAX_PAGE_BYTES`
- `HUGO_MAX_ASSET_BYTES`
- `HUGO_MAX_LIST_PAGES`
- `HUGO_MAX_LIST_ASSETS`
- `PATH=/usr/local/bin:/usr/bin:/bin`

## Systemd Expectations

The installed unit should keep the review-only hardening shape already documented in the repo:

- `User=hugo-mcp`
- `Group=hugo-mcp`
- `NoNewPrivileges=yes`
- `ProtectSystem=strict`
- `ProtectHome=yes`
- `PrivateTmp=yes`
- `CapabilityBoundingSet=`
- `MemoryMax=`
- `CPUQuota=`
- journald-only logs
- restart policy enabled

## Permissions

Recommended file ownership after installation:

- `/srv/hugo-mcp-go` owned by `root:root`
- `/var/lib/hugo-mcp-go` owned by `hugo-mcp:hugo-mcp`
- `/etc/hugo-mcp-go` owned by `root:root`
- service runtime writes limited to the state root only

## Rollback

Rollback must be able to restore:

- the previous binary under `/srv/hugo-mcp-go/bin/`
- the previous env file under `/etc/hugo-mcp-go/`
- the previous unit file if it changed
- the previous state tree snapshot if the staging copy needs to be reverted

Rollback must not touch production content outside the approved root.

## Integration Note

The host install plan is only the local service side. The gateway attachment on the NUC remains a separate operational decision and is intentionally not changed here.

