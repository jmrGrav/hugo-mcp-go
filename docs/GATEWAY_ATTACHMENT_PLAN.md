# Gateway Attachment Plan

## Goal

Define the future attachment path between the NUC gateway and `hugo-mcp-go` without applying any change now.

## Constraints

- `hugo-mcp.service` on the VM stays alive until Go has been proven
- `mcp-runtime.service` on the NUC stays unchanged
- OpenResty/Nginx stay unchanged
- Cloudflare stays unchanged
- no cutover now

## Recommended Attachment Strategy

### Preferred short-term path

Use a VM-local adapter/shim that keeps the current HTTP gateway contract intact while the Go backend remains stdio.

This preserves:

- the NUC gateway process
- the current private VM endpoint shape
- the ability to keep Python live until Go parity is proven

### Long-term steady-state path

Implement native HTTP loopback support in `hugo-mcp-go` and retire the shim later.

This is the cleaner steady-state, but it is a code change and not required for the first safe attachment test.

## Attachment Decision Tree

### Option 1: Shim first

Choose this if the goal is to minimize risk and keep the gateway contract unchanged.

- VM-local shim exposes `/mcp`
- shim spawns or manages `hugo-mcp-go` locally
- NUC `mcp-runtime-go` continues to point to the VM backend URL

### Option 2: Native HTTP in `hugo-mcp-go`

Choose this if the goal is to reduce long-term complexity and remove the shim.

- `hugo-mcp-go` gains a loopback HTTP listener
- NUC gateway continues to use HTTP(S)
- proxying and health checks stay familiar

### Option 3: NUC-side subprocess ownership

Do not choose this for the current topology.

- the backend lives on the VM
- the gateway lives on the NUC
- local subprocess ownership would require moving the backend host or redesigning the gateway

## Validation Before Any Attachment

Before the gateway is changed:

1. run `hugo-mcp-go` in staging on the VM
2. compare `list_pages`, `get_page`, and `list_assets` outputs against the Python oracle
3. validate mutation side effects only on the staging tree
4. prove rollback on the staging tree
5. verify logs are redacted
6. verify no write escapes `/var/lib/hugo-mcp-go`

## Attachment Steps When Ready

### If shim path is chosen

1. deploy the shim on the VM
2. bind it only to loopback or the private VM address
3. keep the Python service live until parity passes
4. point the gateway backend URL at the shim only after staging parity is confirmed
5. maintain rollback to the Python service until the new path is stable

### If native HTTP path is chosen

1. add loopback HTTP to `hugo-mcp-go`
2. validate local staging
3. expose only loopback on the VM
4. update the gateway backend URL only after parity and rollback are proven

## Recommendation

For the current environment, prefer:

1. VM-local shim for the first safe attachment
2. native HTTP in `hugo-mcp-go` as the later cleanup step

This minimizes risk because it avoids any immediate gateway change while preserving the ability to test the Go backend against the live site clone.

## Current Staging Status

- the VM-local shim is installed and has passed local VM validation on the private bind address
- the shim currently remains staging-only and disabled after validation
- direct NUC-side validation was attempted but not completed from this session because SSH access to the reachable host candidate at `192.168.122.1` was denied
- gateway attachment remains deferred until that direct validation gap is closed and the staging evidence is complete
