---
adr: 0003
title: Adopt eidos as the codegen substrate
status: Accepted
date: 2026-08-05
supersedes: none
superseded-by: none
---

# ADR-0003: Adopt eidos as the codegen substrate

## Status

Accepted

## Context

testkit owned a complete code-generation pipeline: package loader, intermediate
representation, metadata attachment with a consumer registry, slot ordering,
template rendering, import management, determinism guarantees, caching, and file
sink. Around 21,700 lines.

None of it is what testkit sells. The product is fourteen generators that know
how to test things. The pipeline is the conveyor belt they stand on.

[eidos](https://go.thesmos.sh/eidos) is that conveyor belt, built and maintained
separately: frontend, typed metadata carrying `(setBy, authority, sourcePos)`
provenance, slots ordered capability-topologically, a configurable directive
parser supporting neutral, `+`, and `-` forms, deterministic output, and a sink.

The overlap was not partial. Thirty-nine of forty-four packages under testkit's
`generator/spec/` had exact upstream counterparts. Two remaining gaps were
audited individually — testkit's `acquire` maps to eidos's `lease`, and its
`two-phase` maps to `tx` — and both adopt cleanly.

Two independent implementations of the same classification vocabulary is drift
waiting to happen: a detector fixed upstream stays broken here, and a
classification added here is invisible there.

## Decision

testkit deletes its own pipeline and builds on eidos. eidos supplies frontend,
intermediate representation, metadata, slots, determinism, caching, and sink.
testkit supplies annotator configuration and generator plugins.

The deleted surface is roughly 21,700 lines. What ports is at most 11,800, and
the directive package's exact split is an output of the classification audit.

## Alternatives Considered

**Keep the pipeline and vendor eidos's detectors.** Rejected: it keeps the
maintenance burden and adds a synchronisation burden on top. The detectors are
the part with an upstream; the pipeline around them is the part that costs.

**Keep the pipeline, contribute nothing upstream.** Rejected: this is the status
quo, and the status quo is that eight of fourteen generators were blocked behind
pipeline work rather than behind anything about testing.

**Extract testkit's pipeline into its own module and adopt that.** Rejected: it
is eidos with a different name and one fewer maintainer. The whole value is that
somebody else owns the conveyor belt.

## Consequences

**Positive:**

- Roughly 21,700 lines stop being testkit's problem.
- Classification fixes land once, upstream, and every consumer gets them.
- Generator work stops queueing behind pipeline work.
- Determinism, caching, and provenance arrive as inherited properties rather
  than as things to build and verify.

**Negative:**

- testkit now has an upstream it does not control. An eidos regression is a
  testkit regression, and an eidos design decision testkit disagrees with is an
  argument to have rather than a file to edit.
- The generator surface is unavailable while the port is in flight. There is no
  `testkit` binary in the tree during the transition.
- Directive semantics are constrained by what eidos's parser supports. See
  [ADR-0008](0008-neutral-directive-form-with-axis-qualifier.md), which depends
  on an upstream change.

**Neutral:**

- The `tool` module carries the eidos dependency. Runtime and engine consumers
  are unaffected — see [ADR-0005](0005-split-into-published-modules.md).
