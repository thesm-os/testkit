---
adr: 0016
title: Directives are positive-only
status: Accepted
date: 2026-08-05
supersedes: 0008
superseded-by: none
---

# ADR-0016: Directives are positive-only

## Status

Accepted

## Decision

testkit's directives use the neutral form with an axis qualifier, batching
several values per line:

```go
//testkit:mixin idempotent concurrent bounded
//testkit:contract lease role=acquire release=Release
```

That much is unchanged from
[ADR-0008](0008-neutral-directive-form-with-axis-qualifier.md).

**The negated form is not part of the vocabulary.** There is no
`//-testkit:mixin`, and testkit does not document, emit, or fixture one.

## Context

ADR-0008 recorded that the `-` form "stays available for suppression, which is
the case where negation is the whole point", and gave
`//-testkit:mixin concurrent-safe` as the example. Both claims are false
against the substrate, and neither was checked before the record was written.

`shape.Plugin.Directives()` declares `DenyNegation()` on the mixin directive.
`//-testkit:mixin idempotent` is therefore not a no-op that suppresses nothing —
it fails to parse. Every mixin fixture in the conformance corpus was written to
ADR-0008's example and would have failed codegen.

The `shape` and `contract` directives do not deny negation, so the form parses
there. It still does nothing: `matchFromDirective` skips any directive with
`Negated` set, and control falls through to signature detection, which stamps
the shape regardless. Negation is accepted and then ignored.

There is a coherent reason for the asymmetry, and it is the reason negation is
not needed. **Mixins are opt-in and shapes are inferred.** A mixin appears only
because someone wrote the directive, so there is nothing to switch off —
deleting the directive is the suppression. A shape is detected from the
signature whether or not anyone asked, so suppression would be meaningful
there; eidos simply has not implemented it.

Recording a capability the substrate does not have is worse than recording its
absence. It propagated into 28 fixtures before anything executed, and it would
have propagated into the generated output and consumer documentation next.

## Alternatives Considered

**Keep ADR-0008 and treat negation as a future capability.** Rejected: a
decision record that documents a form nobody can write is a trap for the next
reader, and this one had already sprung once.

**Request suppression semantics upstream and keep the vocabulary.** Rejected as
a precondition, not as an idea. Shape suppression is worth having and is filed
separately; but the vocabulary should describe what works today, and a directive
that parses and is then ignored is worse than one that is rejected.

**Ask eidos to deny negation on `shape` and `contract` as well.** Deferred to
the upstream discussion. It would make the failure loud, which is right, but it
forecloses the suppression semantics that would make the form useful.

## Consequences

**Positive:**

- The documented vocabulary is what the substrate accepts, so a fixture written
  from the docs parses.
- The mixin corpus tests the real distinction — directive present versus
  directive absent — rather than a form that cannot be written.
- The asymmetry now has a stated reason rather than being an accident of which
  builder call each schema happened to make.

**Negative:**

- A consumer who wants a detected shape ignored has no way to say so, and this
  record does not give them one. That gap is real and now explicit.
- Superseding an ADR eight records after it means readers of the earlier one
  reach a false statement before reaching the correction, unless they read the
  index first.

**Neutral:**

- Nothing generated or shipped depended on the negated form; the error was
  caught while the corpus was being written and before any generator consumed
  it.
