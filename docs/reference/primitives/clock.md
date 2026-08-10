# Clock

```go
import "go.thesmos.sh/testkit/clock"
```

A test asserting on a five-second timeout should not take five seconds. `clock.Clock` is the seam: production code and test doubles read time through it, and a test swaps in a clock it advances by hand.

## The interface

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

`clock.RealClock()` delegates to the standard library. `clock.NewTestClock(origin time.Time)` returns a `*TestClock` frozen at `origin`.

## TestClock

| Method | Effect |
|---|---|
| `Now() time.Time` | The current virtual time. Does not advance on its own. |
| `Advance(d time.Duration)` | Moves virtual time forward and fires every waiter whose deadline has passed |
| `Sleep(d time.Duration)` | Blocks until virtual time reaches `Now() + d` |
| `After(d time.Duration) <-chan time.Time` | A channel that receives when virtual time reaches the deadline |
| `NewTimer(d time.Duration) Timer` | A timer on virtual time |
| `AwaitWaiters(n int)` | Blocks until exactly `n` goroutines are waiting on this clock |

Virtual time moves only when `Advance` is called. Nothing elapses in the background, so a test is deterministic by construction rather than by being fast enough.

## AwaitWaiters is the part that matters

The obvious test is a race:

```go
clk := clock.NewTestClock(time.Unix(0, 0))
go worker(clk)      // will call clk.Sleep(time.Second) — eventually
clk.Advance(time.Second)   // may fire before the worker is waiting
```

If `Advance` runs first, the worker sleeps for a second of virtual time that has already passed and blocks forever. The test hangs, or passes for the wrong reason.

`AwaitWaiters` closes it by waiting for the goroutine to arrive:

```go
clk := clock.NewTestClock(time.Unix(0, 0))
go worker(clk)

clk.AwaitWaiters(1)        // the worker is now parked on the clock
clk.Advance(time.Second)   // and this releases it
```

Advance only after the waiters you expect are registered. That is the whole synchronisation discipline, and skipping it is the one way to make a `TestClock` flaky.

## Wiring it into a double

Generated doubles take a clock through a construction option, which applies it to every method at once:

```go
clk := clock.NewTestClock(time.Unix(0, 0))
s := readertest.NewReaderStub(t, readertest.ReaderStubWithClock(clk))

s.OnGet.Latency(2 * time.Second)

go func() { _, _ = s.Get(t.Context(), "k") }()
clk.AwaitWaiters(1)
clk.Advance(2 * time.Second)
```

Latency and time-windowed faults both read the clock the double was given, so neither costs wall-clock time.

## See also

- [Fault injection](fault-injection.md) — `WindowedFault` reads a clock for its deadline.
- [Method stub](method-stub.md) — `Latency` and `WithClock` on the per-method engine.
- [Recording](recording.md) — `Recorder.WithClock` timestamps recorded calls on virtual time.
