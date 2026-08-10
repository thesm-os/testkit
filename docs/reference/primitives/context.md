# Context assertions

```go
import "go.thesmos.sh/testkit"
```

Four helpers for the context behaviour every `func(ctx, …) error` owes and almost none is tested for. Each takes the method under test as a closure and drives one context condition into it.

## The functions

| Function | Passes a | Asserts |
|---|---|---|
| `AssertCtxCancellation(tb, fn)` | context cancelled before the call | `fn` returns a context error |
| `AssertCtxDeadline(tb, fn)` | context whose deadline has already passed | `fn` returns a deadline error |
| `AssertNilCtx(tb, fn)` | `nil` context | `fn` does not panic |
| `AssertTimeout(tb, deadline, fn)` | context cancelled after `deadline` | `fn` returns before the deadline elapses |

All four take `fn func(ctx context.Context) error`.

```go
testkit.AssertCtxCancellation(t, func(ctx context.Context) error {
    return svc.Fetch(ctx, "id")
})
```

## What each one catches

**`AssertCtxCancellation`** catches the method that accepts a context and never reads it. The common shape is a function that passes `ctx` down to one call and does its own work before that — cancellation is honoured eventually, not promptly, and a caller that cancelled expects promptly.

**`AssertCtxDeadline`** catches the method that checks `ctx.Done()` but not `ctx.Err()`, or that treats an already-expired deadline as "plenty of time" because the select raced the wrong way.

**`AssertNilCtx`** catches the `ctx.Value(...)` on a nil context. A nil context is a caller bug, but a panic in a library turns a caller's bug into an incident in a goroutine nobody can recover. Returning an error is the contract; this asserts the method has one.

**`AssertTimeout`** is the positive form: given a deadline, the method must come back inside it. Use it for a method that does its own bounded wait — a retry loop, a poll, a drain.

```go
testkit.AssertTimeout(t, 500*time.Millisecond, func(ctx context.Context) error {
    return pool.Drain(ctx)
})
```

## Where these come from

The four are what a generated conformance suite asserts on every method carrying a `context.Context`, independent of the method's shape. They are exported so a hand-written test can make the same assertions on code no generator has reached yet.

## See also

- [Directive assertions](directive-assertions.md) — the other contract-shaped helpers.
- [Concurrency](concurrency.md) — `Timeout` for bounding a test that might hang, which is a different job.
- [Shape classification](../generators/shapes.md) — which methods a suite would apply these to.
