# Phase 2 Oracle Capture

## Scope

This capture covers the mutation-oriented Python tools:

- `create_page`
- `update_page`
- `delete_page`
- `upload_asset`
- `build_site`

The capture was produced by executing the Python `main.py` module in a temporary virtual environment with the repo requirements installed. External side effects were neutralized by stubbing:

- `run_deploy()` → `DEPLOY_SKIPPED`
- plugin event hooks → empty list

That keeps the captured outputs reproducible while preserving the Python tool contracts.

## Machine-readable schema capture

The source-derived tool schemas are stored in:

- `testdata/fixtures/oracle_phase2/tool_schemas.json`

This file is the contract source for required/optional arguments and user-visible descriptions.

## Nominal captures

Captured request/response pairs:

- `testdata/fixtures/oracle_phase2/create_page.request.json`
- `testdata/fixtures/oracle_phase2/create_page.response.json`
- `testdata/fixtures/oracle_phase2/update_page.request.json`
- `testdata/fixtures/oracle_phase2/update_page.response.json`
- `testdata/fixtures/oracle_phase2/delete_page.request.json`
- `testdata/fixtures/oracle_phase2/delete_page.response.json`
- `testdata/fixtures/oracle_phase2/upload_asset.request.json`
- `testdata/fixtures/oracle_phase2/upload_asset.response.json`
- `testdata/fixtures/oracle_phase2/build_site.request.json`
- `testdata/fixtures/oracle_phase2/build_site.response.json`

Captured side-effect snapshots:

- `testdata/fixtures/oracle_phase2/create_page.after/content/posts/oracle-phase2/index.fr.md.json`
- `testdata/fixtures/oracle_phase2/update_page.after/content/posts/bonjour/index.fr.md.json`
- `testdata/fixtures/oracle_phase2/delete_page.after.json`
- `testdata/fixtures/oracle_phase2/upload_asset.after/static/images/oracle/oracle-phase2.svg.json`

## Error captures

Captured negative fixtures:

- `testdata/fixtures/oracle_phase2/create_page.error.frontmatter_conflict.request.json`
- `testdata/fixtures/oracle_phase2/create_page.error.frontmatter_conflict.response.json`
- `testdata/fixtures/oracle_phase2/update_page.error.immutable_date.request.json`
- `testdata/fixtures/oracle_phase2/update_page.error.immutable_date.response.json`
- `testdata/fixtures/oracle_phase2/delete_page.error.not_found.request.json`
- `testdata/fixtures/oracle_phase2/delete_page.error.not_found.response.json`
- `testdata/fixtures/oracle_phase2/upload_asset.error.invalid_extension.request.json`
- `testdata/fixtures/oracle_phase2/upload_asset.error.invalid_extension.response.json`

## Deterministic fields

The following response fields are deterministic in the capture and safe for parity checks:

- `status`
- `file`
- `path`
- `public_url`
- `size_bytes`
- `cf_purge`
- `plugins`

## Runtime-normalized fields

The following field is runtime-dependent and is normalized in the fixture capture:

- `deploy` is set to `DEPLOY_SKIPPED` by the harness

## Tool contracts

### `create_page`

Required input fields:

- `route`
- `title`
- `content`

Optional input fields:

- `lang` defaults to `fr`
- `tags` defaults to `[]`
- `draft` defaults to `null`
- `frontmatter` defaults to `null`

Response shape:

- `status: "created"`
- `file`
- `deploy`
- `cf_purge`
- `plugins`

### `update_page`

Required input fields:

- `route`

Optional input fields:

- `lang`
- `title`
- `content`
- `tags`
- `draft`
- `frontmatter`

Response shape:

- `status: "updated"`
- `file`
- `deploy`
- `cf_purge`
- `plugins`

Important contract:

- `date` is immutable
- `null` inside `frontmatter` deletes a key
- `frontmatter` is deep-merged

### `delete_page`

Required input fields:

- `route`

Optional input fields:

- `lang`

Response shape:

- `status: "deleted"`
- `file`
- `deploy`
- `cf_purge`
- `plugins`

### `upload_asset`

Required input fields:

- `filename`
- `data`

Optional input fields:

- `subfolder` defaults to `images`

Response shape:

- `status: "ok"`
- `path`
- `public_url`
- `size_bytes`
- `deploy`

### `build_site`

Optional input fields:

- `purge_cf` defaults to `true`

Response shape:

- `status: "built"`
- `deploy`
- `cf_purge`

## Known limits

- `upload_asset` size-limit behavior is real in Python but the corresponding huge-payload request is intentionally not committed here because it would bloat the repository.
- `deploy` remains normalized in the harness because the deploy script is an external side effect, not a stable oracle datum.

## Related Phase 1 update

The read-only corpus was recaptured alongside this work so that the new Unicode page and uppercase asset are represented consistently across the repository.
