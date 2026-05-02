# REQ Traceability

Verifies end-to-end traceability from requirements to
tests: every REQ in every audit doc has a covering test,
and every test that claims to cover a REQ actually
exercises the relevant code path.

## Command

```
testkit validate reqs
```

## What it checks

### Forward traceability: REQ → test

1. Every `REQ-PKG-*` row in every audit doc's Phase 4
   table has at least one covering test name
2. Every cited test name actually exists in the package's
   test files (catches typos and renamed tests)
3. Every cited test runs successfully (optional: `--run`
   flag to execute)

### Reverse traceability: test → REQ

1. Every `Test*` function in a package with an audit doc
   appears in at least one REQ coverage row (catches
   tests that exist but aren't traced to a requirement)
2. Tests that cover zero REQs are flagged as untraceable
   — they may be legitimate (helpers, benchmarks) but
   should be reviewed

### Cross-reference integrity

1. REQ IDs follow the naming convention
   `REQ-PKG-<PACKAGE>-<NNN>` — malformed IDs are flagged
2. REQ IDs are unique across the entire audit corpus —
   duplicates are flagged
3. REQ numbering is contiguous per package — gaps are
   flagged (suggests a deleted REQ)

## Failure output

```
reqs: FAIL

  myapp/store audit doc:
    REQ-PKG-STORE-007: cited test "TestStore/invariant_chain"
      does not exist — test was renamed or deleted

    TestStore/Delete_idempotent: exists in test files but
      not traced to any REQ — add to Phase 4 table or
      document as non-REQ helper

  myapp/cache audit doc:
    REQ-PKG-CACHE-003 → REQ-PKG-CACHE-005: gap at 004 —
      was a REQ deleted? Add back or renumber
```

## Why

The runbook's Phase 4 is the most-skipped phase. Without
automated verification, REQ-to-test mappings drift:

- Tests are renamed without updating the audit doc
- New methods are added without new REQs
- REQs are removed to "clean up" the mapping instead of
  writing the missing test

The validator makes Phase 4 a CI gate rather than a
human discipline check.

## Configuration

```yaml
# .testkit.yml
validators:
  reqs:
    enabled: true
    doc_root: docs/compliance/package-audit
    # Convention for REQ IDs.
    id_pattern: "REQ-PKG-{PACKAGE}-{NNN}"
    # Whether to flag untraceable tests as errors
    # (vs. warnings).
    strict_reverse: false
```
