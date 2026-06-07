# Post Build Hooks

`hugo-mcp-go` now has an explicit hook subsystem for post-build and post-mutation side effects.

## Goals

- purge Cloudflare cache after relevant mutations
- notify Google Indexing
- submit URLs to IndexNow
- keep secrets file-backed
- keep operational state in SQLite only
- default to dry-run

## Configuration

These settings are read from the environment and should be documented in the deployment env file:

- `HUGO_POST_BUILD_HOOKS_ENABLED`
- `HUGO_CLOUDFLARE_PURGE_ENABLED`
- `HUGO_CLOUDFLARE_ALLOW_PURGE_EVERYTHING`
- `HUGO_CLOUDFLARE_ZONE_ID`
- `HUGO_CLOUDFLARE_TOKEN_FILE`
- `HUGO_GOOGLE_INDEXING_ENABLED`
- `HUGO_GOOGLE_INDEXING_SERVICE_ACCOUNT_FILE`
- `HUGO_INDEXNOW_ENABLED`
- `HUGO_INDEXNOW_KEY_FILE`
- `HUGO_INDEXNOW_ENDPOINT`
- `HUGO_HOOKS_DB`
- `HUGO_HOOKS_DRY_RUN`
- `HUGO_HOOKS_MAX_RETRIES`
- `HUGO_HOOKS_ADMIN_ENABLED`
- `HUGO_SITE_BASE_URL`

## Runtime Model

- secrets are read from files only
- SQLite stores job state, retries, audit, and provider status only
- no secret material is written to SQLite
- no secret material is written to logs
- provider clients apply bounded retries
- the pipeline returns a sanitized summary in MCP responses

## Install Paths

- secrets: `/etc/hugo-mcp-go/secrets/*`
- env example: `/etc/hugo-mcp-go/hugo-mcp-go.env`
- database: `/var/lib/hugo-mcp-go/hooks.db`

## Rollback / Disablement

- set `HUGO_POST_BUILD_HOOKS_ENABLED=false`
- set provider-specific `*_ENABLED=false`
- keep `HUGO_HOOKS_DRY_RUN=true` during validation
- the store can be left in place while hooks are disabled

## Validation

- run `go test ./...`
- run `go test -race ./...`
- run `go vet ./...`
- run `scripts/hooks-smoke.sh` in dry-run/mock mode
