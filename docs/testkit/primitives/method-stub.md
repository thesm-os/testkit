# MethodStub

`MethodStub[C]` is the generic per-method test double that the `stub` generator embeds in every method's stub. It composes recording, fault injection, strict mode, call-count expectations, latency simulation, clock injection, and rand-source injection into a single primitive.

`MethodStub` is the runtime substrate that conformance tiers (`suite`, `model`, `bench`) and pre-prod tiers (`sim`, `chaos`, `replay`) build on top of. Most consumers interact with it through generated stub code rather than directly.

## The Architectural Pattern

Why does the `stub` generator embed `*testkit.MethodStub[Call]` instead of just generating a raw struct for every method? 

The answer is **behavioral uniformity**. Whether your method is a `Reader`, `Writer`, `StreamConsumer`, or `Lifecycle`, the mechanics of recording a call, injecting a time-windowed fault, asserting a call count, or simulating latency are identical. By embedding `MethodStub[C]`, the generator ensures that every stub in your project inherits the exact same, heavily tested observation and fault-injection runtime. The generator only has to emit the type-safe `Call` struct (the `C` type parameter) and the dispatch logic to route arguments into it.

## Construction

```go
stub := testkit.NewMethodStub[StoreGetCall](t, "Store.Get")
```

- `t` is `testing.TB` for auto-verification via `t.Cleanup`. Pass `nil` for a stub without test integration (rare — the generated constructor always passes `t`).
- The name string appears in error messages: `"Store.Get: unexpected call (strict mode)"`.

The generated stub aggregates one `MethodStub[C]` per interface method:

```go
type StoreStub struct {
    OnGet    *getStub      // embeds *testkit.MethodStub[StoreGetCall]
    OnPut    *putStub      // embeds *testkit.MethodStub[StorePutCall]
    OnDelete *deleteStub
}
```

## Recording

`MethodStub[C]` embeds `*Recorder[C]`. All recorder methods are available directly on the stub:

```go
stub.OnGet.CallCount()
stub.OnGet.Filter(func(c StoreGetCall) bool { return c.ID == "x" })
stub.OnGet.AssertCalledOnce(t, "must read once")
stub.OnGet.WaitForN(t, 3, time.Second)
gate := stub.OnGet.NewGate()
```

See [Recording](recording.md) for the full Recorder API.

## Fault injection

Five strategies + composition. Convenience methods install common patterns; `SetFault` accepts any `Fault`.

```go
stub.OnGet.Faults(store.ErrNotFound, 3)                          // every 3rd call
stub.OnGet.FaultsWhen(isHotKey, store.ErrNotFound, 1)            // predicate
stub.OnGet.FaultsWithProbability(store.ErrTransient, 0.05)       // probabilistic
stub.OnGet.FaultsFor(5 * time.Second)                            // time-windowed
stub.OnGet.FaultsUntil(deadline)                                 // until deadline
stub.OnGet.SetFault(testkit.And(predFault, windowFault))         // composition
```

See [Fault injection](fault-injection.md) for strategy details.

## Strict mode

```go
stub.OnGet.Strict()
```

In strict mode, `MethodStub.FailUnexpectedCall(call)` fatals the test. Generated dispatch calls `FailUnexpectedCall` when no behavior is configured (no `Func`, no `Returns`, no fault). Use strict to assert "this method must not be called" or "this method must always have explicit behavior."

The constructor option `<Stub>Strict()` enables strict on every method at once.

## Call-count expectations

```go
stub.OnGet.Times(3)        // exactly 3 calls
stub.OnPut.TimesAtLeast(1) // at least 1 call
```

`Verify()` is registered with `t.Cleanup` automatically — expectations are checked at test end. To check eagerly, call `stub.OnGet.Verify()` directly.

## Clock and rand-source injection

```go
clk := testkit.NewTestClock(time.Unix(0, 0))
rng := testkit.FixedRandSource(0.0)

stub.OnGet.WithClock(clk).WithRandSource(rng)
```

The clock drives windowed faults, latency simulation, recorder timestamps, and wait timeouts. The rand source drives probabilistic faults.

In a generated stub, the constructor option `<Stub>Clock(clk)` propagates the clock to every method's `MethodStub` and to the embedded `Recorder`.

See [Clock](clock.md) and [RandSource](rand.md).

## Latency simulation

```go
stub.OnGet.Latency(5 * time.Millisecond)
```

`Latency(d)` configures a clock-driven sleep before every dispatch path, including fault and unconfigured paths. Composes with `Faults` — `Latency(5*time.Second).Faults(err, 1)` models a slow-then-failing backend.

`Latency(0)` disables (default). Generated dispatch calls `SleepLatency` at the start of every method.

## Reset semantics

```go
stub.OnGet.Reset()
```

Reset clears recorded calls, resets fault counters, clears `Times`/`TimesAtLeast` expectations. It does **not** clear `Func`, `Returns`, or the installed fault — behavior is preserved, only observations are rewound. To remove behavior entirely, call `stub.OnGet.SetFault(nil)` or set `Func`/`Returns` to nil via the generated APIs.

## API summary

| Method | Purpose |
|--------|---------|
| `Strict()` / `IsStrict()` | Enable/check strict mode |
| `Faults(err, n)` | Counter-based fault every Nth call |
| `FaultsWhen(pred, err, every)` | Predicate fault |
| `FaultsWithProbability(err, p)` | Probabilistic fault |
| `FaultsFor(d)` / `FaultsUntil(t)` | Time-windowed fault |
| `SetFault(f Fault)` | Install arbitrary strategy |
| `ShouldFaultFor(call) (bool, error)` | Generated dispatch entry point |
| `WithClock(c)` / `Clock()` | Clock injection / accessor |
| `WithRandSource(r)` | Rand source injection |
| `Latency(d)` / `SleepLatency()` | Latency simulation |
| `Times(n)` / `TimesAtLeast(n)` / `Verify()` | Call-count expectations |
| `FailUnexpectedCall(call)` | Strict-mode fatal |
| `Reset()` | Rewind observations, preserve behavior |
| `Name()` / `TB()` | Identity accessors |

Plus all `Recorder[T]` methods via embedding.

## See also

- [Recording](recording.md) — embedded recorder API
- [Fault injection](fault-injection.md) — strategy details
- [Clock](clock.md) — virtual time
- [RandSource](rand.md) — pluggable RNG
