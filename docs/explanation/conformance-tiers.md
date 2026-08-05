# Conformance tiers

Conformance generators are organised into tiers by how much evidence they
produce and how long they take to produce it. The tier is not a quality ranking —
a tier-1 suite catches real bugs — it is a statement about cost and about what
kind of wrongness the tier can see.

| Tier | Generator | What it proves | Runtime |
|---|---|---|---|
| 1 | `suite` | Single-call contract per documented directive across 21 method shapes | Seconds |
| 2–3 | `model` | Property-based state machine, differential comparison against a reference, workload replay | Seconds to minutes |
| 4 | `bench` | Allocation, mean latency, and per-percentile latency budgets across 21 shapes | Seconds to minutes |
| 5 | `sim`, `chaos`, `differential-rollout`, `replay` | Subsystem simulation, continuous fault injection, shadow traffic, production-trace replay | Minutes to hours |

## Why tiers rather than one suite

Each tier sees a class of defect the tier below it cannot.

**Tier 1 sees contract violations in isolation.** One call, one assertion: a
reader returns the sentinel for a missing key, a writer is idempotent, a
context-aware method honours cancellation. This is fast enough to run on every
save and catches the majority of ordinary mistakes.

What it cannot see is anything requiring *history*. A store that returns the
right value for every individual `Get` but loses a write under a specific
interleaving passes tier 1 completely.

**Tiers 2–3 see violations across sequences.** The model runner drives a
generated sequence of operations against both the subject and a reference
implementation, comparing after each step. A divergence is a bug in one of them,
and the shrinking machinery reduces the sequence to a minimal reproducer.

This is where laws live: read-after-write, monotonic reads, causal ordering,
snapshot isolation. Each is a statement about a *history*, not about a call.

What it cannot see is a cost regression. A correct implementation that allocates
on every read passes every law.

**Tier 4 sees cost.** Allocation and latency budgets, asserted per shape, gated
in CI. A benchmark contract fails the build when a hot path starts allocating.

What it cannot see is emergent behaviour under sustained load and failure.

**Tier 5 sees the system.** Subsystem simulation with injected faults, network
partitions, clock skew, and restarts, run long enough for slow leaks and
degradation to appear. Shadow traffic compares implementations under production
load; trace replay checks behavioural preservation across versions.

The cost is that tier-5 runs are measured in minutes to hours, which puts them on
a schedule rather than on every commit.

## The substrate

`stub` is not a tier. It generates the per-method test doubles every tier
composes with — recording, fault injection, gating, virtual clocks, strict-mode
dispatch — so a tier-5 fault schedule and a tier-1 sentinel assertion drive the
same double.

`sim` is the tier-5 substrate specifically: `chaos` and `replay` are workloads
that run on top of it rather than independent harnesses.

## Picking a tier

Add tiers in order, and only as far as the subject warrants.

- Every interface worth generating a double for is worth a tier-1 suite.
- Reach for tier 2–3 when the subject has *state* — when the answer to a call
  depends on previous calls.
- Reach for tier 4 when the subject is on a hot path and its cost is a
  documented contract rather than an accident.
- Reach for tier 5 when the subject is a subsystem rather than a type, and when
  the failures you care about are the ones that only appear under sustained
  load.

Skipping ahead is usually a mistake: a tier-5 run that fails on a defect tier 1
would have caught has spent an hour proving something a second-long test knew.
