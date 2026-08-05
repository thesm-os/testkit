# Concurrency

The `concurrency` package (`testkit/concurrency`) provides utilities for testing concurrent code.

## ConcurrentStress

Spawns N goroutines that each run M iterations of a callback. Waits for all goroutines to finish before returning. The caller owns per-goroutine state — `ConcurrentStress` only handles fan-out and fan-in.

```go
concurrency.ConcurrentStress(t, 16, 100, func(goroutine, iteration int) {
    _, _ = cache.Get(t.Context(), key)
})
```

Used by the `suite` generator's `concurrent` subtest and by the `bench` generator's `parallel` mode.

## GoroutineLeak

Records live goroutine IDs at setup and returns a teardown closure that fails the test if any new goroutine started during the test is still running at teardown.

```go
func TestSubsystem_ShutsDown(t *testing.T) {
    defer concurrency.GoroutineLeak(t)()
    // test body — every goroutine spawned here must exit by teardown
}
```

ID-based (not count-based), so it tolerates concurrent unrelated tests. Polls briefly (500ms) during teardown to allow goroutines to exit naturally.

## Timeout

Returns a `context.Context` derived from `tb.Context()` with a deadline. If the deadline fires before the test completes, the test **fails loudly** rather than silently cancelling.

```go
ctx := concurrency.Timeout(t, 10*time.Second)
result, err := subject.SlowOperation(ctx)
// On hang: "Timeout: 10s deadline exceeded" instead of silent cancellation
```

Use this for integration tests where a hung operation must be a loud failure, not a quiet skip.

## Goroutine capture utilities

Low-level helpers used internally by `GoroutineLeak` and the model package's goroutine leak detection. Available for advanced use cases.

| Function | Purpose |
|----------|---------|
| `CaptureGoroutineIDs()` | Returns a set of currently-live goroutine IDs (parsed from `runtime.Stack`) |
| `CaptureGoroutineStacks()` | Returns raw `runtime.Stack` output for all goroutines (grow-to-fit buffering, 1MB-8MB) |
| `DiffGoroutineIDs(before, after)` | Returns IDs present in `after` but not in `before` (asymmetric diff) |

```go
before := concurrency.CaptureGoroutineIDs()
// ... run code that may spawn goroutines ...
after := concurrency.CaptureGoroutineIDs()
leaked := concurrency.DiffGoroutineIDs(before, after)
if len(leaked) > 0 {
    stacks := concurrency.CaptureGoroutineStacks()
    t.Errorf("leaked %d goroutines:\n%s", len(leaked), stacks)
}
```
