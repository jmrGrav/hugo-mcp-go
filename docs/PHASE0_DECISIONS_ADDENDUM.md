# Phase 0 Decisions Addendum

This addendum locks the remaining read-only parity decisions so Phase 1 can start on a safe contract.

## Locked Decisions

### 1. `list_pages`

- Do not invent a draft filter.
- Phase 1 must mirror the oracle exactly on input scope:
  - `lang`
  - `section`
- Keep the response shape aligned with the Python oracle:
  - `pages[]`
  - `total`
  - optional `skipped`
  - optional `error`

### 2. `get_page`

- Reproduce the Python fallback language behavior as part of the Phase 1 contract.
- If a language-specific page is missing, fall back to the language-less file exactly as Python does.
- Do not expose the fallback decision in the response; Python does not expose it.

### 3. `list_assets`

- Do not add page association in Phase 1.
- Mirror the Python oracle shape exactly:
  - `count`
  - `truncated`
  - `assets[]`
    - `path`
    - `size_bytes`
    - `mime_type`
    - `modified`

### 4. Ordering

- `list_pages`:
  - use fixtures with distinct dates whenever possible;
  - if dates tie, normalize in tests rather than depending on filesystem ordering.
- `list_assets`:
  - do not compare `modified` in normalized golden fixtures;
  - if two assets share the same mtime, use a stable path sort in tests/normalizers.

### 5. Redaction

- No token, auth header, environment value, production secret, or production-sensitive path may appear in committed fixtures, docs, or logs.
- Any future capture containing sensitive material must be canonicalized or redacted before commit.

### 6. Symlink Policy

- No symlinks are allowed in positive oracle fixtures.
- Any symlink behavior must be covered only by explicit negative tests.
- Phase 1 read-only code must fail closed if a symlink would escape the allowlisted roots.
- If the behavior is ambiguous, fail closed.

## Phase 1 Go/No-Go Change

With these decisions locked, Phase 1 is now **go**.

The implementation must still preserve the documented parity gaps and security guardrails:

- no write tools
- no shell execution
- no plugin discovery
- no Cloudflare/IndexNow/Google/SRI
- no featured image generation
