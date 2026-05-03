# Concurrency

## ConcurrentStress

Spawns N goroutines that each run M iterations of a callback. Waits for all goroutines to finish before returning. The caller owns per-goroutine state — `ConcurrentStress` only handles fan-out and fan-in.

```go
testkit.ConcurrentStress(t, 16, 100, func(goroutine, iteration int) {
    _, _ = cache.Get(t.Context(), key)
})
```

Used by the `suite` generator's `concurrent` subtest and by the `bench` generator's `parallel` mode.

## GoroutineLeak

Records live goroutine IDs at setup and returns a teardown closure that fails the test if any new goroutine started during the test is still running at teardown.

```go
func TestSubsystem_ShutsDown(t *testing.T) {
    defer testkit.GoroutineLeak(t)()
    // test body — every goroutine spawned here must exit by teardown
}
```

ID-based (not count-based), so it tolerates concurrent unrelated tests. Polls briefly during teardown to allow goroutines to exit naturally.

## Timeout

Returns a `context.Context` derived from `tb.Context()` with a deadline. If the deadline fires before the test completes, the test **fails loudly** rather than silently cancelling.

```go
ctx := testkit.Timeout(t, 10*time.Second)
result, err := subject.SlowOperation(ctx)
// On hang: "Timeout: 10s deadline exceeded" instead of silent cancellation
```

Use this for integration tests where a hung operation must be a loud failure, not a quiet skip.
