---
adr: 0013
title: Defer codec, pkgdoc, and smoke
status: Accepted
date: 2026-08-05
supersedes: none
superseded-by: none
---

# ADR-0013: Defer codec, pkgdoc, and smoke

## Status

Accepted

## Context

The catalogue names fourteen generators. Three of them do not fit the substrate
the other eleven are built on, and the mismatch is structural rather than a
matter of effort.

**`codec` and `pkgdoc` need a non-Go backend.** Both emit artifacts that are not
Go source: `codec` produces wire fixtures and fuzz seeds alongside its Go output,
and `pkgdoc` produces a Markdown audit document. eidos's sink writes Go. Adding a
second backend is an upstream design question with consequences for determinism,
formatting, and the sink contract — not something to resolve as a side effect of
porting a generator.

**`smoke` needs value-driven input.** Every other generator reads types: it takes
an interface or a struct and derives what to emit from the shape. `smoke` covers
CLI commands, and what a command does is determined by the values passed to it —
flags, arguments, subcommand paths — which are constructed at runtime and are not
recoverable from the type. The frontend that feeds the other eleven has nothing
useful to hand it.

Attempting all fourteen at once means the three hardest, least-similar ones set
the pace for the eleven that are ready.

## Decision

`codec`, `pkgdoc`, and `smoke` are deferred. They stay in the catalogue and keep
their reference documentation, marked as deferred with the reason above.

The eleven remaining generators are the scope.

Deferral is not cancellation. `codec` and `pkgdoc` become viable if eidos grows a
non-Go backend. `smoke` needs a design for value-driven input that does not
exist yet.

## Alternatives Considered

**Attempt all fourteen.** Rejected: the three blocked ones do not become
unblocked by being scheduled. They would either stall the port or ship in a form
that works around the substrate rather than on it.

**Cut them from the catalogue entirely.** Rejected: the design work is done and
documented, and the blockers are external and plausibly temporary. Deleting the
documentation would mean redoing the thinking when the blocker clears.

**Build `codec` and `pkgdoc` outside eidos, as standalone tools.** Rejected: it
creates a second code-generation path with its own frontend, its own determinism
story, and its own caching — which is the thing
[ADR-0003](0003-adopt-eidos-as-the-codegen-substrate.md) exists to stop.

**Build `smoke` from runtime reflection over the command tree.** Rejected: it
would require importing and executing the consumer's CLI at generation time,
which is a different and much larger trust and determinism problem than reading
source.

## Consequences

**Positive:**

- The scope is eleven generators that share one substrate, one idiom, and one
  set of blockers.
- The blockers are stated, so it is clear what would have to change for the
  three to resume rather than leaving them as unexplained gaps.

**Negative:**

- The catalogue advertises fourteen and delivers eleven, so a reader has to
  check the status of each rather than trusting the list.
- Deferred documentation goes stale. Three reference documents will describe an
  intended design against a substrate that keeps moving.
- Two of the three are blocked on an upstream decision nobody has scheduled, so
  "deferred" has no expected resolution date.

**Neutral:**

- No consumer depends on the three today, so the deferral breaks nothing.
