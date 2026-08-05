---
adr: 0002
title: Support external consumers under semver
status: Accepted
date: 2026-08-05
supersedes: none
superseded-by: none
---

# ADR-0002: Support external consumers under semver

## Status

Accepted

## Context

testkit is published at a public import path from a public repository. Anyone
can take a dependency on it.

Nobody outside has yet. Every consumer today is in-house: the set is known,
enumerable, and reachable by the same people making the changes. That is a fact
about the present, not a design constraint, and it stops being true the first
time someone outside runs `go get`.

Those two facts pull in opposite directions, and the tension has to be resolved
explicitly because the default under ambiguity is the expensive posture. Treat
every consumer as an unknown stranger and each breaking change costs a
deprecation window, a compatibility shim, and two surfaces tested and documented
in parallel for as long as the window lasts — a premium paid every release
against a risk that has not yet materialised. Treat every consumer as in-house
and the public import path advertises a stability that nothing backs.

The restructure sharpens this. Surfaces are moving: modules split, the generator
is being rebuilt, directive forms are changing. Freezing them now would freeze
the wrong thing.

## Decision

testkit supports external consumers under semantic versioning. The promise binds
at `v1.0.0` and covers four things:

- Stable directive vocabulary and composition rules.
- Stable generated-file layout and naming conventions.
- Backward-compatible runtime primitives — additive only within a major.
- A documented deprecation cycle for any directive removal.

Before `v1.0.0` those surfaces move freely. With no external consumers yet, a
breaking change costs a codemod across a known set of repositories rather than
coordination with strangers, and the restructure is exactly when that latitude is
worth having.

A deprecation cycle means a removal is announced in release notes and marked in
godoc for a release before it lands. It does not mean carrying two
implementations of the same thing: the mechanism is notice, not duplication.

## Alternatives Considered

**Treat all consumers as permanently internal.** Rejected: "no external
consumers" is an observation about today. Building the compatibility posture on
it means the posture is wrong the moment it changes, with no signal for when
that happens. The public import path is a promise whether or not it is written
down.

**Full semver discipline starting now, mid-restructure.** Rejected: it would
freeze surfaces that are actively being replaced. The generated-file layout
cannot be promised stable while the generator producing it is being rebuilt.

**Semver with no deprecation cycle — breaking changes land in a major, no
notice.** Rejected: directives live in consumer source code, not in a dependency
graph. A removed directive is a compile-time break in files the consumer wrote by
hand, and discovering that from a version bump alone costs more than one release
of notice.

**Make the repository private.** Rejected: it would make an internal-only
posture honest, at the cost of being able to reference testkit from public work.

## Consequences

**Positive:**

- The restructure keeps the latitude it needs, bounded by an explicit release
  rather than by drift.
- What `v1.0.0` means is written down before it is cut, so the release is a
  promise rather than a number.
- Pre-1.0 breaking changes stay cheap enough to make when they are right, which
  is what keeps [ADR-0009](0009-one-config-filename.md) affordable.

**Negative:**

- An outside adopter who takes a dependency before `v1.0.0` gets no warning
  before a break, and the import path does not say so — only the README does.
- After `v1.0.0` every directive removal carries a release of notice, which
  slows exactly the cleanup work that tends to be deferred anyway.
- The promise is made before the surface it covers is finished. If the generator
  rebuild lands a layout that turns out wrong, it is wrong under a stability
  guarantee.

**Neutral:**

- This ratifies what `README.md` already told readers rather than introducing a
  new posture.
