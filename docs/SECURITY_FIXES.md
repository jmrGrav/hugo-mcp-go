# Security Fixes

## Confirmed findings and fixes

### 1. `list_pages` followed a symlinked `section`

- **Finding:** `internal/hugo/pages.List` accepted a `section` path, joined it directly under the content root, and scanned that path without resolving it through the same existing-path guard used elsewhere.
- **Impact:** a symlinked section inside the content tree could redirect the scan outside the intended content root.
- **Fix:** `List` now canonicalizes the content root and resolves non-empty sections with `pathguard.ResolveExistingPath(...)` before scanning.
- **Tests added:** `TestListPagesRejectsSymlinkSection`

### 2. Page size limits were not enforced

- **Finding:** `MaxPageBytes` existed in config but page reads and page writes did not enforce it.
- **Impact:** a hostile caller could create or request arbitrarily large pages, causing memory and disk pressure.
- **Fix:** page reads now reject oversized files, page writes now reject oversized rendered pages, and the configured `MaxPageBytes` is wired into the page services.
- **Tests added:** `TestGetPageRejectsOversizedFile`, `TestCreatePageRejectsOversizedContent`, `TestUpdatePageRejectsOversizedContent`

### 3. Asset size configuration was not wired

- **Finding:** `MaxAssetBytes` existed in config but `upload_asset` used a fixed internal default.
- **Impact:** deployment-time asset size policy could not be tightened through config.
- **Fix:** `server.New` now wires `validated.MaxAssetBytes` into the asset mutation service.
- **Tests:** covered indirectly by the existing oversized upload tests and server wiring validation.

## Remaining accepted risks

- read-before-write race windows remain before the final anchored mutation hop
- list operations can still scan large trees
- build execution still relies on PATH lookup for the `hugo` binary
- no new features were added

## Refused changes

- no tree-wide dirfd refactor
- no production cutover logic
- no shell-based execution
- no plugin discovery
