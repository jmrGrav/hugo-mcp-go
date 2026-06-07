# Full Tool Parity And Post-Build Hooks Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Restore Python MCP tool parity for the remaining missing tools and add a secure, testable, packageable post-build/post-mutation hook system in Go.

**Architecture:** Keep MCP tool parity and hook execution as two explicit subsystems with a narrow interface between them. Tool parity work stays inside the existing MCP/tooling layers, while hook execution is introduced as a separate package set with file-backed secrets, SQLite only for non-secret operational state, and dry-run defaults.

**Tech Stack:** Go, MCP SDK, SQLite, HTTP clients, systemd unit examples, shell smoke tests.

---

### Task 1: Restore the missing Python-parity tools

**Files:**
- Modify: `internal/tools/tools.go`
- Create: `internal/tools/check_sri_versions_test.go`
- Create: `internal/tools/generate_featured_image_test.go`
- Modify: `scripts/tool-parity-smoke.sh`
- Modify: `docs/SMOKE_TEST_AND_COVERAGE_REPORT.md`
- Create: `docs/TOOL_CATALOG_PARITY_REPORT.md`

- [x] **Step 1: Write the failing tests**

Add unit tests that expect `check_sri_versions` and `generate_featured_image` to appear in `tools/list`, accept the same input shapes as the Python oracle, and return deterministic error text when validation fails.

- [x] **Step 2: Run the tests and verify they fail**

Run:

```bash
go test ./internal/tools -run 'Test.*(SRI|Featured)' -v
```

Expected: failures showing the tools are missing or still unsupported.

- [x] **Step 3: Implement the minimal parity layer**

Register both tools in `internal/tools/tools.go` with explicit schemas and safe validation. Keep all file access inside the Hugo root, reject symlink escapes, reject path traversal, and avoid shell execution.

- [x] **Step 4: Run the tests and verify they pass**

Run:

```bash
go test ./internal/tools -run 'Test.*(SRI|Featured)' -v
go test ./internal/tools ./internal/shim
```

- [x] **Step 5: Update the smoke script**

Change `scripts/tool-parity-smoke.sh` so `check_sri_versions` and `generate_featured_image` must succeed or fail with the Python-matched errors, rather than being treated as unsupported.

---

### Task 2: Add the hook subsystem

**Files:**
- Create: `internal/hooks/config.go`
- Create: `internal/hooks/store.go`
- Create: `internal/hooks/store_test.go`
- Create: `internal/hooks/pipeline.go`
- Create: `internal/hooks/pipeline_test.go`
- Create: `internal/hooks/cloudflare.go`
- Create: `internal/hooks/cloudflare_test.go`
- Create: `internal/hooks/google_indexing.go`
- Create: `internal/hooks/google_indexing_test.go`
- Create: `internal/hooks/indexnow.go`
- Create: `internal/hooks/indexnow_test.go`
- Modify: `internal/tools/tools.go`
- Modify: `cmd/hugo-mcp-go/main.go`
- Modify: `cmd/hugo-mcp-shim/main.go`

- [x] **Step 1: Write the failing tests**

Add tests that assert:

- secrets are loaded from files only
- missing, empty, or world-readable secrets fail closed
- SQLite stores only job state and audit data, never secrets
- dry-run is the default
- retry limits are enforced
- provider clients redact tokens and private material from logs and errors

- [x] **Step 2: Run the tests and verify they fail**

Run:

```bash
go test ./internal/hooks -v
```

- [x] **Step 3: Implement the hook store and pipeline**

Introduce a minimal SQLite-backed outbox/state layer with the tables required for job tracking, attempts, deduplication, and redacted audit records. Keep providers isolated behind interfaces so they can be tested with mocks.

- [x] **Step 4: Implement provider clients**

Add Cloudflare purge-by-URL, Google Indexing URL_UPDATED/URL_DELETED, and IndexNow batch submission with timeout, retry bounds, dry-run support, and no secret logging.

- [x] **Step 5: Run the tests and verify they pass**

Run:

```bash
go test ./internal/hooks ./...
go test -race ./internal/hooks
```

---

### Task 3: Wire hooks into MCP mutation/build flows

