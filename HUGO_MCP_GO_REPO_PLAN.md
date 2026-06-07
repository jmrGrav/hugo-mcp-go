# hugo-mcp-go Repository Plan

> **Scope:** create a dedicated Go repo for the Hugo MCP backend, using the Python service as a behavior oracle, without translating the Python code line by line.

**Goal:** deliver a standalone `hugo-mcp-go` server that implements the Hugo tool surface in Go with explicit filesystem boundaries, deterministic parity fixtures, and a clean handoff to `mcp-runtime-go` as the generic OAuth/MCP gateway.

**Architecture:** keep `mcp-runtime-go` as the OAuth gateway and reverse proxy. Put all Hugo behavior in a separate repo with idiomatic Go package boundaries: thin MCP server/adapters on top, pure Hugo domain packages underneath, and explicit runner/path-guard/config layers around the edges. The Python service remains the oracle for tool contracts, side effects, and error shapes.

**Tech Stack:** Go 1.25.x, `github.com/modelcontextprotocol/go-sdk v1.6.1`, stdlib `net/http`, `os/exec`, `path/filepath`, `gopkg.in/yaml.v3` or equivalent YAML handling, `slog`, `testing`, `testdata` fixtures.

---

## 1. Decision Summary

### Recommended final architecture

- Create a new repo: `hugo-mcp-go`.
- Keep `mcp-runtime-go` unchanged as the generic OAuth/MCP gateway.
- Run `hugo-mcp-go` as the specialized Hugo backend behind `mcp-runtime-go`.
- Do not import `internal/` packages from `mcp-runtime-go`.
- Do not embed Hugo into the runtime proxy package tree.
- Do not recreate Python plugin discovery or shell-driven behavior.

### Why this is the right split

- It preserves the boundary that already exists in production: gateway vs backend.
- It keeps Hugo domain logic isolated from OAuth, token storage, and proxy concerns.
- It makes parity work easier because the backend can be tested and shadowed independently.
- It reduces the blast radius of future Hugo changes.
- It avoids turning `mcp-runtime-go` into a mixed auth/content/runtime monolith.

### Top-level recommendation

Use the official/maintained MCP Go SDK already present in the workspace:

- Module: `github.com/modelcontextprotocol/go-sdk`
- Version observed locally in a sibling Go MCP server: `v1.6.1`

This is the strongest local evidence-based choice because:

- it is already used in `security-automation-go`,
- it gives explicit tool registration and transport abstractions,
- it avoids a homegrown protocol implementation,
- it matches the requested architecture better than a bespoke MCP stack.

### Unknowns that must be kept explicit

- The exact transport shape for `hugo-mcp-go` in production still needs confirmation:
  - HTTP/loopback backend is the recommended production shape for `mcp-runtime-go` proxying.
  - stdio is useful for local/dev and SDK validation, but it is not the natural production backend for the current gateway topology.
- Exact Python response schemas still need oracle capture.
- The precise parity corpus and fixture set are not yet frozen.

---

## 2. Brooks Review

### Findings confirmed

1. `mcp-runtime-go` does not currently include the MCP SDK dependency in its own `go.mod`.
2. `mcp-runtime-go` is already structured as a gateway/proxy runtime, not as a Hugo content engine.
3. A sibling Go project in the workspace already uses `github.com/modelcontextprotocol/go-sdk v1.6.1` with explicit tool registration and stdio transport.
4. The Python `hugo-mcp` service is behaviorally richer but structurally riskier:
   - dynamic plugin discovery,
   - shell-backed integrations,
   - local code execution via `exec(open(...))`,
   - hard-coded filesystem roots.
5. The Go port should not preserve those structural weaknesses.

### Hypotheses

1. The cleanest production transport for `hugo-mcp-go` is HTTP on loopback, fronted by `mcp-runtime-go`.
2. Tool schemas can be frozen from Python oracle responses without copying Python internals.
3. The read-only tool surface can be made parity-stable before any mutating path is enabled.

