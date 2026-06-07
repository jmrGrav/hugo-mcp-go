# Shadow / Staging Report

## Status

Shadow/staging validation completed against the staging-only `hugo-mcp-go` implementation.

- ready for shadow/staging: yes
- production cutover: no-go
- `mcp-runtime-go`: unchanged
- Python oracle: unchanged

## Scope executed

Validated the following in staging-only mode:

- `list_pages`
- `get_page`
- `list_assets`
- `create_page`
- `update_page`
- `delete_page`
- `upload_asset`
- `build_site`

The validation used:

- the existing oracle fixtures under `testdata/fixtures/oracle/`
- the Phase 2 oracle fixtures under `testdata/fixtures/oracle_phase2/`
- isolated temp workspaces created by the test harness

Note: the Phase 2 oracle corpus uses `.after` snapshots rather than a populated `testdata/fixtures/oracle_phase2/staging/` tree.

## Commands executed

```bash
cd /home/jm/Documents/hugo-mcp-go
go test ./...
go test -cover ./internal/...
rg -n "sh -c|bash -lc|exec\\.Command\\(" internal cmd
rg -n "/home/jm/" internal cmd
rg -n "Bearer|token|secret|password|api_key|apikey|x-api-key" docs testdata internal --glob '!**/*.svg' --glob '!**/*.png' --glob '!**/*.jpg' --glob '!**/*.gif'
find testdata/fixtures -type l -print
find testdata/fixtures/minimal-site -maxdepth 1 -type l -print
find testdata/fixtures/oracle_phase2 -maxdepth 2 -type d | sort
```

## Environment

- working directory: `/home/jm/Documents/hugo-mcp-go`
- staging validation used temporary directories created by tests
- no production Hugo tree was mounted or modified
- no gateway routing change was applied

## Results

### Tests

- `go test ./...` passed
- `go test -cover ./internal/...` passed

### Coverage snapshot

- `internal/config` 60.9%
- `internal/hugo/assets` 67.7%
- `internal/hugo/frontmatter` 68.0%
- `internal/hugo/mutations` 70.3%
- `internal/hugo/pages` 72.4%
- `internal/hugo/staging` 67.6%
- `internal/observability` 75.0%
- `internal/runner` 95.5%
- `internal/security/pathguard` 66.3%
- `internal/server` 70.8%
- `internal/tools` 69.0%

### Oracle parity

Accepted normalization and documented drift:

- `deploy` is normalized to `DEPLOY_SKIPPED`
- `list_assets` normalizes non-deterministic `modified` values in golden comparisons
- read-only list behavior preserves the existing skip policy for some file-level errors
- no fields were invented beyond the captured oracle fixtures

Unexpected diffs:

- none observed in the executed validation matrix

Corpus note:

- the intended staging-side comparison is represented by the committed `.after` snapshots in the oracle corpus, not by a separate committed `staging/` directory

### Security audit

Confirmed:

- no `sh -c`
- no `bash -lc`
- no direct `exec.Command` outside the runner wrapper
- no plugin discovery in Go
- no writes outside staging roots in the exercised tests
- no secret leakage observed in the exercised fixtures/docs/tests

Accepted caveats:

- hardcoded `/home/jm/...` references remain in tests and docs only
- residual read-before-write race windows remain documented and accepted for staging-only use

## Files touched by shadow validation

No production files were modified during the shadow run itself. The validation exercised the existing repo state and the following docs were added or updated to record the audit:

- `docs/FINAL_STAGING_AUDIT.md`
- `docs/SHADOW_STAGING_CHECKLIST.md`
- `docs/SECURITY.md`
- `docs/PARITY.md`
- `docs/PHASE2_HARDENING_REPORT.md`

## Errors observed

Observed errors were the expected oracle and validation failures already covered by tests:

- invalid tool name
- invalid argument type
- unknown field rejection by MCP validation
- missing staging wiring
- traversal rejection
- symlink rejection
- invalid base64
- oversized upload rejection
- build timeout
- runner failure

No unexpected runtime errors were observed in the executed matrix.

## Final verdict

- shadow/staging: **go**
- production cutover: **no-go**

## Next recommended step

- Keep shadow/staging as the operating mode until a separate production cutover decision is made.
- If further hardening is desired, the only remaining meaningful discussion is whether the residual read-before-write window justifies a tree-wide dirfd refactor.
