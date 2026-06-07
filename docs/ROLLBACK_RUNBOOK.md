# Rollback Runbook

## Purpose

Restore the previous production state after a failed deployment or bad release of `hugo-mcp-go`.

## What Rollback Must Restore

- the previous `hugo-mcp-go` binary
- the previous environment file
- the previous service unit if it changed
- the previous gateway attachment target if it changed

## What Rollback Must Not Touch

- production content trees outside the approved deployment roots
- the Python oracle
- `mcp-runtime-go` source

## Preconditions

- previous binary is still available
- previous config is still available
- rollback instructions were rehearsed on a staging copy
- service identity and paths are known

## Fast Rollback

1. stop `hugo-mcp-go`
2. replace the current binary with the previous binary
3. restore the previous env file
4. restore the previous service unit if applicable
5. start the service
6. verify the process starts cleanly and logs redact sensitive values

## Data Rollback

If the release wrote unexpected data into staging-like roots:

1. stop the service
2. inspect the `work/` rollback breadcrumb directory
3. restore the original file from the breadcrumb if the breadcrumb exists
4. verify the restored file matches the last known-good snapshot
5. only then restart the service

## Failure Modes

- if the previous binary is missing, do not improvise a fresh build as rollback
- if the env file is missing, restore it from the last known good package
- if rollback cannot be completed quickly, keep the service stopped and escalate

## Rollback Success Criteria

- service starts with the previous binary
- gateway attachment is healthy
- no writes escape the approved roots
- logs remain redacted

## When Rollback Is Not Enough

If the bad release caused irreversible data changes outside the staged roots, stop and declare the deployment unrecoverable through normal rollback. Do not attempt ad hoc repair inside production without a separate incident process.
