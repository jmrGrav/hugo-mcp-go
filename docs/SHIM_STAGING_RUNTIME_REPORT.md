# SHIM Staging Runtime Report

## Runtime Summary

The shim was started on the VM staging address and successfully served the HTTP `/mcp` contract against the stdio `hugo-mcp-go` child.

## Verified HTTP Behavior on the VM

- missing bearer token -> `401 Unauthorized`
- invalid bearer token -> `401 Unauthorized`
- malformed JSON -> `400 Bad Request`
- `initialize` request -> `200 OK`
- `notifications/initialized` notification -> `202 Accepted`
- `tools/list` after initialization -> `200 OK`

## MCP Session Behavior

- the backend required the MCP session initialization sequence before `tools/list`
- `initialize` returned the expected `serverInfo` and `capabilities`
- `tools/list` returned the tool catalog from `hugo-mcp-go`
- the child bridge stayed live after the first successful request and reported `child_generation=1`

## Logging

- request logs were redacted
- no bearer token, raw body, or `/home/jm/...` path leaked in the observed shim logs
- log fields observed during validation were limited to status, latency, method, id type, child generation, and bytes in/out

## Stop Behavior

- the service was stopped after validation
- the shutdown path was corrected so a normal stop now deactivates successfully instead of leaving the unit in a failed state

## NUC Direct Validation

- direct validation from the NUC-equivalent host at `192.168.122.1` could not be completed from this session because SSH public-key authentication was denied
- no gateway or NUC configuration was modified to work around that

## Verdict

- shim responds locally on the VM: yes
- shim is safe to keep staging-only: yes
- gateway attachable now: no
