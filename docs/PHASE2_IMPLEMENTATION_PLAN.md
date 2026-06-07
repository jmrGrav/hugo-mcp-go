# Phase 2 Mutation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development or superpowers:executing-plans. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** implement the five Phase 2 mutation tools behind the existing MCP server, using the staging model and oracle capture as the behavioral contract.

**Architecture:** keep the read-only packages intact and add a separate mutation layer. The mutation layer owns staging, write semantics, build execution, and rollback. The MCP layer stays thin: it validates the request, delegates to domain services, and serializes the result.

**Tech Stack:** Go 1.25, `github.com/modelcontextprotocol/go-sdk v1.6.1`, `gopkg.in/yaml.v3`, `exec.CommandContext`, `os/atomic rename`, existing `internal/config`, `internal/security/pathguard`, and `internal/observability`.

---

### Task 1: Introduce the mutation staging core

**Files:**

- Create: `internal/hugo/staging/staging.go`
- Create: `internal/hugo/staging/staging_test.go`
- Create: `internal/hugo/staging/paths.go`

- [ ] **Step 1: Write the failing tests**

Cover:

- canonicalization of staging roots
- rejection of missing roots
- rejection of symlink roots
- separation of content/static/public/work roots
- path validation for new targets
- path validation for existing targets

- [ ] **Step 2: Implement the minimal staging API**

Add an API that returns a validated staging workspace with:

- `ContentRoot`
- `StaticRoot`
- `PublicRoot`
- `WorkRoot`
- mutation-safe path helpers

- [ ] **Step 3: Run the tests**

Run:

```bash
go test ./internal/hugo/staging
```

Expected: PASS.

### Task 2: Add page mutation services

**Files:**

- Create: `internal/hugo/mutations/pages.go`
- Create: `internal/hugo/mutations/pages_test.go`
- Create: `internal/hugo/mutations/frontmatter.go`

- [ ] **Step 1: Write the failing tests**

Use `testdata/fixtures/oracle_phase2/` to assert:

- `create_page` creates the expected file and frontmatter
- `update_page` deep-merges frontmatter and honors `null` deletion
- `update_page` rejects immutable `date`
- `create_page` rejects frontmatter conflicts
- `delete_page` removes the file and leaves a rollback breadcrumb
- traversal and symlink escapes are rejected

- [ ] **Step 2: Implement the minimal page mutation API**

Implement:

- `Create(ctx, req) -> result`
- `Update(ctx, req) -> result`
- `Delete(ctx, req) -> result`

The service must:

- use the staging workspace
- write via temp file + atomic rename
- preserve the Python oracle field order where the oracle is deterministic
- normalize YAML for comparison

- [ ] **Step 3: Run the tests**

Run:

```bash
go test ./internal/hugo/mutations/...
```

Expected: PASS.

### Task 3: Add the asset upload service

**Files:**

- Create: `internal/hugo/mutations/assets.go`
- Create: `internal/hugo/mutations/assets_test.go`

- [ ] **Step 1: Write the failing tests**

Use `testdata/fixtures/oracle_phase2/` to assert:

- filename validation
- subfolder validation
- extension allowlist
- base64 decode errors
- byte-for-byte write behavior
- path traversal rejection
- symlink escape rejection

- [ ] **Step 2: Implement the minimal asset upload API**

Implement:

- `UploadAsset(ctx, req) -> result`

The service must:

- decode base64
- validate decoded size
- write into the staging static root only
- return the final path and public URL

- [ ] **Step 3: Run the tests**

Run:

```bash
go test ./internal/hugo/mutations/...
```

Expected: PASS.

### Task 4: Add the build runner wrapper

**Files:**

- Create: `internal/runner/runner.go`
- Create: `internal/runner/runner_test.go`
- Create: `internal/hugo/mutations/build.go`
- Create: `internal/hugo/mutations/build_test.go`

- [ ] **Step 1: Write the failing tests**

Cover:

- context timeout propagation
- stdout/stderr capture caps
- no shell invocation
- build failure preservation
- successful build returns the expected shape

- [ ] **Step 2: Implement the runner abstraction**

Define a small interface that can be injected into the build service and later into deploy orchestration.

- [ ] **Step 3: Implement build_site**

The build service must:

- run the build through the injected runner
- keep `exec.CommandContext` the only execution primitive
- never use `sh -c` or `bash -lc`
- return the oracle shape from `PHASE2_ORACLE_CAPTURE.md`

- [ ] **Step 4: Run the tests**

Run:

```bash
go test ./internal/runner ./internal/hugo/mutations/...
```

Expected: PASS.

### Task 5: Wire the MCP tools

**Files:**

- Modify: `internal/tools/tools.go`
- Modify: `internal/server/server.go`
- Modify: `internal/server/server_test.go`

- [ ] **Step 1: Write the failing MCP tests**

Add tests that list and call all five mutation tools through the SDK.

- [ ] **Step 2: Register the new tools explicitly**

Register:

- `create_page`
- `update_page`
- `delete_page`
- `upload_asset`
- `build_site`

Keep the MCP layer thin and keep the mutation logic in the domain packages.

- [ ] **Step 3: Run the tests**

Run:

```bash
go test ./internal/server ./internal/tools
```

Expected: PASS.

### Task 6: Lock parity and safety fixtures

**Files:**

- Create: `testdata/fixtures/oracle_phase2/*.json`
- Create: `testdata/fixtures/oracle_phase2/staging/**`
- Modify: `docs/PHASE2_ORACLE_CAPTURE.md`
- Modify: `docs/PHASE2_STAGING_MODEL.md`

- [ ] **Step 1: Add golden comparisons**

Compare:

- page file contents
- frontmatter normalization
- asset bytes and paths
- build output paths

- [ ] **Step 2: Add negative fixtures**

Cover:

- traversal rejection
- symlink escape rejection
- immutable `date`
- frontmatter conflict
- invalid extension
- invalid base64
- missing page

- [ ] **Step 3: Run the full suite**

Run:

```bash
go test ./...
```

Expected: PASS.

## Acceptance criteria

Phase 2 is complete only when:

- all five mutation tools are exposed by MCP
- staging is isolated from production roots
- path traversal and symlink escape are covered on every mutation path
- rollback is explicit and tested
- build execution is wrapper-injected and shell-free
- oracle parity is documented against `testdata/fixtures/oracle_phase2/`
- `go test ./...` passes
