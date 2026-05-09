# Fault Injection

Fault injection in testkit is a strategy pattern. The `Fault` interface decides whether a fault should fire for a given call; testkit ships five strategies and two composers.

## The Architectural Pattern

Traditional test doubles (mocks) force you to hardcode failure logic directly into your test setup (e.g., `mock.On("Get").Return(ErrNotFound)`). This is brittle and difficult to scale when testing complex, intermittent failure modes like "fail 5% of the time" or "fail exactly on the 3rd retry."

The `testkit` fault injection system completely separates **Domain Behavior** from **Adversarial Failure**. You provide a real, working implementation (like an `InMemoryStore` companion), and then you dynamically attach `Fault` strategies onto the generated stub wrapper. When the strategy fires, it bypasses the companion and returns the error. When it doesn't fire, the real logic executes. This decoupling is the engine that powers the `sim` and `chaos` tier generators, allowing them to ablate faults independently of the domain logic.

## Fault interface

```go
type Fault interface {
    ShouldFire(call any, clock Clock) (bool, error)
    Reset()
}
```

`ShouldFire` is invoked by `MethodStub` on every dispatch. The call parameter is the typed call struct (params only, results not yet populated). The clock parameter is the stub's configured clock — strategies that don't need them ignore them.

A fault that fires returns `(true, err)`; the generated dispatch path then returns `err` to the caller and skips the rest of the method body.

## Strategies

### Counted — every Nth call

```go
fi := testkit.NewCountedFault(errBoom, 3) // fires on 3rd, 6th, 9th, ...
```

Counter-based, context-free. The call and clock parameters are ignored. This is the strategy behind `MethodStub.Faults(err, n)`.

### Retry — fail first N-1, succeed Nth

```go
fi := testkit.NewRetryFault(errTransient, 3) // fires on calls 1, 2; lets call 3 through
```

Models the canonical retry pattern: transient failure for the first N-1 calls, success on the Nth. Pairs with the `retry-succeeds-on-attempt N` directive.

### Probability — fire with probability p

```go
fi := testkit.NewProbabilityFault(errBoom, 0.05, rng) // 5% per call
```

Each call faults with probability `p ∈ [0, 1]`. The `rng` parameter is a `RandSource` — pass `testkit.FixedRandSource(0.0)` for "always fire" or `FixedRandSource(1.0)` for "never fire" in deterministic tests, or a seeded source from your simulation engine.

### Windowed — fire within a time window

```go
fi := testkit.NewWindowedFault(errBoom, start, end) // fires when start <= clock.Now() < end
```

Time-bounded fault — fires only while the clock is inside `[start, end)`. Driven by the stub's configured `Clock`, so `TestClock.Advance` can step in and out of the window deterministically.

`MethodStub.FaultsFor(d)` and `MethodStub.FaultsUntil(deadline)` are convenience wrappers.

### Predicate — fire when call matches

```go
fi := testkit.NewPredicateFault(errNotFound,
    func(c any) bool {
        get, ok := c.(StoreGetCall)
        return ok && get.ID == "missing"
    },
    /*every*/ 1)
```

Fires when the predicate returns true for the typed call value AND the internal counter fires. Pass `every=1` to fire on every matching call; pass `every=N` to fire on every Nth match. This is the strategy behind `MethodStub.FaultsWhen`.

## Composition

### And — all must fire

```go
testkit.And(predFault, windowFault) // fail only when call matches AND time is in window
```

Inner faults are evaluated left-to-right with short-circuit. **Side effects matter here:** if `predFault` doesn't fire, `windowFault.ShouldFire` is never called. So `And(predFault, NewCountedFault(err, 3))` fires on the **3rd matching call**, not the 3rd call overall — the counter advances only on calls that pass the predicate.

### Or — any may fire

```go
testkit.Or(probFault, windowFault) // fail with probability OR within window
```

Fires when any inner strategy fires. The error from the first strategy that fires is returned.

## Wiring into stubs

`MethodStub` exposes both convenience helpers and the raw strategy interface.

```go
stub := storetest.NewStoreStub(t)

// Convenience: counter-based fault on every Nth call.
stub.OnGet.Faults(store.ErrNotFound, 3)

// Convenience: predicate fault.
stub.OnGet.FaultsWhen(func(c StoreGetCall) bool { return c.ID == "missing" },
    store.ErrNotFound, 1)

// Convenience: probabilistic fault.
stub.OnGet.FaultsWithProbability(store.ErrTransient, 0.05)

// Convenience: time-windowed fault.
stub.OnGet.FaultsFor(5 * time.Second)
stub.OnGet.FaultsUntil(deadline)

// Raw strategy: any Fault, including composed.
stub.OnGet.SetFault(testkit.And(
    testkit.NewPredicateFault(errBoom, isHotKey, 1),
    testkit.NewWindowedFault(errBoom, start, end),
))
```

Convenience methods install a single underlying strategy; `SetFault` replaces it with whatever you pass. The most recently installed strategy wins.

## Reset semantics

`MethodStub.Reset()` calls `Reset()` on the installed fault, which rewinds counters but **does not** clear the strategy itself. The stub's behavior is preserved across Reset; only observation state (call counts, fault counters) rewinds.

To remove a fault entirely, call `stub.OnGet.SetFault(nil)`.

## When to use which

| Pattern | Strategy |
|---------|----------|
| Fail every Nth call | `Faults(err, n)` / `NewCountedFault` |
| Recover after N-1 failures | `NewRetryFault(err, n)` |
| Fail X% of calls | `FaultsWithProbability(err, p)` / `NewProbabilityFault` |
| Fail during a time window | `FaultsFor(d)` / `FaultsUntil(t)` / `NewWindowedFault` |
| Fail when arguments match | `FaultsWhen(pred, err, every)` / `NewPredicateFault` |
| Compound condition | `And(...)` / `Or(...)` |

## Concurrency

All strategies are safe for concurrent `ShouldFire` calls. Counter-based strategies use atomic operations or mutex; probability uses the supplied `RandSource` (which must be thread-safe).

## See also

- [MethodStub](method-stub.md) — how stubs invoke faults
- [Clock](clock.md) — drives windowed faults
- [RandSource](rand.md) — drives probabilistic faults
