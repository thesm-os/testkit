# Bench

Tier 4 conformance. Reads a Go interface, classifies each method into a *shape* using the same `gen.DetectShape` rules as `suite`, and emits a `Benchmark<Iface>Contract(b, factory, opts...)` harness with two layers:

- **Auto-detected** `<Method>/hot-path` benchmarks per shape — single `b.Loop()` measurement with `b.ResetTimer` and `b.ReportAllocs`. Run by default; no consumer code required.
- **Plug-in** sub-benchmarks wired through typed extension points that match the method's shape. The consumer composes shape-appropriate primitives from `testkit/bench_*.go` — `BenchReaderHotPath`, `BenchWriterAllocsWithin`, `BenchReaderConcurrentThroughput`, etc.

A `factory func() Iface` is the single required injection. Every bench constructs a fresh implementation per run.

## Directive

```go
//go:generate testkit bench -o storetest/store_bench.gen.go Store
```

`bench` accepts exactly one type argument and emits a single output file.

## Default output

`<package>test/<subject>_bench.gen.go`.

## Method-shape detection

Identical to `suite` — the same nine-rule detection table classifies methods into Reader, Writer, Deleter, Aggregator, Lifecycle, Predicate, Pure, StreamReader, or Unknown. See [Generators / suite — shape detection](suite.md#method-shape-detection) for the table.

The shape determines:

- which auto-detected hot-path benchmark template is emitted,
- which typed bench-context (`BenchReaderContext`, `BenchWriterContext`, etc.) is wired into the plug-in dispatch,
- which bench-library primitives the consumer can plug in.

The header comment at the top of the generated file lists which methods landed in which shape, plus the count of default benchmarks and the available plug-in points.

## What is generated

For an interface with eight methods covering every shape:

```go
type Service interface {
    Get(ctx context.Context, id string) (Item, error)        // Reader
    Put(ctx context.Context, item Item) error                // Writer
    //testkit:deleter
    Delete(ctx context.Context, id string) error             // Deleter
    Count(ctx context.Context) (int, error)                  // Aggregator
    Close(ctx context.Context) error                         // Lifecycle
    Describe() string                                        // Pure
    IsEmpty() bool                                           // Predicate
    List(ctx context.Context) iter.Seq2[Item, error]         // StreamReader
}
```

the generator emits:

```go
// BenchmarkServiceContract runs performance benchmarks against
// implementations of [allshapes.Service] produced by factory.
//
//   Default benchmarks: 8 hot-path measurements across 8 methods
//   Shapes benchmarked: Reader (Get), Writer (Put), Deleter (Delete),
//                       Aggregator (Count), StreamReader (List),
//                       Lifecycle (Close), Pure (Describe), Predicate (IsEmpty)
//   Plug-in points:     ServiceBenchOnClose, ServiceBenchOnCount,
//                       ServiceBenchOnDelete, ServiceBenchOnDescribe,
//                       ServiceBenchOnGet, ServiceBenchOnIsEmpty,
//                       ServiceBenchOnList, ServiceBenchOnPut,
//                       ServiceBenchCustom
func BenchmarkServiceContract(
    b *testing.B,
    factory func() allshapes.Service,
    opts ...ServiceBenchOption,
) {
    b.Helper()
    cfg := newServiceBenchConfig(opts...)

    benchServiceClose(b, factory, &cfg)
    benchServiceCount(b, factory, &cfg)
    benchServiceDelete(b, factory, &cfg)
    benchServiceDescribe(b, factory, &cfg)
    benchServiceGet(b, factory, &cfg)
    benchServiceIsEmpty(b, factory, &cfg)
    benchServiceList(b, factory, &cfg)
    benchServicePut(b, factory, &cfg)

    for _, custom := range cfg.custom {
        b.Run(custom.name, func(b *testing.B) {
            custom.fn(b, factory())
        })
    }
}
```

One `bench<Iface><Method>` per method, plus top-level dispatch for `Custom` sub-benchmarks.

## Auto-detected hot-path benchmarks

Every method gets a `<Method>/hot-path` benchmark with the same shape:

```go
b.Run("Get/hot-path", func(b *testing.B) {
    impl := factory()
    if cfg.prePopulate != nil {
        cfg.prePopulate(b.Context(), impl)
    }
    b.ResetTimer()
    b.ReportAllocs()
    for b.Loop() {
        _, _ = impl.Get(b.Context(), "")
    }
})
```

The hot-path body calls the method with zero-value inputs in a tight `b.Loop()`. `ResetTimer` excludes setup; `ReportAllocs` enables alloc accounting. The result of any return value is discarded so the benchmark measures call overhead, not the consumer's downstream work.

The hot-path wraps the call in `defer func() { if r := recover(); r != nil { b.Skipf(...) } }()`. If a method panics on zero-value inputs, the benchmark skips with a diagnostic instead of failing. Annotate the method with `//testkit:sample` (see [suite — sample directive](suite.md#sample-directive)) to replace zero values with valid samples.

Hot-path is a measurement, not a gate — no allocation or latency ceiling is asserted. Gates come from the plug-in primitives below.

## Plug-in extension points (typed by shape)

Each method gets a `<Iface>BenchOn<Method>` option whose argument type depends on the method's shape. The consumer composes assertions from `testkit/bench_*.go`.

### Reader plug-ins

```go
servicetest.ServiceBenchOnGet(
    bench.ReaderHotPath[allshapes.Service, string, allshapes.Item]("seed-1"),
    bench.ReaderAllocsWithin[allshapes.Service, string, allshapes.Item]("seed-1", 0),
    bench.ReaderConcurrentThroughput[allshapes.Service, string, allshapes.Item]("seed-1", 4),
)
```

`...bench.Reader[T, K, V]`. Library:

- `bench.ReaderHotPath(key)` — single-thread measurement against a known key (replaces the auto-detected zero-key hot-path with a realistic one).
- `bench.ReaderAllocsWithin(key, maxAllocs)` — gate: fail if allocs/op exceeds `maxAllocs`.
- `bench.ReaderConcurrentThroughput(key, parallelism)` — `b.RunParallel`-style stress at the given parallelism.

### Writer / Deleter / Lifecycle / Aggregator / Pure / Predicate / Stream

| Method shape | `BenchOn<Method>` accepts |
|--------------|---------------------------|
| Reader | `bench.Reader[T, K, V]` |
| Writer | `bench.Writer[T, V]` |
| Deleter | `bench.Deleter[T, K]` |
| Lifecycle | `bench.Lifecycle[T]` |
| Aggregator | `bench.Aggregator[T, R]` |
| Predicate | `bench.Predicate[T]` |
| Pure | `bench.Pure[T, R]` |
| StreamReader | `bench.Stream[T, V]` |
| Unknown | `func(*testing.B, T)` (free-form) |

Bench-library inventories per shape (in `bench/`):

- **Writer** — `WriterHotPath(sample)`, `WriterAllocsWithin(sample, maxAllocs)`
- **Deleter** — `DeleterHotPath(key)`, `DeleterAllocsWithin(key, maxAllocs)`
- **Lifecycle** — `LifecycleAllocsWithin(maxAllocs)`
- **Aggregator** — `AggregatorAllocsWithin(maxAllocs)`
- **Predicate** — `PredicateAllocsWithin(maxAllocs)`
- **Pure** — `PureAllocsWithin(maxAllocs)`, `PureConcurrentThroughput(parallelism)`
- **Stream** — `StreamHotPath()`, `StreamAllocsWithin(maxAllocs)`

Each shape ships at least an `AllocsWithin` gate; Reader/Writer/Deleter/Stream additionally ship hot-path measurements that target a specific key/sample (the auto-detected hot-path uses zero-value inputs, which may not be representative for a backing store with real data). Adding a custom bench primitive is a closure over the shape's typed `Context`.

## PrePopulate

```go
servicetest.ServiceBenchPrePopulate(func(ctx context.Context, s allshapes.Service) {
    _ = s.Put(ctx, allshapes.Item{ID: "pre-1", Name: "prepopulated"})
})
```

`PrePopulate` runs before each benchmark against a fresh impl from the factory. State is not shared across benchmarks; every `<Method>/hot-path` and every plug-in primitive sees a freshly-populated impl. This keeps benchmarks isolated and stable across runs.

PrePopulate runs *outside* the timed region — `b.ResetTimer` is called after PrePopulate completes, so seed cost is excluded from the measurement.

## Custom sub-benchmarks

```go
servicetest.ServiceBenchCustom("put-then-get-round-trip", func(b *testing.B, s allshapes.Service) {
    b.ResetTimer()
    b.ReportAllocs()
    for b.Loop() {
        _ = s.Put(b.Context(), allshapes.Item{ID: "rt"})
        _, _ = s.Get(b.Context(), "rt")
    }
})
```

For benchmarks that aren't shape-aligned — multi-method round-trips, workload mixes, scenario benchmarks. Use sparingly; most performance contracts have a shape primitive that fits.

## Wiring against an implementation

```go
// servicetest/bench_test.go
package servicetest_test

func BenchmarkAllShapesContract(b *testing.B) {
    factory := func() allshapes.Service {
        s := allshapes.NewInMemoryService()
        _ = s.Put(context.Background(), allshapes.Item{ID: "seed-1", Name: "seed"})
        return s
    }

    servicetest.BenchmarkServiceContract(b, factory,
        servicetest.ServiceBenchPrePopulate(func(ctx context.Context, s allshapes.Service) {
            _ = s.Put(ctx, allshapes.Item{ID: "pre-1", Name: "prepopulated"})
        }),

        // Reader: Get
        servicetest.ServiceBenchOnGet(
            bench.ReaderHotPath[allshapes.Service, string, allshapes.Item]("seed-1"),
            bench.ReaderAllocsWithin[allshapes.Service, string, allshapes.Item]("seed-1", 0),
        ),

        // Writer: Put
        servicetest.ServiceBenchOnPut(
            bench.WriterHotPath[allshapes.Service, allshapes.Item](
                allshapes.Item{ID: "bench-w", Name: "bench"},
            ),
        ),

        // Deleter: Delete (declared with //testkit:deleter)
        servicetest.ServiceBenchOnDelete(
            bench.DeleterHotPath[allshapes.Service, string]("seed-1"),
        ),

        // Aggregator: Count — gate allocs.
        servicetest.ServiceBenchOnCount(
            bench.AggregatorAllocsWithin[allshapes.Service, int](0),
        ),

        // Lifecycle: Close — gate allocs.
        servicetest.ServiceBenchOnClose(
            bench.LifecycleAllocsWithin[allshapes.Service](0),
        ),

        // Pure: Describe — gate allocs.
        servicetest.ServiceBenchOnDescribe(
            bench.PureAllocsWithin[allshapes.Service, string](0),
        ),

        // Predicate: IsEmpty — gate allocs.
        servicetest.ServiceBenchOnIsEmpty(
            bench.PredicateAllocsWithin[allshapes.Service](0),
        ),

        // Stream: List
        servicetest.ServiceBenchOnList(
            bench.StreamHotPath[allshapes.Service, allshapes.Item](),
        ),
    )
}
```

A second implementation plugs in by changing the `factory` closure — every benchmark runs against every implementation. Comparing benchmark output across factories produces a per-method performance comparison without writing more code.

## Relationship to suite

`suite` and `bench` share `gen.DetectShape`, the same `On<Method>` plug-in pattern, and the same per-shape typed bindings (`ReaderBindings`, `WriterBindings`, etc.) — but `bench` uses `*testing.B`-rooted `BenchXContext` types, while `suite` uses `*testing.T`-rooted `XContext` types. Plug-in primitives are not interchangeable between them: a `ReaderAssertion` cannot be passed to `BenchOnGet`, and a `BenchReader` cannot be passed to `OnGet`. The split keeps the bench primitives free of `t.Run` overhead and lets each side compose differently with the auto-detected layer.

## Symbol naming

Every option is prefixed with the interface name and `Bench` (`ServiceBenchOption`, `ServiceBenchPrePopulate`, `ServiceBenchOnGet`, `ServiceBenchCustom`, etc.) so multiple interfaces — and both suite and bench output for the same interface — can land in the same `*test` package without symbol collisions. The internal accumulator type is unexported (`serviceBenchConfig`).

## See also

- [Primitives / Contract](../primitives/benchmarking.md) — `StartContract` is the runtime gate primitive bench primitives layer on top of
- [Generators / suite](suite.md) — Tier 1 conformance with the same shape detection
- Shape-specific bench files in `bench/`: `reader.go`, `writer.go`, `deleter.go`, `lifecycle.go`, `aggregator.go`, `predicate.go`, `pure.go`, `stream.go`
- [Validators / benchmarks](../validators/benchmarks.md) — enforces every benchmark uses `StartContract`
- [Validators / bench-regression](../validators/bench-regression.md) — compares benchmark output against pinned baseline