### Probable risks

1. Overfitting the Go repo to the Python service shape would reintroduce Python-era coupling.
2. A direct port of the build/deploy hooks could reintroduce shell-injection and secret-leak issues.
3. Symlink escape will remain a risk unless path resolution is checked at the write boundary, not only at input parsing.
4. The transport decision can become over-engineered if it is delayed too long; it must be settled before implementation starts.

### Decisions recommended

1. Keep `hugo-mcp-go` separate from `mcp-runtime-go`.
2. Use the MCP Go SDK, not a custom protocol layer.
3. Start with read-only tools only.
4. Freeze behavior from Python fixtures, not from line-by-line translation.
5. Make filesystem and command execution injectable and bounded.

### Open questions

1. Should production backend transport be HTTP/loopback only, or should stdio be preserved as a first-class dev transport?
2. What exact Python response fields are deterministic enough to freeze as parity fixtures?
3. Which mutation tools need staging-only gating beyond the generic allowlist/root checks?

---

## 3. SDK Selection

### Recommended SDK

- `github.com/modelcontextprotocol/go-sdk v1.6.1`

### Why this module

- Already present in the workspace in `security-automation-go/go.mod`.
- Already used successfully for a Go MCP server.
- Provides explicit `mcp.Server` construction and tool registration.
- Keeps the repo aligned with a maintained SDK rather than a handwritten protocol shim.

### What is confirmed locally

- `security-automation-go/cmd/security-automation-mcp/main.go` uses:
  - `mcp.NewServer(...)`
  - `mcp.StdioTransport{}`
  - explicit tool registration through SDK helpers.
- That makes it a good local reference for server wiring style.

### What is not yet confirmed

- The exact transport variant to prefer for `hugo-mcp-go` in production.
- Whether the SDK version should remain exactly `v1.6.1` or be bumped at implementation time.

### Recommendation on version pinning

- Pin `github.com/modelcontextprotocol/go-sdk v1.6.1` initially.
- Re-evaluate only if:
  - the SDK lacks the required transport,
  - the SDK lacks a needed MCP primitive,
  - or an implementation-time compatibility issue appears.

---

## 4. Proposed Repo Architecture

### Recommended layout

- `cmd/hugo-mcp-go/`
  - process entrypoint, config bootstrapping, transport selection, shutdown.
- `internal/server/`
  - MCP server construction, tool registration, transport wiring.
- `internal/tools/`
  - tool handlers and request/response adapters.
- `internal/hugo/pages/`
  - page discovery, read/write logic, path mapping.
- `internal/hugo/assets/`
  - asset discovery and upload logic.
- `internal/hugo/build/`
  - build execution wrapper, status capture, log normalization.
- `internal/hugo/frontmatter/`
  - YAML parse/merge/serialize, field filtering, deterministic normalization.
- `internal/security/pathguard/`
  - root allowlists, symlink escape checks, path normalization.
- `internal/runner/`
  - bounded command execution, timeouts, environment control, output capture.
- `internal/config/`
  - root paths, limits, build settings, staging roots, backend transport, feature flags.
- `internal/observability/`
  - structured logging, redaction helpers, counters, request correlation.
- `testdata/fixtures/`
  - minimal Hugo site, oracle captures, golden responses, staging samples.

### Architectural boundary rules

- `internal/tools` knows the MCP schemas.
- `internal/hugo/*` knows Hugo domain rules.
- `internal/security/pathguard` knows only path safety.
- `internal/runner` knows only process execution.
- `internal/server` knows only SDK/server wiring.
- Nothing in `internal/hugo/*` should depend on MCP types.

### Recommended package split challenge to the user example

- Keep `internal/server/` and `internal/tools/` as thin shells.
- Do not put business logic there.
- Keep the pure domain inside `internal/hugo/*`.
- Keep command execution separate from content manipulation.

---

