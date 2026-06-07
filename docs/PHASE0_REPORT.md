# Phase 0 Report

## Outcome

Phase 0 is complete for the read-only oracle capture set.

This report is superseded by [PHASE0_DECISIONS_ADDENDUM.md](PHASE0_DECISIONS_ADDENDUM.md), which locks the remaining parity decisions and clears Phase 1 to start.

## Files Created

### Repo plan and Phase 0 docs

- `HUGO_MCP_GO_REPO_PLAN.md`
- `docs/PHASE0_ORACLE_FIXTURES.md`
- `docs/PHASE0_REPORT.md`

### Minimal Hugo fixture site

- `testdata/fixtures/minimal-site/content/_index.fr.md`
- `testdata/fixtures/minimal-site/content/posts/bonjour/index.fr.md`
- `testdata/fixtures/minimal-site/content/posts/bonjour/index.en.md`
- `testdata/fixtures/minimal-site/content/posts/bonjour/hero.svg`
- `testdata/fixtures/minimal-site/static/images/site-logo.svg`

### Oracle fixtures

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
- Error fixtures:
  - `testdata/fixtures/oracle/get_page_route_traversal.raw-response.json`
  - `testdata/fixtures/oracle/get_page_bad_lang.raw-response.json`
  - `testdata/fixtures/oracle/get_page_missing.raw-response.json`
  - `testdata/fixtures/oracle/list_assets_path_traversal.raw-response.json`
  - `testdata/fixtures/oracle/unknown_tool.raw-response.json`

## Schemas Confirmed From Python Oracle

### `list_pages`

Input schema:

- `lang` string, optional
- `section` string, optional

Actual response fields:

- `pages[]`
  - `route`
  - `lang`
  - `file`
  - `title`
  - `date`
  - `draft`
  - `tags`
- `total`
- optional `skipped`
- optional `error`

No draft filter exists in the oracle.

### `get_page`

Input schema:

- `route` string, required
- `lang` string, optional

Actual response fields:

- `route`
- `file`
- `frontmatter`
- `content`

Important behavior:

- `route="/"` and `route="_index"` are normalized to the root page.
- If a language-specific file is missing, the oracle falls back to the language-less `index.md` or `_index.md`.
- The response does not indicate whether a fallback occurred.

### `list_assets`

Input schema:

- `type` string enum: `image`, `document`, `all`
- `path_prefix` string, optional
- `max_results` integer, optional

Actual response fields:

- `count`
- `truncated`
- `assets[]`
  - `path`
  - `size_bytes`
  - `mime_type`
  - `modified`

Actual scan behavior:

- scans both `static/` and `content/`
- excludes `index.*` and `_index.*`
- rejects `path_prefix` traversal attempts
- uses filesystem `modified` timestamps for ordering

## Deterministic vs Non-Deterministic

### Deterministic enough for committed parity

- `list_pages.route`
- `list_pages.lang`
- `list_pages.file`
- `list_pages.title`
- `list_pages.draft`
- `list_pages.tags`
- `list_pages.total`
- `get_page.route`
- `get_page.file`
- `get_page.frontmatter`
- `get_page.content`
- `list_assets.count`
- `list_assets.truncated`
- `list_assets.assets[].path`
- `list_assets.assets[].size_bytes`
- `list_assets.assets[].mime_type`

### Fixture-sensitive and normalized or controlled

- `list_pages.date`
- `list_pages` ordering when dates tie
- `list_pages.skipped`
- `list_pages.error`
- `list_assets.assets[].modified`
- `list_assets` ordering when mtimes tie

## Oracle Capture Commands

The capture was generated locally with a source-derived Python harness and the repo-owned minimal site fixture.

Key inputs:

- `hugo-mcp/main.py`
- `testdata/fixtures/minimal-site/`

Key outputs:

- raw request/response JSON under `testdata/fixtures/oracle/`
- normalized golden JSON under `testdata/fixtures/oracle/`

## Findings From Review

### Oracle/Parity review

- The phase 0 corpus is sufficient for a first read-only pass, but only after the addendum locks the remaining decisions is it safe to start Phase 1 implementation.
- Missing or ambiguous items:
  - `list_pages` has no draft filter in the real oracle.
  - `get_page` language fallback must be documented explicitly.
  - `list_assets` has no page-association field.
  - ordering depends on fixed dates/mtimes.
- Verdict: no-go before the addendum; go after the addendum.

### Security/Brooks review

- The capture docs must explicitly ban secrets, auth headers, shell transcripts, and environment-bearing strings from committed fixtures.
- The capture docs must define symlink policy before Phase 1.
- Verdict: no-go before the addendum; go after the addendum.

## Remaining Unknowns

1. Should Phase 1 mirror the oracle exactly on `get_page` fallback behavior, or intentionally diverge?
2. Should committed fixtures preserve raw `modified` values or keep them only in raw captures and strip them from normalized parity files?
3. Should read-only symlinked content be rejected or treated as a negative-test-only case?
4. Should `list_pages` and `list_assets` tie ordering be preserved or normalized out?
5. `hugo-mcp-go` now has a `go.mod` and an initial scaffold; the remaining unknowns are limited to implementation details and later parity extensions.

## Go / No-Go Decision

**Go for Phase 1 implementation start.**

The read-only fixtures exist, the addendum locks the ambiguous contracts, and the scaffold now exists. Remaining work is implementation detail, not decision making:

- explicit redaction enforcement in code and tests
- explicit symlink enforcement in code and tests
- parity tests for `get_page` fallback
- parity tests for `list_pages` and `list_assets` ordering

Once those are implemented, Phase 1 can proceed on a parity-safe basis.
