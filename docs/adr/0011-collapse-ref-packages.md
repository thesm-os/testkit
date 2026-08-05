---
adr: 0011
title: Collapse the reference-implementation packages
status: Accepted
date: 2026-08-05
supersedes: none
superseded-by: none
---

# ADR-0011: Collapse the reference-implementation packages

## Status

Accepted

## Context

Differential testing needs a reference: a small, obviously-correct implementation
of a shape that the subject under test can be compared against. The engine ships
seventeen of them — an append-only chain, a compare-and-swap cell, a snapshot-
isolated store, a saga, a scheduler, and so on.

Each lived in its own package, named by prefixing the shape: `refcas`, `reftxn`,
`refsaga`, `refsched`, `refpubsub`. Seventeen packages, 2,995 lines, averaging
176 lines and roughly one exported type each.

The names carried no information. `refcas` says "reference" twice — once in the
prefix and once in the import path — and then abbreviates the actual subject.
Worse, the package name occupied the namespace the type wanted: the type had to
be called `refcas.Cell` or `refcas.CAS` because `refcas.AtomicCell` reads as
stuttering, when `ref.AtomicCell` is exactly right.

Seventeen import lines to use five references is friction with nothing on the
other side of it. The packages have no independent lifecycle, no separate
consumers, and no dependency reason to be apart.

The precedent was already in the tree: `model/law` is one package across
thirty-five files and roughly the same total size, and it reads fine.

## Decision

The seventeen `ref*` packages collapse into one `ref` package, one file per
shape, and the types take the names the prefixes were hiding:

| Was | Is |
|---|---|
| `refcas.Cell` | `ref.AtomicCell` |
| `reftxn.Store` | `ref.SnapshotIsolation` |
| `refsaga.Saga` | `ref.CompensatingSaga` |
| `refsched.Scheduler` | `ref.PureScheduler` |

File-per-shape is kept, so the physical organisation is unchanged and the
navigation cost is the same. Only the import surface collapses.

## Alternatives Considered

**Keep seventeen packages, rename them better.** Rejected: it fixes the names
and keeps the seventeen import lines. The package boundary was never doing
anything; renaming preserves the cost and addresses only half the problem.

**One package, one file.** Rejected: 2,995 lines in one file is worse to
navigate than seventeen packages. The file boundary is carrying real weight even
though the package boundary is not.

**Group into three or four packages by category — stores, chains, coordination.**
Rejected: the categories are not obvious, so a reader has to learn the taxonomy
to find `AtomicCell`. It trades seventeen arbitrary boundaries for four
arbitrary boundaries.

## Consequences

**Positive:**

- Type names stop being truncated by their package. `ref.AtomicCell` says what
  it is.
- One import line instead of up to seventeen.
- Matches `model/law`, so the engine has one convention for this rather than
  two.

**Negative:**

- One package namespace shared by seventeen shapes means name collisions are now
  possible and have to be resolved by hand. Two shapes both wanting `Entry` is a
  problem the package split solved for free.
- The compiler can no longer tell a reader which shapes a consumer actually
  uses; a single `ref` import says nothing about which references are in play.
- Every consumer's import block and every reference to these types changes at
  once.

**Neutral:**

- The references are engine-internal in practice; nothing outside the engine's
  own tests imports them today.
