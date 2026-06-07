# MCP Hugo -> Go Migration Plan

Date: 2026-06-06

Scope:
- `hugo-mcp` Python runtime in `/home/jm/Documents/hugo-mcp`
- `mcp-runtime-go` in `/home/jm/Documents/mcp-runtime-go`
- No source changes in this pass

## Executive Summary

The Python `hugo-mcp` service is a single FastAPI monolith that combines MCP transport, page/file manipulation, Cloudflare orchestration, plugin discovery, and a few external-script integrations. The Go runtime is already a mature OAuth proxy with good security primitives, structured audit logging, storage abstraction, and test coverage, but it does not yet contain a Hugo domain.

The migration should not copy the Python structure as-is. In particular:
- Do not replicate Python-style dynamic plugin discovery in Go.
- Do not repurpose `internal/storage` for Hugo content state; that package should remain token-persistence only.
- Keep OAuth proxy, Hugo MCP, storage, auth, and config as separate domains.
- Make filesystem and command execution explicit, injectable, and bounded by context/timeouts.
- Use shadow parity on read-only operations first, then gated write operations on a fixture/staging tree.

Most importantly, the phase-1 implementation should be a pure-domain Hugo package plus MCP tool adapters. The current Go OAuth runtime should remain untouched in behavior until Hugo parity is demonstrated.

## Evidence-Based State: Python `hugo-mcp`

### Entry point and transport

Confirmed in `main.py:1-1285`:
- FastAPI app with `/mcp`, `/health`, `/healthz`, `/readyz`, and `/metrics`.
- JSON-RPC MCP handling is implemented directly in the FastAPI app.
- Interactive docs are disabled (`docs_url=None`, `redoc_url=None`, `openapi_url=None`).
- Rate limiting is applied at 60/minute per client IP.
- A request-body cap exists at 10 MB.

### Tools exposed

Confirmed in `main.py:640-809` and `README.md:5-18`:
- `list_pages`
- `get_page`
- `create_page`
- `update_page`
- `delete_page`
- `build_site`
- `upload_asset`
- `list_assets`
- `generate_featured_image`
- `check_sri_versions`

### File access and content model

Confirmed in `main.py:96-114`, `main.py:851-1279`, and `README.md:30-55`:
- Hugo content root is hard-coded to `/home/jm/hugo-site/content`.
- Static assets are hard-coded under `/home/jm/hugo-site/static`.
- Page routing is language-aware and uses `_index` plus leaf bundle patterns.
- Frontmatter is YAML-serialized and validated with a max size of 10 KB and max depth of 3.
- `create_page` and `update_page` support free-form frontmatter with a small forbidden-field list.
- Mermaid blocks are frozen to SVG before write/build.

### Shell and external execution

Confirmed in `main.py:224-246`, `main.py:439-450`, `main.py:584-589`, `main.py:1144-1174`, and `plugins/sri-check/plugin.py:68-136`:
- `deploy.sh` is executed via `subprocess.run(["bash", DEPLOY_SH], ...)`.
- Mermaid rendering shells out to `mmdc`.
- `generate_featured_image` executes a local Python skill file with `exec(open(...).read(), ns)`.
- `check_sri_versions` delegates to a shell script with `--json` and optional flags.

### Plugin model

Confirmed in `core/plugin_loader.py:13-159`, `core/plugin_base.py:1-32`, and `plugins/*/plugin.py`:
- Plugins are discovered dynamically from `plugins/*/plugin.py`.
- Plugin config is read from `config/plugins.yaml`.
- `cloudflare`, `indexnow`, `google-indexing`, and `sri-check` are implemented as separate plugins.
- `sri-check` is an audit-only plugin; page-event plugins run in parallel with per-plugin timeouts.

### Security model

Confirmed in `main.py:394-433` and `main.py:185-201`:
- Auth is bearer-token based.
- Tokens are read from `tokens.json` with bcrypt hashes.
- There is a fallback plain token via `MCP_TOKEN` for migration/backward compatibility.
- Route and language validation exist for several handlers.
- `upload_asset` and `list_assets` have explicit path checks.

### Tests and production use

Confirmed in the repo:
- No Python test files were found (`rg --files` returned no `*_test.py` / `test*.py` files).
- Production deployment is described in `systemd/hugo-mcp.service` and `README.md:20-28`.
- `docs/postmortems/2026-05-09-double-stack-mcp-oauth.md` documents real Claude.ai usage and a stale-session incident.
- `README.md:61-76` documents the `sri-check` audit plugin as an on-demand façade while a weekly cron continues independently.

