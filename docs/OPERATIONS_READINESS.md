# Operations Readiness

## Scope

This document describes the operating procedures for `hugo-mcp-go` as a release candidate.

## Start

1. run `deploy/validation/preflight.sh`
2. confirm the service user and group are correct
3. confirm the Hugo binary resolves to the approved path
4. confirm content/static roots are canonical and non-symlinked
5. start the approved systemd unit
6. confirm logs reach journald

## Stop

1. stop the systemd unit cleanly
2. confirm the process exits
3. inspect journald for shutdown errors
4. preserve any `work/` rollback breadcrumbs if a mutation was in progress

## Upgrade

1. install the new binary alongside the old one
2. validate the new binary hash and version
3. run the preflight script
4. restart the service
5. verify the gateway reconnects
6. compare the first post-start logs for redaction and error shape

## Rollback

1. stop the service
2. restore the previous binary
3. restore the previous env file
4. restore the previous unit file if it changed
5. rerun the preflight script
6. start the service again

## Supervision

- systemd restart policy handles process crashes
- journald is the primary log source
- there is no app-level readiness endpoint in the current repo
- operational readiness is inferred from preflight plus clean startup

## Common Incidents

### Hugo binary missing

- preflight fails
- service must not start

### Hugo version mismatch

- preflight fails
- operator must approve the version before retrying

### Wrong content/static root

- preflight fails
- correct the env file and retry

### Mutation failure

- inspect journald
- inspect the `work/` rollback breadcrumb
- replay only after the underlying filesystem issue is fixed

## Recovery After Crash

1. restart the service through systemd
2. if the crash occurred during a mutation, inspect `work/`
3. confirm no write escaped the approved roots
4. run a smoke test against read-only tools

## Post-Deployment Validation

- `list_pages` returns expected content
- `get_page` returns the expected page record
- `list_assets` returns deterministic normalized fields
- `create_page` and `delete_page` operate only in staging paths
- `build_site` uses the approved Hugo binary

## Observability

- journald is required
- stderr redaction must remain enabled
- no dedicated metrics endpoint exists yet
- alerting should key off service restarts, build failures, and unexpected mutation errors
