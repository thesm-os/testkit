# Stub

Generates a per-method test double for a Go interface. Each method gets a stub type that embeds `*testkit.MethodStub[Call]` — composing recording, fault injection, latency, strict mode, call-count expectations, clock injection, and rand-source injection — and adds two type-safe dispatch entries: `Func(fn)` and `Returns(values...)`.

The aggregate stub composes these per-method stubs into a single value implementing the target interface and provides constructor options, `Reset`, and a compile-time interface check.

## Directive

```go
//go:generate testkit stub -o storetest/store_stub.gen.go Store
//go:generate testkit stub -o storetest/cache_stub.gen.go Cache

// Multiple types in one file:
//go:generate testkit stub -o storetest/stubs.gen.go Store Cache
```

## Default output

`<package>test/<subject>_stub.gen.go` — one file per type when `-o` is omitted.

## What is generated

For

```go
type Store interface {
    Get(ctx context.Context, id string) (Item, error)
    Put(ctx context.Context, item Item) error
    Delete(ctx context.Context, id string) error
}
```

the generator emits two files: the stub (`store_stub.gen.go`) and tests for the generated plumbing (`store_stub.gen_test.go`).

### Call types — one per method

```go
type StoreGetCall struct {
    Ctx    context.Context
    ID     string
    Result Item   // populated after dispatch
    Err    error
}
```

The call struct holds inputs and outputs. The `Recorder[StoreGetCall]` embedded in the per-method stub captures the populated call after every dispatch — tests inspect both the arguments passed in and the values returned.

Result fields use the return-value names from the source interface when named; unnamed returns produce positional names (`Result0`, `Result1`, ...). `Err` is the conventional name for the error position.

### Per-method stubs

```go
type StoreGetStub struct {
    *testkit.MethodStub[StoreGetCall]
    fn       func(context.Context, string) (Item, error)
    fallback *storeGetReturn
}

func (s *StoreGetStub) Returns(result Item, err error) *StoreGetStub
func (s *StoreGetStub) Func(fn func(context.Context, string) (Item, error)) *StoreGetStub
```

Each per-method stub embeds `*MethodStub[Call]`, so the full primitive surface is available without indirection: `Faults`, `FaultsWhen`, `FaultsWithProbability`, `FaultsFor`, `FaultsUntil`, `SetFault`, `Latency`, `Strict`, `Times`, `TimesAtLeast`, `Verify`, `WithClock`, `WithRandSource`, `BenchMode`, plus the entire `Recorder[Call]` API (`CallCount`, `Calls`, `Filter`, `WaitForN`, `OnRecord`, `NewGate`, `Timestamped`, ...).

`Returns` stores a fallback return value; `Func` stores a function override.

### Aggregate stub + constructor

```go
type StoreStub struct {
    OnDelete *StoreDeleteStub
    OnGet    *StoreGetStub
    OnPut    *StorePutStub
    strict   bool
}

func NewStoreStub(tb testing.TB, opts ...StoreStubOption) *StoreStub
```

`NewStoreStub` constructs each per-method stub via `testkit.NewMethodStub[Call](tb, "Store.<Method>")`, applies options, and registers `tb.Cleanup` that invokes `Verify` on every method (auto-verifies `Times`/`TimesAtLeast` expectations at test end).

### Constructor options

```go
StoreStubStrict()                       // turn on strict mode for every method
StoreStubDelegateTo(impl Store)         // forward every method to a real implementation
StoreStubWithClock(clk testkit.Clock)   // propagate clock to every method
StoreStubWithRandSource(testkit.RandSource)
StoreStubBenchMode()                    // disable recording on every method

WithStoreGet(fn func(...) (...))        // per-method Func override at construction time
WithStorePut(fn func(...) error)
WithStoreDelete(fn func(...) error)
```

Notice the option-naming convention: `<StubName><Verb>(...)` for stub-wide options, `With<StubName><Method>(...)` for per-method dispatch overrides. The full stub name (`StoreStub`, not `Store`) is used as the prefix on `With*` options to avoid collisions when multiple stubs live in the same package.

### Compile-time interface check

