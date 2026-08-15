---
adr: 0004
title: Consume only eidos's annotator plugin
status: Accepted
date: 2026-08-05
supersedes: none
superseded-by: none
---

# ADR-0004: Consume only eidos's annotator plugin

## Status

Accepted

## Context

[ADR-0003](0003-adopt-eidos-as-the-codegen-substrate.md) settles that testkit
builds on eidos. It does not settle how much of eidos's plugin set testkit
consumes.

eidos ships two plugin families. `plugins/annotator` classifies Go declarations
and stamps typed metadata onto them. `plugins/generator` emits code from that
metadata, and three of its generators — `builder`, `enum`, `sentinel` — share
names with three of testkit's fourteen.

The name collision is nominal. eidos's `builder` emits a fluent builder;
testkit's emits a builder *and* the tests that prove it round-trips, and reads
directives (`//testkit:default`) that eidos has no reason to know. The same
asymmetry holds for `enum` and `sentinel`. Every testkit generator emits the
artifact plus its proof; that property is the product, and it is not what
eidos's generators do.

Annotation is the opposite case. The `shape` annotator's three orthogonal axes —
twenty detectors, twenty-four contracts, twenty-eight mixins — produce
seventy-two classifications (as of this record; see ADR-0018 for the live
count), all of which the CLI already registers via
`.All()`. testkit's own classification packages reimplemented thirty-nine of
forty-four equivalents, and the audited gaps close upstream.

## Decision

testkit consumes `plugins/annotator` and nothing else from eidos's plugin set.

All fourteen generator plugins are testkit's own, including `builder`, `enum`,
and `sentinel`.

testkit writes no annotators. Classification is configured, not implemented.

## Alternatives Considered

**Adopt eidos's three overlapping generators.** Rejected: they emit the artifact
without the test that exercises it, which is the property that distinguishes
testkit's output. Closing the gap would mean either pushing testkit's directive
vocabulary upstream — making eidos carry testing concerns it has no reason to —
or forking the three, which is adoption in name only.

**Write testkit-specific annotators alongside eidos's.** Rejected: forty-one of
testkit's forty-four classification packages already exist upstream and are
already registered.
Writing more would recreate the drift that
[ADR-0003](0003-adopt-eidos-as-the-codegen-substrate.md) exists to end. The three
testkit-only classifications become upstream contributions or configuration, not
a parallel annotator set.

**Consume both families and override selectively.** Rejected: two plugin idioms
across fourteen generators means every contributor learns both and every
generator's provenance becomes a lookup. One idiom for all fourteen is worth
more than three generators of reuse.

## Consequences

**Positive:**

- One plugin idiom across all fourteen generators. A contributor learns the
  shape once.
- The classification vocabulary has a single owner, so a detector fix lands
  once.
- The boundary is stateable in a sentence, which means it can be enforced by a
  depguard rule rather than by review.

**Negative:**

- Three generators are written here that already exist upstream. That is
  duplicated intent even though it is not duplicated behaviour.
- testkit's classification needs now arrive as upstream feature requests. A
  classification eidos declines to add has no local escape hatch short of
  forking.
- The seventy upstream classifications have not yet been audited one by one
  against what the fourteen generators actually need. That audit is outstanding
  and may surface gaps this record assumes away.

**Neutral:**

- The annotator is configured through testkit's own directive namespace, so the
  consumer-facing vocabulary is unaffected by which side implements what.
