# Production Readiness Review

## Scope

This review answers a narrow question: what is still missing to move `hugo-mcp-go` from staging-ready to production-ready?

Out of scope:

- `mcp-runtime-go`
- the Python oracle
- production cutover
- any live production change

## Verdict

- **Production ready:** no
- **Staging ready:** yes

## Status Update

The release-candidate artifacts now exist in the repo:

- [`docs/TRANSPORT_CONTRACT.md`](/home/jm/Documents/hugo-mcp-go/docs/TRANSPORT_CONTRACT.md)
- [`docs/HUGO_BINARY_CONTRACT.md`](/home/jm/Documents/hugo-mcp-go/docs/HUGO_BINARY_CONTRACT.md)
- [`deploy/README.md`](/home/jm/Documents/hugo-mcp-go/deploy/README.md)
- [`deploy/systemd/hugo-mcp-go.service`](/home/jm/Documents/hugo-mcp-go/deploy/systemd/hugo-mcp-go.service)
- [`deploy/systemd/hugo-mcp-go.env.example`](/home/jm/Documents/hugo-mcp-go/deploy/systemd/hugo-mcp-go.env.example)
- [`deploy/validation/preflight.sh`](/home/jm/Documents/hugo-mcp-go/deploy/validation/preflight.sh)

This makes the repo reviewable as a release candidate, but not yet production-ready without external deployment execution and approval.

## What Is Already Ready

- read-only tools are implemented and parity-validated
- staging-only mutation tools are implemented and parity-validated
- security hardening has been applied and tested
- `go test ./...` and `go test -cover ./internal/...` are green
- shell execution is forbidden in the repo code
- filesystem writes are staged and anchored
- oracle fixtures exist for read-only and mutation parity

## Blockers for Production

### 1. Production transport contract must be executed in the target environment

The repository currently runs as a stdio MCP server. The docs recommend `mcp-runtime-go` as the gateway, with a production preference for a loopback HTTP backend, but the repo does not yet provide that production transport itself.

Impact:

- there is no production-ready attachment point for the gateway
- deployment topology is still a design decision, not a committed contract

Why this blocks production:

- a production backend must have a stable, documented transport contract with the gateway
- without that contract, the service can be staged but not safely handed to operations as a production dependency

### 2. Production packaging must be installed and validated on the target host

There is no committed, production-grade service package for:

- dedicated user and group
- file ownership and permissions
- `ReadWritePaths`
- `ProtectSystem`
- `PrivateTmp`
- `NoNewPrivileges`
- resource caps
- restart policy
- journald logging contract

Impact:

- operations do not yet have a single installable service definition
- deployment behavior is still inferred from docs rather than a release artifact

Why this blocks production:

- production readiness requires a repeatable install and startup contract, not just code that can run locally

### 3. Hugo binary execution must be validated on the target host

The build runner executes `hugo` through `exec.CommandContext` with PATH lookup.

Impact:

- production runtime correctness depends on environment PATH rather than an explicit binary contract
- a wrong or shadowed `hugo` binary can change build behavior

Why this blocks production:

- production needs a pinned, deterministic build toolchain or an explicit service-level PATH contract that is validated in deployment

## Accepted Risks

- residual read-before-write windows before the final anchored write/delete hop
- large list scans can still traverse large trees
- no app-level readiness endpoint exists yet

## Not Acceptable For Production

- any implicit gateway attachment
- any undeclared PATH dependency for the build binary
- any deployment that does not pin root paths and service identity

## Minimal Next Step

Before production can be approved, freeze the deployment topology and install contract in a non-active systemd package or equivalent release artifact, and decide the gateway attachment mode definitively.
