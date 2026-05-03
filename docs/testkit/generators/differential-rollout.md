# Differential Rollout

> **Status:** not yet implemented. Targeted for a subsequent dev cycle. The behavior described below is the design intent — code, flags, and output may differ once shipped.

Tier 5 shadow-traffic harness. Runs a production interface across N implementations in parallel, compares responses, and reports divergences. Migration-grade testing — answers "can I cut over from impl A to impl B without breakage?" Pluggable equivalence relations handle non-deterministic fields (timestamps, generated IDs).

## Planned directive

```go
//go:generate testkit differential-rollout -o storetest/store_diffrollout.gen.go Store
```

## Planned injection point

Implementation list, equivalence-class declarations, divergence threshold.

## Consumed directives

`errors` (classify), `partition` (shard-aware), `pure` (side-free comparison), `bounded` (range-equiv), `cacheable` (memo-equiv), `monotonic` (ordered-equiv), `sideeffect` (effect-equiv), `eventually` (window-equiv), `pagination` (page-equiv), `latency` (regression), `network-safe` (partition-equiv), `req`, plus skip-on `integration-only` and `consistency` model-aware comparison.

## See also

- [Generators / replay](replay.md) — captures production traffic this harness can replay against multiple impls
- [Generators / sim](sim.md) — drives the harness when no production traffic source is available
