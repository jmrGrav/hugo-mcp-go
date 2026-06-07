# Govulncheck Triage

## Summary

The failing `govulncheck ./...` run was caused by the Go 1.25.0 toolchain, not by a need to remove the vulnerability scan.
The correct fix is to validate with a patched toolchain.

## Classification

### Stdlib vulnerabilities reachable by current code

These were the 20 findings reported against Go 1.25.0.
The traces show reachable paths through:

- `internal/shim/server.go`
- `internal/hugo/mutations/pages.go`
- `internal/hugo/pages/pages.go`
- `internal/tools/tools.go`
- `cmd/hugo-mcp-go/main.go`

Affected stdlib areas:

- `crypto/tls`
- `crypto/x509`
- `encoding/asn1`
- `encoding/pem`
- `net`
- `net/http`
- `net/textproto`
- `net/url`
- `os`

### Stdlib vulnerabilities non-reachable

No separate non-reachable stdlib findings were shown in the failing symbol report.
The patched run on Go 1.25.11 reported no code vulnerabilities.

### Modules direct

No direct third-party module vulnerability was reported as reachable by our code in the failing run.

### Modules indirect

The patched Go 1.25.11 run still reported one module vulnerability in the dependency graph:

- `golang.org/x/sys@v0.41.0`
- vulnerability: `GO-2026-5024`
- platform: Windows only
- reachable by code: no, under the current Linux build path

### Toolchain-caused findings

All 20 failing stdlib findings were caused by the toolchain being too old.
Each finding was fixed in a later Go 1.25 patch release, and they disappeared when `GOTOOLCHAIN=go1.25.11` was used.

## Recommendation

- Force Go 1.25.11 or newer during release validation
- Keep the vulnerability scan
- Do not add a global allowlist
- Do not suppress the module vulnerability without a platform-specific justification

