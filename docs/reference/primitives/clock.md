# Clock

`Clock` abstracts time operations for deterministic testing. testkit defines the interface; consumers inject implementations. The default is real wall-clock time (`RealClock`); tests use `TestClock` for manual advancement; simulation engines plug in their own clock that drives every method, fault, and recorder timestamp from a single virtual timebase.

## The Architectural Pattern

Why require a `Clock` interface instead of just using `time.Now()` and `time.Sleep()`?

Real-time tests in distributed systems are fundamentally flawed. A `time.Sleep(100ms)` race condition that passes consistently on a developer's fast laptop will inevitably fail on a busy, resource-constrained CI runner. Furthermore, wall-clock fault windows are impossible to reproduce—a fault configured to fire at "12:00:00" cannot be replayed deterministically if the test fails.

The `Clock` interface separates the *concept* of time ("I need a timestamp" or "I need to wait") from the *execution* of time. In production, your code uses the real wall-clock. In tests, you inject a virtual `TestClock`. Time only advances when your test explicitly calls `clk.Advance()`, completely eliminating scheduling flakes and ensuring 100% reproducible execution traces.

## Interface

```go
type Clock interface {
    Now() time.Time
    Sleep(d time.Duration)
    After(d time.Duration) <-chan time.Time
    NewTimer(d time.Duration) Timer
}

type Timer interface {
    C() <-chan time.Time
    Stop() bool
    Reset(d time.Duration) bool
}
```

`testkit.RealClock()` returns a `Clock` backed by `time` stdlib. `testkit.NewTestClock(epoch)` returns a virtual clock controlled by `Advance`.

## TestClock — deterministic virtual time

```go
clk := testkit.NewTestClock(time.Unix(0, 0))
clk.Advance(5 * time.Second)
clk.Now() // time.Unix(5, 0)
```

Time does not advance on its own. Goroutines blocked in `Sleep`, `After`, or on a `Timer` are released when virtual time crosses their deadline.

### Synchronizing with sleepers

```go
go func() { clk.Sleep(5 * time.Second) }()
clk.AwaitWaiters(1)         // spin until the goroutine is parked
clk.Advance(6 * time.Second) // wake it up
```

`AwaitWaiters(n)` spins until at least `n` goroutines are blocked on the clock. This eliminates the `time.Sleep`-based race that plagues clock-based tests.

## Wiring into stubs and recorders

A single clock should drive everything in a test — fault windows, stub latencies, recorder timestamps, wait timeouts. Pass it once at construction:

```go
clk := testkit.NewTestClock(time.Unix(0, 0))
stub := storetest.NewStoreStub(t,
    storetest.WithStoreClock(clk),
)

stub.OnGet.FaultsFor(5 * time.Second) // window driven by clk
stub.OnGet.Latency(2 * time.Millisecond) // simulated by clk.Sleep
stub.OnGet.WaitForN(t, 3, time.Second) // timeout driven by clk
```

Internally, `MethodStub.WithClock` propagates to the embedded `Recorder`, so `Recorder.Timestamped()` and `Recorder.WaitForN` use the same clock.

## Why a Clock interface

Real-time tests are flaky: a `time.Sleep(100ms)` race that passes on a fast laptop fails on a busy CI runner. Wall-clock fault windows are unreproducible — a windowed fault that fires at 12:00:00 cannot be replayed deterministically.

`Clock` separates "I need a timestamp" from "I need to wait" from "I need wall-clock semantics." Production code calls a real clock; tests substitute a virtual one and step through scenarios that would otherwise require real time.

## Standard injection points

| Surface | How to inject |
|---------|---------------|
| MethodStub fault windows + latency | `MethodStub.WithClock(clk)` or generated `WithIfaceClock(clk)` constructor option |
| Recorder timestamps + waits | Inherits from `MethodStub.WithClock`; standalone via `Recorder.WithClock(clk)` |
| Polling helpers | `RetryUntil` / `AssertEventually` use real time by design — for virtual-time waits, use `Recorder.WaitForN` or `TestClock.AwaitWaiters` |

## See also

- [Fault injection](fault-injection.md) — windowed faults are clock-driven
- [Recording](recording.md) — `Timestamped` reads from the configured clock
- [MethodStub](method-stub.md) — `WithClock`, `Latency`
