# Concurrency

```go
import "go.thesmos.sh/testkit/concurrency"
```

Three things a concurrent test needs and the standard library does not supply: a way to run work from many goroutines and collect the failures, a way to notice a goroutine that outlived the test, and a context that expires.

## ConcurrentStress

```go
func ConcurrentStress(tb testing.TB, goroutines, iterations int, work func(g, i int))
```

Runs `work` from `goroutines` goroutines, `iterations` times each, and waits for all of them.

```go
c := cache.New()
concurrency.ConcurrentStress(t, 8, 1000, func(g, i int) {
    c.Put(fmt.Sprintf("k%d", g), i)
    _ = c.Get(fmt.Sprintf("k%d", g))
})
```

Both indices are passed so `work` can partition by goroutine — writing to `k<g>` keeps each goroutine on its own key, which is what separates a test of concurrent access from a test of last-write-wins.

Run it under `-race`. Without the race detector this exercises the code without checking the thing it exists to check.

## GoroutineLeak

```go
func GoroutineLeak(tb testing.TB) func()
```

Captures the live goroutine set and returns a function that compares against it. Defer the returned function:

```go
func TestWatcherStopsCleanly(t *testing.T) {
    defer concurrency.GoroutineLeak(t)()

    w := watcher.New()
    w.Start()
    w.Stop()
}
```

A goroutine present at the end and absent at the start fails the test and reports its ID. `Stop` returning before its goroutine has actually exited is the usual cause, and it is invisible without this.

The comparison is by goroutine ID, so a goroutine that finished and had its ID reused does not read as a leak.

## Timeout

```go
func Timeout(tb testing.TB, d time.Duration) context.Context
```

A context cancelled after `d`, with cleanup registered on `tb`.

```go
ctx := concurrency.Timeout(t, 2*time.Second)
err := svc.Drain(ctx)
```

This is real wall-clock time, not virtual — it bounds a test that might otherwise hang, rather than testing timeout behaviour. For that, use [`testkit.AssertTimeout`](context.md#asserttimeout) or a [`TestClock`](clock.md).

## Inspecting the goroutine set directly

The pieces `GoroutineLeak` is built from are exported, for a test that needs to assert something more specific than "nothing leaked":

| Function | Returns |
|---|---|
| `CaptureGoroutineIDs() map[uint64]struct{}` | The live goroutine IDs |
| `CaptureGoroutineStacks() []byte` | The raw stack dump |
| `ParseGoroutineIDs(stack []byte) map[uint64]struct{}` | The IDs in a stack dump |
| `DiffGoroutineIDs(before, after map[uint64]struct{}) []uint64` | IDs in `after` and not in `before` |

```go
before := concurrency.CaptureGoroutineIDs()
pool.Start(4)
after := concurrency.CaptureGoroutineIDs()

testkit.Len(t, concurrency.DiffGoroutineIDs(before, after), 4, "the pool must start exactly four workers")
```

## See also

- [Polling](polling.md) — for waiting on a condition rather than a goroutine count.
- [Recording](recording.md) — `Recorder.WaitForN` blocks until a double has been called N times, which is usually what a concurrent test is really waiting for.