## 5. Tool Matrix: Python -> Go

### Legend

- **Phase 1** = read-only core.
- **Phase 2** = mutations and build side effects.
- **Deferred** = explicitly postponed until the core is stable.
- **Out-of-scope** = not planned for this repo.

| Tool | Status | Input schema | Output schema | Effects | Security risks | Parity tests |
|---|---|---|---|---|---|---|
| `list_pages` | Phase 1 | Read-only selectors for section/language/draft scope; exact Python fields must be frozen from oracle captures. | Deterministic page summary list: path, kind, title, language, section, route, and other stable metadata captured from oracle. | Filesystem read only. | Path traversal via section filters; symlink escape while scanning. | Golden response parity on minimal fixtures; traversal rejection tests; symlink scan tests. |
| `get_page` | Phase 1 | Page locator (path/slug/route) plus optional language selector; exact Python field names to be captured. | Single page record with frontmatter, content body, and stable metadata. | Filesystem read only. | Reading outside allowed roots; symlink dereference if path resolution is weak. | Golden parity on leaf bundle, branch bundle, and `_index` cases. |
| `create_page` | Phase 2 | Page target, content body, frontmatter payload, layout/type data, and optional draft/publish semantics as captured from Python. | Created page record plus deterministic status metadata and path. | Writes content tree; may trigger build/deploy wrapper. | Path traversal, symlink escape, accidental overwrite, secret leakage in logs. | Mutation parity on temp tree; overwrite rejection tests; write-root tests. |
| `update_page` | Phase 2 | Page locator plus partial update payload and merge semantics. | Updated page record plus deterministic status metadata. | Writes content tree; may trigger build/deploy wrapper. | Same as create plus merge bugs and null/empty-field ambiguity. | Deep-merge parity tests; field-delete semantics; idempotence tests. |
| `delete_page` | Phase 2 | Page locator and deletion intent. | Confirmation/status payload with removed path and stable metadata. | Deletes content tree entry; may trigger build/deploy wrapper. | Destructive write outside allowlist; symlink-target deletion. | Temp-tree delete parity; safety rejection tests; non-existent page errors. |
| `build_site` | Phase 2 | Build options, if any, including output/refresh toggles. | Build status, timing, captured stderr/stdout slice, and post-build summary. | Executes Hugo build and any explicit wrapper logic. | Shell injection, excessive log capture, false success on partial failure. | Build parity on fixture tree; wrapper timeout tests; output redaction tests. |
| `upload_asset` | Phase 2 | Asset target path, bytes or base64 payload, and optional overwrite semantics. | Asset metadata with final path and size. | Writes static/bundle asset file. | Path traversal, symlink escape, oversized payloads, binary/log confusion. | Upload parity on static and bundle trees; size-limit tests; symlink reject tests. |
| `list_assets` | Phase 1 | Asset scope/filter selectors for static and bundle assets. | Deterministic asset summary list: path, size, MIME-ish metadata if available, and page association. | Filesystem read only. | Traversal in scan roots; false inclusion of generated files. | Golden parity on static and bundle fixture trees. |
| `generate_featured_image` | Deferred | Image generation prompt/options and target page association. Exact Python behavior must be re-captured before any design is finalized. | Generated image artifact or updated page metadata; exact response shape to be frozen later. | Potentially writes image assets and page frontmatter. | Highest-risk path for unsafe execution and oversized artifacts. | Deferred until the core filesystem and build path are stable. |
| `check_sri_versions` | Deferred | Audit/run options for dependency/version checks. | Audit result summary, probably structured findings and exit status. | Read-only audit plus optional wrapper execution. | Shell use, supply-chain output leakage, false confidence if results are not normalized. | Deferred until the core page/asset/build behavior is stable. |

### Notes on schema discipline

- Exact field names should be captured from Python oracle calls, not inferred from intuition.
- The Go repo should use typed request structs and explicit validation.
- Unknown Python fields should be preserved only if the oracle proves they are part of the real contract.

