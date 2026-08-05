# Sim

> **Status:** not yet implemented. Targeted for a subsequent dev cycle. The behavior described below is the design intent — code, flags, and output may differ once shipped.

Tier 5 substrate. Subsystem-shaped deterministic simulation harness wrapping the full production stack: stubs auto-wrapped with recording-stamped `OnRecord` hooks emitting into the engine trace; `Clock` and `RandSource` plumbed from engine seeds; completion-event sinks; capture-on-failure with minimal-reproducer seed extraction; `Workload[T]` and `Invariant[T]` registration verbs; cooperative-quiescence `AssertAll`. Per-subsystem composition — one `Sim` per top-level interface — replaces hand-rolled per-package sim packages.

`sim` is the harness `chaos` and `replay` build on top of.

## Planned directive

```go
//go:generate testkit sim -o storetest/store_sim.gen.go Store
```

## Planned shape

```go
sim := storetest.NewStoreSim(t, seed, cfg)
sim.RegisterWorkload(workload)
sim.RegisterInvariant(invariant)
sim.AssertAll()
```

The harness composes the generated stub with the engine's clock, rand source, and trace, so a single seed reproduces an entire run.

## See also

- [Primitives / Clock](../primitives/clock.md), [RandSource](../primitives/rand.md)
- [Generators / chaos](chaos.md), [replay](replay.md), [differential-rollout](differential-rollout.md)
