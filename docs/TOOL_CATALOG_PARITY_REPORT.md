# Tool Catalog Parity Report

Date: 2026-06-07

## Status

The native MCP tool catalog now covers the historical Python-era tools and adds the two native chunk helpers:

- `check_sri_versions`
- `generate_featured_image`
- `get_page_chunk`
- `get_asset_chunk`

`check_sri_versions` is now implemented natively in Go and no longer depends on the historical bash script at runtime.
The live native catalog now also exposes MCP tool `title` and `annotations` metadata after redeploy.

## Behavioral Notes

- `check_sri_versions`
  - accepts `auto_fix` and `dry_run`
  - returns a structured `sri-check` report with `plugin`, `success`, `exit_code`, `report`, and `downstream`
  - performs native version and SRI evaluation in Go
  - returns deterministic validation errors for invalid input types

- `generate_featured_image`
  - accepts `style`, `title`, `subtitle`, `tags`, `accent`, `slug`, `route`, and `lang`
  - validates safe slugs and requires an accent color in strict hex form like `#7aa2f7`
  - generates a local featured image file under `static/images/`
  - updates matched page frontmatter with `featuredImage`
  - fails closed on missing services or missing pages

- `list_assets`
  - `path_prefix` is root-relative under the Hugo `content/` or `static/` roots
  - an exact directory prefix matches, partial substrings do not
  - traversal attempts are rejected

## Ordering

The tool catalog is stable and deterministic. The current registration order matches the SDK’s observed ordering used by the smoke tests:

1. `build_site`
2. `check_sri_versions`
3. `create_page`
4. `delete_page`
5. `generate_featured_image`
6. `get_asset_chunk`
7. `get_page`
8. `get_page_chunk`
9. `list_assets`
10. `list_pages`
11. `update_page`
12. `upload_asset`

## Validation

- `go test ./...` passes
- `go test -race ./...` passes
- `go vet ./...` passes
- `internal/tools` coverage is now `91.4%`
- live native `tools/list` after redeploy shows `title`, `annotations.readOnlyHint`, and `annotations.destructiveHint`

## Remaining Gaps

- The remaining coverage in `internal/tools` is concentrated in defensive helper branches.
- The future hooks subsystem is still pending and will introduce new public-facing configuration and runtime paths.

## Verdict

- historical Python-era tools preserved: `yes`
- native-only chunk helpers added: `yes`
- smoke/runtime validation against the native HTTP backend: `yes`
- ready for the next subsystem: `yes`
