---
adr: 0017
title: Every classification owes an assertion
status: Accepted
date: 2026-08-07
supersedes: none
superseded-by: none
---

# ADR-0017: Every classification owes an assertion

## Status

Accepted

## Context

eidos registers seventy-two classifications across three orthogonal axes:
twenty detectors, twenty-eight mixins, twenty-four contracts. The counts in this record are as of its acceptance; the vocabulary grows upstream and `conformance/gate.Registered` reads the live registries. It stood at one hundred and four when ADR-0018's evidence census landed.
A callable carries
one detector stamp and any number of mixin and contract memberships, so the axes
combine rather than compete.

The vocabulary is not testkit's.
[ADR-0004](0004-consume-only-the-annotator-plugin.md) committed to consuming it
whole, which means it grows and changes upstream and testkit learns about the
change when the dependency moves.

`conformance/gate` already treats that as an obligation rather than an event. It
builds its expected set from the live registries rather than from a checked-in
table, so a classification added upstream appears in the gate on the next build
and fails it until the corpus carries a fixture. There is one fixture directory
per classification today.

The classifications differ in how much a single call can demonstrate. One call
settles `nilsafe` completely: the method either tolerates zero inputs or it does
not. One call settles nothing about `idempotent`, which needs a second call to be
stateable at all and is only *proved* by comparing a generated sequence of
operations against a reference implementation.

The conformance tiers differ in cost and in the kind of wrongness each can see,
not in which classifications they cover. A cheap direct assertion and an
exhaustive behavioural proof of the same classification are different evidence
for the same claim, and a subject can warrant one without warranting the other.

That leaves a question the corpus cannot answer on its own: whether a
classification whose direct assertion is weak should get one at all.

## Decision

Every classification eidos registers owes an assertion in the generated
conformance suite. The corpus fixture for that classification is the proof
obligation, and a classification with no assertion fails the gate.

An assertion is the direct form: the shortest sequence of calls that can
demonstrate the classification, run once. Where the direct form is weaker than
the classification's full statement, the generated assertion says so at the point
a reader will look — in its own documentation — rather than leaving the strength
to be inferred from a passing subtest.

The exhaustive form is separate evidence and belongs to the model tier, which
compares generated histories against a reference implementation. Both tiers cover
the same classification. The suite tier's unit is a `Check`; `law` is the model
tier's word and is not reused.

## Alternatives Considered

**Curate: assert only where the direct form is strong.** Rejected on two
grounds. A generated file that is silent about a classification the source
declared is indistinguishable from one where the generator failed to handle it,
so the reader cannot tell a principled omission from a defect. And curation has
no owner — the judgement falls to whoever adds the next classification, with
nothing recording why the previous ones went the way they did.

**Assert only what the model tier cannot reach, and defer the rest.** Rejected:
the model tier costs a reference implementation, which many subjects will never
have. Deferring on that basis leaves classifications checked by nothing at all,
for subjects whose authors declared them deliberately.

**Keep a table of which classifications are covered, and grow it
deliberately.** Rejected: `gate.Registered()` reads the registries specifically
so that upstream growth surfaces as a failure. A table reintroduces the drift
that design exists to prevent, one build behind.

**Give the suite tier the exhaustive form too, and have no model tier for
classifications.** Rejected: `engine/model/law` implements that form against a
reference with shrinking. Two implementations of one property is drift, and the
weaker one would be the one that runs by default.

## Consequences

**Positive:**

- A classification added upstream fails the gate by name, so the work it implies
  is discovered rather than remembered.
- A consumer reading a generated file finds every classification their source
  declared, which makes the file answer "what is being checked here" without
  reference to the catalogue.
- No per-classification judgement about whether an assertion is worth emitting,
  so there is nothing to adjudicate and nothing to drift.

**Negative:**

- Some assertions are weak. The direct form of `idempotent` — call twice, the
  second must not newly fail — passes for implementations that are not
  idempotent. The obligation to state that limit in the generated documentation
  is what keeps the assertion from overclaiming, and it is an obligation on
  prose, which no gate enforces.
- Seventy-two assertions is a large surface, and each needs a generated
  violation to demonstrate it can fail, which roughly doubles it.
- An upstream classification whose direct form is not obvious blocks the gate
  until someone writes one. The only unblocking move is to write a weak
  assertion, which is a pressure toward exactly the assertions this record
  concedes are unsatisfying.

**Neutral:**

- How assertions are emitted, extended, and proven is design, and lives in
  [RFC-0002](../rfc/0002-the-suite-generator.md).
- The composite axis is unaffected: a method carrying a detector, a mixin and a
  contract owes all three assertions, because the axes are orthogonal and the
  obligation is per classification.
