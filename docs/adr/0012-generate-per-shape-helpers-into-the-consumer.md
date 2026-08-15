---
adr: 0012
title: Generate per-shape helpers into the consumer
status: Accepted
date: 2026-08-05
supersedes: none
superseded-by: none
---

# ADR-0012: Generate per-shape helpers into the consumer

## Status

Accepted

## Context

Conformance suites and benchmarks are generated per shape. A method classified as
a reader gets reader assertions; a lease-shaped method gets lease assertions.
That per-shape logic lived in a shipped `harness/` package — roughly 8,800 lines
of suite and bench helpers, one entry point per shape, imported by the generated
code.

That put the shape vocabulary into a published API. And the shape vocabulary is
not testkit's to fix: it comes from eidos's annotator
([ADR-0004](0004-consume-only-the-annotator-plugin.md)), where twenty
detectors, twenty-four contracts, and twenty-eight mixins — the vocabulary as
of this record, since grown; ADR-0018 carries the live count — are actively
maintained. Upstream adds a mixin and the published surface grows; upstream
renames one and the published surface breaks.

The failure is worse than churn. Under
[ADR-0002](0002-support-external-consumers-under-semver.md) the runtime surface
is additive-only within a major, so every upstream vocabulary change would either
be blocked or would force a major version. A published API whose shape is decided
in another repository cannot honour a stability promise.

Generated code that calls a shared helper also has a second failure mode: the
generated file and the helper it calls can be at different versions. Regenerating
is not enough; the runtime has to be upgraded in step.

## Decision

Per-shape suite and bench helpers are generated into the consumer's package as
part of the generated file. They are not shipped as a shared runtime API.

The 8,800 lines leave `harness/` and become plugin templates in the `tool`
module.

The consumer's generated file is self-contained with respect to shape logic: it
depends on runtime primitives, which are stable, not on per-shape entry points,
which are not.

## Alternatives Considered

**Keep `harness/` as a published package.** Rejected: it puts a vocabulary
testkit does not own into a surface testkit promises to keep stable. Every
upstream classification change becomes a compatibility event.

**Keep `harness/` but mark it unstable and exempt from the additive-only
promise.** Rejected: an exemption carved out of a stability promise is the part
consumers discover after it breaks them. A surface that cannot be stable should
not be published.

**Ship `harness/` in the `tool` module rather than the runtime module.**
Rejected: generated code has to import it at test time, so it would drag the
generator's dependency graph — including eidos — into every consumer's test
binary, undoing [ADR-0005](0005-split-into-published-modules.md).

## Consequences

**Positive:**

- The shape vocabulary can change upstream without touching a published surface.
- Generated code and its shape logic are always at the same version, because
  they are the same file.
- 8,800 lines leave the maintained runtime surface.

**Negative:**

- Generated files get substantially larger, and the same helper is emitted into
  every package that needs it. Duplication is the direct cost of this decision.
- Within a package, deduplication is mandatory rather than merely desirable. Two
  interfaces in one package that share a shape must emit that shape's helper
  exactly once or the package does not compile. That forces an emit kind which
  aggregates per package rather than per source file, and it has to be designed
  in from the start: retrofitting it means reworking the emit model for `suite`
  and `bench` together.
- A bug in a shape helper is fixed by regenerating every consumer, not by
  upgrading a dependency. Consumers who do not regenerate keep the bug.
- Generated output is read by humans during review, and more of it means more to
  read.

**Neutral:**

- Runtime primitives that the helpers call — assertions, stubs, clocks — stay
  published and stable. Only the per-shape composition moves.
