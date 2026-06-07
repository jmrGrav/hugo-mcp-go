# CLAUDE-REFRESH-001 Reproduction

## Prerequisites

- a configured Claude.ai connector for Hugo MCP
- the native Hugo MCP backend is active and healthy
- no assumption that the refresh banner means the backend is down

## Steps

1. Open the Hugo MCP connector in Claude.ai.
2. Confirm the tool catalog loads and remains usable.
3. Click the refresh button in the connector panel.
4. Observe whether Claude shows `Impossible de recharger les outils depuis le serveur`.
5. Keep the connector open and verify that the tools are still present and callable.

## Expected result

- the refresh completes without any banner
- tool permissions remain stable

## Observed result

- the refresh banner can appear even when the tools remain visible and usable
- the live backend still receives successful calls
- the connector can keep working after the banner appears

## Server-side logs to check

- gateway discovery and OAuth logs
- proxy hits for `/mcp`
- backend native HTTP logs for `initialize`, `notifications/initialized`, and `tools/list`

## What not to conclude without logs

- do not conclude the backend failed just because the banner appeared
- do not conclude the gateway is broken without matching proxy or discovery errors
- do not conclude IP blocking without an explicit deny line

## Sanitized captures

- [`docs/assets/screenshots/claude-refresh-banner.png`](../assets/screenshots/claude-refresh-banner.png)
- [`docs/assets/screenshots/claude-permissions.png`](../assets/screenshots/claude-permissions.png)

## Notes

- the tools remain usable while this issue is present
- the suspected cause remains client-side refresh/cache or connector permission state
