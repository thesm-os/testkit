# Benchmarking

## Contract

Scope-based allocation and latency assertion for
benchmarks. Declare ceilings once, drive iterations with
`c.Loop()`, call `c.End()`.

```go
func BenchmarkScheduler_Ready(b *testing.B) {
    s := setup()
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
| `StartContract(b)` | Begin contract; b is `*testing.B` or `BenchTB` stub |
| `AllocsMax(n)` | Max allocations per op; 0 = zero-alloc |
| `LatencyMax(d)` | Max p99 per-iteration latency |
| `Loop() bool` | Drop-in replacement for `b.Loop()` |
| `End()` | Report metrics, assert ceilings |

The scope form avoids closure-escape contamination that a
callback form imposes on 0-alloc claims.
