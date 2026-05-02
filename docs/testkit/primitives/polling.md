# Polling

## RetryUntil

Polls a condition with exponential backoff until true or
timeout. For eventually-consistent assertions in
integration tests.

```go
testkit.RetryUntil(t, 5*time.Second, func() bool {
    return cache.Len() > 0
}, "cache must be populated")
```

## AssertEventually

Stronger than `RetryUntil`. Instead of polling a `bool`,
takes a `func(t testing.TB)` — the assertion itself. On
timeout, reports the *last assertion failure message*, not
just "condition not met".

```go
testkit.AssertEventually(t, 5*time.Second, 100*time.Millisecond,
    func(t testing.TB) {
        entries, err := store.List(ctx)
        testkit.NoError(t, err, "List")
        testkit.Len(t, entries, 3, "replication must converge to 3 entries")
    },
    "store must converge",
)
```

On timeout, the failure reads:

```
store must converge (after 5s, 50 attempts)
  last failure: replication must converge to 3 entries
    got len 1, want len 3
```

Uses `FailableTB` internally to capture assertion failures
without aborting between attempts.
