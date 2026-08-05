---
adr: 0006
title: Tag published modules in lockstep
status: Accepted
date: 2026-08-05
supersedes: none
superseded-by: none
---

# ADR-0006: Tag published modules in lockstep

## Status

Accepted

## Context

[ADR-0005](0005-split-into-published-modules.md) creates three published modules
in one repository. Go tags them independently: `v1.2.0`, `engine/v1.2.0`,
`tool/v1.2.0`. Nothing forces those numbers to match.

Independent versioning is the flexible option and it is also the one that
creates version skew. `engine` depends on the runtime module, and `tool`
generates code that imports both. A consumer on runtime `v1.4.0` with engine
`v1.2.0` is a combination nobody tested, and the set of such combinations grows
multiplicatively with each release.

Handling skew properly means a compatibility matrix, minimum-version constraints
expressed in each `go.mod`, and CI that tests the combinations rather than the
tip. That is real machinery, and it buys the ability to release a patch to one
module without touching the others.

The three modules are developed together, released together, and change together
— a generator change usually implies a runtime or engine change, because the
generated code calls into both.

## Decision

All published modules are tagged in lockstep. One release cuts `vX.Y.Z`,
`engine/vX.Y.Z`, and `tool/vX.Y.Z` at the same commit with the same numbers.

A module with no changes in a release is tagged anyway. Gaps in a module's tag
sequence would make "which engine goes with runtime v1.4.0" a question again,
which is the thing being avoided.

## Alternatives Considered

**Independent versioning per module.** Rejected: it buys per-module patch
releases at the cost of a compatibility matrix, minimum-version constraints, and
combinatorial CI. The modules do not change independently often enough for that
to pay.

**One module, no split.** Rejected by
[ADR-0005](0005-split-into-published-modules.md) on dependency-weight grounds.

**Lockstep major and minor, independent patch.** Rejected: it is independent
versioning with extra rules. The moment patch numbers diverge, the matrix is
back.

## Consequences

**Positive:**

- Version skew is structurally impossible. The engine that goes with runtime
  `vX.Y.Z` is `engine/vX.Y.Z`, always.
- No compatibility matrix, no minimum-version constraints, no combinatorial CI.
- A consumer upgrading one module knows exactly what to upgrade alongside it.

**Negative:**

- A patch to one module forces a release of all three, including modules with an
  empty diff. Changelogs will carry releases where two of three modules changed
  nothing.
- Version numbers stop conveying per-module change magnitude. A major bump means
  something somewhere broke, not that this module broke.
- Release tooling has to tag three refs atomically; a partial tag push leaves the
  repository in a state that contradicts this decision.

**Neutral:**

- The unpublished `conformance` module is untagged and unaffected.
