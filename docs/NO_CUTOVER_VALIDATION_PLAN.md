# No Cutover Validation Plan

## Purpose

Validate `hugo-mcp-go` against the live Hugo content tree without changing routing, gateway configuration, or the live Python service.

## Invariants

- Python service stays active
- Go RC stays isolated on the VM staging layout until explicitly started for validation
- NUC gateway stays unchanged
- no routing changes
- no Cloudflare changes
- no Nginx changes
- no production cutover

## Safe Validation Sequence

### Phase 1: Local Go staging start on the VM

1. start `hugo-mcp-go` only in a staging context on the VM
2. do not attach it to the NUC gateway
3. keep the Python service live

### Phase 2: Read-only parity

1. call `list_pages`
2. call `get_page`
3. call `list_assets`
4. compare deterministic fields against the Python oracle fixtures
5. normalize mtimes and other non-deterministic fields

### Phase 3: Mutation parity on the staging tree only

1. call `create_page`
2. call `update_page`
3. call `delete_page`
4. call `upload_asset`
5. check only `/var/lib/hugo-mcp-go`
6. verify rollback breadcrumbs in `work/`
7. verify nothing writes into `/home/jm/hugo-site`

### Phase 4: Build validation

1. run `build_site` only against the staging tree
2. use the approved Hugo binary path
3. verify redacted logs
4. verify timeout behavior and runner failure shape

### Phase 5: Optional local-only HTTP bridge test

If a shim or native HTTP mode exists:

1. bind it only to loopback or the private VM address
2. call it manually from the VM
3. do not point the NUC gateway at it yet

### Phase 6: Compare against Python

1. keep the Python backend as the current reference
2. compare only the deterministic fields
3. document accepted drift explicitly
4. do not change live routing

## Manual Checks

- verify current live Python service still responds
- verify Go staging responds only on the staging path
- verify logs do not leak tokens or host-sensitive values
- verify the NUC gateway still points at the existing backend

## Exit Criteria

The validation is successful if:

- Go staging passes read-only parity
- Go staging passes mutation parity on the staging tree
- build validation succeeds with the approved Hugo binary
- no writes escape the staging roots
- the Python live service remains untouched
- the NUC gateway remains untouched

## What This Plan Does Not Do

- no gateway routing change
- no service disable/stop
- no cutover
- no production install change

