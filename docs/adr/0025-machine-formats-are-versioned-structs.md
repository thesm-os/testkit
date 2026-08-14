---
adr: 0025
title: Machine-read formats are versioned structs
status: Accepted
date: 2026-08-14
supersedes: none
superseded-by: none
---

# ADR-0025: Machine-read formats are versioned structs

## Status

Accepted

## Context

Three artifacts are read by machines: the run report (CI and tooling), the
conformance statement (published claims, badges), and the lock file
(review tooling). The first draft derived the conformance statement from
the test log text. The review pointed out the problem: as soon as one
script parses log output, the wording becomes an API and can never be
improved.

The draft also called the run report a "census", but that word already
names the generation-time gates in `conformance/gate`.

## Decision

Every machine-read format is a versioned struct owned by the `suite`
package:

- `testkit-report v1` — the run report.
- `testkit-conformance v1` — the conformance statement, built from the
  report and the lock digest.
- `testkit-checks v1` — the lock file
  ([ADR-0023](0023-a-lost-manifest-line-fails-check.md)).

All three are additive-only: fields can be added, never changed or
removed. A breaking change gets a new version header. Log text is
rendered from the structs and is never parsed.

The run report type is named `Report`. "Census" keeps its existing
meaning (the gates).

## Alternatives Considered

**Log text as the interface.** No new API surface, but wording becomes
frozen the moment anyone scripts against it, and a public conformance
claim would rest on prose. Rejected.

**One combined format.** Fewer schemas, but the three have different
lifecycles: the report is per run, the statement is per published claim,
the lock is per regeneration. Combining them couples their evolution.
Rejected.

**Version later, when someone needs it.** By then the first consumer has
frozen whatever shape existed by accident. A version header costs minutes
now; a migration costs much more later. Rejected.

## Consequences

**Positive:**

- Log wording can be improved at any time.
- Each format has an explicit, testable stability promise.

**Negative:**

- Three schemas with golden tests to maintain, and additive-only means a
  badly named field stays forever.

**Neutral:**

- The remaining uses of "census" in the tree all mean the gates, which
  keep the name.
