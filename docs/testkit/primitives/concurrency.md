# Concurrency

## ConcurrentStress

Runs N goroutines x M iterations. The caller owns
per-goroutine state; ConcurrentStress only plumbs
fan-out/fan-in.

```go
testkit.ConcurrentStress(t, 16, 100, func(goroutine, iteration int) {
    _ = cache.Get(key)
})
```

## GoroutineLeak

Records live goroutine IDs at setup and returns a teardown
closure that fails the test if any new goroutine appeared
and is still live. ID-based (not count-based), so robust
to goroutines from concurrent tests.

```go
func TestSubsystem_ShutsDown(t *testing.T) {
    defer testkit.GoroutineLeak(t)()
    // test body — spawned goroutines must exit by teardown
}
```

Polls briefly (~100ms) so in-flight goroutines exiting as
part of natural shutdown are not flagged.
