# Audit Completeness

Verifies that every package has an audit document and
that every audit document is complete — no placeholder
sections, no uncovered REQs, no missing evidence.

## Command

```
testkit validate audit
```

## What it checks

### Coverage: every package has an audit doc

1. Every directory with a `go.mod` or `doc.go` has a
   corresponding `docs/compliance/package-audit/<path>.md`
2. No orphaned audit docs exist for packages that have
   been deleted or renamed

### Completeness: no placeholder gaps

For each audit doc:

1. Frontmatter has `status` set to `elite`, `complete`,
   or `in-progress` (not `draft` or missing)
2. No `<!-- manual -->` placeholders remain in required
   sections (Scope, REQ inventory, Evidence)
3. Every REQ row has a non-empty Behavior column
4. Every REQ row in the Phase 4 coverage table has status
   `covered` (no `**GAP**` markers)
5. Evidence table has actual values (not `<!-- from -->`)

### Freshness: auto-filled sections match current code

1. Exported methods in the package match the REQ method
   rows — new methods without REQs are flagged
2. `//testkit:errors` sentinels match the failure REQ
   rows — new sentinels without REQs are flagged
3. Fan-in count matches current `go list -deps` output

## Failure output

```
audit: FAIL

  myapp/store
    REQ-PKG-STORE-007: status is **GAP** — no covering test
    new method Delete added since last audit — no REQ row

  myapp/cache
    no audit doc found — create with: testkit pkgdoc myapp/cache

  myapp/legacy
    audit doc exists but package was deleted — remove stale doc
```

## Severity levels

| Level | Meaning |
|-------|---------|
| ERROR | Uncovered REQ, missing audit doc, stale doc |
| WARN | `draft` status, placeholder in optional section |
| INFO | Fan-in count changed, new method detected |

Errors fail CI. Warnings are reported but don't fail.

## Configuration

```yaml
# .testkit.yml
validators:
  audit:
    enabled: true
    doc_root: docs/compliance/package-audit
    # Packages exempt from audit (e.g., internal helpers).
    exclude:
      - internal/testutil
    # Minimum status to pass CI (default: complete).
    min_status: complete
```
