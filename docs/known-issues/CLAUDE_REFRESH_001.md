# CLAUDE-REFRESH-001

## Title

Claude.ai connector refresh banner despite working tools

## Severity

minor

## Impact

low

## Symptom

- the refresh button displays an error banner
- tools remain visible and usable

## Proven state

- OAuth flow is healthy
- `tools/list` succeeds
- 12 tools are visible in the live native catalog
- tool calls succeed
- the native HTTP backend receives traffic
- no shim traffic was observed during validation
- no Python traffic was observed during validation

## Workaround

- reconnect or remove/re-add the connector if the banner blocks the UI flow
- continue using the tools if they remain visible and functional

## Suspect

- Claude.ai connector UI cache or permission state

## Status

- deferred investigation

## Release impact

- non-bloquant

## References

- [`docs/MCP_RUNTIME_REFRESH_DIAGNOSTIC.md`](/home/jm/Documents/hugo-mcp-go/docs/MCP_RUNTIME_REFRESH_DIAGNOSTIC.md)
- [`docs/CLAUDE_REFRESH_AND_PERMISSION_DIAGNOSTIC.md`](/home/jm/Documents/hugo-mcp-go/docs/CLAUDE_REFRESH_AND_PERMISSION_DIAGNOSTIC.md)