### Confirmed Python discrepancies and risks

1. `tool_list_pages` path handling is not fully normalized.
   - Evidence: `main.py:851-906` builds `scan_path = Path(CONTENT_DIR) / section.strip('/')` with no `resolve()` boundary check.
   - Consequence: `section` can escape the content root during scan.

2. `build_site` advertises `purge_cf` but ignores it.
   - Evidence: `main.py:1130-1141`.
   - Consequence: schema/runtime mismatch; the tool returns `cf_purge` as skipped regardless of input.

3. `generate_featured_image` executes local Python code.
   - Evidence: `main.py:584-589`.
   - Consequence: this is not a safe pattern to copy into Go.

4. Dynamic plugin loading is high flexibility, high coupling.
   - Evidence: `core/plugin_loader.py:24-87`.
   - Consequence: this is acceptable in Python for the current service, but should not become the default Go design.

5. `upload_asset` and page writes are validated, but symlink escape remains a probable risk.
   - Evidence: `main.py:1177-1230` and `main.py:979-1072`.
   - Consequence: a pre-existing symlink inside an allowed tree could redirect writes if not blocked separately.

## Evidence-Based State: `mcp-runtime-go`

### Current architecture

Confirmed in `README.md`, `docs/architecture/ARCHITECTURE.md:3-19`, and `internal/runtime/app.go:28-183`:
- The Go runtime is already the authoritative OAuth proxy in production.
- It is structured as isolated internal packages.
- `internal/runtime` wires the app, HTTP server, storage, audit, and proxy.
- `internal/httpserver` owns server lifecycle.
- `internal/oauthproxy` owns OAuth discovery, registration, authorization, token exchange, and reverse proxying.

### Config and compatibility

Confirmed in `internal/config/config.go:17-176`:
- Canonical env names are `HUGO_*`, with legacy `GRAV_*` aliases still supported.
- `HUGO_MCP_URL`, `HUGO_HOST`, and `HUGO_TOKEN` are required.
- `USE_SQLITE=true` is the default token-store mode.
- `TRUSTED_AUTHORIZE_CIDRS` already exists for secondary `/authorize` enforcement.

### Security and observability

Confirmed in `internal/oauthproxy/handlers.go:15-264`, `internal/oauthproxy/proxy.go:48-140`, `internal/observability/audit.go:1-131`, and `internal/security/*`:
- `/authorize` has Go-level CIDR enforcement.
- Redirect URIs are allowlisted.
- OAuth tokens are SHA-256 hashed.
- Audit logs redact token/secret/code values.
- Request IDs are generated and propagated.
- Reverse proxy strips hop-by-hop headers and sets the backend Authorization header explicitly.
- `/metrics` is loopback-oriented in deployment docs.

### Storage

Confirmed in `internal/storage/json_store.go:1-113`, `internal/storage/sqlite_store.go:1-125`, and `internal/storage/interface.go:1-10`:
- Token persistence already has JSON and SQLite implementations.
- SQLite WAL is the production path.
- JSON store is legacy/rollback only.
- Storage is narrowly scoped to token persistence and should stay that way.

### Tests and maturity

Confirmed by the test tree:
- `internal/config/config_test.go`
- `internal/context/context_test.go`
- `internal/httpserver/server_test.go`
- `internal/oauthproxy/*_test.go`
- `internal/observability/audit_test.go`
- `internal/runtime/*_test.go`
- `internal/security/*_test.go`
- `internal/storage/*_test.go`

This is a materially stronger test base than the Python repo.

### Brooks-style findings on the Go side

1. The runtime already has the right boundaries, but the Hugo domain is missing.
   - The current code is a strong shell for integration, not the integration itself.

2. `internal/storage` should not absorb Hugo page content.
   - Storage is already correctly focused on token state.
   - Reusing it for pages, assets, or audit results would blur ownership and likely create accidental coupling.

3. The runtime is configured for a single authoritative OAuth proxy plus a remote backend.
   - The Hugo migration will require a transport decision, not just a package addition.
   - That decision is the main architectural unknown.

4. The Go codebase is already testable because command execution, request IDs, and storage are explicit.
   - The Hugo addition should preserve that property with injectable filesystem, clock, command runner, and HTTP client abstractions.

## Python Feature -> Go Target Matrix

