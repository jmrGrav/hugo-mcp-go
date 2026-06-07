# Security Threat Model

## Scope

This threat model covers `hugo-mcp-go` only.

- `mcp-runtime-go` is out of scope and unchanged
- the Python oracle is out of scope and unchanged
- production cutover is out of scope and forbidden

## Assets

- Hugo staging content tree
- Hugo staging static tree
- Hugo staging public tree
- Hugo staging work tree
- read-only parity fixtures
- mutation parity fixtures
- MCP tool inputs and outputs
- build runner execution boundary
- redacted logs and error messages

## Trust Boundaries

- MCP caller is hostile by default
- filesystem contents under staging roots may be adversarial
- build output is untrusted until normalized
- oracle fixtures are trusted only as behavioral snapshots, not as runtime state
- shell execution is not permitted
- no direct dependency boundary exists to `mcp-runtime-go`

## Attacker Capabilities

Assume an attacker can:

- call any exposed MCP tool repeatedly
- send malformed JSON/MCP payloads
- supply long strings, nested structures, and invalid encodings
- choose paths, filenames, and section selectors within the schema
- create large content or asset payloads within request limits
- attempt traversal, symlink abuse, and rollback abuse through staging inputs
- trigger build failures and inspect returned error text

## Attack Surface

### MCP

- malformed tool arguments
- unknown fields
- unknown tools
- invalid types
- oversized payloads
- repeated mutation requests

### Filesystem

- path traversal
- symlink escape
- directory replacement races
- rollback breadcrumb handling
- temp file allocation
- massive tree scans

### Runner

- build command execution
- stderr/stdout leakage
- timeout handling
- PATH-based executable lookup

### Staging

- staged content, static, public, and work roots
- write confinement within allowlisted roots
- rollback restoration
- staging configuration miswiring

### Oracle Abuse

- fixture drift masking behavioral changes
- using oracle snapshots to justify unsafe normalizations
- replaying malicious payloads from captured error fixtures

## Threats and Assessment

| Threat | Impact | Exploitability | Likelihood |
|---|---:|---:|---:|
| Symlink section scan in `list_pages` | High | Medium | Medium |
| Unbounded page body reads/writes | High | High | Medium |
| Large staging tree scans | Medium | Medium | Medium |
| Runner stderr leakage | Medium | Low | Low |
| MCP malformed payload rejection gaps | High | Low | Low |
| Rollback breadcrumb abuse | Medium | Low | Low |

## Key Controls

- canonical root validation at startup
- symlink rejection on existing and new targets
- dirfd/openat-style final mutation anchoring
- bounded upload payload decoding
- bounded build execution with injected runner and timeout
- redacted runner output
- explicit staging isolation
- schema-validated MCP inputs

## Residual Risks

- read-before-write race windows remain before the final anchored write/delete hop
- list operations can still traverse large trees
- build execution still uses PATH lookup for the `hugo` binary
- oracle fixtures do not model hostile filesystem state exhaustively

## Security Posture

- staging-only shadow execution: acceptable with the current controls
- production cutover: not acceptable
