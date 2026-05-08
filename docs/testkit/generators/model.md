# Model

Tier 2-3 conformance. Reads a Go interface, classifies each method by shape, and emits a `rapid`-driven property-based state-machine harness. Every iteration constructs a fresh SUT and (optionally) a reference implementation, runs a random sequence of actions drawn from the interface methods, and checks:

- **Differential correctness** — SUT and reference produce identical results for every action.
- **Auto-derived laws** — shape-specific invariants (ReadAfterWrite, DeleteReturnsNotFound, PureDeterminism, etc.) checked after every action.
- **Trace combinators** — temporal assertions across the action sequence (AfterEvery, EventuallyAfter, Never).
- **Goroutine leak detection** — compares goroutine sets before and after the property run.

When a failure is found, rapid shrinks the action sequence to the minimal reproducer.

## Directive

```go
//go:generate testkit model -o storetest/store_model.gen.go Store
```

`model` accepts exactly one type argument and emits a single output file.

## Default output

`<package>test/<subject>_model.gen.go`.

## What is generated

For a CRUD interface:

```go
type Store interface {
    //testkit:errors ErrNotFound
    Get(ctx context.Context, id string) (Item, error)
    Put(ctx context.Context, item Item) error
    //testkit:deleter
    Delete(ctx context.Context, id string) error
    Count(ctx context.Context) (int, error)
    Close(ctx context.Context) error
    Describe() string
    IsEmpty() bool
    List(ctx context.Context) iter.Seq2[Item, error]
}
```

the generator emits four entry points:

```go
// Run as a standard test with default 100 iterations.
func StoreModelTest(t *testing.T, factory func() store.Store, opts ...StoreModelOption)

// Run as a rapid property test (custom iteration control).
func StoreModelAssert(t *testing.T, factory func() store.Store, opts ...StoreModelOption)

// Run as a fuzz target for corpus-driven exploration.
func StoreModelFuzz(f *testing.F, factory func() store.Store, opts ...StoreModelOption)

// Exported property function for advanced composition.
func StoreModelProperty(factory func() store.Store, opts ...StoreModelOption) func(*model.T)
```

## Actions by shape

Each method is classified into a shape and gets an auto-generated action helper. CRUD shapes (Reader, Writer, Deleter) produce comparison actions that call both SUT and reference and diff results. Non-CRUD shapes produce stress-only actions when no reference is available.

| Shape | Action behavior | Example methods |
|-------|----------------|-----------------|
| Reader | `action.Reader` — compare `(V, error)` from SUT and ref | `Get(ctx, K) (V, error)` |
| Writer | `action.Writer` — compare `error` from SUT and ref | `Put(ctx, V) error` |
| Deleter | `action.Deleter` — compare `error` from SUT and ref | `Delete(ctx, K) error` |
| Aggregator | `action.Aggregator` — compare `(V, error)` | `Count(ctx) (int, error)` |
| Lifecycle | `action.Lifecycle` — compare `error` | `Close(ctx) error` |
| Pure | `action.Pure` — compare return values | `Describe() string` |
| Predicate | `action.Predicate` — compare `bool` | `IsEmpty() bool` |
| StreamReader | `action.Stream` — collect and compare elements | `List(ctx) iter.Seq2[V, error]` |
| Unknown | `action.Stress` — call SUT only, no comparison | Anything else |

Methods annotated with `//testkit:nondeterministic` emit `action.Stress` instead of comparison actions, even for shapes that would normally compare.

## Auto-derived laws

Laws are invariants checked after every action in the sequence. The generator auto-derives laws from the interface shape:

| Law | Condition | What it checks |
|-----|-----------|----------------|
| `ReadAfterWrite` | Has Reader + Writer + KeyField | Write then read with same key returns the written value |
| `DeleteReturnsNotFound` | Has Reader + Deleter + errors ErrNotFound | Delete then read returns ErrNotFound |
| `CountEqualsReference` | Has Aggregator + RefFactory | SUT count matches reference count |
| `PureDeterminism` | Has Pure methods (not `nondeterministic`) | N calls with same input produce identical output |
| `PredicateConsistency` | Has Predicate methods (not `nondeterministic`) | N calls produce same bool |
| `StreamReentrancy` | Has StreamReader methods | Two collect passes produce same elements |

Laws are suppressed per-method with `//testkit:nondeterministic` (e.g., `Clock.Now()`).

## Concurrent stress

