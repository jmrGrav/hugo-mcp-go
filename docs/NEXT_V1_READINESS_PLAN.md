# Next V1 Readiness Plan

Status: post-merge planning document for the work that remains before any `v1.0.0` decision.

## Purpose

The `feature/full-tool-parity-and-hooks` branch has been merged, the Go backend is live, and the Python rollback path remains available. This document records the next discrete checks and decisions needed before a formal `v1.0.0` release discussion.

## Current state

- Go runtime is live behind `mcp-runtime-go`.
- Python backend remains available for rollback.
- Tool parity blockers are fixed.
- Post-build hooks are integrated with dry-run defaults.
- Coverage is materially improved, but there is still room to harden non-critical paths.

## Remaining work before `v1.0.0`

1. Validate a live post-merge smoke run against the deployed runtime after any operational changes.
2. Decide whether hooks should be enabled progressively in production or remain dry-run until explicit operator approval.
3. Verify real secret file permissions on the VM before any live hook activation.
4. Confirm the production `HUGO_SITE_BASE_URL` and other environment values used by link generation and hook payloads.
5. Review `internal/hugo/*` coverage and decide whether additional tests are worthwhile before release.
6. Confirm whether Cloudflare purge, Google Indexing, and IndexNow should be enabled together or staged independently.
7. Decide the release policy for `v1.0.0` after the above checks are complete.

## Recommended sequence

1. Keep the current runtime live and rollback path intact.
2. Run a live smoke validation after any environment change.
3. Validate secret file ownership and permissions on the VM.
4. Enable hooks progressively, starting with dry-run or the lowest-risk provider.
5. Re-run the relevant smoke checks after each activation step.
6. Reassess release readiness only after the live operational state is stable.

## Non-goals

- No tag creation.
- No automatic release.
- No automatic live hook activation.
- No modification of `mcp-runtime-go`.
- No removal of the Python backend.
- No production runtime changes in this planning step.

## Decision gate

`v1.0.0` should remain blocked until:

- live validation passes on the deployed runtime,
- hook activation strategy is explicitly approved,
- secret file permissions are verified,
- and the operator confirms the final release policy.

