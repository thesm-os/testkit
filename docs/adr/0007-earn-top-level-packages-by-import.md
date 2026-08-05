---
adr: 0007
title: Earn top-level packages by import
status: Accepted
date: 2026-08-05
supersedes: none
superseded-by: none
---

# ADR-0007: Earn top-level packages by import

## Status

Accepted

## Context

The runtime module had nineteen top-level packages. A top-level package is the
most expensive naming slot a module has: it appears in every import block, it is
the first thing a reader of the module root sees, and it implies to a consumer
that the package is something they will use directly.

Most of those nineteen were not. Plumbing that one other package imported sat
next to `clock` and `golden`, which every consumer touches. The directory
listing gave no signal about which was which, so the reader had to open each one
to find out.

Nineteen entries is also past the point where a flat list reads as a menu. It
reads as a pile.

## Decision

A package earns a top-level slot by being imported directly by users. Everything
else groups under `core/`.

The test is import provenance, not size or importance: `core/failure` is
load-bearing and nobody imports it by hand, so it groups. `clock` is small and
consumers construct a `TestClock` in their own test files, so it stays.

The root package is a façade over the surface those packages provide, so the
common case is one import rather than several.

This brought the runtime module from nineteen top-level packages to nine.

## Alternatives Considered

**Flat, all nineteen at top level.** Rejected: the directory listing carries no
information if everything is in it. A reader cannot tell entry points from
plumbing without opening files.

**`internal/` for everything not user-facing.** Rejected: `internal/` is an
enforcement mechanism, and enforcement is the wrong tool here. Some of these
packages are legitimately importable by a consumer doing something unusual;
grouping says "you probably do not need this", where `internal/` says "you may
not have this". Also, the generated code and the sibling modules import across
this boundary, which `internal/` would forbid.

**Group by subject rather than by audience — `assert/`, `stub/`, `time/`.**
Rejected: it produces a taxonomy that has to be learned before the first import.
Audience grouping produces exactly two categories, and a consumer only needs to
know one of them exists.

## Consequences

**Positive:**

- The module root is a menu of nine things a consumer might want, not a pile of
  nineteen.
- "Do I need this?" is answerable from the path. `core/` means probably not.
- Adding plumbing no longer costs a top-level slot, so the pressure to jam
  helpers into existing packages goes away.

**Negative:**

- The rule is a judgement call, and judgement calls drift. "Imported directly by
  users" has no test that fails when it stops being true.
- Moving a package between the two tiers is a breaking import-path change, so a
  wrong initial placement is expensive to correct after `v1.0.0`.
- `core/` is a grab bag by construction. It will accumulate unrelated things
  whose only shared property is that consumers do not import them.

**Neutral:**

- The engine module applies the same rule under `engine/`, so the convention is
  one rule rather than two.