When configured with `StoreModelConcurrent(opts...)`, the runner spawns multiple goroutines executing random actions against a shared SUT. CRUD shapes (Reader, Writer, Deleter) get Porcupine linearizability checking — the recorded operation history is verified against a sequential specification. Non-CRUD shapes are stress-tested without linearizability (call-only, checking for panics and races).

## Trace combinators

Laws can include temporal assertions over the action trace:

- **`AfterEvery(trigger, check)`** — after every action matching `trigger`, the next action must satisfy `check`.
- **`EventuallyAfter(trigger, check, budget)`** — after `trigger`, `check` must hold within `budget` subsequent actions.
- **`Never(check)`** — `check` must never hold in any action.

Trace combinators are attached via `StoreModelExtraLaws`.

## TimeDriven integration

Interfaces annotated with `//testkit:time-aware` on any method get `TestClock` integration. The generator emits `StoreModelClockFactory(func(clock.Clock) Store)` — the runner creates paired `TestClock` instances for SUT and reference, advances both by the same random duration between actions, and injects the clock into the factory.

## Directives consumed

| Directive | Effect |
|-----------|--------|
| `errors ErrX` | Auto-derives `DeleteReturnsNotFound` law when Reader+Deleter present |
| `deleter` | Marks method as Deleter shape |
| `mutator` | Marks method as Mutator shape |
| `keyfield FieldName` | Key extraction for reference map synthesis |
| `nondeterministic` | Suppresses determinism laws; emits `action.Stress` instead of comparison |
| `time-aware` | Enables `TestClock` pair injection |
| `appends` | Chain: marks append operation |
| `verifies` | Chain: marks integrity verifier |
| `replays` | Chain: marks replay/stream operation |
| `partition-by Field` | Chain: partition key field |
| `entry-id Field` | Chain: unique entry ID field |
| `depends-on Field` | Chain: causal dependency field |
| `hash Pkg.Func` | Chain: custom hash function |
| `integration-only` | Skips method entirely |

## Extension points

```go
storetest.StoreModelTest(t, factory,
    // Differential mode: compare SUT against a reference.
    storetest.StoreModelReference(func() store.Store {
        return store.NewInMemoryStore()
    }),

    // Add custom actions beyond the auto-generated ones.
    storetest.StoreModelExtraActions(
        storetest.StoreGetAction(),    // auto-generated per-method helper
        storetest.StorePutAction(),
        storetest.StoreDeleteAction(),
    ),

    // Add custom laws.
    storetest.StoreModelExtraLaws(
        customInvariant,
    ),

    // Concurrent stress with linearizability.
    storetest.StoreModelConcurrent(
        model.Workers(4),
        model.OpsPerWorker(50),
    ),

    // TimeDriven (for //testkit:time-aware interfaces).
    storetest.StoreModelClockFactory(func(c clock.Clock) store.Store {
        return store.NewTTLStore(c)
    }),
)
```

## Rapid facade

Generated code imports `go.thesmos.sh/testkit/model` instead of `pgregory.net/rapid` directly. The `model` package re-exports all rapid types and generator functions (`model.T`, `model.Generator[V]`, `model.StringMatching`, `model.SliceOf`, etc.) so consumers extending model specs with custom actions or laws never need rapid in their `go.mod`.

## Stub-as-reference pattern

The recommended reference implementation for model testing is a configured stub:

```go
storetest.StoreModelReference(func() store.Store {
    return storetest.NewStoreStub(nil, storetest.StoreStubDelegateTo(store.NewInMemoryStore()))
})
```

Pass `nil` as the `testing.TB` argument (not `f` or `t`) when the reference is created inside a property iteration — this skips cleanup registration which is forbidden inside fuzz bodies.

## Fuzz safety

Stubs detect `*testing.F` and skip `tb.Cleanup` registration. This allows stubs to be constructed inside fuzz iteration bodies without panicking. The model's `Fuzz` entry point passes `*testing.F` to the outer harness; rapid's per-iteration `*rapid.T` (which implements `testing.TB`) is available inside the property function for per-iteration cleanup.

## See also

- [Generators / suite](suite.md) — Tier 1 single-call conformance (same shape detection)
- [Generators / bench](bench.md) — Tier 4 benchmarking (same shape detection)
- [Primitives / concurrency](../primitives/concurrency.md) — goroutine leak detection primitives
- [Primitives / clock](../primitives/clock.md) — `TestClock` for deterministic time testing
