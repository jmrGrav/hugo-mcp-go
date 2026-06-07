# Deployment Package

This directory contains review-only deployment artifacts for `hugo-mcp-go`.

These files are not installed automatically and do not change the running system.

## Contents

- `systemd/hugo-mcp-go.service` - example hardened service unit for operations review
- `systemd/hugo-mcp-go.env.example` - example environment file with non-secret values
- `validation/preflight.sh` - non-destructive validation script for deployment-time checks

## Intended Use

1. review the transport contract in [`../docs/TRANSPORT_CONTRACT.md`](../docs/TRANSPORT_CONTRACT.md)
2. review the Hugo binary contract in [`../docs/HUGO_BINARY_CONTRACT.md`](../docs/HUGO_BINARY_CONTRACT.md)
3. review the service and env examples
4. run `validation/preflight.sh` against a staging or clone environment
5. install only after operations approval

## Non-Goals

- no service is enabled by these files
- no system state is changed by these files
- no production cutover is implied
- no gateway changes are made here

## Release Note

The package is deliberately conservative:

- dedicated user/group
- journald logging
- strict filesystem write boundaries
- no shell execution
- no dynamic plugin discovery
