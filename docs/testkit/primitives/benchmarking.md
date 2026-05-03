# Benchmarking

## Contract

Scope-based allocation and latency assertions for benchmarks. A drop-in replacement for the `b.Loop()` pattern that adds explicit performance ceilings.

```go
func BenchmarkScheduler_Ready(b *testing.B) {
    s := setup(b)
    c := testkit.StartContract(b).
        AllocsMax(0).
        LatencyMax(5 * time.Microsecond)
    for c.Loop() {
        _ = s.Ready()
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

## BenchTB

`BenchTB` is the subset of `*testing.B` that `Contract` depends on. Defining it as an interface allows testing the contract machinery itself with a stub. Most code passes `*testing.B` directly.

## Why this exists

Benchmarks without contracts are vanity metrics. A benchmark that reports `0 allocs/op` today silently becomes `3 allocs/op` tomorrow with no test failure. The `Contract` primitive makes the budget explicit and asserts on it.

The [`benchmarks` validator](../validators/benchmarks.md) and the [`bench` generator](../generators/bench.md) build on this primitive.
