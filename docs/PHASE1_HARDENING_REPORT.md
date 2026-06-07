# Phase 1 Hardening Report

## Summary

Phase 1 remains complete and green after hardening the read-only surface and recapturing the oracle fixtures affected by the expanded minimal site.

## What changed

Hardening additions:

- added a plain `content/posts/plain/index.md` fixture to verify `get_page` fallback behavior
- added a Unicode page fixture at `content/posts/unicode/index.fr.md`
- added an uppercase asset fixture at `content/posts/unicode/COVER.SVG`
- fixed `list_pages` to ignore plain `index.md` files so it stays aligned with the Python oracle
- wired startup error redaction through `main.go`
- extended redaction to cover a simple Windows-path pattern
- made invalid numeric env vars fail closed instead of silently falling back to defaults
- added context cancellation checks to read-only page and asset scans
- added tests for all of the above

## Validation

Current command results:

```bash
go test ./...
go test -cover ./internal/...
```

Coverage reported by `go test -cover ./internal/...`:

- `internal/config` 59.4%
- `internal/hugo/assets` 67.7%
- `internal/hugo/frontmatter` 68.0%
- `internal/hugo/pages` 70.7%
- `internal/observability` 75.0%
- `internal/security/pathguard` 65.3%
- `internal/server` 66.7%
- `internal/tools` 17.6%

## Oracle recapture

The Phase 1 oracle fixtures were updated to reflect the expanded minimal corpus:

- `list_pages_all.*` now includes the Unicode page and still excludes `posts/plain/index.md`
- `list_pages_posts_fr.*` now includes the Unicode page
- `list_assets_all.*` now includes `content/posts/unicode/COVER.SVG`

The following files are the active oracle set:

- `testdata/fixtures/oracle/list_pages_all.normalized.json`
- `testdata/fixtures/oracle/list_pages_posts_fr.normalized.json`
- `testdata/fixtures/oracle/list_assets_all.normalized.json`

## Security notes

Resolved in code:

- startup errors are redacted before being written to stderr
- invalid numeric environment variables now fail closed
- read-only scans stop promptly when the context is canceled

Still deferred by design:

- mutation-specific staging and rollback
- path validation for new targets
- build wrapper injection
- file promotion semantics

## Go / no-go

Go for Phase 2 planning docs.

No-go for Phase 2 implementation until the staging model document is approved and the mutation oracle is fully consumed by the implementation plan.
