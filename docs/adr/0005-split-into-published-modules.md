---
adr: 0005
title: Split into published modules behind a go.work
status: Accepted
date: 2026-08-05
supersedes: none
superseded-by: none
---

# ADR-0005: Split into published modules behind a go.work

## Status

Accepted

## Context

testkit was one module. A consumer who imported it for `testkit.Equal` and a
`MethodStub` took the property-testing engine, the linearizability checker, and
the code generator's dependency graph with it — in `go.sum`, in
`govulncheck` surface, and in build time.

The three tiers have genuinely different consumers. Runtime primitives are used
by every test in every package. The model-checking engine is used by the subset
of packages that have a state machine worth checking. The generator runs at
build time and ships nothing.

Dependency weight was the single loudest complaint about adopting testkit.

## Decision

Three published modules in one repository, joined by a `go.work`:

| Module | Contains | Adds |
|---|---|---|
| `go.thesmos.sh/testkit` | Runtime primitives | go-cmp |
| `go.thesmos.sh/testkit/engine` | Model checking | rapid, porcupine |
| `go.thesmos.sh/testkit/tool` | CLI and generator plugins | eidos |

Plus one unpublished module, `conformance`, holding the generated corpus that
proves the generators work. Unpublished modules cost no tags.

The second module is named `engine` rather than `model`: it houses `engine/model`
and `engine/sim`, and "engine" is the vocabulary the simulation design already
used.

`go.work` lists only the modules in this repository. Sibling checkouts such as
eidos are added locally with `go work use ../eidos` rather than committed, so a
clone builds without them present.

## Alternatives Considered

**One module with build tags.** Rejected: build tags do not remove entries from
`go.sum`. The dependency is still declared, still audited, still resolved.

**Separate repositories.** Rejected: three repositories means three CI setups,
three release pipelines, and cross-repository changes that cannot be reviewed as
one diff. The dependency isolation is a module property, not a repository
property.

**Two modules — runtime and everything else.** Rejected: it puts the eidos
dependency on every consumer who wants model checking, which is the majority of
the interesting use cases. The generator is the heaviest dependency and the one
fewest consumers need at runtime.

## Consequences

**Positive:**

- A consumer who wants assertions pays for go-cmp and nothing else.
- The boundary is enforced by the module graph, so it cannot erode through a
  convenience import.
- Each tier can be vetted, vulnerability-scanned, and gated independently.

**Negative:**

- Three `go.mod` files to keep aligned, and a `go.work` that must not be
  committed with local `use` directives pointing at sibling checkouts.
- Tooling that assumes one module per repository needs per-module invocation.
  Branch-coverage instrumentation currently cannot measure `engine/model`
  because it copies the module to a temporary directory and loses the workspace
  context that resolves the relative replace.
- A change spanning two modules cannot be released atomically without
  [ADR-0006](0006-tag-published-modules-in-lockstep.md).

**Neutral:**

- Import paths stay unsuffixed. See
  [ADR-0010](0010-first-stable-release-is-v1.md).
