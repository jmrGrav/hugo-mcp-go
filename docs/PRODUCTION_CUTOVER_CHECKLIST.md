# Production Cutover Checklist

## Pre-cutover

- [ ] `go test ./...` passes
- [ ] `go test -cover ./internal/...` passes
- [ ] production deployment package reviewed and approved
- [ ] systemd unit or equivalent release artifact reviewed and approved
- [ ] gateway attachment mode approved
- [ ] exact `hugo` binary path or PATH contract approved
- [ ] rollback binary and rollback config are available
- [ ] staging clone validation passed on a copy of the real site

## Site-Clone Validation

On a copy of the real site only:

- [ ] `list_pages` matches expected section/lang slices
- [ ] `get_page` matches the expected fallback behavior
- [ ] `list_assets` matches normalized deterministic fields
- [ ] `create_page` creates the expected page file
- [ ] `update_page` preserves immutable field rules
- [ ] `delete_page` leaves and records rollback artifacts in `work/`
- [ ] `upload_asset` writes only to the expected static path
- [ ] `build_site` succeeds with the pinned `hugo` binary

## Cutover Steps

1. install the approved binary
2. install the approved systemd unit or equivalent supervisor config
3. install the approved env file
4. start the service in staging clone
5. verify logs, limits, and error redaction
6. switch gateway attachment only after the staging clone matches

## Go / No-Go Criteria

### Go

- deployment is deterministic
- transport contract is frozen
- rollback is rehearsed
- staging clone matches oracle expectations

### No-Go

- transport contract is still ambiguous
- `hugo` binary path is still implicit and uncontrolled
- rollback cannot be executed in a bounded time
- any unexpected write outside the allowed roots is observed

## Post-cutover Watch

- monitor journald for startup and build failures
- verify redaction in error paths
- verify the gateway can reconnect after a process restart
- verify mutation paths stay confined to the intended roots
