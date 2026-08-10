# Polling

```go
import "go.thesmos.sh/testkit/polling"
```

Two helpers for a condition that becomes true some time after the call that causes it. Both replace the `time.Sleep` that would otherwise be there — a sleep is either too short and flaky or too long and slow, and it is always both on someone else's machine.

## AssertEventually

```go
func AssertEventually(tb testing.TB, timeout, interval time.Duration, fn func(tb testing.TB), msg string)
```

Runs `fn` every `interval` until it makes no assertion failures, or `timeout` elapses.

```go
polling.AssertEventually(t, 2*time.Second, 50*time.Millisecond, func(tb testing.TB) {
    testkit.Equal(tb, store.Count(), 3, "all three writes must land")
}, "the writer must drain within two seconds")
```

`fn` receives a `tb` that captures failures rather than aborting, so an assertion inside it can fail on an early attempt without ending the test. Use that `tb`, not the outer `t` — an assertion against the outer one fails the test on the first attempt, which defeats the retry entirely.

The final attempt's failure message is what surfaces on timeout, so the assertion inside `fn` still says what went wrong.

## RetryUntil

```go
func RetryUntil(tb testing.TB, timeout time.Duration, pred func() bool, msg string)
```

Polls `pred` until it returns true or `timeout` elapses, failing with `msg` on timeout.

```go
polling.RetryUntil(t, time.Second, func() bool {
    return conn.State() == state.Ready
}, "the connection must reach Ready")
```

Use `RetryUntil` when the condition is a boolean and the failure message is the whole diagnosis. Use `AssertEventually` when you want the assertion's own diff — `Equal` reporting `3 != 2` beats a message saying the count was wrong.

## Choosing a timeout

Long enough that a slow CI runner does not fail it; short enough that a genuine hang is reported rather than waited out. Seconds, not minutes.

The interval matters less. A short one costs a few wasted iterations; a long one adds latency to every passing run. Tens of milliseconds is usually right.

## When not to poll

Polling is for a condition with no observable event behind it. Where there is one, wait on it instead:

- A double that will be called — [`Recorder.WaitForN`](recording.md#waiting) blocks until it has been.
- Time passing in code under a virtual clock — [`TestClock.AwaitWaiters`](clock.md#awaitwaiters-is-the-part-that-matters) then `Advance`.
- A goroutine finishing — a `sync.WaitGroup` the test owns.

Each of those is exact. Polling is the fallback for when nothing exact is available.

## See also

- [Recording](recording.md) — `WaitFor` and `WaitForN` on a double.
- [Clock](clock.md) — deterministic waiting under virtual time.
- [Concurrency](concurrency.md) — `Timeout` for bounding a test overall.
