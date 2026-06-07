# Tool Catalog Parity Report

Date: 2026-06-07

## Status

The MCP tool catalog now includes the two previously missing Python tools:

- `check_sri_versions`
- `generate_featured_image`

## Behavioral Notes

- `check_sri_versions`
  - accepts `auto_fix` and `dry_run`
  - returns the structured `no_handlers` response when no audit handlers are configured
  - returns deterministic validation errors for invalid input types

- `generate_featured_image`
  - accepts `style`, `title`, `subtitle`, `tags`, `accent`, `slug`, `route`, and `lang`
  - validates safe slugs and accent colors
  - generates a local featured image file under `static/images/`
  - updates matched page frontmatter with `featuredImage`
  - fails closed on missing services or missing pages

## Ordering

The tool catalog is stable and deterministic. The current registration order matches the SDK’s observed ordering used by the smoke tests:

1. `build_site`
2. `check_sri_versions`
3. `create_page`
4. `delete_page`
5. `generate_featured_image`
6. `get_page`
7. `list_assets`
8. `list_pages`
9. `update_page`
10. `upload_asset`

## Validation

- `go test ./...` passes
- `go test -race ./...` passes
- `go vet ./...` passes
- `internal/tools` coverage is now `92.7%`

## Remaining Gaps

- The remaining coverage in `internal/tools` is concentrated in defensive helper branches.
- The future hooks subsystem is still pending and will introduce new public-facing configuration and runtime paths.

## Verdict

- Python tool parity complete: `yes`
- smoke/runtime validation against the branch-local shim: `yes`
- ready for the next subsystem: `yes`
