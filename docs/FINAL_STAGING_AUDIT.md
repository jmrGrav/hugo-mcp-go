# Final Staging Audit

## Scope

This audit checked whether `hugo-mcp-go` is ready for a shadow/staging run before any production decision.

What was audited:

- read-before-write and rollback orchestration for the five mutation tools
- staging isolation and allowlist behavior
- MCP protocol and error robustness
- parity drift against the captured Python oracle
- security grep for shell, direct exec, hardcoded prod paths, secrets, and plugin discovery

## Findings

### Confirmed

- final file I/O for page and asset mutations is anchored through dirfd/openat-style helpers
- writes stay inside staging roots
- rollback breadcrumbs stay inside `work/`
- missing staging wiring fails closed
- invalid arguments, bad types, unknown tools, unknown fields, oversized uploads, and build failures are covered by tests
- mutation errors are serialized through MCP without leaking production secrets or prod paths in the observed test matrix
- no `sh -c`, `bash -lc`, or direct `exec.Command` calls exist outside the injected runner
- no plugin discovery system exists in the Go repo

### Accepted risks

- there is still a residual read-before-write race window before the final anchored mutation hop
- delete rollback staging still depends on a pre-delete read of the existing file
- `list_assets` still skips some file-level stat/read errors to preserve oracle parity
- full tree-wide dirfd refactor remains deferred because the remaining race window is contained to staging-only writes

### False positives

- hardcoded `/home/jm/...` references in tests and docs are expected and do not appear in production code paths
- references to plugin names or `plugins` in docs and response payloads are oracle/protocol artifacts, not dynamic plugin loading

## Verdict

- Ready for shadow/staging run: **yes**
- Ready for production cutover: **no**

## Validation to rerun

```bash
cd /home/jm/Documents/hugo-mcp-go
go test ./...
go test -cover ./internal/...
rg -n "sh -c|bash -lc|exec\\.Command\\(" internal cmd
rg -n "/home/jm/" internal cmd
```

## Next prompt recommended

- Start a shadow/staging execution plan that keeps the production gateway unchanged and compares only staging-tree side effects against the oracle fixtures.