| Python feature | Current behavior | Go target | Phase | Notes |
|---|---|---|---|---|
| `list_pages` | Scans Hugo content tree and returns metadata | `internal/hugo/pages.List` + MCP tool adapter | 1 | Read-only parity first |
| `get_page` | Reads frontmatter + Markdown | `internal/hugo/pages.Get` + MCP tool adapter | 1 | Must preserve `_index` semantics |
| `create_page` | Validates, writes, deploys, purges via plugins | `internal/hugo/pages.Create` + deploy wrapper + hook interface | 2 | Mutating path; shadow only first |
| `update_page` | Deep merge frontmatter and write | `internal/hugo/pages.Update` + same wrappers | 2 | Needs null-deletion semantics |
| `delete_page` | Remove page and rebuild | `internal/hugo/pages.Delete` + deploy wrapper | 2 | Needs symlink-safe delete checks |
| `build_site` | Rebuild only; `purge_cf` currently ignored | `internal/hugo/build.Run` + explicit CF hook semantics | 2 | Fix the schema/runtime mismatch |
| `upload_asset` | Base64 decode, path validation, write static asset | `internal/hugo/assets.Upload` | 2 | Must harden against symlink escape |
| `list_assets` | Scan static + content bundle assets | `internal/hugo/assets.List` | 1 | Read-only parity candidate |
| `generate_featured_image` | Executes local Python skill, rewrites featuredImage | Deferred adapter or separate integration package | 3 or out-of-scope | Do not port the `exec()` pattern |
| `check_sri_versions` | Delegates to audit plugin and shell script | Separate integration or audit hook, not core Hugo domain | 3 | Keep orchestration separate from page CRUD |

## Proposed Architecture

### Minimal target

I recommend the following package boundaries:

- `internal/hugo/`
  - Pure Hugo domain logic: page CRUD, asset listing/upload, frontmatter parse/merge, safe paths, Mermaid freeze, build wrapper.
  - No MCP parsing here.
  - No OAuth knowledge here.
  - No plugin discovery here.

- `internal/mcp/tools/hugo/`
  - MCP tool schema and JSON argument translation.
  - Calls into `internal/hugo`.
  - Contains tool registration only.

- `internal/hugo/hooks/` or `internal/integrations/hugo/`
  - Optional Cloudflare, IndexNow, Google Indexing, SRI, or other side-effect adapters.
  - Each adapter should be explicit and opt-in, not filesystem-discovered.

- `internal/security/`
  - Reuse for path validation helpers and request/source IP policy checks.
  - Do not add Hugo storage here.

- `internal/config/`
  - Add a dedicated `HugoConfig` subtree rather than extending OAuth proxy config indefinitely.
  - Keep `oauthproxy` config and Hugo config separate.

- `internal/runtime/`
  - Wire the OAuth proxy and Hugo domain together.
  - Own startup order, lifecycle, and shadow mode.

### Interfaces to introduce

Suggested interfaces for testability:

- `PageStore`
  - `List(ctx, filter) ([]PageSummary, error)`
  - `Get(ctx, route, lang) (Page, error)`
  - `Create(ctx, input) (PageResult, error)`
  - `Update(ctx, input) (PageResult, error)`
  - `Delete(ctx, route, lang) error`

- `AssetStore`
  - `List(ctx, filter) ([]Asset, error)`
  - `Upload(ctx, input) (AssetResult, error)`

- `Builder`
  - `Build(ctx) (BuildResult, error)`

- `DeployRunner`
  - `Run(ctx, args...) (stdout string, stderr string, err error)`

- `HookRegistry`
  - `OnPageEvent(ctx, event, urls, meta) []HookResult`
  - `OnAudit(ctx, auditType, meta) []HookResult`

- `Filesystem`
  - Abstract path resolution, read/write, and symlink checks for tests.

### Boundary rules

- All user-controlled paths must be validated against an allowlist root.
- No shell execution via `sh -c` or `bash -lc`.
- No direct `exec()`-style evaluation.
- No implicit filesystem traversal outside the Hugo root.
- No secrets in returned tool results or logs.

## Migration Phases

### Phase 0 - Freeze and model parity

Goal:
- Capture the Python runtime as a reference model before porting code.

Deliverables:
- Tool inventory.
- Request/response fixtures for read-only tools.
- File tree snapshot of a minimal Hugo content sample.
- Shadow comparison harness design.

Exit criteria:
- Read-only fixtures are stable and versioned.
- No implementation changes yet.

### Phase 1 - Read-only Hugo domain

Goal:
- Implement pure read-only Hugo operations in Go.

Deliverables:
- `list_pages`
- `get_page`
- `list_assets`
- Frontmatter parser/serializer
- Safe path utilities

Exit criteria:
- Go read-only responses match Python on the fixture tree.
- Path traversal and symlink tests pass.
- No shell execution in this phase.