---

## 6. Parity Strategy

### Oracle model

- Python `hugo-mcp` is the behavior oracle.
- Go does not define the contract first; it mirrors the oracle contract.
- The parity harness compares responses from the same request set against Python and Go.

### Fixture strategy

1. Create a minimal Hugo site fixture tree.
2. Capture oracle responses for read-only tools first.
3. Normalize nondeterministic fields out of the comparison.
4. Freeze the stable part of each response as a golden file.
5. Only after read-only parity is stable, add mutation fixtures on a staging tree.

### What to compare

- Deterministic response fields only.
- File paths after normalization.
- Content/frontmatter data after canonicalization.
- Error class, error code, and stable message fragments.

### What not to compare

- Timestamps unless intentionally part of the contract.
- Random IDs unless oracle behavior proves they are stable.
- Raw shell output.
- Environment-specific absolute paths unless they are part of the documented result.

### Testing order

1. Read-only parity against minimal fixtures.
2. Path traversal rejection tests.
3. Symlink escape rejection tests.
4. Mutation parity on temporary staging trees.
5. Build wrapper behavior and timeout tests.
6. Shadow comparison against the live Python service only after fixture parity is clean.

### Fixture corpus recommendations

- `testdata/fixtures/minimal-site/`
- `testdata/fixtures/minimal-site-content/`
- `testdata/fixtures/oracle/list-pages/*.json`
- `testdata/fixtures/oracle/get-page/*.json`
- `testdata/fixtures/oracle/list-assets/*.json`
- `testdata/fixtures/staging-tree/`

---

## 7. Security Rules

### Mandatory rules

1. Path traversal must be impossible.
   - Clean paths.
   - Resolve final target paths.
   - Reject relative escapes and invalid separators.

2. Symlink escape must be impossible.
   - Check parent directories and final targets.
   - Refuse writes through pre-existing symlinks.
   - Treat the filesystem tree as untrusted input.

3. Writes are only allowed under allowlisted roots.
   - Separate allowlists for content and static assets.
   - No implicit fallback to broader paths.

4. `exec.CommandContext` only.
   - Never use `sh -c`.
   - Never use `bash -lc`.
   - Pass arguments explicitly.

5. Timeouts are mandatory.
   - Build and external hooks must always use context deadlines.
   - No unbounded subprocesses.

6. Payload size limits are mandatory.
   - Recommended initial caps:
     - MCP request body: 1 MiB
     - tool args JSON: 256 KiB
     - page content/frontmatter: 1 MiB
     - asset upload: 25 MiB
     - captured build output: 64 KiB
   - These caps should be configurable, but not disabled.

7. Logs must be redacted.
   - No tokens.
   - No secrets.
   - No raw payloads unless explicitly safe.

8. Fixtures and docs must be secret-free.
   - No production tokens.
   - No private URLs that reveal secrets.
   - No copied environment files.

9. Build/deploy must go through an injectable wrapper.
   - The domain should depend on a `Runner` interface, not raw shell calls.
   - Wrapper output should be bounded and sanitized.

10. Featured image and SRI remain deferred.
    - Do not stabilize these paths before the read-only and mutation core is solid.

### Security posture for the repo

- Fail closed on missing config.
- Fail closed on missing allowlist roots.
- Fail closed on unsafe paths.
- Fail closed on missing staging roots for mutation tests.
- Prefer refusal over partial guessing.

---

## 8. Relation to `mcp-runtime-go`

### Role split

- `mcp-runtime-go`:
  - OAuth 2.0 / PKCE gateway.
  - token persistence.
  - authz/authn boundary.
  - reverse proxy.
  - operational and audit responsibilities.

- `hugo-mcp-go`:
  - Hugo content domain.
  - tool execution.
  - path safety.
  - build and asset logic.
  - parity fixtures and domain-specific tests.

### Routing strategy

