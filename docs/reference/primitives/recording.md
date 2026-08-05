# Recording

`Recorder[T]` is the thread-safe call log that captures values of type `T`. It is the core observation primitive used by `MethodStub`, simulation drivers, and integration tests.

## The Architectural Pattern

Why use a generic `Recorder[T]` instead of just appending to a slice, or using a traditional mock expectation like `mock.AssertCalled("Method", arg)`?

1. **Thread Safety & Isolation:** In concurrent tests (especially under Tier 2/3 `model` linearizability checking), thousands of goroutines might hit your stub simultaneously. Slices race; traditional mocks often hold global locks that skew performance. `Recorder[T]` is highly optimized for concurrent appends.
2. **Asynchronous Causality:** Distributed systems are asynchronous. A call to `Put` might eventually trigger a call to `Notify`. `Recorder` provides `WaitForN` and `WaitFor` (backed by condition variables, not polling) to deterministically synchronize tests across goroutine boundaries without `time.Sleep()`.
3. **Simulation Hooks & Gating:** In Tier 5 simulation, you need to precisely control the ordering of events. `Recorder` allows you to install synchronous `Hooks` to broadcast events to a trace engine, and `Gates` to deliberately block execution until a specific condition is met, allowing you to synthesize and test impossible race conditions.

## Construction and basic recording

```go
rec := testkit.NewRecorder[PutCall]()
rec.Record(PutCall{Key: "a", Value: "1"})
calls := rec.Calls()         // defensive copy
testkit.Len(t, calls, 1, "expected 1 call")
```

| Method | Purpose |
|--------|---------|
| `NewRecorder[T]()` | Create empty recorder |
| `Record(v T)` | Append; fire hooks; unblock waiters; respect gates |
| `Calls() []T` | Defensive copy of all recorded values |
| `Timestamped() []RecordedCall[T]` | Defensive copy with per-call timestamps |
| `CallCount() int` | Number of recorded values |
| `LastCall() T` | Most recent value; `tb.Fatalf` if none |
| `Reset()` | Clear log; preserve hooks, gates, clock |

## Assertion helpers

Combine count check with value retrieval. Replaces the common three-line pattern.

```go
call := rec.AssertCalledOnce(t, "single put")
calls := rec.AssertCalledN(t, 3, "three puts")
rec.AssertNotCalled(t, "Put must not run yet")
```

| Method | Purpose |
|--------|---------|
| `AssertCalledOnce(t, msg) T` | Exactly 1 call; returns it |
| `AssertCalledN(t, n, msg) []T` | Exactly n calls; returns them |
| `AssertNotCalled(t, msg)` | No calls recorded |

## Filtering

```go
active := rec.Filter(func(c StorePutCall) bool { return c.Status == StatusActive })
testkit.Len(t, active, 2, "two active puts")

c, ok := rec.First(func(c StorePutCall) bool { return c.ID == "x" })
present := rec.Any(func(c StorePutCall) bool { return c.ID == "x" })
allActive := rec.All(func(c StorePutCall) bool { return c.Status == StatusActive })
```

| Method | Returns |
|--------|---------|
| `Filter(pred)` | All matching values |
| `First(pred)` | First match + ok bool |
| `Any(pred)` / `All(pred)` | Bool |

## Waiting

For asynchronous tests where calls arrive on goroutines outside the test's control. Uses condition variables — no polling, no `time.Sleep`.

```go
rec.WaitForN(t, 3, 5*time.Second)

rec.WaitFor(t, func(c StorePutCall) bool { return c.Status == StatusActive },
    5*time.Second, "active put must arrive")
```

Timeouts route through the configured `Clock` — pass a `TestClock` and use `Advance` to step over the deadline deterministically.

## Hooks

Synchronous callbacks fired on every `Record`. Used to wire observation into simulation engines.

```go
rec.OnRecord(func(c PutCall) {
    trace.Append(tick, Event{Kind: "put", Data: c})
})
```

Multiple hooks are allowed. They run under the recorder mutex — keep them fast. For expensive work, send to a channel inside the hook.

## Gates

Block `Record` calls until released. Used to create deterministic race conditions.

```go
gate := rec.NewGate()

go func() {
    _ = store.Put(ctx, req) // blocks at recording layer
}()

_, _ = store.Get(ctx, key) // happens while Put is blocked
gate.Release()              // let the Put through
```

| Method | Behavior |
|--------|----------|
| `NewGate()` | Install a gate; only one may be active at a time |
| `gate.Release()` | Unblock all waiting Record calls |
| `gate.ReleaseOne()` | Unblock exactly one waiting Record call |

Gates compose with hooks and waiters — a gated `Record` fires hooks and unblocks waiters only after the gate releases.

## Timestamping

```go
rec.WithClock(clk) // optional — defaults to RealClock

rec.Record(c1)
rec.Record(c2)

stamped := rec.Timestamped()
// stamped[0].Time = clock.Now() at the moment c1 was recorded
// stamped[1].Time = clock.Now() at the moment c2 was recorded
```

`RecordedCall[T]` wraps the value with the timestamp of when it was recorded. The clock is the one configured via `WithClock` (default: real time). When the recorder is embedded in a `MethodStub`, the stub's clock propagates here automatically.

## Bench mode

```go
rec.BenchMode()
```

Disables call recording — `Record` becomes a no-op. No allocation, no hooks, no gate checks. Dispatch (Func, Returns, Faults) still works through the enclosing `MethodStub`. The `bench` generator enables this automatically; consumers writing their own benchmarks against generated stubs do the same.

## Concurrency

All Recorder methods are thread-safe via mutex. `WaitForN` and `WaitFor` use condition variables. Gates use condition variables internally. Generated stub code inherits this safety.

## See also

- [MethodStub](method-stub.md) — embeds `*Recorder[T]`
- [Clock](clock.md) — drives `Timestamped` and `WaitForN` timeouts
