# Recording

## Recorder[T]

Thread-safe call log with filtering, waiting, hooks, and
gating. The core observation primitive for stubs,
recording wrappers, simulation, and integration tests.

```go
rec := testkit.NewRecorder[MyRequest]()
rec.Record(req)
calls := rec.Calls()  // defensive copy
testkit.Len(t, calls, 3, "expected 3 calls")
```

## Core methods

| Method | Description |
|--------|-------------|
| `NewRecorder[T]()` | Create empty recorder |
| `Record(v T)` | Append value, fire hooks, unblock waiters |
| `Calls() []T` | Defensive copy of all recorded values |
| `CallCount() int` | Number of recorded values |
| `LastCall() T` | Most recent value |
| `Reset()` | Clear the log |

## Assertion helpers

Convenience methods that combine count checking with
value retrieval. Reduce the most common three-line
pattern to one line.

| Method | Description |
|--------|-------------|
| `AssertCalledOnce(t, msg) T` | Fails unless exactly 1 call; returns it |
| `AssertCalledN(t, n, msg) []T` | Fails unless exactly n calls; returns them |
| `AssertNotCalled(t, msg)` | Fails if any calls were recorded |

```go
// Before:
testkit.Equal(t, rec.Puts.CallCount(), 1, "one put")
call := rec.Puts.LastCall()

// After:
call := rec.Puts.AssertCalledOnce(t, "one put")
```

## Filtering

Query recorded calls by predicate. Returns a new slice
(does not mutate the recorder).

```go
active := rec.Puts.Filter(func(c StorePutCall) bool {
    return c.Req.Status == StatusActive
})
testkit.Len(t, active, 2, "two puts with active status")
```

| Method | Description |
|--------|-------------|
| `Filter(pred func(T) bool) []T` | Returns calls matching predicate |
| `First(pred func(T) bool) (T, bool)` | First matching call |
| `Any(pred func(T) bool) bool` | True if any call matches |
| `All(pred func(T) bool) bool` | True if all calls match |

## Waiting

For asynchronous tests where calls arrive on goroutines
outside the test's control (integration tests, sim
engines). Blocks until the condition is met or the
timeout fires.

```go
// Wait until 3 Put calls have arrived:
rec.Puts.WaitForN(t, 3, 5*time.Second)

// Wait until a specific call arrives:
rec.Puts.WaitFor(t, func(c StorePutCall) bool {
    return c.Req.Status == StatusActive
}, 5*time.Second, "active put must arrive")
```

| Method | Description |
|--------|-------------|
| `WaitForN(t, n, timeout)` | Block until n calls recorded |
| `WaitFor(t, pred, timeout, msg)` | Block until predicate matches |

Internally uses a condition variable signalled by
`Record`. No polling, no sleep.

## Hooks

Callbacks fired synchronously on every `Record` call.
For wiring observation into sim engines without polling.

```go
rec.Puts.OnRecord(func(c StorePutCall) {
    trace.Append(tick, Event{Kind: "put", Data: c.Req})
})
```

| Method | Description |
|--------|-------------|
| `OnRecord(fn func(T))` | Register a hook (multiple allowed) |

Hooks run under the recorder's mutex — keep them fast.
For expensive work, send to a channel inside the hook.

## Gating

Blocks `Record` calls until the gate is released. For
testing race conditions and ordering-sensitive scenarios
where you need to control exactly when a method completes.

```go
gate := rec.Puts.NewGate()

// This goroutine blocks inside Record until gate.Release():
go func() {
    _ = store.Put(ctx, req)  // blocks at recording layer
}()

// Do something that should happen while Put is blocked:
_, _ = store.Get(ctx, key)

// Now let the Put through:
gate.Release()
```

| Method | Description |
|--------|-------------|
| `NewGate() *Gate` | Install a gate that blocks Record |
| `gate.Release()` | Unblock all waiting Record calls |
| `gate.ReleaseOne()` | Unblock exactly one waiting Record call |

Gates compose with hooks and waiting — a gated `Record`
fires hooks and unblocks waiters only after the gate
releases.

## Concurrency

All Recorder methods are thread-safe via mutex.
`WaitForN` and `WaitFor` use condition variables (not
polling). Gates use condition variables internally.
Generated recording wrappers inherit this safety.
