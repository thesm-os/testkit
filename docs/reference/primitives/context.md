# Context

## t.Context() (Go 1.24+)

Use `t.Context()` directly. testkit does not wrap it. The returned context is cancelled when the test finishes; child contexts and goroutines respect that cancellation.

```go
func TestStore_Put(t *testing.T) {
    ctx := t.Context()
    err := store.Put(ctx, item)
    testkit.NoError(t, err, "Put must succeed")
}
```

All testkit examples use `t.Context()` rather than `context.Background()`.

## Timeout

`testkit.Timeout(t, d)` wraps `t.Context()` with a deadline and **fails the test loudly** if the deadline fires before the test completes — not a quiet cancellation.

```go
ctx := testkit.Timeout(t, 10*time.Second)
result, err := subject.SlowOperation(ctx)
// On hang: "Timeout: 10s deadline exceeded" instead of silent cancel
```

For integration tests where a hung operation must be a loud failure. See [concurrency.md](concurrency.md).

## AssertCtxCancellation / AssertTimeout

To assert that a method respects context cancellation or honors a deadline, use the directive-driven helpers:

```go
testkit.AssertCtxCancellation(t, func(ctx context.Context) error {
    _, err := store.Get(ctx, "id")
    return err
})

testkit.AssertTimeout(t, 5*time.Second, func(ctx context.Context) error {
    return runner.Run(ctx)
})
```

See [directive-assertions.md](directive-assertions.md).
