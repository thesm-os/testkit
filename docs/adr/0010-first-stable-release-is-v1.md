---
adr: 0010
title: The first stable release is v1.0.0
status: Accepted
date: 2026-08-05
supersedes: none
superseded-by: none
---

# ADR-0010: The first stable release is v1.0.0

## Status

Accepted

## Context

The restructure recorded in this directory — new module topology, a different
codegen substrate, a changed directive form — was designed under a working name
that implied it was a second major version. It is not. The most recent release is
`v0.10.0`, and everything published so far has been `v0.x`.

Under semantic versioning `v0.x` promises nothing. The restructure is therefore
not a breaking change to a stable contract; it is the work that produces the
first stable contract.

This matters mechanically, because Go's import-path rule is tied to the major
version. A `v2` would require every published module path to carry a `/v2`
suffix: `go.thesmos.sh/testkit/v2`, `go.thesmos.sh/testkit/v2/engine`. That is a
permanent tax on every import block in every consumer, paid to signal a break
from a version that never promised anything.

The working name also leaked into the vocabulary — commit messages and documents
started referring to the current tree as one version and the target as another,
which is confusing when neither number is a release.

## Decision

The restructure releases as `v1.0.0`, succeeding `v0.10.0`. It is the first
stable release, not a second major version.

Module paths stay unsuffixed: `go.thesmos.sh/testkit`,
`go.thesmos.sh/testkit/engine`, `go.thesmos.sh/testkit/tool`.

The design-era version names are retired. Documents and commit messages describe
what changed, not which internal generation it belonged to.

`v1.0.0` is when the promises in
[ADR-0002](0002-support-external-consumers-under-semver.md) start binding.

## Alternatives Considered

**Release as `v2.0.0`.** Rejected: it costs a `/v2` suffix on every module path
forever, to mark a break from a `v0` line that made no compatibility promise. The
suffix is the permanent cost of a one-time signal.

**Stay on `v0.x` after the restructure.** Rejected: it defers the stability
promise indefinitely with no forcing function. The surfaces are settling; a
version that says "nothing is stable" would be inaccurate and would keep adopters
away for no reason.

**Cut `v1.0.0` now, before the generator port completes.** Rejected: the
generated-file layout is one of the four things
[ADR-0002](0002-support-external-consumers-under-semver.md) promises to hold
stable, and it cannot be promised while the generator producing it is being
rebuilt.

## Consequences

**Positive:**

- Import paths stay short and stable. No `/v2` in any consumer's import block.
- The release number means what semver says it means: this is the first version
  with a contract.
- A single, unambiguous marker for when the stability promises begin.

**Negative:**

- The jump from `v0.10.0` to `v1.0.0` skips the usual gradual `v0` ramp, so
  there is no intermediate release where adopters can try the new shape under an
  explicit "still unstable" label.
- Anything wrong in the surface at `v1.0.0` is wrong under a stability
  guarantee, and correcting it costs a major version and the `/v2` suffix this
  decision was made to avoid.

**Neutral:**

- All three published modules take the tag together. See
  [ADR-0006](0006-tag-published-modules-in-lockstep.md).
