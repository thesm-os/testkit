# Bench

> **Status:** not yet implemented. Targeted for a subsequent dev cycle. The behavior described below is the design intent — code, flags, and output may differ once shipped.

Tier 4 conformance. Reads an interface and its `//testkit:` directives, emits a `BenchmarkContract(b, factory, opts...)` function with `AllocsMax(N)` and `LatencyMax(X)` gates per directive. Adds performance-shape benchmarks (`O(1)`, `O(log n)`, `O(n)`) for methods carrying the `complexity` directive by varying input size across runs. Auto-enables stub `BenchMode` so recording overhead does not contaminate measurements.

## Planned directive

```go
//go:generate testkit bench -o storetest/store_bench.gen.go Store
```

## Planned injection point

Consumer-supplied `Factory func() Iface` closure.

## Consumed directives

- `allocs N` — `StartContract(b).AllocsMax(N)`
- `latency D` — `StartContract(b).LatencyMax(D)`
- `complexity` — vary input size, assert shape against documented complexity
- `concurrent` / `concurrent-readers` — parallel benchmark variant
- `timeout` — bench harness deadline gate
- `req` — REQ-ID embedded in benchmark name

## See also

- [Primitives / Contract](../primitives/benchmarking.md) — the runtime ceiling primitive bench wraps
- [Validators / benchmarks](../validators/benchmarks.md) — enforces every benchmark has a contract
- [Validators / bench-regression](../validators/bench-regression.md) — compares against pinned baseline
