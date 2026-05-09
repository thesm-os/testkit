# Benchmarking

Two layers of benchmarking primitives:

- **`testkit.StartContract`** — scope-based allocation and latency contract, a drop-in replacement for `for b.Loop()` that adds explicit ceilings.
- **`testkit/bench` runtime helpers** — a thin layer over `b.Loop` / `b.RunParallel` that drives single-goroutine measurement, parallel throughput, and the three opt-in budget gates (`allocs`, `latency`, `percentiles`). The shape-typed primitives consumed by [`bench` generator](../generators/bench.md) sit on top of these helpers.

## Contract (`testkit.StartContract`)

Scope-based contract with chained ceilings:

```go
func BenchmarkStore_Get(b *testing.B) {
    s := setup(b)
    c := testkit.StartContract(b).
        AllocsMax(0).
        LatencyMax(5 * time.Microsecond)
    for c.Loop() {
        _ = s.Get(b.Context(), key)
    }
    c.End()
}
```

| Method | Description |
|--------|-------------|
| `StartContract(b)` | Begin contract; `b` is `*testing.B` or `BenchTB` stub |
| `AllocsMax(n)` | Max allocations per iteration; `0` = zero-alloc |
| `LatencyMax(d)` | Max p99 per-iteration latency |
| `Loop() bool` | Drop-in replacement for `b.Loop()` |
| `End()` | Report metrics and assert ceilings; fatals on violation |

The scope form avoids the closure-escape contamination that a callback form imposes on zero-alloc claims.

### BenchTB

`BenchTB` is the subset of `*testing.B` that `Contract` depends on. Defining it as an interface allows testing the contract machinery itself with a stub. Most code passes `*testing.B` directly.

## Runtime helpers (`testkit/bench`)

The `bench` package ships a small set of `*testing.B`-rooted primitives the [`bench` generator](../generators/bench.md) calls into. Consumers can call them directly when hand-writing benchmarks that don't fit the generator's shape vocabulary.

```go
import "go.thesmos.sh/testkit/bench"

func BenchmarkCache_Lookup(b *testing.B) {
    c := newCache()
    bench.HotPath(b, "hot-path", func() {
        _ = c.Lookup("known-key")
    })
    bench.AllocsWithin(b, "allocs-within-0", 0, func() {
        _ = c.Lookup("known-key")
    })
    bench.LatencyPercentilesWithin(b, "percentiles", map[float64]time.Duration{
        0.50: 1 * time.Microsecond,
        0.99: 10 * time.Microsecond,
    }, func() {
        _ = c.Lookup("known-key")
    })
}
```

Each helper opens its own `b.Run(name, ...)` namespace and exits cleanly on assertion failure. The helpers are deliberately untyped over the call signature (`func()`) so they compose with any Go function, regardless of arity or return type.

### Always-emitted primitives

| Helper | Purpose |
|--------|---------|
| `HotPath(b, name, call)` | Single-goroutine `b.Loop` measurement; reports `ns/op` and `allocs/op`. |
| `ConcurrentThroughput(b, name, parallelism, call)` | `b.RunParallel`-style stress at the given parallelism. |

### Opt-in budget gates

| Helper | Purpose |
|--------|---------|
| `AllocsWithin(b, name, maxAllocs, call)` | Fails the benchmark when `allocs/op` exceeds `maxAllocs`. `0` is a valid alloc-free assertion. |
| `LatencyWithin(b, name, maxLatency, call)` | Fails when mean `ns/op` exceeds the duration ceiling. |
| `LatencyPercentilesWithin(b, name, budgets, call)` | Records per-iteration durations into a pre-allocated slice, sorts the distribution, reports `p50/p95/p99` via `b.ReportMetric` (regardless of which percentiles are budgeted), and fails when any budgeted percentile exceeds its ceiling. Percentile keys in `(0, 1)`, e.g. `0.50`, `0.99`. |

### `*WithBytes` variants for I/O shapes

`HotPathWithBytes`, `AllocsWithinWithBytes`, `LatencyWithinWithBytes`, and `ConcurrentThroughputWithBytes` accept a `bytesPerOp int64` argument and call `b.SetBytes` so `MB/s` is reported alongside `ns/op`. Use these for `BatchReader` / `StreamReader` / `StreamConsumer` shapes where throughput is the meaningful metric.

The bench generator wires `BytesPerOp` automatically for `StreamConsumer` methods whose stream parameter is `io.Reader` (defaults to the synthesized sample length); other I/O shapes accept a per-method `<Iface>BenchSetBytes(method, n)` option.

### Utilities

| Helper | Purpose |
|--------|---------|
| `SubtestKey(v)` | Renders an arbitrary value into a benchmark-safe subtest segment (used by the generator to name subtests like `Get/hot-path/seed-1`). |
| `ReportRunningMetric(b, unit, value)` | Thin wrapper over `b.ReportMetric` for surfacing custom metrics from inside a `for b.Loop()` body. |

## Choosing between layers

| Need | Use |
|------|-----|
| One ad-hoc benchmark with mean-latency / alloc ceilings | `testkit.StartContract` |
| Hand-written benchmark with shape-style primitives (multiple `b.Run` namespaces, percentile gating, parallel sweep) | `bench` runtime helpers |
| Generated benchmarks for a whole interface | [`bench` generator](../generators/bench.md) — emits typed `<Iface>BenchOn<Method>` plug-ins on top of the helpers |

## Why this exists

Benchmarks without ceilings are vanity metrics. A benchmark that reports `0 allocs/op` today silently becomes `3 allocs/op` tomorrow with no test failure. Both layers make the budget explicit and assert on it; the [`benchmarks` validator](../validators/benchmarks.md) enforces every `Benchmark*` function uses one of them, and the [`bench-regression` validator](../validators/bench-regression.md) (planned) compares benchmark output against a pinned baseline so any performance degradation surfaces in CI.
