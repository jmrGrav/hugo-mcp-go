# VM-Local Shim Validation Plan

## Purpose

Validate the VM-local shim against `hugo-mcp-go` without stopping the live Python service, without changing the NUC gateway, and without modifying production routing.

## Invariants

- Python remains live on VM port `8000`
- shim uses a different staging port
- NUC gateway remains unchanged
- `mcp-runtime.service` remains unchanged
- OpenResty, Nginx, and Cloudflare remain unchanged
- no cutover
- no deployment to `/home/jm/hugo-site`

## Pre-Validation Checks

Before starting the shim:

1. confirm the Python service is still active on `8000`
2. confirm the Go RC binary is installed and runnable
3. confirm the shim port is unused
4. confirm the staging tree is isolated from the live site tree
5. confirm the backend token is available only through the intended staging secret source

## Validation Phase 1: Local Shim Bring-Up

Goal: prove the shim can start and own the child stdio process.

1. start the shim on the VM staging port
2. confirm the child `hugo-mcp-go` process is spawned
3. confirm the MCP initialization handshake completes
4. confirm the shim reports healthy only after the child is ready

Expected result:

- shim is reachable on the staging port
- Python remains untouched on `8000`
- no gateway configuration changes are required

## Validation Phase 2: Manual VM Calls

Goal: prove the shim answers HTTP `/mcp` requests correctly from the VM.

1. call `POST /mcp` manually from the VM
2. include the staging bearer token
3. verify a read-only MCP request returns the expected JSON-RPC response shape
4. verify malformed requests are rejected with the expected HTTP status
5. verify oversized requests are rejected before child dispatch

Recommended manual check:

```bash
curl -sS -D - \
  -H "Authorization: Bearer $HUGO_MCP_SHIM_BACKEND_TOKEN" \
  -H "Content-Type: application/json" \
  --data '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}' \
  http://192.168.122.69:18180/mcp
```

## Validation Phase 3: Manual NUC Calls Without Gateway Changes

Goal: prove the NUC can reach the shim directly without editing the gateway.

1. from the NUC, call the VM shim endpoint directly
2. do not change `mcp-runtime.service`
3. do not point the gateway at the shim yet
4. compare the response shape to the existing Python backend for the same request

Only use direct calls for validation. The gateway remains on its current target during this phase.

## Validation Phase 4: Python vs Go Comparison

Goal: compare deterministic behavior only.

Compare the shim-backed Go result against the Python oracle for:

- `list_pages`
- `get_page`
- `list_assets`

Rules:

- compare only deterministic fields
- normalize timestamps and other known drift fields
- document any accepted differences explicitly
- do not treat non-deterministic ordering as a failure unless the contract requires it

## Validation Phase 5: Mutation Checks on Staging Only

Goal: prove that writes affect only the staging tree.

Run only against the staging tree:

- `create_page`
- `update_page`
- `delete_page`
- `upload_asset`
- `build_site`

Checks:

- no writes under `/home/jm/hugo-site`
- no writes outside the shim-owned staging tree
- rollback breadcrumbs are created where expected
- error paths are redacted

## Validation Phase 6: Log Review

Goal: prove the shim logs enough to operate, but not enough to leak secrets.

Check:

- request method
- request id
- elapsed time
- child restart count
- error class

Verify absence of:

- bearer token values
- raw request bodies
- file contents
- absolute paths from input

## Validation Phase 7: Restart and Failure Tests

Goal: prove child restart behavior.

Test cases:

1. kill the child only
2. confirm the shim restarts it
3. confirm in-flight requests fail cleanly
4. confirm the shim recovers after the new child initializes
5. kill the shim and confirm rollback is simply a stop

## Validation Exit Criteria

The validation passes if:

- the shim starts cleanly on the staging port
- direct VM and NUC calls succeed
- Python remains active on port `8000`
- deterministic read-only parity is acceptable
- staging mutations stay inside the staging tree
- logs are redacted
- child restart behavior is stable
- stop-the-shim rollback is immediate and clean

## Stop Condition

After validation:

1. stop the shim
2. confirm the staging port is closed
3. confirm Python is still running
4. leave the NUC gateway unchanged

