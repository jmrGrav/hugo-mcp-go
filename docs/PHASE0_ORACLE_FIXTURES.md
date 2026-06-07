# Phase 0 Oracle Fixtures

## Scope

This document records how the read-only oracle fixtures for `list_pages`, `get_page`, and `list_assets` were captured for `hugo-mcp-go`.

The fixtures live under:

- `testdata/fixtures/minimal-site/`
- `testdata/fixtures/oracle/`

The oracle capture was generated from the behavior defined in `hugo-mcp/main.py`, replayed against a repo-owned minimal Hugo site fixture. No production Hugo files were modified.

## Capture Method

The captured fixtures were generated with a local Python harness that:

1. Replayed the helper logic from `main.py` for the three read-only tools.
2. Used the repo-owned minimal site under `testdata/fixtures/minimal-site/`.
3. Emitted raw MCP-style request/response JSON alongside normalized golden payloads.

This is a source-derived oracle capture, not a live production edit.

## Captured Files

- `testdata/fixtures/oracle/tools_list.request.json`
- `testdata/fixtures/oracle/tools_list.raw-response.json`
- `testdata/fixtures/oracle/list_pages_all.request.json`
- `testdata/fixtures/oracle/list_pages_all.raw-response.json`
- `testdata/fixtures/oracle/list_pages_all.normalized.json`
- `testdata/fixtures/oracle/list_pages_posts_fr.request.json`
- `testdata/fixtures/oracle/list_pages_posts_fr.raw-response.json`
- `testdata/fixtures/oracle/list_pages_posts_fr.normalized.json`
- `testdata/fixtures/oracle/get_page_bonjour_fr.request.json`
- `testdata/fixtures/oracle/get_page_bonjour_fr.raw-response.json`
- `testdata/fixtures/oracle/get_page_bonjour_fr.normalized.json`
- `testdata/fixtures/oracle/get_page_root_fr.request.json`
- `testdata/fixtures/oracle/get_page_root_fr.raw-response.json`
- `testdata/fixtures/oracle/get_page_root_fr.normalized.json`
- `testdata/fixtures/oracle/list_assets_all.request.json`
- `testdata/fixtures/oracle/list_assets_all.raw-response.json`
- `testdata/fixtures/oracle/list_assets_all.normalized.json`
- `testdata/fixtures/oracle/list_assets_post_images.request.json`
- `testdata/fixtures/oracle/list_assets_post_images.raw-response.json`
- `testdata/fixtures/oracle/list_assets_post_images.normalized.json`
- Error fixtures for invalid routes, invalid languages, missing pages, invalid asset prefixes, and unknown tools.

## Deterministic Fields

### `list_pages`

Keep in committed parity fixtures:

- `pages[].route`
- `pages[].lang`
- `pages[].file`
- `pages[].title`
- `pages[].date`
- `pages[].draft`
- `pages[].tags`
- `total`

Optional error-only fields to keep when present:

- `skipped`
- `error`

### `get_page`

Keep in committed parity fixtures:

- `route`
- `file`
- `frontmatter`
- `content`

### `list_assets`

Keep in raw capture:

- `count`
- `truncated`
- `assets[].path`
- `assets[].size_bytes`
- `assets[].mime_type`
- `assets[].modified`

Keep in normalized golden fixtures:

- `count`
- `truncated`
- `assets[].path`
- `assets[].size_bytes`
- `assets[].mime_type`

`assets[].modified` is captured in the raw response but removed from normalized parity fixtures because it depends on filesystem mtimes.

## Non-Deterministic or Fixture-Sensitive Fields

- `list_pages` ordering when two pages share the same `date`
- `list_pages.skipped` when a filesystem read error occurs
- `list_pages.error` when the scan root cannot be walked
- `get_page` fallback choice when a language-specific file is missing
- `list_assets.modified` unless the fixture tree has pinned mtimes
- `list_assets` ordering when `modified` ties

## Error Shapes Captured

The committed error fixtures capture the real MCP error envelope:

- `content[0].type = "text"`
- `content[0].text = <error message>`
- `isError = true`

Observed examples:

- path traversal: `Invalid route (path traversal): ../escape`
- invalid language: `Invalid lang (must match ^[a-z]{2,3}$): english`
- missing page: `Page not found: posts/missing (lang=fr)`
- invalid asset prefix: `Invalid path_prefix: must be relative without '..'`
- unknown tool: `Tool not found: not_a_tool`

## Redaction Policy

Committed fixtures must stay free of:

- bearer tokens
- auth headers
- shell transcripts
- environment values
- production secrets

If a future capture contains absolute paths or environment-bearing strings in an error message, canonicalize or redact them before writing the committed fixture.

## Symlink Policy

For Phase 0 committed fixtures:

- no symlinks are allowed in `testdata/fixtures/minimal-site/`
- no symlinks are allowed in `testdata/fixtures/oracle/`

Symlinked fixtures should be treated as explicit negative tests later, not silently followed in the committed oracle corpus.

## Open Decisions Before Phase 1

1. `list_pages` does not expose a draft filter in the oracle. Do not invent one for Phase 1 parity.
2. `get_page` language fallback is real behavior and must be documented as a contract, even though the fallback is not surfaced in the response.
3. `list_assets` does not expose page association in the oracle. Do not add one during parity work.
4. Tie ordering for `list_pages` and `list_assets` must be controlled by fixture data or normalized out of the comparison.
