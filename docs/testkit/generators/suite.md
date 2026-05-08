# Suite

Tier 1 conformance. Reads a Go interface, classifies each method into a *shape* by signature pattern, and emits an `Assert<Iface>Contract(t, factory, opts...)` test harness with two layers of subtests:

- **Auto-detected** subtests derived from the method's shape and `//testkit:` directives. The consumer writes nothing for these.
- **Plug-in** subtests wired through typed extension points that match the method's shape. The consumer composes shape-appropriate primitives from `testkit/*_assert.go`.

A `factory func() Iface` is the single required injection. Every subtest constructs a fresh implementation per run; tests are parallelizable.

## Directive

```go
//go:generate testkit suite -o storetest/store_spec.gen.go Store
```

`suite` accepts exactly one type argument and emits a single output file.

## Default output

`<package>test/<subject>_spec.gen.go`.

## Method-shape detection

Each method is classified by `gen.DetectShape` from its signature. The shape determines:

- which auto-detected subtests run for that method,
- which typed context (`ReaderContext`, `WriterContext`, etc.) is wired into plug-in dispatch,
- which assertion-library primitives the consumer can plug in.

Detection rules (first match wins):

| Rule | Signature | Shape |
|------|-----------|-------|
| 1 | returns `iter.Seq[V]` or `iter.Seq2[V, error]` | **StreamReader** |
| 2 | no ctx, returns `bool` only | **Predicate** |
| 3 | no ctx, no error return | **Pure** |
| 4 | `func(ctx, K) (V, error)` with `V != error` | **Reader** |
| 5 | `func(ctx, K) error` (default) | **Writer** |
| 5 | `func(ctx, K) error` + `//testkit:deleter` | **Deleter** |
| 6 | `func(ctx, V) (R, error)` with `R != error` | **Writer** (with result) |
| 7 | `func(ctx) (T, error)` | **Aggregator** |
| 8 | `func(ctx) error` | **Lifecycle** |
| 9 | none of the above | **Unknown** |

The header comment at the top of the generated file lists which methods landed in which shape, so a misclassification is obvious on inspection.

## What is generated

Given

```go
type Store interface {
    //testkit:errors ErrNotFound
    //testkit:ctx
    //testkit:pure
    Get(ctx context.Context, id string) (Item, error)

    //testkit:nilsafe
    //testkit:ctx
    Put(ctx context.Context, item Item) error

    //testkit:ctx
    Delete(ctx context.Context, id string) error

    //testkit:bounded 0 1000
    Count(ctx context.Context) int

    //testkit:timeout 5s
    Ping(ctx context.Context) error

    //testkit:deprecated PutBatch
    LegacyPut(ctx context.Context, item Item) error
}
```

the generator emits a header documenting subtest count, shape classification, applied directives, and plug-in extension points:

```
// AssertStoreContract runs conformance assertions against
// implementations of [basic.Store] produced by factory.
//
//   Default subtests: 27 across 6 methods
//   Shapes detected:  Reader (Get), Writer (Delete, LegacyPut, Put),
//                     Lifecycle (Ping), Unknown (Count)
//   Directives:       bounded (Count: 0..1000), errors (Get→ErrNotFound),
//                     pure (Get), deprecated (LegacyPut→PutBatch),
//                     timeout (Ping: 5s), nilsafe (Put)
//   Plug-in points:   StoreOnCount, StoreOnDelete, StoreOnGet,
//                     StoreOnLegacyPut, StoreOnPing, StoreOnPut,
//                     StoreOnAll, StoreCustom
```

Then the entry point:

```go
func AssertStoreContract(
    t *testing.T,
    factory func() basic.Store,
    opts ...StoreOption,
) {
    t.Helper()
    cfg := newStoreConfig(opts...)

    runStoreCount(t, factory, &cfg)
    runStoreDelete(t, factory, &cfg)
    runStoreGet(t, factory, &cfg)
    runStoreLegacyPut(t, factory, &cfg)
    runStorePing(t, factory, &cfg)
    runStorePut(t, factory, &cfg)

    // Cross-method assertions (StoreOnAll).
    for _, a := range cfg.onAll { ... }

    // Free-form custom subtests (StoreCustom).
    for _, custom := range cfg.custom { ... }
}
```

One `run<Iface><Method>` per method, plus the top-level cross-method and custom dispatch.

## Auto-detected subtests

These run without any directive.

**All ctx-taking shapes** (Reader, Writer, Deleter, Aggregator, Lifecycle):

