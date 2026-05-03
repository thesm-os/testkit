# Polling

## RetryUntil

Polls a predicate with exponential backoff (starting at 1ms, doubling) until it returns true or the timeout expires.

```go
testkit.RetryUntil(t, 5*time.Second, func() bool {
    return cache.Len() > 0
}, "cache must be populated")
```

For eventually-consistent assertions in integration tests where no virtual clock is available.

## AssertEventually

Stronger than `RetryUntil`. Instead of polling a `bool`, takes a `func(testing.TB)` — the assertion itself. On timeout, it reports the **last assertion failure message**, not just "condition not met".

```go
testkit.AssertEventually(t, 5*time.Second, 100*time.Millisecond,
    func(tb testing.TB) {
        entries, err := store.List(t.Context())
        testkit.NoError(tb, err, "List")
        testkit.Len(tb, entries, 3, "replication must converge to 3 entries")
    },
    "store must converge",
)
```

On timeout, the failure message reads:

```
store must converge (after 5s, 50 attempts)
  last failure: replication must converge to 3 entries
    got len 1, want len 3
```

Uses `FailableTB` internally so individual attempts can fail without aborting the polling loop.

## Virtual time alternative

For tests with a virtual clock (`TestClock`), prefer `Recorder.WaitForN` / `Recorder.WaitFor` and `TestClock.AwaitWaiters` — they synchronize on goroutine scheduling rather than wall-clock polling and are fully deterministic.
