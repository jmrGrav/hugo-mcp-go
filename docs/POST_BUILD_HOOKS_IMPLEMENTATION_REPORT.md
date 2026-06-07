# Post Build Hooks Implementation Report

## Architecture Chosen

The hook subsystem is split into:

- file-backed secrets
- a small SQLite store for non-secret state
- provider clients for Cloudflare, Google Indexing, and IndexNow
- a pipeline that enqueues work and returns sanitized MCP summaries

## Why This Design

- file-backed secrets avoid persisting credentials in SQLite
- SQLite keeps retry and job state local and inspectable
- dry-run is the default so operators can validate safely
- provider clients can be tested with mocks without calling live services

## Current Configuration Contract

- `HUGO_POST_BUILD_HOOKS_ENABLED=true`
- `HUGO_CLOUDFLARE_PURGE_ENABLED=false`
- `HUGO_CLOUDFLARE_ALLOW_PURGE_EVERYTHING=false`
- `HUGO_CLOUDFLARE_ZONE_ID=`
- `HUGO_CLOUDFLARE_TOKEN_FILE=/etc/hugo-mcp-go/secrets/cloudflare_api_token`
- `HUGO_GOOGLE_INDEXING_ENABLED=false`
- `HUGO_GOOGLE_INDEXING_SERVICE_ACCOUNT_FILE=/etc/hugo-mcp-go/secrets/google_indexing_service_account.json`
- `HUGO_INDEXNOW_ENABLED=false`
- `HUGO_INDEXNOW_KEY_FILE=/etc/hugo-mcp-go/secrets/indexnow_key`
- `HUGO_INDEXNOW_ENDPOINT=https://api.indexnow.org/indexnow`
- `HUGO_HOOKS_DB=/var/lib/hugo-mcp-go/hooks.db`
- `HUGO_HOOKS_DRY_RUN=true`
- `HUGO_HOOKS_MAX_RETRIES=5`
- `HUGO_HOOKS_ADMIN_ENABLED=false`
- `HUGO_SITE_BASE_URL=https://example.com`

## Validation Status

- unit tests: pass
- race tests: pass
- vet: pass
- provider clients: mocked only
- live provider calls: disabled by default
- local protocol smoke: pass against the branch shim with a temporary fake `hugo`

## Remaining Risks

- production secret permissions still need to be provisioned correctly on each host
- live provider endpoints should be enabled one at a time
- the site base URL must be correct for absolute URL generation
- the local validation shim uses `HUGO_MCP_CHILD_PATH` only for smoke testing; production defaults remain unchanged

## Rollback / Disablement

- set `HUGO_POST_BUILD_HOOKS_ENABLED=false`
- leave the SQLite file in place
- leave the secret files in place
- keep `HUGO_HOOKS_DRY_RUN=true` while diagnosing