```go
var _ basic.Store = (*StoreStub)(nil)
```

If the source interface gains or loses a method, the next regeneration changes this line and any obsolete dispatch — but the check ensures the stub never silently drifts from the interface.

### Reset

```go
func (s *StoreStub) Reset() {
    s.OnDelete.Reset()
    s.OnGet.Reset()
    s.OnPut.Reset()
}
```

`Reset` rewinds observation state on every method (recorded calls, fault counters, `Times`/`TimesAtLeast`). It does **not** clear `Func`, `Returns`, or `Faults` — behavior is preserved across resets so a single `NewStoreStub` can drive multiple test phases.

### Interface method implementations

For methods returning an error, dispatch is:

```go
func (s *StoreStub) Get(ctx context.Context, id string) (Item, error) {
    s.OnGet.SleepLatency()
    call := StoreGetCall{Ctx: ctx, ID: id}
    if fired, faultErr := s.OnGet.ShouldFaultFor(call); fired {
        call.Err = faultErr
        s.OnGet.Record(call)
        return Item{}, faultErr
    }
    if s.OnGet.fn != nil {
        r0, r1 := s.OnGet.fn(ctx, id)
        call.Result = r0
        call.Err = r1
        s.OnGet.Record(call)
        return r0, r1
    }
    if s.OnGet.fallback != nil {
        f := s.OnGet.fallback
        call.Result = f.Result
        call.Err = f.Err
        s.OnGet.Record(call)
        return f.Result, f.Err
    }
    s.OnGet.FailUnexpectedCall(call)
    s.OnGet.Record(call)
    return Item{}, nil
}
```

Order: `SleepLatency` → fault check → `Func` → `Returns` fallback → `FailUnexpectedCall` (strict mode fatal) → zero-value return. The call struct is populated with results in every successful path and recorded.

Methods that don't return an error skip the fault check.

### iter.Seq / iter.Seq2 detection

When a method returns `iter.Seq[T]` or `iter.Seq2[V, error]`, the per-method stub gains a `Yields` helper that constructs a `Func` returning a single-pass iterator:

```go
func (s *ScannerKeysStub) Yields(items ...string) *ScannerKeysStub
```

For `iter.Seq2[V, error]`, an additional helper yields values then a final error:

```go
func (s *ScannerScanStub) YieldsError(items []Item, err error) *ScannerScanStub
```

The fault path is omitted on iterator methods — errors flow through the iterator's pair, not the return.

### Generated tests

Alongside the stub, a `_test.go` file exercises the generated plumbing — every `Func` path, every `Returns` path, every fault path, every constructor option, recording, and `Reset`. The tests don't depend on domain logic; they verify the generator's output is internally consistent.

## Directive-driven additions

The generator reads `//testkit:` directives and emits per-method helpers or alters dispatch logic:

| Directive | Effect |
|-----------|--------|
| `errors ErrA ErrB` | Emits a `Fault<ShortName>()` helper per sentinel (e.g., `s.OnGet.FaultNotFound()` calls `s.Faults(ErrNotFound, 1)`). |
| `wrapped-via Target` | Modifies the `Fault<ShortName>()` helpers to wrap the sentinel via the specified target error struct. |
| `deprecated <Replacement>` | Injects `tb.Logf("<Method> is deprecated, use <Replacement> instead")` at the top of the generated dispatch (if `tb` is non-nil). |
| `integration-only` | Skips stub emission for that method. The compile-time interface check still includes it, so consumers must wire it via `DelegateTo`. |
| `retry-succeeds-on-attempt N` | Emits a `RetrySchedule(err)` helper that returns a fault sequence simulating a transient failure that succeeds on the Nth attempt. |
| `partition Field` | Emits `FaultForPartition` and `FaultForOtherPartitions` helpers to inject faults isolated to a specific request parameter (e.g., isolating a network fault to a specific tenant ID). |
| `order-after Method` | Emits an `AssertAfter` check inside the dispatch body (active in strict mode) to fatal the test if the method is called out of order. |

## Usage patterns

### Vanilla stub