### Phase 2 - Mutating page and asset operations

Goal:
- Port the write path carefully.

Deliverables:
- `create_page`
- `update_page`
- `delete_page`
- `upload_asset`
- `build_site`
- Command wrapper for deploy/build/mermaid with context deadlines

Exit criteria:
- Mutation tests pass on a temp Hugo tree.
- Writes stay inside the allowlisted roots.
- Failure paths are explicit and non-leaky.

### Phase 3 - Optional integrations

Goal:
- Reintroduce side effects as explicit adapters.

Candidates:
- Cloudflare purge hook
- IndexNow hook
- Google indexing hook
- SRI audit hook
- Featured-image generation adapter

Exit criteria:
- Each adapter is independently testable.
- Missing config is fail-closed.
- No plugin autodiscovery by filesystem scan.

### Phase 4 - Shadow and cutover

Goal:
- Compare Python and Go behavior under mirrored traffic and fixture-driven tests.

Deliverables:
- Shadow deployment of the Hugo domain.
- Golden diff comparison for read-only tools.
- Controlled staging runs for write tools.
- Rollback switch documented and exercised.

Exit criteria:
- Read-only parity is clean over a representative request set.
- Controlled write parity is clean on a fixture/staging tree.
- Operator rollback is one config change, not a code revert.

## Security Audit

### Confirmed issues in Python to avoid copying

| Severity | Finding | Evidence | Consequence |
|---|---|---|---|
| High | `list_pages.section` path traversal risk | `main.py:851-906` | Scanning can escape the Hugo tree |
| High | `generate_featured_image` uses `exec(open(...).read(), ns)` | `main.py:584-589` | Local code execution boundary is unsafe to replicate |
| Medium | `build_site` ignores `purge_cf` | `main.py:1130-1141` | Tool schema and behavior diverge |
| Medium | Dynamic plugin discovery | `core/plugin_loader.py:24-87` | Hard to reason about and easy to over-couple |
| Medium | Potential symlink escape on writes | `main.py:979-1072`, `main.py:1177-1230` | A pre-existing symlink could redirect writes |

### Go-side security requirements for the port

1. Path traversal
   - Use allowlisted roots only.
   - Resolve and verify the final parent path after cleaning.
   - Reject relative segments, backslashes, and symlink escapes.

2. Command injection
   - Use `exec.CommandContext`.
   - Pass arguments directly.
   - Never invoke a shell for build/deploy/mermaid.

3. Symlink escape
   - Check parent directories and destination paths for symlink behavior.
   - Prefer a write strategy that refuses pre-existing symlinks.

4. Dangerous writes
   - Only permit writes under `content/` and `static/` allowlists.
   - Make destructive operations explicit and narrow.

5. Mutator exposure
   - Keep all write tools behind the existing auth boundary.
   - Preserve fail-closed authz if the backend is reachable directly.

6. Schema validation
   - Enforce JSON schemas at the MCP boundary and typed validation in the Hugo domain.
   - Keep the Python validation semantics, but do not copy the permissive parts blindly.

7. Sensitive logs
   - Never log secret values or raw tokens.
   - Do not return deploy output unless it is sanitized.

8. Timeouts and limits
   - Add explicit timeouts for build/deploy/external hooks.
   - Enforce request size caps for content and asset payloads.
   - Add rate limiting per tool class if the domain becomes user-facing at scale.

### Confirmed Go strengths to reuse

- Constant-time comparisons and request-ID propagation are already in place.
- Audit logging already redacts token/secret/code data.
- TLS backend verification is already explicit.
- `TRUSTED_AUTHORIZE_CIDRS` already exists for defense in depth.
- SQLite WAL token storage is already production ready.

## Brooks / Superpower Review

### Architecture review

Confirmed:
- The Go codebase has the right separation of concerns for a second domain.
- The existing `internal/runtime` and `internal/httpserver` layers are suitable integration points.

Recommendation:
- Add a new Hugo domain package rather than folding Hugo logic into OAuth proxy code.
- Keep `oauthproxy` as the auth/data-plane boundary, not the content engine.

Probable risk:
- If Hugo is grafted into `oauthproxy` directly, the package will become a mixed auth/content/service layer and lose the clarity the Go runtime currently has.

### Security review

Confirmed:
- Python contains one clearly unsafe pattern to avoid: `exec(open(...).read(), ns)`.
- Dynamic filesystem plugin loading is not a good default for Go.

Recommendation:
- Prefer typed registries and explicit adapters.
- Make all external effects injectable and observable.

