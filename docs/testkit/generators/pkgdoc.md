# Package Audit Skeleton

Reads a Go package's exported symbols, doc comments, test
files, and `//testkit:` directives, emits a compliance
audit document pre-filled with everything derivable from
static analysis. The developer fills in the domain-
specific sections (REQ descriptions, design notes,
exceptions).

## go:generate directive

```go
//go:generate testkit pkgdoc -o ../../../docs/compliance/package-audit/myapp-store.md
```

## Default output

`docs/compliance/package-audit/<package-path>.md`
(relative to repository root, derived from `.testkit.yml`
`doc_root` setting)

## What is generated

The output matches the project's package audit template
exactly. Sections that require human judgment are emitted
as placeholders; sections derivable from code are
pre-filled.

```yaml
---
package: myapp/store
audited: 2026-05-02
auditor: pending
status: draft
---
```

```markdown
<!-- Changelog: created by testkit pkgdoc -->

# myapp/store

## Purpose

<!-- from doc.go package comment -->
Store provides a key-value persistence abstraction with
transactional semantics and conflict detection.

## Scope (Phase 1)

| Property | Value |
|----------|-------|
| Interfaces | Store, Cache |
| []byte boundary | yes (Get return, Put param) |
| Stateful | yes (struct with mutex) |
| Allocation contracts | Put (0 allocs), Get (1 alloc) |
| Latency contracts | Get (< 5µs p99) |
| Reference impl | none |
| MC/DC | not applicable |
| Distributed invariants | <!-- manual --> |
| Fan-in | 12 packages import this |

## REQ inventory (Phase 1)

Auto-populated from exported methods and `//testkit:`
directives. Each method generates one contract REQ; each
`//testkit:errors` sentinel generates one failure REQ.

| REQ | Type | Behavior | Testable surface |
|-----|------|----------|-----------------|
| REQ-PKG-STORE-001 | Contract | Put persists an item | <!-- manual --> |
| REQ-PKG-STORE-002 | Contract | Get retrieves by ID | <!-- manual --> |
| REQ-PKG-STORE-003 | Contract | Delete removes by ID | <!-- manual --> |
| REQ-PKG-STORE-004 | Contract | List returns all items | <!-- manual --> |
| REQ-PKG-STORE-005 | Failure | Put returns ErrConflict on duplicate | from //testkit:errors |
| REQ-PKG-STORE-006 | Failure | Get returns ErrNotFound on missing | from //testkit:errors |
| REQ-PKG-STORE-007 | Invariant | <!-- manual --> | <!-- manual --> |

### Out-of-scope

<!-- manual: items deferred with rationale -->

## Design notes (Phase 2)

| Aspect | Notes |
|--------|-------|
| Spec fields | Factory (required); <!-- manual --> |
| Simulator tiers | <!-- manual --> |
| Fixtures | <!-- manual --> |
| StateMachine / PBT | <!-- manual: if stateful --> |

## Evidence (Phase 3)

Auto-populated from CI output when `testkit pkgdoc
--refresh` is run:

| Metric | Value |
|--------|-------|
| Coverage | <!-- from go test -cover --> |
| Mutation kill rate | <!-- from gremlins --> |
| Lint | <!-- from golangci-lint --> |
| Race | <!-- from go test -race --> |
| Bench baseline | <!-- from bench/baseline.txt --> |
| Fuzz | <!-- if []byte boundary --> |
| PBT | <!-- if stateful --> |
| Differential | <!-- if reference exists --> |
| Simulation | <!-- if distributed invariants --> |

## REQ coverage (Phase 4)

Auto-populated by joining REQ IDs against test function
names:

| REQ | Covering test(s) | Status |
|-----|-----------------|--------|
| REQ-PKG-STORE-001 | TestStore/Put_persists | covered |
| REQ-PKG-STORE-002 | TestStore/Get_retrieves | covered |
| REQ-PKG-STORE-005 | TestStore/Put_duplicate_returns_ErrConflict | covered |
| REQ-PKG-STORE-007 | <!-- no test found --> | **GAP** |

## Refactor in this audit round

<!-- manual: narrative of code changes -->

## Exceptions

<!-- manual: deviations with reviewer justification -->

## References

- [Package doc](../../store/doc.go)
- <!-- manual: ADRs, RFCs -->
```

## What is auto-filled vs. manual

| Section | Auto-filled | Manual |
|---------|------------|--------|
| Frontmatter | package, audited, status:draft | auditor, status upgrade |
| Purpose | From `doc.go` comment | Refinement |
| Scope table | Interfaces, []byte, stateful, alloc/latency contracts, fan-in | Distributed invariants, MC/DC, reference |
| REQ inventory | One row per method + one per sentinel | Behavior description, invariant REQs, out-of-scope |
| Design notes | Spec fields (Factory always) | Everything else |
| Evidence | Skeleton with metric names | Actual values (via `--refresh`) |
| REQ coverage | Join REQ IDs against test names | Review status |
| Refactor/Exceptions | Empty sections | All content |

## Refresh mode

`testkit pkgdoc --refresh` re-scans the package and
updates the auto-filled sections without touching manual
content. It uses marker comments (`<!-- from ... -->`,
`<!-- manual -->`) to distinguish auto-filled cells from
hand-written ones.

This allows the audit doc to stay current as the package
evolves — new methods get REQ rows, new sentinels get
failure REQs, coverage numbers update — without
overwriting the human analysis.

## Scale

~50 audit docs x ~120 lines = ~6,000 lines scaffolded.
~50% of each doc is auto-filled; the rest is domain analysis.