- `<Method>/smoke` — call with zero-value inputs, ignore the result.
- `<Method>/ctx cancellation` — pre-cancelled context, expect a context error.
- `<Method>/ctx deadline` — context with deadline already past, expect a context error.
- `<Method>/nil context` — pass `nil` ctx, must not panic.

**StreamReader**:

- `<Method>/smoke`
- `<Method>/iterate no error`
- `<Method>/break mid-stream`
- `<Method>/double iteration`

**Predicate, Pure, Unknown**:

- `<Method>/smoke` only.

The fact that every ctx-taking method gets cancellation, deadline, and nil-context coverage by default is load-bearing — these are the cheapest tests with the highest payoff for catching context-related bugs.

### Smoke test recovery

Every `<Method>/smoke` subtest wraps the call in `defer func() { if r := recover(); r != nil { t.Skipf(...) } }()`. If a method panics on zero-value inputs (common for methods with parameter preconditions like `Combine(a, b Digest)` where `Digest{}` is invalid), the smoke test skips with a diagnostic message instead of failing:

```
--- SKIP: TestHasherContract/Combine/smoke (0.00s)
    smoke: Combine panicked on zero-value args (Combine: zero-value Digest)
      — supply sample values via HasherOnCombine
```

To eliminate the skip, annotate the method with `//testkit:sample` (see below) so the smoke test calls with valid values.

## Sample directive

Methods whose parameters have preconditions that make zero-value args panic need sample builder functions. The `//testkit:sample` directive provides one builder per non-context parameter:

```go
type Hasher interface {
    //testkit:sample SampleDigest SampleDigest
    Combine(left, right Digest) Digest
}
```

Every builder is `func(I) T` — takes the SUT as its sole argument, returns a value of the parameter type:

```go
// In the source package (qualified in generated code):
func SampleDigest(h Hasher) Digest {
    return h.Hash([]byte{})  // impl-aware: correct size for any Hasher
}

// Or in the output test package (emitted unqualified):
func TestSampleDigest(_ crypto.Hasher) Digest {
    var d Digest; d[0] = 0x42; return d
}
```

The generator resolves unqualified names by scope: if the name exists in the source package, it's emitted with the source qualifier (`crypto.SampleDigest(impl)`). If not found in the source package, it's emitted unqualified (assumes the output test package). Fully qualified names (`crypto.SampleDigest`) are emitted as-is.

The sample expressions replace zero values in:

- **Smoke tests**: `s.Combine(crypto.SampleDigest(s), crypto.SampleDigest(s))`
- **Plug-in Call closures**: `impl.Combine(crypto.SampleDigest(impl), crypto.SampleDigest(impl))`
- **Bench hot-path**: same substitution with `impl` as receiver

Context parameters are not affected — they always get `t.Context()` or `b.Context()`.

### Where to put sample functions

- **Source package** — when the builder uses SUT methods to construct impl-aware values (e.g., `SampleDigest(h Hasher) Digest { return h.Hash(nil) }`). This is the recommended default.
- **Output test package** — when the builder is test-only infrastructure that doesn't belong in the public API. Define it in a hand-written file alongside `spec_test.go` (e.g., `hashertest/sample_helpers.go`).

## Directive-driven subtests

Added when the directive is present on the method:

