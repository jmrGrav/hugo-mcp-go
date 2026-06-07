# Live Cutover Report

## Summary

The runtime was routed to the Go backend and validated live against the local authoritative MCP runtime path.

## Commit and Tag

- Commit SHA: `645c60c658043b1df97b8a6f0ad69c58152c3c89`
- Branch: `main`
- Tag: `v0.9.0-rc1`

## Service State

- `mcp-runtime.service`: active on the NUC
- `hugo-mcp-shim.service`: active on the VM
- `hugo-mcp-go` child process: active under the shim service
- Python backend: still available for rollback

## Live Validation

Validated through the live runtime path:

- `initialize`: success
- `tools/list`: success
- `list_pages`: success
- `get_page`: success
- `build_site`: success

Observed results:

- No timeout
- No 504
- No panic
- No visible MCP transport error during the final live checks

## Incident Encountered

`build_site` initially failed on the VM with filesystem permission errors in Hugo-generated directories:

- `mkdir /var/lib/hugo-mcp-go/resources/_gen/assets: permission denied`
- `chtimes /var/lib/hugo-mcp-go/public: operation not permitted`

Root cause:

- The runtime path executes the build as `hugo-mcp-shim`.
- Hugo needs write and ownership-compatible access to `public` and generated cache directories.
- The directories were owned for the backend user path, not the live shim user.

## Corrections Applied

Applied on the VM only:

- `chgrp -R hugo-mcp /var/lib/hugo-mcp-go/resources/_gen`
- `chmod -R g+rwX /var/lib/hugo-mcp-go/resources/_gen`
- `chown -R hugo-mcp-shim:hugo-mcp /var/lib/hugo-mcp-go/public`
- `chmod -R g+rwX /var/lib/hugo-mcp-go/public`
- `chmod g+s` on writable build directories to keep group inheritance stable

After that, the live `build_site` call returned:

- `status: built`
- `deploy: DEPLOY_SKIPPED`
- `cf_purge.skipped: use cloudflare plugin`

## Rollback

- Rollback document created: yes
- Rollback executed during this session: no
- Rollback readiness: yes

## Remaining Gaps

- `tools/list` metadata is still not a full parity clone of the Python oracle
- `check_sri_versions` and `generate_featured_image` remain out of scope
- Tool ordering and schema metadata still have documented differences

## Verdict

- GitHub private synchronized: yes
- Tag created: yes
- Service Go active: yes
- Claude functional: yes
- Rollback ready: yes
- Production usable: yes