Probable risk:
- A naive port of the deploy/build/plugin flow will reintroduce shell-injection and secret-leak risks the Go runtime currently avoids.

### Testability review

Confirmed:
- The Go repo already has meaningful unit coverage in config, security, storage, observability, runtime, and oauthproxy.

Recommendation:
- Add injectable filesystem, clock, HTTP client, and command runner abstractions before porting Hugo.
- Use temp directories and golden fixtures for all content operations.

Probable risk:
- If the port keeps global state or hard-coded paths, it will become difficult to test and impossible to shadow cleanly.

## Shadow / Parity Strategy

### Read-only parity

1. Mirror representative requests against Python and Go.
2. Compare response payloads for:
   - `list_pages`
   - `get_page`
   - `list_assets`
3. Match request IDs in logs.
4. Diff only normalized, deterministic fields.

### Write parity

Do not compare live destructive effects against production content.

Use a fixture/staging Hugo tree and compare:
- Output files
- Frontmatter changes
- Asset paths
- Error codes and messages
- Build/deploy wrapper behavior

### Shadow deployment model

Recommended:
- Keep the Python service authoritative until the Hugo Go domain is proven.
- Run the Go Hugo domain in shadow mode on a separate port or hostname.
- Mirror traffic at the edge only for read-only tools first.
- Introduce write tools only after deterministic parity is established on fixtures.

### Rollback strategy

Rollback should be configuration-first:
- Keep the Python backend available until cutover.
- Use an environment switch or routing switch to return the `/mcp` backend to Python if needed.
- Keep the Go OAuth runtime untouched while toggling the Hugo backend target.

Open question:
- Whether the final topology should be "Go OAuth proxy -> local in-process Hugo handler" or "Go OAuth proxy -> loopback Hugo backend" is not yet decided in the repo. That decision must be made before implementation.

## Readiness Criteria

The next implementation phase is ready only when all of the following are true:

- The target Hugo package boundaries are agreed and documented.
- Read-only fixtures exist for the Python current behavior.
- Path traversal and symlink-escape tests are specified before code is written.
- A command runner abstraction is defined for deploy/build/mermaid hooks.
- The Go runtime config split between OAuth and Hugo is agreed.
- The rollback path is a single config change, not a binary replacement strategy.
- No secrets are introduced into logs, docs, fixtures, or tests.

## Files Likely To Change In The Next Phase

These are the precise files I would expect to touch first, in order:

### Go runtime wiring

- `internal/config/config.go`
- `internal/runtime/app.go`
- `internal/httpserver/server.go` if the Hugo domain needs a distinct router mount strategy
- `internal/oauthproxy/service.go` if the backend dispatch model changes
- `internal/oauthproxy/handlers.go` if proxy/auth boundaries need to be reworked
- `internal/oauthproxy/proxy.go` if `/mcp` stops being a pure reverse proxy

### New Hugo domain

- `internal/hugo/*.go`
- `internal/mcp/tools/hugo/*.go`
- `internal/hugo/hooks/*.go` or `internal/integrations/hugo/*.go`

### Supporting helpers

- `internal/security/*.go` if path-guard helpers are added there
- `internal/observability/*.go` only if Hugo-specific metrics or audit helpers are needed

### Tests

- `internal/hugo/*_test.go`
- `internal/mcp/tools/hugo/*_test.go`
- `internal/runtime/*_test.go`
- `internal/security/*_test.go`

### Deployment and docs

- `deploy/env/mcp-runtime-shadow.env.example`
- `deploy/systemd/mcp-runtime-shadow.service`
- `README.md`
- `docs/architecture/HUGO_MCP_INTEGRATION.md`
- `docs/migration/MIGRATION_PLAN.md`
- `docs/operations/SHADOW_RUNBOOK.md`

## Open Questions

1. Should `generate_featured_image` be ported as a Go-native image pipeline or left as a separately wrapped external asset generator?
2. Should `check_sri_versions` remain an external audit adapter, or be reimplemented inside Go after the core Hugo port is stable?
3. Is the final Hugo topology remote-backend compatible, or should the backend become in-process behind the OAuth boundary?
4. Which production tools are actually used day-to-day by Claude.ai versus which are only documented?

## Bottom Line

The migration is feasible and the Go runtime is a good foundation, but the safe path is a domain carve-out, not a feature-copy. Port the Hugo logic as a pure package with explicit boundaries, then wire the MCP adapter and optional integrations around it. Keep the Python service authoritative until read-only parity is proven, then move write operations only after fixture-based and shadow-based validation is clean.
