# Bench

Tier 4 conformance. Reads a Go interface, classifies each method into a *shape* by signature pattern, and emits a `Benchmark<Iface>Contract(b, factory, opts...)` harness.

The generated `bench` harness evaluates implementations against three layers of performance contracts:
1. **Baseline throughput:** Auto-emitted `hot-path` and `concurrent-4` sub-benchmarks for every method.
2. **Deterministic budget gates:** Opt-in directives (e.g., `//testkit:allocs 0`, `//testkit:latency 100us`) that fail the CI run if an implementation violates performance ceilings.
3. **Shape-specific plug-ins:** Extensible `BenchOn<Method>` hooks to inject domain-specific load tests (e.g., measuring a `Reader` against 10,000 keys instead of just 1).

A `factory func() Iface` is the single required injection. Every per-method benchmark constructs a fresh implementation via the factory, isolating state across runs.

## Directive

```go
//go:generate testkit bench -o storetest/store_bench.gen_test.go Store
```

`bench` accepts exactly one type argument and emits a single test file (`_test.go`).

## The Conformance Pattern

Like the `suite` generator, `bench` produces entry points designed for the **Multi-Implementation Conformance Pattern**. 

```go
func BenchmarkStoreContract(
    b *testing.B, 
    factory StoreFactory, 
    opts ...StoreBenchOption,
)
```

You write the benchmark wiring once. By passing different factories (e.g., `InMemoryStore`, `RedisStore`, `PostgresStore`), you instantly generate a comparative performance matrix across all your implementations.

## Auto-Emitted Primitives

Every method is classified by `shape.Detect` from its signature (see [Shapes](shapes.md)). Based on this shape, the generator emits default sub-benchmarks without requiring any consumer code.

For every non-skipped method, the harness emits:
- `<Method> / hot-path` — Single-goroutine `b.Loop()` measurement reporting `ns/op` and `allocs/op`.
- `<Method> / concurrent-4` — `b.RunParallel`-style throughput measurement at a parallelism of 4.

> **Note on Lifecycle Shapes:** `Lifecycle`, `VoidLifecycle`, and `Unknown` shapes skip the `concurrent-4` emission by default. Concurrent invocation of `Init()` or `Close()` on a shared implementation is rarely safe.

### The Zero-Allocation Guarantee
The auto-emitted benchmark helpers are strictly optimized. The harness guarantees zero allocations from the testing infrastructure inside the timed `b.Loop()` and `b.RunParallel` blocks. Every reported allocation (`allocs/op`) belongs strictly to your implementation.

### Smoke Values & `//testkit:sample`
To benchmark a method, the generator must pass arguments to it. By default, it synthesizes zero-value literals. If your method fast-fails on zero values (e.g., `if id == "" { return ErrInvalid }`), you will inadvertently benchmark the error path instead of the success path.

Use the `//testkit:sample` directive to provide a valid builder function:

```go
//testkit:sample SampleRecord
Put(ctx context.Context, item Record) error
```

The generator will invoke `SampleRecord()` once, outside the timed loop, and feed the valid result into the hot-path continuously.

## Opt-in Budget Gates (CI Enforcement)

The most powerful feature of the `bench` generator is turning performance metrics into hard CI failures. You apply these gates via directives on the interface methods.

| Directive | Sub-benchmark Emitted | Behavior |
|-----------|-----------------------|----------|
| `//testkit:allocs N` | `<Method> / allocs-within-N` | Fails the benchmark when `allocs/op` exceeds `N`. `N=0` enforces an alloc-free hot-path. |
| `//testkit:latency D` | `<Method> / latency-within-D` | Fails the benchmark when the mean `ns/op` exceeds the duration ceiling `D` (e.g., `100us`). |
| `//testkit:percentiles pX=D...` | `<Method> / percentiles` | Records the duration distribution and reports percentiles (e.g., `p50`, `p95`, `p99`) via `b.ReportMetric()`. Fails if any budgeted percentile exceeds its ceiling (e.g., `p99=500us`). |

When a method carries these directives, the generator emits the corresponding `allocs-within` or `latency-within` gate tests. These gates are strict assertions—if the implementation violates the budget, the `go test -bench` command exits non-zero.

## Injecting State via `BenchOption`

Like `suite`, you configure the benchmark harness by passing `StoreBenchOption` values into the entry point.

### `StoreBenchPrePopulate(func(impl T))`
Because you cannot benchmark a `Get` method on an empty database, you must seed state. `PrePopulate` registers a one-shot seeder that runs against every freshly-instantiated implementation *before* the per-method benchmarks observe it. 

**Crucially**, the cost of `PrePopulate` is paid *outside* the timed `b.Loop()`. It does not pollute your `ns/op` measurements.

```go
BenchmarkStoreContract(b, factory, 
    storetest.StoreBenchPrePopulate(func(ctx context.Context, s basic.Store) {
        _ = s.Put(ctx, basic.Item{ID: "known-1"})
    }),
)
```

### Shape-Specific Plug-ins
You can override the auto-emitted hot-paths or add new load profiles by injecting primitives into the `<Iface>BenchOn<Method>` slots. These slots are strongly typed by the method's detected shape.

```go
BenchmarkStoreContract(b, factory,
    storetest.StoreBenchOnGet(
        // Override the auto-emitted hot-path to use a specific seeded key
        bench.ReaderHotPath[basic.Store, string, basic.Item]("known-1"),
    ),
)
```

### Free-Form Custom Benchmarks
For measurements outside the shape vocabulary (e.g., multi-method scenarios like reading while writing), use the `Custom` extension point:

```go
storetest.StoreBenchCustom("read-write-contention", func(b *testing.B, s basic.Store) {
    // Custom b.RunParallel block...
})
```

## Integration with Stubs

When your factory returns a generated `testkit stub` (often used to benchmark the overhead of decorators or intermediate layers), the `bench` harness automatically triggers the stub's `BenchMode()`. 

In `BenchMode`, the stub disables all call recording and expectation tracking. The per-call overhead drops to near-zero, ensuring your benchmark reflects the underlying implementation's speed, not the testkit recorder's overhead.

## Layout Conventions

A typical interface generates its bench harness into a `<pkg>test/` sub-package. 

**What goes where:**

| File | Owner | Contents |
|------|-------|----------|
| `*_bench.gen_test.go` | Generator | The generated benchmark harness (DO NOT EDIT). |
| `bench_test.go` | Developer | Hand-written `func Benchmark<Iface>Contract(b *testing.B)` wiring the harness with the factory and `BenchOption` injections. |
| `sample_helpers.go` | Developer | Hand-written `func TestSampleX(_ Iface) X` builder functions for `//testkit:sample`. |

Keep `bench_test.go` separated from `spec_test.go` (the Tier 1 suite). Benchmarks often require different environment conditions, build tags, or execution flags (e.g., `-benchmem`).

## See also

- [Primitives / Benchmarking](../primitives/benchmarking.md) — How to write custom `StartContract` primitives.
- [Shape Classification](shapes.md) — The 21 signature shapes that drive the defaults.
- [Generators / suite](suite.md) — Tier 1 conformance testing.
