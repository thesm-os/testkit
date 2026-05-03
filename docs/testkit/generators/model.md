# Model

> **Status:** not yet implemented. Targeted for a subsequent dev cycle. The behavior described below is the design intent — code, flags, and output may differ once shipped.

Tier 2-3 conformance. Reads an interface and emits a `rapid`-driven state-machine harness with three modes:

- **`RunStateMachine`** — oracle-based parity testing: every command runs against the real implementation and a consumer-supplied oracle; results compared at every step.
- **`RunDifferential`** — N-implementation comparison without an oracle: every command runs against every impl; pairwise divergence fails the test.
- **`RunWorkload`** — random-dispatch driver for simulation engines; emits operation streams without per-step assertions.

Drives property-based input exploration, cross-method invariants, sequence assertions, atomicity rollback, and partition isolation from `//testkit:` directives.

## Planned directive

```go
//go:generate testkit model -o storetest/store_model.gen.go Store
```

## Planned injection point

Consumer-supplied `Oracle` interface implementation; optional command generators, custom invariants, and per-method weights.

## See also

- [Generators / suite](suite.md) — Tier 1 single-call conformance
- [Generators / sim](sim.md) — Tier 5 subsystem simulation that consumes model workloads
