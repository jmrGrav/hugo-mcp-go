# NUC to Shim Connectivity Report

## Scope

Validate the real path:

`NUC -> hugo-mcp-shim:18180 -> hugo-mcp-go (stdio)`

without touching the live Python backend or the NUC gateway.

## Validation Sequence

1. confirm NUC reachability from the VM
2. start the shim staging service on the VM
3. open the VM firewall only for the NUC source IP and port `18180` during staging validation
4. exercise HTTP `/mcp` from the NUC
5. verify MCP initialize and tool listing
6. inspect journald and child process stability
7. stop only `hugo-mcp-shim.service`

## Results

- ping NUC -> VM: success
- TCP NUC -> shim port `18180`: success after temporary validation allowances
- HTTP no token: `401 Unauthorized`
- HTTP bad token: `401 Unauthorized`
- MCP `initialize`: `200 OK`
- MCP `notifications/initialized`: `202 Accepted`
- MCP `tools/list`: `200 OK`
- shim child process: stable, with `hugo-mcp-go` running under the shim cgroup

## Latency

- `initialize` response:
  - `time_connect`: about `0.00067s`
  - total: about `0.00287s`
- `tools/list` response:
  - `time_connect`: about `0.00070s`
  - total: about `0.00584s`

## Logs

- request logs showed `status=401`, `status=200`, and `status=202` at the expected points
- logs remained redacted
- no restart loop or crash loop was observed during the successful validation window

## Stop State

- the shim was stopped after validation
- Python remained active throughout

## Verdict

- NUC -> Shim TCP: yes
- NUC -> Shim HTTP: yes
- MCP initialize: yes
- MCP tools/list: yes
- Shim stable: yes
- Backend Go stable: yes
