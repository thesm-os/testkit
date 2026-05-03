# Chaos

> **Status:** not yet implemented. Targeted for a subsequent dev cycle. The behavior described below is the design intent — code, flags, and output may differ once shipped.

Tier 5 fault simulation. Built on top of [`sim`](sim.md). Drives randomized fault schedules, network partitions, clock skew, and process restarts across operation sequences. Seeded reproducible runs; on failure, emits a trace plus a minimal-reproducer seed extracted from the failing prefix. Integrates with the sim engine via `OnRecord` hooks for trace correlation.

## Planned directive

```go
//go:generate testkit chaos -o storetest/store_chaos.gen.go Store
```

## Planned injection point

`Faults` configuration, `RunFault` / `PartitionSpec` declarations, soak-budget hints.

## Consumed directives

`errors`, `partition`, `order-after`, `retry-succeeds-on-attempt`, `idempotent`, `concurrent`, `atomic`, `sideeffect`, `eventually`, `invariant`, `consistency`, `lease`, `latency`, `crash-safe`, `network-safe`, `clock-skew`, `req`, plus skip-on `integration-only`. See the directive matrix in the top-level README.

## See also

- [Generators / sim](sim.md) — the harness chaos runs on
- [Generators / replay](replay.md) — production-trace replay sibling
