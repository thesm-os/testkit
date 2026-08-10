# Replay

> **Status: planned.** This generator is not implemented. No directive is registered for it, and `testkit run` does not produce the output described below. This page records the intended design so adoption can be planned against a stable target — the directive, the output paths and the generated surface may all differ once it ships.

Tier 5 trace-replay harness. Consumes captured production call traces (or sim-engine traces) and replays them through implementations to verify behavioral preservation across versions. Answers "does the new impl behave the same way the old one did on real workloads?"

## Planned directive

```go
//go:generate testkit replay -o storetest/store_replay.gen.go Store
```

## Planned injection point

Trace source (file path or producer function), version-skew tolerance hints.

## Consumed directives

`errors` (assert), `order-after` (replay-order), `pure` (deterministic), `monotonic` (ordering), `idempotent` (replay-safe), `atomic` (no-partial), `sideeffect` (effect-replay), `eventually` (window), `invariant` (replay-stable), `consistency` (window), `clock-skew` (drift), `crash-safe` (recovery), `req`, plus skip-on `integration-only`.

## See also

- [Generators / sim](sim.md) — produces traces this harness can replay
- [Generators / differential-rollout](differential-rollout.md) — for replaying the same trace against multiple impls in parallel
