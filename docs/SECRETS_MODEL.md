# Secrets Model

The hook subsystem uses file-backed secrets only.

## Secret Files

- `/etc/hugo-mcp-go/secrets/cloudflare_api_token`
- `/etc/hugo-mcp-go/secrets/google_indexing_service_account.json`
- `/etc/hugo-mcp-go/secrets/indexnow_key`

## Permission Model

Secrets must be fail-closed:

- the file must exist
- the file must not be empty
- the file must not be world-readable or world-writable
- the file must stay inside the approved secrets directory
- symlink escapes are rejected
- access should remain restricted to `root` or the service user, with group hugo-mcp

Recommended modes:

- `0640` when the group needs read access
- `0600` when only the service user reads the file

## What Never Goes into SQLite

- Cloudflare API tokens
- Google private keys
- IndexNow keys
- access tokens

SQLite is reserved for:

- hook jobs
- hook attempts
- hook provider state
- hook audit records

## Logging Rules

- no bearer token in logs
- no private key in logs
- no raw secret file contents in logs
- redact before returning error text

## Operational Notes

- keep hooks in dry-run mode until validation is complete
- when a secret file is missing or invalid, fail closed rather than skipping silently
- rotate files out-of-band; the database does not hold secret history
