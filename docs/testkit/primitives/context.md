# Context

## t.Context() (Go 1.24+)

Go 1.24 added `testing.TB.Context()` which returns a
context cancelled when the test finishes. Use `t.Context()`
directly — testkit does not wrap it.

```go
func TestStore_Put(t *testing.T) {
    ctx := t.Context()
    err := store.Put(ctx, item)
    testkit.NoError(t, err, "Put must succeed")
}
```

All testkit code examples use `t.Context()` rather than
`context.Background()`.

## Timeout

Wraps `t.Context()` with a deadline and **fails the test**
with a clear message if the deadline fires. For
integration tests where a hung operation should be a loud
failure, not a quiet cancellation.

```go
ctx := testkit.Timeout(t, 10*time.Second)
// If the subject hangs, the test fails with:
//   "Timeout: 10s deadline exceeded"
// instead of silently cancelling.
result, err := subject.SlowOperation(ctx)
```

Internally wraps `t.Context()` with `context.WithDeadline`
and registers a `t.Cleanup` that checks whether the
deadline cause was the timeout (vs. test completion) and
calls `t.Fatal` if so.
