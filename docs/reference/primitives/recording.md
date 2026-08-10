# Recording

```go
import "go.thesmos.sh/testkit/stub"
```

`Recorder[T]` captures calls for after-the-fact inspection. Generated doubles embed one per method through [`MethodStub`](method-stub.md); a hand-written double can use it directly.

```go
func NewRecorder[T any]() *Recorder[T]
func (r *Recorder[T]) Record(v T)
```

`T` is whatever the call is worth recording as — for a generated double, the `<Iface><Method>Call` struct.

## Reading calls back

| Method | Returns |
|---|---|
| `Calls() []T` | every recorded call, in order |
| `CallCount() int` | how many |
| `LastCall(tb) T` | the most recent, failing the test when there is none |
| `Timestamped() []RecordedCall[T]` | each call with the time it was recorded |

```go
testkit.Equal(t, s.OnGet.CallCount(), 1, "the reader must be consulted once")
testkit.Equal(t, s.OnGet.LastCall(t).Key, "absent", "the key must reach the reader")
```

`LastCall` takes `tb` because "the last call" of an empty recorder has no answer. Failing there names the real problem — nothing was called — rather than returning a zero value that fails a field comparison three lines later.

## Asserting

| Method | Fails when |
|---|---|
| `AssertCalledOnce(tb, msg) T` | the count is not exactly 1. Returns the call. |
| `AssertCalledN(tb, n, msg) []T` | the count is not `n`. Returns the calls. |
| `AssertNotCalled(tb, msg)` | anything was recorded |

`AssertCalledOnce` returning the call is the point — the assertion and the inspection are one statement:

```go
call := s.OnPut.AssertCalledOnce(t, "a flush must write exactly once")
testkit.Equal(t, call.Record.ID, "42", "the right record must be written")
```

## Querying

| Method | Returns |
|---|---|
| `Filter(pred func(T) bool) []T` | every matching call |
| `First(pred func(T) bool) (T, bool)` | the first match, and whether there was one |
| `Any(pred func(T) bool) bool` | whether any call matches |
| `All(pred func(T) bool) bool` | whether every call matches |

```go
testkit.True(t, s.OnPut.All(func(c PutCall) bool { return c.Ctx != nil }),
    "every write must carry a context")
```

`All` on an empty recorder is true, as it is for any universal quantifier. Where "and there was at least one" is part of the claim, assert the count too — otherwise a double that was never called passes.

## Waiting

| Method | Blocks until |
|---|---|
| `WaitForN(tb, n int, timeout)` | `n` calls have been recorded, or the timeout elapses |
| `WaitFor(tb, pred func(T) bool, timeout, msg)` | a recorded call matches `pred`, or the timeout elapses |

These are what a concurrent test should wait on. A background worker that will call the double is ready when the double has been called — waiting on a sleep instead is a flake with a timer attached.

```go
go svc.Run(ctx)
s.OnFetch.WaitForN(t, 3, time.Second)
```

## Gates

A gate blocks a recorded call until the test releases it, which is how a test controls interleaving rather than hoping for it.

```go
func (r *Recorder[T]) NewGate() *Gate
func (g *Gate) Release()
func (g *Gate) ReleaseOne()
```

`Release` frees every waiter; `ReleaseOne` frees one, which is what lets a test step a queue and observe the state between steps.

```go
gate := s.OnGet.NewGate()

go func() { _, _ = s.Get(ctx, "a") }()
go func() { _, _ = s.Get(ctx, "b") }()

s.OnGet.WaitForN(t, 2, time.Second)   // both are parked
gate.ReleaseOne()                      // let exactly one through
```

## Hooks

```go
func (r *Recorder[T]) OnRecord(fn func(T))
```

Runs `fn` for every recorded call, as it is recorded. Use it to drive a side channel — an [order tracker](order-tracker.md), a counter, a log the test asserts on afterwards.

## Timestamps

```go
func (r *Recorder[T]) WithClock(clk clock.Clock) *Recorder[T]
```

Binds the clock `Timestamped` reads. Under a [`TestClock`](clock.md) the recorded times are virtual, so a test can assert that two calls were 200ms apart without waiting 200ms.

## Bench mode

```go
func (r *Recorder[T]) BenchMode()
func (r *Recorder[T]) IsBenchMode() bool
```

Stops recording. The call log's allocations are what a benchmark measuring the surrounding code would otherwise be measuring — see [Benchmarking](benchmarking.md).

Every query method still works and reports an empty recorder, so a benchmark that accidentally asserts on calls gets zero rather than a panic.

## Reset

```go
func (r *Recorder[T]) Reset()
```

Clears recorded calls. Hooks, the clock and bench mode survive.

## See also

- [Method stub](method-stub.md) — the engine that records through this.
- [Order tracker](order-tracker.md) — cross-method ordering, usually driven from `OnRecord`.
- [Concurrency](concurrency.md) — for the goroutines a gate coordinates.
