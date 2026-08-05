---
adr: 0001
title: Record decisions as ADRs
status: Accepted
date: 2026-08-05
supersedes: none
superseded-by: none
---

# ADR-0001: Record decisions as ADRs

## Status

Accepted

## Context

The design that produced the current module split, the eidos boundary, and the
generator catalogue was worked out in a scratch document under
`docs/superpowers/`, which is gitignored. Twenty decisions were made there. None
of them were in the repository.

That is the worst of both worlds. The reasoning existed — it was written down,
argued through, and revised — but no clone contained it. Anyone reading the code
six months later would find the *outcome* of twenty decisions with no record of
the alternatives, and would either re-litigate them or route around them.

Prose in `README.md` does not fix this. A README is edited to stay current, so
the reasoning that applied when a decision was made gets overwritten by the
reasoning that applies now. Those are different documents and only one of them
answers "why did we do this?"

The sibling `ergon` repository already reached this conclusion and keeps
`docs/adr/` with a fixed record shape.

## Decision

Architectural decisions are recorded as ADRs in `docs/adr/`, one decision per
file, numbered and never renumbered. An ADR is never edited in place once
Accepted; a decision that changes is recorded as a new ADR that supersedes the
old one.

Broader statements of shape — what the platform is, how the pieces relate — are
recorded as RFCs in `docs/rfc/` and index the ADRs that fix them.

`docs/superpowers/` stays gitignored scratch. It is where thinking happens, not
where it lands.

The record shape matches `ergon`'s so the two repositories read the same way.

## Alternatives Considered

**Commit the scratch design document as-is.** Rejected: it mixes settled
decisions, volatile catalogue, and working notes in one 1,200-line artifact. It
would go stale on the first plugin and the stable decisions would be buried
under the volatile ones.

**One RFC per subsystem, no ADRs.** Rejected: an RFC that carries both the shape
and every decision behind it is edited as the shape evolves, which destroys the
frozen rationale. Splitting frozen decisions from living reference is the whole
point.

**ADRs only, no RFC layer.** Rejected: thirteen records with no document stating
what they add up to leaves a newcomer with no entry point.

## Consequences

**Positive:**

- Decisions acquire a permanent, searchable record that survives whoever made
  them.
- Re-litigating a settled question costs a link rather than an argument.
- The `Alternatives Considered` section forces rejected options to be written
  down while they are still remembered.
- `reference/` is freed to describe only the present, because the past has
  somewhere else to live.

**Negative:**

- Every architectural change now carries a documentation step, and a process
  that is inconvenient under deadline pressure is the kind that gets skipped
  first.
- ADRs accrete. Some will be wrong, and they stay on disk being wrong, because
  deleting them destroys the record.
- Backfilling twenty existing decisions is work that produces no running code.

**Neutral:**

- This ratifies a practice the project was already reaching for in commit
  bodies, rather than introducing a new one.