**Files:**
- Modify: `internal/tools/tools.go`
- Modify: `internal/hugo/mutations/*.go` as needed
- Modify: `internal/server/*.go` or equivalent call path
- Create: `internal/hooks/mcp_summary.go`
- Create: `internal/hooks/mcp_summary_test.go`

- [x] **Step 1: Write the failing tests**

Add tests that confirm `create_page`, `update_page`, `delete_page`, `upload_asset`, and `build_site` enqueue URLs and return a non-secret hook summary with:

- `hooks.enabled`
- `cloudflare_purge.status`
- `google_indexing.status`
- `indexnow.status`
- `queued_urls_count`
- `failed_jobs_count`

- [x] **Step 2: Run the tests and verify they fail**

Run:

```bash
go test ./internal/tools ./internal/hooks -run 'Test.*Hook' -v
```

- [x] **Step 3: Implement the hook trigger path**

After mutation/build success, derive impacted URLs, enqueue them in the hook store, and execute immediately when enabled, otherwise leave them pending. Ensure the MCP response includes only sanitized status metadata.

- [x] **Step 4: Run the tests and verify they pass**

Run:

```bash
go test ./internal/tools ./internal/hooks ./...
```

---

### Task 4: Add administrative MCP tools

**Files:**
- Modify: `internal/tools/tools.go`
- Create: `internal/tools/hooks_admin_test.go`

- [x] **Step 1: Write the failing tests**

Add tests for:

- `list_hook_jobs`
- `retry_hook_jobs`
- `get_hook_status`
- `run_post_build_hooks`

Verify they are present only when enabled by config and that they never expose secrets.

- [x] **Step 2: Run the tests and verify they fail**

Run:

```bash
go test ./internal/tools -run 'Test.*Hook.*' -v
```

- [x] **Step 3: Implement the admin tools**

Expose read-only status and explicit retry hooks, with config-based disablement and no secret material in outputs.

- [x] **Step 4: Run the tests and verify they pass**

Run:

```bash
go test ./internal/tools ./...
```

---

### Task 5: Packaging, systemd, and docs

**Files:**
- Modify: `deploy/systemd/hugo-mcp-go.service`
- Create: `deploy/systemd/hugo-mcp-go.env.example`
- Create: `deploy/tmpfiles.d/hugo-mcp-go.conf`
- Create: `deploy/sysusers.d/hugo-mcp-go.conf`
- Create: `docs/POST_BUILD_HOOKS.md`
- Create: `docs/SECRETS_MODEL.md`
- Create: `docs/POST_BUILD_HOOKS_IMPLEMENTATION_REPORT.md`
- Modify: `docs/README.md`
- Modify: `README.md`

- [x] **Step 1: Write the documentation-first tests**

Add checks that the example env file contains only non-secret defaults and that docs state dry-run defaults, file-backed secrets, and rollback/delay behavior.

- [x] **Step 2: Update packaging files**

Document the secret file paths, permissions, tmpfiles directories, and system user/group layout required to install the hook system on another host.

- [x] **Step 3: Run the tests and verify they pass**

Run:

```bash
go test ./...
```

---

### Task 6: Smoke validation

**Files:**
- Create: `scripts/hooks-smoke.sh`

- [x] **Step 1: Write the smoke script**

The script must validate in dry-run/mock mode:

- `build_site` enqueues hook work
- `update_page` enqueues impacted URLs
- Cloudflare dry-run
- Google Indexing dry-run
- IndexNow dry-run
- retry job
- job status

- [x] **Step 2: Run the smoke script**

Run:

```bash
scripts/hooks-smoke.sh
```

Expected: no real provider calls, no secrets logged, deterministic success.

- [x] **Step 3: Final coverage pass**

Run:

```bash
go test -race ./...
go vet ./...
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
```

---

### Task 7: Final report

**Files:**
- Create: `docs/FULL_TOOL_PARITY_AND_HOOKS_REPORT.md`

- [x] **Step 1: Write the report**

Summarize:

- implemented Python tools
- hook integration status
- secret storage model
- SQLite state model
- dry-run defaults
- smoke results
- coverage results
- remaining risks

- [x] **Step 2: Re-read and verify against the spec**

Check that every requirement in the mission has a corresponding implementation or a documented intentional non-goal.

- [x] **Step 3: Stop before any merge/tag decision**

Do not push, merge, or tag automatically. Wait for explicit user instruction once validation is complete.