- `mcp-runtime-go` should route to `hugo-mcp-go` as a backend.
- Production recommendation: loopback HTTP backend for the Hugo server.
- Development/testing can also use stdio if the SDK/server harness benefits from it.
- Do not import `mcp-runtime-go/internal/*` from the new repo.

### Sharing strategy

- Do not create a shared internal package dependency now.
- If later reuse is needed, extract a small public module explicitly and intentionally.
- Keep that decision out of phase 1.

### Shadow/cutover strategy

1. Keep Python authoritative while the Go backend is being built.
2. Run Go in shadow mode for read-only parity first.
3. Use fixture/staging parity for mutations.
4. Cut over the backend target only when the parity corpus is stable.
5. Keep rollback as a routing/configuration switch, not a code rollback plan.

---

## 9. Phase 1 Backlog: Read-Only Core

### Backlog items

1. Create repo skeleton and Go module.
2. Add MCP SDK dependency and minimal server boot.
3. Implement config loading with explicit Hugo roots and limits.
4. Implement `internal/security/pathguard`.
5. Implement frontmatter parsing/serialization helpers.
6. Implement page discovery and page read logic.
7. Implement asset discovery logic.
8. Add deterministic response normalization helpers.
9. Add fixture tree under `testdata/fixtures/`.
10. Add parity tests against captured Python responses.
11. Add path traversal and symlink rejection tests.
12. Add logging redaction and request correlation helpers.

### Phase 1 exit criteria

- `list_pages`, `get_page`, and `list_assets` match Python on the fixture corpus for deterministic fields.
- Path traversal is rejected.
- Symlink escape is rejected.
- No shell execution exists in the repo.
- No mutating tool is exposed yet.

---

## 10. Phase 2 Backlog: Mutations and Build

### Backlog items

1. Implement `create_page`.
2. Implement `update_page`.
3. Implement `delete_page`.
4. Implement `upload_asset`.
5. Implement `build_site`.
6. Add the bounded command runner wrapper.
7. Add staging-tree mutation fixtures.
8. Add overwrite/merge/delete safety tests.
9. Add build timeout and redaction tests.
10. Add explicit error mapping for filesystem and runner failures.

### Phase 2 exit criteria

- Mutation tools pass on a temporary staging tree.
- Writes remain inside allowlisted roots.
- Build output is bounded and sanitized.
- External command execution is entirely context-bound.
- No mutation path is enabled without explicit intent.

---

## 11. Deferred Items

### Keep deferred until the core is stable

1. `generate_featured_image`
2. `check_sri_versions`
3. Cloudflare purge integration
4. IndexNow integration
5. Google indexing integration
6. Any plugin-like extensibility mechanism

### Why these stay deferred

- They introduce the highest risk of external side effects and weak parity.
- They are not needed to prove the core Hugo domain.
- They would distract from the repository boundary and safety model.

---

## 12. Ready-to-Implement Criteria

The repo is ready for implementation only when all of the following are true:

- The SDK choice is pinned: `github.com/modelcontextprotocol/go-sdk v1.6.1` or an explicitly justified replacement.
- The production transport shape is decided.
- The fixture corpus has been captured from Python.
- Read-only deterministic parity fields are frozen.
- Pathguard and symlink tests are specified before code is written.
- Command execution wrapper semantics are agreed.
- Payload caps are documented.
- The `mcp-runtime-go` routing contract is documented.
- Shadow mode and rollback expectations are documented.
- No secrets exist in fixtures, docs, or tests.

---

## 13. Final Recommendation

Proceed with a separate `hugo-mcp-go` repository, built on `github.com/modelcontextprotocol/go-sdk v1.6.1`, with read-only parity first and mutations only on staging trees. Keep `mcp-runtime-go` as the generic gateway and route Hugo traffic to the new backend over loopback HTTP. Do not fold Hugo into the runtime repo, and do not copy Python shell/plugin patterns into Go.
