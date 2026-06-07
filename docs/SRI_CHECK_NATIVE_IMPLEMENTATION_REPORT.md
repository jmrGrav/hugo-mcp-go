# SRI Check Native Implementation Report

Date: 2026-06-07

## Summary

The `sri-check` plugin has been ported to native Go in `internal/sri/` and wired into the MCP tool `check_sri_versions`.

## Behavior Implemented

- scans Hugo roots for jsDelivr references
- compares package versions against jsDelivr latest tags
- verifies SRI hashes for tracked CDN URLs
- supports `auto_fix` and `dry_run`
- updates `themes/LoveIt/assets/data/cdn/jsdelivr.yml` and `data/sri.yaml` on autofix
- rebuilds through the injected Hugo build service
- triggers downstream hook orchestration when autofix is applied and hooks are enabled
- returns a structured JSON result compatible with the historical plugin shape

## Validation

- `go test ./internal/sri -count=1` passes
- `go test ./...` passes
- `go test -race ./...` passes
- `go vet ./...` passes
- `scripts/tool-parity-smoke.sh` passes against the local validation shim on `127.0.0.1:18182`
- `scripts/hooks-smoke.sh` passes

## Coverage

- `internal/sri`: `90.7%`
- coverage is concentrated in useful branches:
  - config parsing and normalization
  - version comparison
  - SRI verification
  - scan-root guards
  - autofix rollback and downstream trigger paths

## Remaining Differences

- the native implementation does not depend on the historical bash script at runtime
- the historical script remains available as audit reference only
- the remaining coverage gap is concentrated in defensive helper branches and broader repository packages

## Verdict

- plugin SRI Python remplacé: `yes`
- script bash encore requis: `no`
- prêt pour merge: `yes`
- prêt pour v1.0.0: `no`
