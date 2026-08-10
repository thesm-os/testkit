# Benchmarking

```go
import "go.thesmos.sh/testkit"
```

A benchmark reports numbers. A benchmark *contract* fails when the numbers move the wrong way — which is the difference between a measurement someone has to read and a gate that holds a budget.

## Contract

```go
func StartContract(tb BenchTB) *Contract

func (c *Contract) AllocsMax(n uint64) *Contract
func (c *Contract) LatencyMax(d time.Duration) *Contract
func (c *Contract) Loop() bool
func (c *Contract) End()
```

`Loop` is a drop-in replacement for `testing.B.Loop`, so the benchmark body looks like any other:

```go
func BenchmarkStoreGet(b *testing.B) {
    s := store.New()
    c := testkit.StartContract(b).
        AllocsMax(2).
        LatencyMax(500 * time.Microsecond)
    defer c.End()

    for c.Loop() {
        _, _ = s.Get("k")
    }
}
```

`End` reports the measured allocation count and p99 latency as benchmark metrics, then fails the benchmark if either ceiling is exceeded. Defer it — a ceiling that only applies when the body returns normally is not a ceiling.

A contract with no ceiling set still reports. `AllocsMax` and `LatencyMax` are independent; set either, both, or neither.

## What the ceilings mean

`AllocsMax(n)` is **heap allocations per iteration**, the same number `-benchmem` prints as `allocs/op`. It is the more stable of the two: allocation count is a property of the code, not of the machine, so a budget that holds on a laptop holds in CI.

`LatencyMax(d)` is **p99 per iteration**, not the mean. The mean hides the tail, and the tail is what a latency budget is about — a p50 that never moves alongside a p99 that doubled is a regression the mean will not show.

Set the latency ceiling with headroom. CI runners are slower and noisier than a development machine, and a p99 budget pinned to the number you measured locally will fail on the first shared runner.

## BenchTB

```go
type BenchTB interface { ... }
```

The subset of `*testing.B` that `Contract` needs. Taking the interface rather than the concrete type is what lets the contract be driven from a test — including testkit's own tests of the contract itself.

## Benchmarking through a double

A generated double records every call, and those allocations are what the benchmark would otherwise measure. Turn recording off:

```go
s := readertest.NewReaderStub(nil, readertest.ReaderStubBenchMode())
```

Two things there:

**`BenchMode` drops the call log**, keeping dispatch. What remains is the cost of the double's method call, which is what a benchmark measuring the code around it wants.

**The `tb` is `nil`.** A non-nil one registers a cleanup that verifies call-count expectations, which a benchmark has none of — and the constructor skips the cleanup on `nil`, so passing it is correct rather than a shortcut.

## Recording a baseline

Budgets in the code are the gate. Tracking the numbers over time is [`ergon bench`](https://go.thesmos.sh/ergon)'s job — `ergon bench baseline` pins the current results and `ergon bench regression` fails when a run drifts beyond noise. The two are complementary: the contract catches an absolute breach at the point it happens, the baseline catches a slow drift that never breaches anything.

## See also

- [Recording](recording.md) — `BenchMode` on the recorder.
- [Stub](../generators/stub.md) — the `BenchMode` construction option.
