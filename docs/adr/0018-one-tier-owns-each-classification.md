---
adr: 0018
title: One tier owns each classification
status: Accepted
date: 2026-08-10
supersedes: 0017
superseded-by: none
---

# ADR-0018: One tier owns each classification

## Status

Accepted

## Context

eidos registers its classifications across three orthogonal axes — detectors,
mixins and contracts — and a callable carries one detector stamp and any number
of mixin and contract memberships. The vocabulary was seventy-two when this
record was accepted and is one hundred and four as of the last revision
(twenty-two, fifty-six, twenty-six); the mixin axis has roughly doubled. No
number here is authoritative, and that is the point of the next paragraph.
[ADR-0004](0004-consume-only-the-annotator-plugin.md) committed to consuming the
vocabulary whole, so it grows upstream and testkit learns of the change when the
dependency moves. `conformance/gate` builds its expected set from the live
registries rather than a checked-in table, so a classification added upstream
appears in the gate on the next build and fails it until the corpus carries a
fixture.

testkit has two tiers of evidence, and they differ in what they can reach.

The suite tier makes a fixed sequence of calls against one subject, obtained
from a `factory func() T`. That is everything a consumer must supply.

The model tier drives rapid-generated action sequences against a subject and a
reference implementation, with shrinking, linearizability checking and goroutine
leak detection. `engine/model/law` implemented seventy-one laws for it when
this record was accepted and implements eighty-three now. They are
observational — a law reads from the subject and the reference and never writes,
because mutation belongs to the action handlers — and most compare the two:
`AUTO-READ-AFTER-WRITE` asserts that subject and reference agree on every key
drawn from the pool, after any sequence the run generated.

**Classifications differ in what evidence can establish them at all.** One call
settles `nilsafe` completely: the method either tolerates zero inputs or it does
not. Several cannot be *stated* against a single subject making a fixed call —
`atomic` and `transaction` need a call that fails on demand, `singleflight`
needs concurrent callers, `crdtmerge` needs two replicas, `cas` needs
accumulated version state, `rate-limit` needs controlled time. A tier that
cannot produce those conditions does not check such a classification weakly; it
checks nothing, and reports success.

The remedy would be for the cheaper tier to grow failure injection, concurrent
drivers, multi-subject setup and a clock — which is the model runner, rebuilt
beside the model runner, and paid for by every consumer whether or not their
interfaces carry those classifications.

[ADR-0017](0017-every-classification-owes-an-assertion.md) placed the obligation
on the suite tier specifically. Discharging it as written means implementing
sixty-odd properties a second time, as generated templates, against Go
implementations that already exist and are tested — with nothing relating the
two copies.

## Decision

**Every classification owes an assertion in at least one tier**, and the
conformance gate measures the union. A classification covered by neither tier
fails the gate.

**Each classification is owned by exactly one tier** — the cheapest that can
state it without vacuity and without borrowing machinery from another tier.

**The suite tier implements no property `engine/model/law` already carries.**
Where a law exists, the classification is the model tier's. Where none does —
signature-derived hygiene, and the classifications the law catalogue does not
reach — the suite tier owns it.

**A generated file names what its own tier does not cover**, and which tier
does, in the header a reader meets before the checks. An uncovered
classification is stated, never omitted.

## Alternatives Considered

**Keep ADR-0017 as written: both tiers cover every classification, direct form
and exhaustive form.** Rejected on two grounds. It implements sixty-odd
properties twice in two languages with nothing relating the copies, which is the
drift this repository has spent its history removing. And for the
classifications with no honest single-call form it does not produce weak
evidence but absent evidence: a check for "a stale version is refused", written
where no stale version can be produced, passes against every implementation
including a broken one.

**Let the suite tier grow the machinery.** Rejected: failure injection,
concurrent drivers, multi-subject setup and a controlled clock are what
`engine/model` is. A second implementation would be the weaker one, and it would
be the one that runs by default.

**Drop the suite tier and generate only model bindings.** Rejected: the model
tier costs a reference implementation, which many subjects will never have.
Every subject without one would then be checked by nothing, including for the
signature-derived properties that need no reference at all.

**Curate: assert only where the evidence is strong.** Rejected for the reasons
ADR-0017 gave, which stand. A generated file silent about a classification the
source declared is indistinguishable from one where the generator failed to
handle it, and the judgement has no owner — it falls to whoever adds the next
classification, with nothing recording why the previous ones went as they did.
This record does not restore that judgement: assignment follows from what the
claim needs, which is a property of the classification rather than an opinion
about it.

## Consequences

**Positive:**

- One implementation per property. A law fixed in `engine/model/law` is fixed
  for every consumer, with no template carrying a second answer.
- The suite tier stays cheap and always runnable: a `factory` is the whole
  input, so a subject with no reference implementation still gets every
  signature-derived and shape-derived check.
- A reader of a generated file learns which tier covers what, so an absent check
  is a stated boundary rather than a suspected defect.
- A classification added upstream still fails the gate by name, so the work it
  implies is discovered rather than remembered.
- The union this record decided on is now measured rather than intended.
  `conformance/gate.Evidenced` runs the real generators over the corpus and
  asks, per registered classification, whether either tier asserts it; anything
  neither asserts must carry a row in
  `conformance/gate.UnevidencedClassifications` saying what the claim is
  waiting on. Both directions fail: an unargued gap reddens, and so does a row
  for a classification a tier has since covered. Fifteen rows opened it, of
  which the four this record already named in prose are now four of the
  fifteen — which is the difference the register buys. Prose stales silently.

**Negative:**

- A consumer without a reference implementation gets no evidence for the
  model-tier classifications their source declares. The header says so, which is
  honest, but it remains a gap that only writing a reference closes.
- The assignment is a judgement made once per classification, and the gate
  measures the union without measuring whether each was assigned correctly. A
  classification parked in the model tier to avoid designing a suite check
  would not be caught — the census sees evidence, not whether the evidence
  came from the right tier.
- The register's rows are arguments, and an argument can be wrong in a way a
  build cannot see. What the census enforces is that one exists, is specific,
  and stops being accepted the moment a tier covers the classification.
- `engine/model`'s laws have to be reachable from a generated binding. The
  model generator produces one, so this is discharged; the census reads those
  bindings back to decide whether a classification's laws are evidence at all,
  and a law in the catalogue for a fixture the model tier never runs is
  evidence nowhere.

**Neutral:**

- How each tier emits, extends and proves its checks is design, and lives in
  [RFC-0002](../rfc/0002-the-suite-generator.md).
- The composite axis is unaffected: a method carrying a detector, a mixin and a
  contract owes all three assertions, in whichever tier owns each.
