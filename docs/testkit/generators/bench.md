# Bench

Tier 4 conformance. Reads a Go interface, classifies each method into one of 21 *shapes* via signature matching, and emits a `Benchmark<Iface>Contract(b, factory, opts...)` harness with two layers:

- **Always-emitted primitives** per method — `<Method>/hot-path` (single-goroutine ns/op + allocs/op) and, for shapes where it's safe, `<Method>/concurrent-4` (multi-goroutine throughput). No consumer code required.
- **Opt-in budget gates** driven by directives — `<Method>/allocs-within-N`, `<Method>/latency-within-D`, `<Method>/percentiles`. Each fails the benchmark when the measured value exceeds the consumer-declared ceiling.
- **Plug-in primitives** through typed `<Iface>BenchOn<Method>(...)` slots that match the method's shape, plus a free-form `<Iface>BenchCustom(name, fn)` escape hatch.

A `factory func() Iface` is the single required injection. Every per-method bench constructs a fresh implementation per iteration via the seeded factory.

## Directive

```go
//go:generate testkit bench -o storetest/store_bench.gen.go Store
```

`bench` accepts exactly one type argument and emits a single output file.

## Default output

`<package>test/<subject>_bench.gen.go`.

## Method-shape detection

Identical to `suite` — the same `shape.Detect` priority table classifies every method into one of 21 shapes. See [Generators / suite — shape detection](suite.md#method-shape-detection) for the full table.

The shape determines:

- which typed bench Context (`bench.ReaderContext`, `bench.WriterContext`, ...) is wired into the per-method helper,
- which primitives the helper calls (HotPath / Concurrent / AllocsWithin / LatencyWithin / LatencyPercentilesWithin),
- which typed primitives consumers can plug in through `<Iface>BenchOn<Method>`.

A per-method docblock at the top of each generated helper lists the detected shape, the directives applied, the synthesized sample inputs, and the exact list of subtests emitted — so a misclassification or a missing seed shows up by reading the file.

## Always-emitted primitives

Every non-skipped method produces:

- `<Method>/hot-path` — single-goroutine `b.Loop()` measurement reporting `ns/op` and `allocs/op`. The hot-path calls the method with sample literals (synthesized from the parameter types, or supplied via `//testkit:sample`).
- `<Method>/concurrent-4` — `b.RunParallel`-style throughput at parallelism 4.

`Lifecycle`, `VoidLifecycle`, and `Unknown` shapes skip the concurrent emission by default — concurrent invocation of `Init` / `Close` / `Reset` on a shared impl is rarely safe, and the typed primitive can't guarantee otherwise. Consumers that need concurrent measurement on those methods supply a primitive through the `<Iface>BenchOn<Method>` plug-in slot.

For methods whose hot-path uses synthesized sample literals (no `//testkit:sample` directive), the generated helper emits a `b.Logf` warning at the top of the run so consumers reading the output don't silently measure the not-found / error path:

```
Hot: hot-path uses synthesized sample literals; declare //testkit:sample <Func>...
and seed the factory with matching values, or the benchmark may measure the
not-found / error path instead of the success path
```

`StreamConsumer` methods whose stream type is `io.Reader` automatically wire `BytesPerOp` from the synthesized stream sample (`bytes.NewReader([]byte("test-data"))` = 9 bytes) so MB/s reports without per-method configuration. Override via the consumer's `<Iface>BenchSetBytes` option.

## Directives consumed

| Directive | Adds subtest | Behavior |
|-----------|--------------|----------|
| `//testkit:allocs N` | `<Method>/allocs-within-N` | Fails when the measured allocs/op exceeds `N`. `N=0` is a valid alloc-free assertion. |
| `//testkit:latency D` | `<Method>/latency-within-D` | Fails when the mean ns/op exceeds the duration ceiling. |
| `//testkit:percentiles p<N>=D...` | `<Method>/percentiles` | Records each iteration's duration, sorts the distribution, reports `p50-ns/op` / `p95-ns/op` / `p99-ns/op` via `b.ReportMetric`, and fails when any budgeted percentile exceeds its ceiling. Multiple budgets supported (e.g. `p50=10us p95=50us p99=100us`); percentiles in [1, 99]. |
| `//testkit:sample <Func>...` | *(no extra subtest)* | Replaces the synthesized sample literals in the hot-path / concurrent / budget-gate calls with `<Func>()` invocations. One builder per non-context parameter; called once at construction time. See [suite — sample directive](suite.md#sample-directive) for the resolution rules. |
| `//testkit:integration-only` | *(method skipped)* | The method's helper is omitted entirely. Use for methods that can't be benched in a hermetic harness (network, hardware, persistence). |

`bench` shares the directive registry with every other generator, so a method may carry directives consumed by `suite` (`errors`, `pure`, `nilsafe`, ...) without affecting the bench output — `bench` reads only the five directives above.

## What is generated

For an interface declaring three budget gates plus a sample directive on a key parameter:

```go
type Perf interface {
    //testkit:allocs 0
    //testkit:latency 100us
    //testkit:percentiles p50=50us p95=200us p99=500us
    //testkit:sample SeededKey
    Hot(ctx context.Context, key string) (Item, error)
}
```

the generator emits a per-method helper:

```go
// benchPerfHot measures Perf.Hot(ctx context.Context, key string) (basic.Item, error).
//
//   Shape:      Reader
//   Directives: //testkit:allocs 0
//               //testkit:latency 100us
//               //testkit:percentiles p50=50us p95=200us p99=500us
//               //testkit:sample SeededKey
//   Sample inputs: SeededKey()
//   Emits:      Hot/hot-path
//               Hot/concurrent-4
//               Hot/allocs-within-0     (//testkit:allocs gate)
//               Hot/latency-within-100us     (//testkit:latency gate)
//               Hot/percentiles    (//testkit:percentiles p50=50us p95=200us p99=500us)
//               Hot/<consumer-supplied>     (via cfg.onHot)
func benchPerfHot(b *testing.B, factory func() basic.Perf, cfg *perfBenchConfig) {
    b.Helper()
    b.Run("Hot", func(b *testing.B) {
        seededFactory := func() basic.Perf {
            impl := factory()
            if cfg.prePopulate != nil {
                cfg.prePopulate(b.Context(), impl)
            }
            return impl
        }
        rctx := bench.ReaderContext[basic.Perf, string, basic.Item]{ /* ... */ }
        bench.ReaderHotPath[basic.Perf, string, basic.Item](SeededKey())(rctx)
        bench.ReaderConcurrentThroughput[basic.Perf, string, basic.Item](SeededKey(), 4)(rctx)
        bench.ReaderAllocsWithin[basic.Perf, string, basic.Item](SeededKey(), 0)(rctx)
        bench.ReaderLatencyWithin[basic.Perf, string, basic.Item](SeededKey(), time.Duration(100000) /* 100us */)(rctx)
        bench.ReaderLatencyPercentilesWithin[basic.Perf, string, basic.Item](SeededKey(), map[float64]time.Duration{
            0.50: time.Duration(50000)  /* p50=50us */,
            0.95: time.Duration(200000) /* p95=200us */,
            0.99: time.Duration(500000) /* p99=500us */,
        })(rctx)
        for _, p := range cfg.onHot { p(rctx) }
    })
}
```

The driver then dispatches one helper per method, plus the top-level `Custom` and option-folding scaffolding.

## Plug-in extension points (typed by shape)

Each method gets a `<Iface>BenchOn<Method>` option whose argument type depends on the method's shape. The consumer composes typed bench primitives from `bench/`:

| Method shape | `BenchOn<Method>` accepts |
|--------------|---------------------------|
| Reader | `bench.Reader[T, K, V]` |
| ReaderNoError | `bench.ReaderNoError[T, K, V]` |
| ReaderWithBool | `bench.ReaderWithBool[T, K, V]` |
| Lookup | `bench.Lookup[T, K, V, R]` |
| PointerReader | `bench.PointerReader[T, K, V]` |
| MultiReader | `bench.MultiReader[T, K, V1, V2]` |
| BatchReader | `bench.BatchReader[T, K, V]` |
| Writer | `bench.Writer[T, V]` |
| CompositeWriter | `bench.CompositeWriter[T, K1, V]` |
| Mutator | `bench.Mutator[T, V]` |
| Deleter | `bench.Deleter[T, K]` |
| MultiArgWriter (arity 3) | `bench.MultiArgWriter[T, P1, P2, P3]` |
| MultiArgWriter (arity ≠ 3) | `func(*testing.B, T)` (free-form) |
| Aggregator | `bench.Aggregator[T, R]` |
| MultiAggregator | `bench.MultiAggregator[T, V1, V2]` |
| StreamReader | `bench.Stream[T, V]` |
| StreamConsumer | `bench.StreamConsumer[T, S, V]` |
| Pure | `bench.Pure[T, R]` |
| Predicate | `bench.Predicate[T]` |
| PoisonAccessor | `bench.PoisonAccessor[T]` |
| Lifecycle | `bench.Lifecycle[T]` |
| VoidLifecycle | `bench.VoidLifecycle[T]` |
| Unknown | `func(*testing.B, T)` (free-form) |

Every shape ships the same five-primitive vocabulary (where applicable):

- `<Shape>HotPath(...)` — single-goroutine measurement (overrides the auto-emitted hot-path).
- `<Shape>ConcurrentThroughput(..., parallelism)` — `b.RunParallel`-style stress.
- `<Shape>AllocsWithin(..., maxAllocs)` — gate: fail when allocs/op exceeds `maxAllocs`.
- `<Shape>LatencyWithin(..., maxLatency)` — gate: fail when ns/op exceeds `maxLatency`.
- `<Shape>LatencyPercentilesWithin(..., budgets map[float64]time.Duration)` — gate: fail when any budgeted percentile exceeds its ceiling; reports `p50/p95/p99` via `b.ReportMetric` regardless of which percentiles are budgeted.

Consumer-defined primitives are one-line closures over the shape's typed `Context`.

## PrePopulate

```go
servicetest.ServiceBenchPrePopulate(func(ctx context.Context, s allshapes.Service) {
    _ = s.Put(ctx, allshapes.Item{ID: "pre-1", Name: "prepopulated"})
})
```

`PrePopulate` is wrapped into a `seededFactory` closure that the per-method helper calls every time a primitive needs a fresh impl. State is not shared across iterations — `b.Loop()` and `b.RunParallel` see the seeded baseline on every reconstruction. PrePopulate cost is paid inside the timed region; if you want to exclude seeding from the measurement, do the work in the `factory` closure instead.

## Custom sub-benchmarks

```go
servicetest.ServiceBenchCustom("put-then-get-round-trip", func(b *testing.B, s allshapes.Service) {
    for b.Loop() {
        _ = s.Put(b.Context(), allshapes.Item{ID: "rt"})
        _, _ = s.Get(b.Context(), "rt")
    }
})
```

For benchmarks outside the shape vocabulary — multi-method round-trips, workload mixes, scenario benchmarks. Each `Custom` sub-benchmark receives a fresh `factory()`-produced impl. Use sparingly; most performance contracts have a shape primitive that fits.

## Stub interaction

When the consumer's factory returns a generated stub (from [`testkit stub`](stub.md)), the bench harness automatically benefits from `BenchMode` — recording is skipped for the duration of the run so per-call overhead reflects the underlying implementation, not the recorder. `BenchMode` is enabled per stub via the stub's `Bench()` mode option.

## Wiring against an implementation

```go
// servicetest/bench_test.go
package servicetest_test

func BenchmarkServiceContract(b *testing.B) {
    factory := func() allshapes.Service {
        s := allshapes.NewInMemoryService()
        _ = s.Put(context.Background(), allshapes.Item{ID: "seed-1", Name: "seed"})
        return s
    }

    servicetest.BenchmarkServiceContract(b, factory,
        servicetest.ServiceBenchPrePopulate(func(ctx context.Context, s allshapes.Service) {
            _ = s.Put(ctx, allshapes.Item{ID: "pre-1", Name: "prepopulated"})
        }),

        // Override the synthesized hot-path with a real seeded key.
        servicetest.ServiceBenchOnGet(
            bench.ReaderHotPath[allshapes.Service, string, allshapes.Item]("seed-1"),
            bench.ReaderLatencyPercentilesWithin[allshapes.Service, string, allshapes.Item](
                "seed-1",
                map[float64]time.Duration{
                    0.50: 5 * time.Microsecond,
                    0.99: 100 * time.Microsecond,
                },
            ),
        ),

        // Free-form scenario benchmark.
        servicetest.ServiceBenchCustom("put-then-get-round-trip", func(b *testing.B, s allshapes.Service) {
            for b.Loop() {
                _ = s.Put(b.Context(), allshapes.Item{ID: "rt"})
                _, _ = s.Get(b.Context(), "rt")
            }
        }),
    )
}
```

A second implementation plugs in by changing the `factory` closure — every emitted bench runs against every implementation. Comparing benchmark output across factories produces a per-method performance comparison without writing more code.

## Relationship to suite

`suite` and `bench` share `shape.Detect`, the same `On<Method>` plug-in pattern, and the same per-shape typed bindings (`ReaderBindings`, `WriterBindings`, ...) — but `bench` uses `*testing.B`-rooted `<Shape>Context` types, while `suite` uses `*testing.T`-rooted `<Shape>Context` types. Plug-in primitives are not interchangeable: a `suite.ReaderAssertion` cannot be passed to `BenchOnGet`, and a `bench.Reader` cannot be passed to `OnGet`. The split keeps bench primitives free of `t.Run` overhead and lets each side compose differently with the auto-emitted layer.

## Symbol naming

Every option is prefixed with the interface name and `Bench` (`ServiceBenchOption`, `ServiceBenchPrePopulate`, `ServiceBenchOnGet`, `ServiceBenchCustom`, ...) so multiple interfaces — and both suite and bench output for the same interface — can land in the same `*test` package without symbol collisions. The internal config struct is unexported (`serviceBenchConfig`).

## See also

- [Primitives / Benchmarking](../primitives/benchmarking.md) — `StartContract` is the runtime gate primitive bench primitives layer on top of.
- [Generators / suite](suite.md) — Tier 1 conformance with the same shape detection.
- Shape-specific bench files in `bench/` — one file per shape (`reader.go`, `writer.go`, `aggregator.go`, ...) carrying the typed Context and the five primitives.
- [Validators / benchmarks](../validators/benchmarks.md) — enforces every benchmark uses `StartContract`.
- [Validators / bench-regression](../validators/bench-regression.md) — compares benchmark output against a pinned baseline (planned).
