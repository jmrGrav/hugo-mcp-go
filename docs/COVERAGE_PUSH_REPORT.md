# Coverage Push Report

Date: 2026-06-07

## Cycle 1: Restore the missing MCP tools

- Coverage before:
  - `internal/tools`: `80.5%`
  - `internal/tools/parity_tools.go::generateFeaturedImage`: `68.4%`
- Coverage after:
  - `internal/tools`: `93.0%`
  - `internal/tools/parity_tools.go::generateFeaturedImage`: `87.3%`
- Functions targeted:
  - `checkSriVersions`
  - `generateFeaturedImage`
  - `featuredImageTargetLangs`
  - `stagingFromDeps`
  - `safeFeaturedImagePath`
  - `renderFeaturedImage`
  - `featuredImagePalette`
  - `drawCenteredText`
  - `wrapText`
  - `blendColor`
  - `colorFromHex`
  - `hexToBytes`
- New tests:
  - `internal/tools/parity_tools_test.go`
  - `internal/tools/parity_tools_internal_test.go`
- Justification:
  - The new tools were missing and the smoke path still reported them as unsupported.
  - The tests focus on real inputs, page/frontmatter updates, safe path resolution, and deterministic error branches.

## Cycle 2: Harden the hooks subsystem

- Coverage before:
  - `internal/hooks`: `88.7%`
  - `internal/shim`: `90.0%`
- Coverage after:
  - `internal/hooks`: `90.0%`
  - `internal/shim`: `90.5%`
- Functions targeted:
  - `LoadConfigFromEnv`
  - `LoadSecretFile`
  - `OpenStore`
  - `HasColumn`
  - `Enqueue`
  - `ListJobs`
  - `SetJobStatus`
  - `RecordAudit`
  - `JobCount`
  - `AuditMessages`
  - `createSchema`
  - `newID`
  - `PurgeURLs`
  - `Publish`
  - `Submit`
  - `Process`
  - `childEnv`
  - `Config.Validate`
- New tests:
  - `internal/hooks/hardening_test.go`
  - `internal/shim/config_test.go`
  - `internal/shim/child_test.go`
- Justification:
  - `internal/hooks` was still the last merge blocker because it sat below the minimum coverage target.
  - The new tests exercise real provider success/error branches, secret permission hardening, nil-store behavior, and schema/ID fallback logic.
  - The shim PATH override is testability-only and defaults to the original production-safe behavior.

## Observations

- The coverage gains are real and tied to the new tool behavior.
- The remaining uncovered code in `internal/tools/parity_tools.go` is mostly in the image-generation helper and a few defensive branches.
- Additional gains are still possible, but each remaining point now requires increasingly specific branch coverage rather than broad missing functionality.

## Current Snapshot

- Global coverage: `80.5%`
- `internal/tools`: `92.7%`
- `internal/hooks`: `90.0%`
- `internal/shim`: `90.5%`
- `internal/runner`: `95.5%`

## Coverage Table

| Package | Coverage |
| ------- | -------- |
| `cmd/hugo-mcp-go` | `0.0%` |
| `cmd/hugo-mcp-shim` | `0.0%` |
| `internal/config` | `60.9%` |
| `internal/hugo/assets` | `67.7%` |
| `internal/hugo/frontmatter` | `68.0%` |
| `internal/hugo/mutations` | `71.0%` |
| `internal/hugo/pages` | `72.4%` |
| `internal/hugo/staging` | `67.6%` |
| `internal/observability` | `75.0%` |
| `internal/runner` | `95.5%` |
| `internal/security/pathguard` | `66.3%` |
| `internal/server` | `69.4%` |
| `internal/shim` | `90.5%` |
| `internal/tools` | `92.7%` |

## Judgment

- More useful coverage is still possible in `internal/tools`, especially around the remaining defensive branches.
- `internal/hooks` has reached the target floor, but more branch coverage is still possible if future hook providers are added.
- The gains so far are not artificial and exercised real functionality.
- The next meaningful cycle should target the remaining `internal/hugo/*` packages or the docs/operational surface, not more synthetic hook branches.
