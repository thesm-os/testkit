---
adr: 0028
title: One tier owns each obligation
status: Accepted
date: 2026-08-20
supersedes: 0018
superseded-by: none
---

# ADR-0028: One tier owns each obligation

## Status

Accepted

## Context

[ADR-0018](0018-one-tier-owns-each-classification.md) settled how testkit
divides evidence between the suite tier and the model tier, and its reasoning
holds: a tier that cannot produce a condition does not check a classification
weakly, it checks nothing and reports success. Nothing here disturbs that.

What it got wrong is the unit. Its decision line reads:

> **Each classification is owned by exactly one tier** — the cheapest that can
> state it without vacuity and without borrowing machinery from another tier.

The shipped code already disagreed with that sentence on the day it was written.
`ttl` is law-backed in the tiers catalogue *and* emits a `Read/miss` row under
`ClassReader`, because the sentinel the ttl declaration names is what turns that
row's body from the zero arm into the sentinel arm. `bounded` is law-backed and
its real check is a suite row. Neither is a mistake — both are classifications
carrying more than one obligation, with the obligations landing in different
tiers.

Reading ADR-0018 literally makes those look like violations, which invites the
wrong repair: pick a tier and drop the other row. That loses a real check to
tidy a sentence.

Building out the remaining derivation rules made the shape plain. `validates`
carries two obligations — that the method and its named validator agree about a
value, and that a refused value left nothing behind. The first needs nothing a
caller does not have; the second needs a reader the directive names no parameter
for. `accumulates` carries two — that a repeat is accepted, and that N calls
compound. The first is a fixed sequence on one subject; the second needs a
reference. In both cases the classification is not owned by a tier at all. Its
obligations are.

## Decision

**Every classification owes an assertion in at least one tier**, and the
conformance gate measures the union. Unchanged from ADR-0018.

**Each OBLIGATION a classification carries is owned by exactly one tier** — the
cheapest that can state it without vacuity and without borrowing machinery from
another tier. A classification carrying several may therefore be checked in
several tiers, each stating the part it can reach.

The deciding question, per obligation: *what does it need that a caller does not
have?* Nothing, and it is the suite tier's. A reference implementation,
generated sequences, a clock, concurrency, induced failure, or a second subject,
and it is the model tier's. The process or the medium dying, and it is the sim
tier's.

**The suite tier implements no property `engine/model/law` already carries.**
Unchanged, and now read at obligation granularity: a law covering one obligation
of a classification says nothing about its others.

**A derived claim states the obligation it covers, not the classification.**
Where the suite tier reaches part of a classification, the emitted claim says
which part — `Add is accepted a second time rather than refused as a repeat`,
not `Add accumulates`. A claim wider than its check is the vacuity this record
and ADR-0018 both exist to prevent.

**A generated file names what its own tier does not cover**, and which tier
does, in the header a reader meets before the checks. Unchanged.

`docs/internal/classification-checks.md` is the register: every detected shape,
mixin and contract, with its obligations split across the three tiers.

## Alternatives Considered

**Leave ADR-0018 as written and treat the two-tier classifications as
exceptions.** Rejected. There are eighty of them out of roughly a hundred — a
rule with eighty exceptions is not a rule, and the exceptions were being
discovered one at a time by whoever next hit one.

**Narrow the classifications instead: split `validates` into
`validates-agrees` and `validates-discards` upstream.** Rejected. The
vocabulary is eidos's and describes what an author means, not how testkit
checks it. An author writing `//testkit:mixin validates fn=Validate` is stating
one thing about their method; making them state two so our tier boundary lands
cleanly moves our problem into their annotation.

**Drop the ownership rule and let each tier check whatever it can.** Rejected
for the reason ADR-0018 gave: it reinstates the duplicate-implementation problem
at obligation granularity, and the weaker copy is the one that runs by default.

## Consequences

**Positive:**

- The register is checkable. Each row names an obligation and a tier, and the
  deciding question above settles disputes without an opinion.
- Real checks stop looking like violations. `ttl`'s miss row and `bounded`'s
  suite row are the rule working, not exceptions to it.
- The census in `generator/suite/derive_stamps_test.go` gains an honest third
  state: a classification whose suite-tier obligation is discharged and whose
  model-tier obligation waits on the model generator is recorded as owned, not
  as pending.

**Negative:**

- A classification can now appear in two tiers' output, and a reader counting
  coverage has to read the claims rather than the classification names. The
  emitted claims are worded to make that possible; nothing enforces it beyond
  review.
- Splitting obligations is a judgement, and a wrong split is a claim wider than
  its check. The deciding question narrows it but does not remove it.

**Neutral:**

- No generated output changes on adoption. This records what the derivation
  rules already do and what the register already documents.
