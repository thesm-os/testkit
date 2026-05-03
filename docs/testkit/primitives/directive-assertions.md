# Directive Assertions

testkit ships free-standing assertion helpers that match the semantics of common `//testkit:` directives. The `suite` generator uses them internally; consumers can call them directly when wiring suite logic by hand or testing a property the generator doesn't yet cover.

Each assertion is a one-call wrapper: pass a function that exercises the subject; the helper asserts the expected behavioral property.

## AssertNilSafe

```go
testkit.AssertNilSafe(t, func() {
    _ = store.Put(ctx, Item{})
})
```

Asserts that `fn` does not panic. The function may return an error — that's expected. It must not crash. Used by the `nilsafe` directive to verify methods handle zero-value or nil inputs gracefully.

## AssertCtxCancellation

```go
testkit.AssertCtxCancellation(t, func(ctx context.Context) error {
    _, err := store.Get(ctx, "id")
    return err
})
```

Calls `fn` with an already-cancelled context. Asserts the returned error wraps `context.Canceled` or `context.DeadlineExceeded`. Methods that respect context cancellation must return promptly with a context error. Used by the `ctx` directive.

## AssertTimeout

```go
testkit.AssertTimeout(t, 5*time.Second, func(ctx context.Context) error {
    return runner.Run(ctx)
})
```

Calls `fn` with a context that has the given deadline. Asserts `fn` returns before the deadline fires. If `fn` returns `context.DeadlineExceeded`, the method did not honor the deadline — the test fails. Used by the `timeout` directive.

## AssertPure

```go
testkit.AssertPure(t,
    func() []Item { return store.List(ctx) },        // observe
    func() { _, _ = store.Get(ctx, "id") },          // exercise
)
```

Calls `observe` before and after `fn`, then asserts the observable state did not change. Used by the `pure` directive.

Observers must return values comparable via `cmp.Equal`. For types with unexported fields, return a deep-copied projection of the public fields. For non-deterministic state (timestamps, random IDs), exclude them from the projection.

## AssertBounded

```go
testkit.AssertBounded(t, 0, 1000, func() int {
    return counter.Count(ctx)
})
```

Calls `fn` and asserts the result is in `[min, max]` inclusive. Used by the `bounded` directive.

## See also

- [Configuration](../configuration.md) — directive vocabulary reference
- [Generators / suite](../generators/suite.md) — how `suite` consumes directives to call these helpers