| Directive | Adds subtest | Behavior |
|-----------|--------------|----------|
| `errors ErrX` | `<Method>/returns X` | Calls method with zero-value, asserts `errors.Is(err, ErrX)`. One subtest per sentinel listed. |
| `nilsafe` | `<Method>/nilsafe` | `testkit.AssertNilSafe` wrapping the method call. |
| `pure` | `<Method>/pure` | `testkit.AssertPure` with `prePopulate` + observation closure. Skipped if `PrePopulate` not configured. |
| `bounded N M` | `<Method>/bounded N M` | `testkit.AssertBounded(t, N, M, fn)`. |
| `timeout D` | `<Method>/timeout D` | `testkit.AssertTimeout(t, D, fn)`. |
| `deprecated <Replacement>` | `<Method>/deprecated` | `t.Logf("<Method> is deprecated, use <Replacement> instead")` + `t.Skip("deprecated method")`. |
| `sample Fn [Fn...]` | *(no extra subtest)* | Replaces zero-value arguments with `Fn(impl)` calls in smoke tests, hot-path benchmarks, and plug-in dispatch `Call` closures. See [Sample directive](#sample-directive) below. |

The `ctx` directive doesn't add a subtest — the auto-detected `ctx cancellation`/`ctx deadline`/`nil context` subtests already cover its semantics. Marking a method with `//testkit:ctx` is a documentation hint that the consumer expects context handling, not a new test.

## Plug-in extension points (typed by shape)

Each method gets a `<Iface>On<Method>` option whose argument type is determined by the method's shape. The consumer composes assertions from `testkit/<shape>_assert.go` and passes them in.

### Reader plug-ins

```go
storetest.StoreOnGet(
    suite.AssertReturnsForKey[basic.Store, string, basic.Item]("known-1", basic.Item{...}),
    suite.AssertReturnsSentinel[basic.Store, string, basic.Item]("nonexistent", basic.ErrNotFound),
    suite.AssertConsistentReads[basic.Store, string, basic.Item]("known-1", 3),
)
```

`StoreOnGet` accepts `...suite.ReaderAssertion[basic.Store, string, basic.Item]`. The shipped reader-assertion library:

- `AssertReturnsForKey(key, want)`
- `AssertReturnsSentinel(unknown, sentinel)`
- `AssertConsistentReads(key, n)`
- `AssertReadsAreNonMutating(key, observe)`
- `AssertReaderConcurrentSafe(key, n)`

### Writer plug-ins

```go
storetest.StoreOnPut(
    suite.AssertWriteSucceeds[basic.Store, basic.Item](basic.Item{ID: "new-1"}),
)
```

`...suite.WriterAssertion[basic.Store, basic.Item]`. Library:

- `AssertWriteSucceeds(sample)`
- `AssertWriteIsObservable(sample, observe)`
- `AssertWriteRejectInvalid(invalid, sentinel)`
- `AssertWriteOverwrite(...)`

### Deleter / Lifecycle / Aggregator / Predicate / Pure / Stream

Each shape has its own typed plug-in. The dispatch table:

| Method shape | `On<Method>` accepts |
|--------------|----------------------|
| Reader | `ReaderAssertion[T, K, V]` |
| Writer | `WriterAssertion[T, V]` |
| Deleter | `DeleterAssertion[T, K]` |
| Lifecycle | `LifecycleAssertion[T]` |
| Aggregator | `AggregatorAssertion[T, R]` |
| Predicate | `PredicateAssertion[T]` |
| Pure | `PureAssertion[T, R]` |
| StreamReader | `StreamAssertion[T, V]` |
| Unknown | `func(*testing.T, T)` (free-form) |

Library inventories per shape:

- **Deleter** — `AssertDeleteSucceeds`, `AssertDeleteIdempotent`, `AssertDeleteReturnsNotFound`
- **Lifecycle** — `AssertLifecycleSucceeds`, `AssertLifecycleIdempotent`, `AssertLifecycleRespectsContext`
- **Aggregator** — `AssertAggregatorReturns`, `AssertAggregatorBounded`, `AssertAggregatorConsistent`
- **Predicate** — `AssertPredicateReturns`, `AssertPredicateConsistent`
- **Pure** — `AssertDeterministic`, `AssertNoSideEffects`
- **Stream** — `AssertStreamCompletes`, `AssertStreamRespectsBreak`, `AssertStreamReentrant`, `AssertStreamYieldsInOrder`, `AssertStreamHasNoDuplicates`

Adding a custom assertion is just writing a function that matches the shape's `Assertion` type — a one-line closure over the shape's `Context`.

### Cross-method plug-ins

`<Iface>OnAll(...)` accepts `CrossMethodAssertion[T]` — assertions that span multiple methods of the same interface. These are wired without a `prePopulate` step (cross-method primitives manage their own state).

```go
storetest.StoreOnAll(
    suite.AssertReadAfterWrite[basic.Store, string, basic.Item](
        basic.Item{ID: "cross-1", Name: "cross"},
        func(ctx context.Context, s basic.Store, item basic.Item) error { return s.Put(ctx, item) },
        func(ctx context.Context, s basic.Store, id string) (basic.Item, error) { return s.Get(ctx, id) },
        func(item basic.Item) string { return item.ID },
    ),
)
```

Cross-method library:

- `AssertReadAfterWrite(sample, write, read, key)`
- `AssertDeleteRemovesValue(sample, write, read, delete, key, sentinel)`
- `AssertStreamReflectsMutations(sample, write, stream, key)`

### Free-form custom subtests

For contracts not expressible via the shape primitives:

```go
storetest.StoreCustom("custom subtest", func(t *testing.T, s basic.Store) {
    testkit.NoError(t, s.Put(t.Context(), basic.Item{ID: "c"}), "custom put")
})
```

The function receives a fresh `factory()`-produced impl. Use this sparingly — most contracts have a shape primitive that fits.

## PrePopulate

```go
storetest.StorePrePopulate(func(ctx context.Context, s basic.Store) {
    _ = s.Put(ctx, basic.Item{ID: "known-1", Name: "test"})
})
```

`PrePopulate` runs before subtests that need pre-existing state. It's invoked once per relevant subtest against a fresh impl from the factory — never reused across subtests, so tests stay isolated.

Subtests that depend on `PrePopulate` (notably `<Method>/pure` and reader/writer plug-ins that assert behavior against existing keys) skip with a diagnostic when `PrePopulate` is not configured rather than fatal — consumers ramp up their seed function as their plug-in vocabulary grows.

## Wiring against an implementation

```go
// storetest/spec_test.go
package storetest_test

func TestInMemoryStoreContract(t *testing.T) {
    t.Parallel()
    factory := func() basic.Store { return basic.NewInMemoryStore() }

    storetest.AssertStoreContract(t, factory,
        storetest.StorePrePopulate(func(ctx context.Context, s basic.Store) {
            _ = s.Put(ctx, basic.Item{ID: "known-1", Name: "test"})
        }),
        storetest.StoreOnGet(
            suite.AssertReturnsForKey[basic.Store, string, basic.Item]("known-1", basic.Item{ID: "known-1", Name: "test"}),
            suite.AssertReturnsSentinel[basic.Store, string, basic.Item]("nonexistent", basic.ErrNotFound),
        ),
        storetest.StoreOnPut(
            suite.AssertWriteSucceeds[basic.Store, basic.Item](basic.Item{ID: "new-1"}),
        ),
        storetest.StoreOnDelete(
            suite.AssertWriteSucceeds[basic.Store, string]("known-1"),
        ),
        storetest.StoreOnAll(
            suite.AssertReadAfterWrite[basic.Store, string, basic.Item](...),
        ),
    )
}
```

A second implementation (production, integration-tested against a real backend, etc.) plugs in by changing the `factory` closure — every contract subtest runs against every implementation that satisfies the interface. This is the conformance-suite pattern: one set of tests, N implementations.

## Symbol naming

Every option is prefixed with the interface name (`StoreOption`, `StorePrePopulate`, `StoreOnGet`, `StoreCustom`, etc.) so multiple interfaces can generate into the same `*test` package without symbol collisions. The internal accumulator type is unexported (`storeConfig`).

## Layout conventions

A typical interface generates into a `<pkg>test/` sub-package. The layout after generation:

```
crypto/
  hasher.go              # interface + //go:generate directives + sample functions
  inmemory.go            # production implementation
  cryptotest/
    hasher_spec.gen.go   # generated suite (DO NOT EDIT)
    hasher_bench.gen.go  # generated bench (DO NOT EDIT)
    hasher_stub.gen.go   # generated stub (DO NOT EDIT)
    hasher_stub.go       # hand-written stub companion (DelegateTo wiring)
    sample_helpers.go    # hand-written sample builders (if in test package)
    spec_test.go         # hand-written: TestHasherContract, BenchmarkHasherContract
```

**What goes where:**

| File | Owner | Contents |
|------|-------|----------|
| `*.gen.go` | Generator | Never edit. Regenerated by `go generate`. |
| `*_stub.go` | Developer | Stub companion: `NewStdlibHasherStub` wrapping `DelegateTo`. Optional. |
| `sample_helpers.go` | Developer | `func TestSampleDigest(_ Hasher) Digest` — sample builders that don't belong in the source package. |
| `spec_test.go` | Developer | Test functions wiring `AssertHasherContract` and `BenchmarkHasherContract` with factory + options. |

**Naming the test file.** Use `spec_test.go` for the suite+bench wiring. If the interface has a model harness too, add `model_test.go` for the model wiring. One test file per harness keeps diffs clean.

**Multiple interfaces in one package.** When a package has several interfaces (e.g., `crypto.Hasher`, `crypto.Signer`, `crypto.MAC`), all generators write into the same `cryptotest/` directory. Symbol collisions are avoided by the interface-name prefix on every generated symbol.

## See also

- [Primitives / directive-assertions](../primitives/directive-assertions.md) — the runtime helpers suite calls (`AssertNilSafe`, `AssertCtxCancellation`, `AssertCtxDeadline`, `AssertNilCtx`, `AssertTimeout`, `AssertPure`, `AssertBounded`)
- Shape-specific assertion files in `suite/`: `reader.go`, `writer.go`, `deleter.go`, `lifecycle.go`, `aggregator.go`, `predicate.go`, `pure.go`, `stream.go`, `cross.go`
- [Generators / model](model.md) — Tier 2-3 follow-up that pairs with suite for stateful interfaces
