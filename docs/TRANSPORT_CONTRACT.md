# Transport Contract

## Scope

This document compares the two transport shapes discussed for `hugo-mcp-go` behind `mcp-runtime-go`.

- no implementation changes are made here
- no protocol changes are introduced
- `mcp-runtime-go` remains unchanged

## Option A: stdio subprocess behind `mcp-runtime-go`

### Advantages

- matches the current `hugo-mcp-go` executable shape
- avoids adding a new network listener
- keeps the attack surface smaller than an HTTP backend
- simplifies rollback to a binary replacement
- keeps the dependency chain short: gateway spawns backend

### Inconvenients

- weaker independent observability than a networked backend
- readiness/health are process-level, not HTTP-level
- build/runtime failures are only visible through the gateway and journald
- no natural external health endpoint for separate probes

### Security

- smallest transport attack surface of the two options
- no open backend port
- fewer protocol translation layers

### Observability

- process logs only unless the gateway adds extra telemetry
- no backend health endpoint
- error visibility depends on gateway and journald

### Maintenance

- easiest to maintain with the current codebase
- fewer moving parts than a loopback HTTP service

### Upgrade

- upgrade is a binary replacement plus restart of the gateway-managed backend process
- simple rollback to previous binary

### Rollback

- rollback is a binary/config revert
- no network routing change is needed

## Option B: loopback HTTP backend behind `mcp-runtime-go`

### Advantages

- better backend observability and independent readiness semantics
- easier to add dedicated health checks
- easier to separate gateway and backend lifecycle at the process boundary
- future-friendly if the backend needs its own HTTP-level control plane

### Inconvenients

- requires an additional backend transport implementation
- adds a listener surface, even if only on loopback
- adds reverse-proxy and HTTP lifecycle complexity
- not currently implemented in this repo

### Security

- more surface area than stdio
- local listener still needs hardening and binding discipline

### Observability

- better than stdio
- can support explicit readiness/health and access logging

### Maintenance

- higher operational complexity
- more code paths and more integration failure modes

### Upgrade

- easier to restart independently if implemented
- but requires stable HTTP contract and routing discipline

### Rollback

- rollback can be a routing switch if the gateway supports it
- but only after the HTTP backend exists

## Comparison Summary

| Criterion | stdio subprocess | loopback HTTP |
|---|---|---|
| Attack surface | smaller | larger |
| Current repo fit | direct fit | not implemented |
| Observability | weaker | stronger |
| Operational simplicity | stronger | weaker |
| Independent health checks | weaker | stronger |
| Upgrade complexity | lower | higher |
| Rollback simplicity | high | high, once implemented |

## Recommendation

**Recommended production transport for the current release candidate: stdio subprocess behind `mcp-runtime-go`.**

### Why

- it matches the current executable and test harness without requiring an additional transport implementation
- it keeps the transport surface minimal for the current release
- it allows the release candidate to be reviewed operationally without introducing a new network service

### Why not loopback HTTP for this RC

- the repo does not implement HTTP transport
- introducing HTTP now would change scope and delay the release candidate without improving the current code path

## Final Contract

- the backend is a dedicated `hugo-mcp-go` binary
- the gateway owns attachment and process supervision
- the backend remains stdio-based in the current release candidate
- loopback HTTP is deferred to a future transport redesign if the operations team needs it