```go
stub := storetest.NewStoreStub(t)
stub.OnGet.Returns(Item{ID: "x"}, nil)

result, err := stub.Get(ctx, "x")
testkit.NoError(t, err, "Get must succeed")
testkit.Equal(t, result.ID, "x", "result")

stub.OnGet.AssertCalledOnce(t, "single Get")
```

### Strict mode

```go
stub := storetest.NewStoreStub(t, storetest.StoreStubStrict())
// Calling any method without configuring it fatals the test.
```

### DelegateTo

```go
inMem := companion.NewInMemoryStore()
stub := storetest.NewStoreStub(t, storetest.StoreStubDelegateTo(inMem))
// Every call forwards to inMem; the recorder captures every call.
```

See [Wiring a companion](#wiring-a-companion) below for the full pattern.

### Fault injection

```go
stub.OnGet.Faults(store.ErrNotFound, 3)              // counter
stub.OnGet.FaultNotFound()                            // directive helper
stub.OnGet.FaultsWhen(isHotKey, store.ErrTransient, 1) // predicate
stub.OnGet.FaultsWithProbability(store.ErrTransient, 0.05)
stub.OnGet.FaultsFor(5 * time.Second)                 // windowed
stub.OnGet.SetFault(testkit.And(predFault, windowFault))
```

### Virtual time

```go
clk := testkit.NewTestClock(time.Unix(0, 0))
stub := storetest.NewStoreStub(t, storetest.StoreStubWithClock(clk))

stub.OnGet.Latency(2 * time.Millisecond)
stub.OnGet.FaultsFor(5 * time.Second)
// Both driven by clk — Advance to test windows deterministically.
```

### Bench mode

```go
stub := storetest.NewStoreStub(b, storetest.StoreStubBenchMode())
// Recording is no-op; dispatch (Func/Returns/Faults) still works.
```

## Wiring a companion

A *companion* is a hand-written implementation of the source interface — typically an in-memory or fake variant — that the stub wraps via `DelegateTo`. The companion supplies real behavior; the generated stub adds recording, fault injection, call-count verification, strict mode, virtual clock, and rand-source injection on top, without the consumer writing any of that plumbing.

This is the load-bearing pattern for integration tests and for the consumer side of the planned `sim`, `chaos`, and `replay` generators.

### Step 1 — write the companion

Place the companion next to the interface (or in any package — the import path is irrelevant to the stub generator):

```go
// store/inmemory.go — hand-written
package store

import (
    "context"
    "sync"
)

type InMemoryStore struct {
    mu   sync.Mutex
    data map[string]string
}

func NewInMemoryStore() *InMemoryStore {
    return &InMemoryStore{data: make(map[string]string)}
}

func (s *InMemoryStore) Get(_ context.Context, key string) (string, error) {
    s.mu.Lock()
    defer s.mu.Unlock()
    v, ok := s.data[key]
    if !ok {
        return "", ErrNotFound
    }
    return v, nil
}

func (s *InMemoryStore) Put(_ context.Context, key string, value string) error {
    s.mu.Lock()
    defer s.mu.Unlock()
    s.data[key] = value
    return nil
}

func (s *InMemoryStore) Delete(_ context.Context, key string) error {
    s.mu.Lock()
    defer s.mu.Unlock()
    if _, ok := s.data[key]; !ok {
        return ErrNotFound
    }
    delete(s.data, key)
    return nil
}
```

The companion does not need to be in `*test` — it can live anywhere as long as it satisfies the interface. Keeping it next to the interface keeps the wire-up trivially importable.

### Step 2 — wrap with `DelegateTo`

```go
inner := store.NewInMemoryStore()
s := storetest.NewStoreStub(t, storetest.StoreStubDelegateTo(inner))
```

`StoreStubDelegateTo(impl)` is the constructor option the stub generator emits; it sets `OnGet.Func`, `OnPut.Func`, `OnDelete.Func` to forward to the companion's matching method. Every method dispatches through the inner implementation; every call is recorded on the outer stub.

### Step 3 — exercise through the stub

Writes go through to the companion, reads come back from the companion's state, and the stub records every call:

```go
err := s.Put(t.Context(), "greeting", "hello")
testkit.NoError(t, err, "Put must succeed")

got, err := s.Get(t.Context(), "greeting")
testkit.NoError(t, err, "Get must succeed")
testkit.Equal(t, got, "hello", "must return stored value")

s.OnPut.AssertCalledOnce(t, "must record Put")
s.OnGet.AssertCalledOnce(t, "must record Get")

call := s.OnGet.LastCall(t)
testkit.Equal(t, call.Key, "greeting", "must capture arg")
```

### Composition rules

The stub layers on top of the companion in a defined order. Once `DelegateTo` is wired, you can configure any of these without touching the companion:

**Fault injection takes precedence over delegation.** Once a fault is configured, the companion is bypassed for matching calls:

```go
inner := store.NewInMemoryStore()
s := storetest.NewStoreStub(t, storetest.StoreStubDelegateTo(inner))

err := s.Put(t.Context(), "key", "value")
testkit.NoError(t, err, "real Put succeeds")

s.OnGet.Faults(testkit.TestError("transient"), 1)
_, err = s.Get(t.Context(), "key")
// Fault fires — companion never runs.
```

**Per-method `Func` overrides delegation for that method.** Use this to keep the companion for some methods and stub others:

```go
s := storetest.NewStoreStub(t,
    storetest.StoreStubDelegateTo(inner),
    storetest.WithStoreGet(func(ctx context.Context, key string) (string, error) {
        return "stubbed", nil // overrides the companion just for Get
    }),
)
```

**Call-count expectations apply to delegated calls just like any other:**

```go
s.OnPut.Times(2) // verified at t.Cleanup
s.Put(t.Context(), "a", "1")
s.Put(t.Context(), "b", "2") // OK
```

**`OnRecord` hooks observe every delegated call** — useful for streaming traces into a sim engine without modifying the companion:

```go
s.OnPut.OnRecord(func(c storetest.StorePutCall) {
    trace.Append(tick, c)
})
```

**Strict mode + DelegateTo: DelegateTo wires every method**, so strict mode does not fire on any of them. Strict catches unconfigured methods; once `DelegateTo` is in play, every method is configured. To make strict mode meaningful with a companion, omit `DelegateTo` for the methods that should fail on call:

```go
// Test that asserts Delete is NEVER called.
s := storetest.NewStoreStub(t,
    storetest.StoreStubStrict(),
    storetest.WithStoreGet(inner.Get),
    storetest.WithStorePut(inner.Put),
    // OnDelete intentionally not wired — calling it fatals the test.
)
```

### Why use the companion pattern at all?

| Without companion | With companion |
|---|---|
| Tests configure `Returns` for every method on every test | Companion provides real behavior; tests configure deviations only |
| Each test re-implements key-value semantics | Companion implements them once |
| Refactoring the interface breaks every test | Refactoring breaks only the companion |
| No way to assert "the data the test wrote is what it reads back" | Real state means real round-trip works |

The companion is the substrate for any integration-grade test where the stub would otherwise need an unreasonable number of `Returns` calls. The stub layer remains useful — it adds recording, fault injection, expectations, observation hooks — without forcing the test to also write the domain logic.

## Layout Conventions

A typical interface generates its stub into a `<pkg>test/` sub-package. This prevents test-infrastructure code from bloating the production binary and clearly delineates the testing surface.

**What goes where:**

| File | Owner | Contents |
|------|-------|----------|
| `*_stub.gen.go` | Generator | The generated recording stub (DO NOT EDIT). |
| `*_stub.gen_test.go` | Generator | The self-verifying test suite for the generated stub (DO NOT EDIT). |
| `companion.go` | Developer | The hand-written fake or in-memory implementation (e.g., `InMemoryStore`). |
| `setup.go` | Developer | Test helpers that construct the `NewStoreStub(t, StoreStubDelegateTo(companion))` wiring. |

## See also

- [Primitives / MethodStub](../primitives/method-stub.md)
- [Primitives / Recording](../primitives/recording.md)
- [Primitives / Fault injection](../primitives/fault-injection.md)
- [Primitives / Clock](../primitives/clock.md)
