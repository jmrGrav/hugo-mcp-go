# Second Canary Fix Plan

## Problem

The first gateway canary failed because the NUC runtime and the VM shim did not share the same backend bearer token.

- `mcp-runtime.service` injects `Authorization: Bearer <HUGO_TOKEN>`
- Python accepts that same shared secret via `MCP_TOKEN`
- `hugo-mcp-shim.service` currently expects a different value in `HUGO_MCP_SHIM_BACKEND_TOKEN`

## Options

### Option A: Align the shim to the runtime token

Change only `HUGO_MCP_SHIM_BACKEND_TOKEN` in the shim env so it matches the shared runtime/Python secret already used by `mcp-runtime.service`.

Pros:

- no NUC-side config change
- no `HUGO_MCP_URL` change
- no `mcp-runtime.service` restart
- Python rollback remains immediate and unchanged
- single-host rollback is simple: restore shim env and restart shim only

Cons:

- the shared token is mirrored in one additional VM env file

### Option B: Change the runtime to send the shim token

Change the NUC runtime so it injects the shim-specific token instead.

Pros:

- keeps the shim token as-is

Cons:

- requires a NUC-side secret change
- requires a `mcp-runtime.service` restart
- weakens rollback simplicity because Python currently expects the shared runtime/Python token
- increases the number of moving parts for a temporary canary

## Recommendation

Choose **Option A**.

It is the least risky path because it changes only the shim, preserves the live Python rollback path, and avoids any NUC-side service restart.

## Gate

Do not attempt the second gateway canary until the shim and the runtime share the same backend bearer token and a direct NUC -> shim test with the runtime token succeeds.

