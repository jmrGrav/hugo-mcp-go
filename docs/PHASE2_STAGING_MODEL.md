# Phase 2 Staging Model

## Goal

Define a mutation-safe staging environment for `create_page`, `update_page`, `delete_page`, `upload_asset`, and `build_site` without reusing the production tree directly.

## Roots

The staging model must use explicitly configured roots only:

- `STAGING_HUGO_ROOT`
- `STAGING_CONTENT_ROOT`
- `STAGING_STATIC_ROOT`
- `STAGING_PUBLIC_ROOT`
- `STAGING_WORK_ROOT`

All roots must be canonicalized at startup and rejected if:

- missing
- symlinked
- outside the allowlist
- not nested as expected

## Isolation

Each mutation run gets its own isolated staging workspace:

- `work/` for temp files, backups, and command outputs
- `content/` for page mutations
- `static/` for asset uploads
- `public/` for build output

Production roots are never mutated directly.

## Mutation workflow

### Create / update page

1. validate the target route against the staging content root
2. load or create the target file in staging
3. write to a sibling temp file
4. fsync and atomically rename
5. capture the resulting file content for parity

### Delete page

1. validate the target route
2. confirm the file exists inside the staging allowlist
3. move the file to a rollback location in `work/`
4. delete the original atomically
5. if any downstream step fails, restore from rollback

### Upload asset

1. validate filename and subfolder separately
2. resolve the destination under `static/`
3. write the decoded bytes to a sibling temp file
4. atomically rename into place
5. compare byte-for-byte with the captured oracle

### Build site

1. execute the build via an injected runner
2. collect stdout/stderr with bounded buffers
3. on success, publish or compare the generated `public/` tree
4. on failure, preserve the staging tree for inspection and do not promote it

## Path validation rules

The mutation path guard must be separate from the read-only `ResolveExistingPath` helper.

It must support:

- new files that do not exist yet
- parent-directory validation
- traversal rejection
- absolute-path rejection
- symlink escape rejection

It must reject:

- `..`
- absolute user input
- backslashes in user-controlled segments
- symlinked parents
- targets outside the allowlist after resolution

## Comparison rules

### Pages

Compare:

- route
- file path
- frontmatter map after canonical YAML normalization
- content body after newline normalization

Ignore:

- raw YAML formatting
- map key ordering
- filesystem mtimes

### Assets

Compare:

- path
- bytes
- mime type

Ignore:

- `modified`
- filesystem ordering unless the staging harness intentionally pins mtimes

### Build output

Compare:

- promoted files present in `public/`
- expected paths and byte contents

Ignore:

- nondeterministic timestamps unless they are explicitly normalized

## Rollback rules

Rollback must be explicit and bounded:

- write temp file first
- rename atomically
- keep rollback copies in `work/`
- restore on failure
- clean temp artifacts on success

If rollback itself fails, the operation must fail closed and leave the staging tree intact for inspection.

## Build wrapper

The build runner must be injected behind an interface and use `exec.CommandContext` only.

Requirements:

- no `sh -c`
- no `bash -lc`
- explicit timeout
- bounded stdout/stderr
- no command construction from raw user strings
- no access to production paths unless they are in the staging allowlist

## Fixture layout

Mutation fixtures should live under:

- `testdata/fixtures/oracle_phase2/`
- `testdata/fixtures/oracle_phase2/staging/`

Recommended sublayout:

- `create_page/`
- `update_page/`
- `delete_page/`
- `upload_asset/`
- `build_site/`

Each subdir should contain:

- request JSON-RPC input
- raw response JSON-RPC output
- side-effect snapshots
- error fixtures
- a short note on any normalization

## Testing model

Phase 2 tests should cover:

- nominal mutation flow
- invalid route / invalid filename rejection
- frontmatter conflict and immutable field rejection
- rollback on write failure
- rollback on build failure
- symlink escape rejection
- traversal rejection
- byte-for-byte asset upload comparison
- deterministic output comparison for page content and frontmatter

## Exit criteria

The staging model is ready when:

- every mutation has a dedicated staging path
- every mutation has an explicit rollback story
- every mutation comparison rule is documented
- every mutation path has a negative traversal/symlink test
- the build runner is injected, context-aware, and shell-free
