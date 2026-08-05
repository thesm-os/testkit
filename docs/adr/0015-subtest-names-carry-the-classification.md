---
adr: 0015
title: Subtest names carry the classification
status: Accepted
date: 2026-08-05
supersedes: none
superseded-by: none
---

# ADR-0015: Subtest names carry the classification

## Status

Accepted

## Context

The conformance corpus proves a generated suite *detects* violations, not merely
that it runs, by pairing each fixture with a deliberately-broken implementation
and asserting the suite fails against it.

That assertion is worth very little if it checks failure *existence*. A broken
implementation that panics in its constructor fails the suite. So does one that
violates a different law than the one its fixture exists to prove. Both produce
a red suite and a green gate, and the gate has verified nothing beyond "some
code is wrong somewhere". The obligation to break the *right* law becomes a
matter of care rather than something enforced, and care does not survive
seventy-odd fixtures.

Checking failure *identity* fixes it: `iface/mixin/idempotent`'s broken
implementation must fail the `idempotent` subtest specifically. A constructor
panic fails a different subtest, or none, and the gate reports it.

That only works if subtest names carry the classification in a form a matcher
can read. The previous generator's names were already close — `Count/bounded 0
1000`, `Get/errors ErrNotFound` — a method segment followed by prose that
usually begins with the classification. Usually is not a guarantee, and
`Cancel/ctx cancellation` maps to no classification at all.

The timing matters more than the rule. Generated-file layout and naming is one
of the four surfaces `v1.0.0` freezes
([ADR-0002](0002-support-external-consumers-under-semver.md)), and subtest names
are what a consumer's CI output, test filters, and REQ traceability all key on.
Fixing the convention before the suite plugin exists costs a paragraph. Fixing
it afterwards is a retrofit across every generated file and every consumer's
`-run` expressions.

## Decision

A generated subtest that exists because of a classification is named
`<Method>/<classification>`, optionally followed by detail:

```go
t.Run("Put/idempotent", ...)
t.Run("Count/bounded 0 1000", ...)
t.Run("Get/errors ErrNotFound", ...)
```

The guarantee is narrow and mechanical: **the second path segment begins with
the classification's canonical name**, as eidos reports it, followed by either
nothing or a space and human-readable detail.

Subtests that exist for reasons other than a classification — the baseline
smoke case, context cancellation, nil-context handling — keep descriptive names
and carry no such guarantee. They are not classification subtests, so nothing
needs to match them.

The corpus gate matches on that prefix. A fixture whose broken implementation
fails any other subtest is a gate failure.

## Alternatives Considered

**Check that the suite failed, without checking which subtest.** Rejected: it
is the current design and it is why this record exists. It cannot distinguish a
correctly-broken fixture from a fixture broken by accident, which is precisely
the case that accumulates.

**Name subtests `<classification>/<Method>`.** Rejected: it groups output by
law rather than by method, so `go test -v` on a twenty-method interface
interleaves methods under each law heading. Reading a failure means
reconstructing which method it belonged to. Method-first also matches the
previous generator, so consumers' existing `-run` expressions survive.

**Emit the classification as a structured attribute rather than in the name —
`t.Attr` or a log line.** Rejected: subtest names are what `-run` filters,
CI report parsers, and REQ traceability all read. An attribute would be more
precise and invisible to every one of them.

**Require the whole second segment to equal the classification name.**
Rejected: it discards the detail that makes a failure readable. `bounded 0
1000` says which bound was violated; `bounded` alone sends the reader to the
source.

## Consequences

**Positive:**

- The corpus gate enforces that a broken fixture breaks the law it claims to,
  turning a discipline problem into a test failure.
- `go test -run 'Put/idempotent'` addresses one law on one method, which is the
  granularity a developer debugging a failure actually wants.
- REQ traceability keys on a name whose structure is now stated rather than
  incidental.

**Negative:**

- The suite plugin must thread the canonical classification name through to
  every emitted subtest, so the name is no longer free-form prose the template
  can phrase for readability.
- Renaming a classification upstream renames subtests here, which breaks
  consumers' `-run` expressions. That is a break the deprecation cycle in
  [ADR-0002](0002-support-external-consumers-under-semver.md) has to cover.
- The convention binds before a single suite has been generated, so it is a
  prediction about what reads well at scale rather than an observation.

**Neutral:**

- The shape matches the previous generator's output, so this constrains an
  existing convention rather than introducing one.
