# Brooks-Lint Review

**Mode:** PR Review
**Scope:** `internal/security`, `internal/hugo`, `internal/tools`, `internal/server`, `internal/runner`
**Health Score:** 90/100

The code is materially safer than the previous pass: one real symlink-scan escape was fixed, and page-size bounds are now enforced end to end. The remaining issues are accepted staging-only risks rather than immediate blockers for shadow use.

---

## Findings

### 🟡 Medium

**Dependency Disorder — Symlinked section scan in `list_pages`**
Symptom: `internal/hugo/pages.List` previously joined `section` directly under the content root and scanned the resulting path, which allowed a symlinked section path to be traversed without the existing-path guard used elsewhere.
Source: Martin — Clean Architecture — Acyclic Dependencies Principle (ADP)
Consequence: a hostile staging filesystem entry could redirect `list_pages` outside the intended content root and expose or scan unintended files.
Remedy: resolve non-empty sections through `pathguard.ResolveExistingPath(...)` and canonicalize the content root before scanning, which is now implemented.
Proof: `internal/hugo/pages/pages_test.go::TestListPagesRejectsSymlinkSection`

**Change Propagation — Unbounded page body handling**
Symptom: `MaxPageBytes` existed in config, but `pages.Get`, `create_page`, and `update_page` did not enforce it before reading or writing page bodies.
Source: Winters et al. — Software Engineering at Google — Hyrum's Law
Consequence: callers could drive memory and disk growth by sending or requesting arbitrarily large pages, and the implicit limit became undefined behavior instead of policy.
Remedy: reject oversized files on read and oversized rendered pages on write, and wire the configured page-size bound into both read and write services. This is now implemented.
Proof: `internal/hugo/pages/pages_test.go::TestGetPageRejectsOversizedFile`, `internal/hugo/mutations/pages_test.go::TestCreatePageRejectsOversizedContent`, `internal/hugo/mutations/pages_test.go::TestUpdatePageRejectsOversizedContent`

---

## Critical

None.

## High

None.

## Medium

- Symlinked section scan in `list_pages`
- Unbounded page body handling

## Low

None.

---

## Summary

The most important security correction was the `list_pages` section scan escape; it is now closed. The other meaningful hardening is the page-size bound enforcement, which removes a clear resource-exhaustion path. No production-cutover blockers were introduced by this review, and the remaining risks are documented staging-only tradeoffs rather than open exploits.

## False Positives

- `/home/jm/...` matches in tests and docs are expected and are only present as redaction fixtures or explanatory text.
- references to `Bearer <redacted>` in tests are intentional assertions for log redaction.
- `plugins` fields in oracle responses are protocol artifacts, not dynamic plugin loading.

## Accepted Risks

- residual read-before-write windows remain before the final anchored mutation hop
- large list scans can still traverse substantial directory trees
- build execution still uses PATH lookup for `hugo`

## Remaining Risks

- the threat model still assumes staging roots are not simultaneously manipulated by an external local attacker
- oracle fixtures do not model every hostile filesystem shape

## Verdict

- ready for shadow/staging: yes
- ready for production: no

### Production blockers

- residual read-before-write windows are still accepted only for staging-only use
- list scan volume is not fully bounded for arbitrarily large trees
- build execution remains a staging-only operation behind an injected runner and is not approved for production cutover
