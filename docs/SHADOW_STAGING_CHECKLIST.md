# Shadow Staging Checklist

## Preconditions

- `go test ./...` passes
- `go test -cover ./internal/...` passes
- final audit concludes ready for shadow/staging, not production
- `mcp-runtime-go` remains unchanged
- Python oracle remains unchanged

## Environment

- staging roots are configured and canonicalized
- staging content, static, public, and work roots are present
- no production tree is mounted as a staging root
- no symlinked staging root is allowed

## Tool scope

- `list_pages`
- `get_page`
- `list_assets`
- `create_page`
- `update_page`
- `delete_page`
- `upload_asset`
- `build_site`

## Checks before each run

- confirm no new shell execution was added
- confirm no direct `exec.Command` exists outside the runner wrapper
- confirm no hardcoded production paths entered code
- confirm no secret material entered fixtures, docs, or tests
- confirm no plugin discovery was introduced

## Runtime checks during shadow/staging

- compare only deterministic fields against oracle fixtures
- normalize timestamps and other non-deterministic fields before comparison
- keep write activity limited to staging roots
- keep rollback artifacts inside `work/`
- inspect rejected MCP calls for redaction and leakage

## Failure policy

- fail closed on traversal
- fail closed on symlink escape
- fail closed on missing staging wiring
- fail closed on oversized upload payloads
- fail closed on build timeout or runner failure

## Exit criteria

- shadow/staging run completed with no write outside staging
- parity diffs are only the documented oracle gaps
- security audit remains clean
- no production cutover decision is implied by the shadow run
