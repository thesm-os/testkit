# Fault injection

```go
import "go.thesmos.sh/testkit/fault"
```

A `Fault` decides, per call, whether to return an error. Strategies compose, which is how a test says "fail the third write, but only while the circuit is open" without writing a bespoke double.

```go
type Fault[C any] interface {
    ShouldFire(call C, clk clock.Clock) (bool, error)
}
```

`C` is the recorded-call type — for a generated double, `<Iface><Method>Call`. A strategy that inspects the call sees exactly what the caller passed.

## Strategies

| Constructor | Fires |
|---|---|
| `NewCountedFault[C](err, n)` | every `n`th call |
| `NewRetryFault[C](err, n)` | on the first `n` calls, then stops — the mirror of counted |
| `NewPredicateFault[C](err, pred func(C) bool)` | when the call matches `pred` |
| `NewProbabilityFault[C](err, p float64, src rand.Source)` | with probability `p`, drawn from `src` |
| `NewWindowedFault[C](err, deadline time.Time)` | until `deadline`, read from the clock passed to `ShouldFire` |

Each returns a concrete type (`*CountedFault[C]`, `*RetryFault[C]`, …) implementing `Fault[C]`.

`NewRetryFault` is the one worth naming: it fails the first `n` calls and then succeeds, which is the shape a retry test needs. `NewCountedFault(err, 3)` fails call 3, 6, 9; `NewRetryFault(err, 3)` fails calls 1, 2, 3 and nothing after.

## Composition

```go
func And[C any](faults ...Fault[C]) *AndFault[C]
func Or[C any](faults ...Fault[C]) *OrFault[C]
```

`And` fires when every member fires; `Or` when any does. Both return the first non-nil error from a firing member.

```go
// Fail writes for the first 200ms, but only for the tenant under test.
f := fault.And(
    fault.NewWindowedFault[PutCall](ErrUnavailable, clk.Now().Add(200*time.Millisecond)),
    fault.NewPredicateFault[PutCall](ErrUnavailable, func(c PutCall) bool {
        return c.Record.Tenant == "acme"
    }),
)
s.OnPut.SetFault(f)
```

## Determinism

Two rules, and both exist so a failing run can be replayed.

**`NewProbabilityFault` draws from a caller-supplied source**, not the global generator. A seeded [`rand.Source`](rand.md) reproduces the same firing pattern every run. A failure nobody can replay is not a finding.

**Time-based strategies read the clock passed to `ShouldFire`**, not the wall clock. Under a [`TestClock`](clock.md) a windowed fault's deadline arrives when the test advances time, not when the machine happens to be slow.

## Reset

`CountedFault`, `RetryFault`, `AndFault` and `OrFault` carry a call counter and expose `Reset()`. `PredicateFault`, `ProbabilityFault` and `WindowedFault` are stateless — `Reset` on a composite propagates to members that have one and is a no-op for the rest.

A double's `ResetCalls` resets its faults along with its recorded calls.

## Concurrency

Strategies guard their counters internally, so one instance may back a double shared across goroutines. The count is global to the strategy, not per goroutine: `NewCountedFault(err, 3)` under eight concurrent callers fails the third call overall, and which goroutine receives it is not defined.

## Reading the clock safely

```go
func ClockNow(clk clock.Clock) time.Time
```

Returns `clk.Now()`, or `time.Now()` when `clk` is nil. A custom strategy reading time should go through it — `ShouldFire` receives whatever clock the stub was given, and that is nil for a stub nobody configured one on.

## Through a generated double

Most tests never construct a strategy. The [method stub](method-stub.md) exposes the common cases directly:

| Method | Equivalent |
|---|---|
| `Faults(err, n)` | `NewCountedFault(err, n)` |
| `FaultsWhen(pred, err, n)` | a predicate strategy |
| `FaultsWithProbability(p, err)` | `NewProbabilityFault(err, p, src)` |
| `FaultsUntil(deadline, err)` | `NewWindowedFault(err, deadline)` |
| `FaultsFor(d, err)` | a window of `d` from the clock's now |
| `SetFault(f)` | anything you built by hand |

A method that declares `//testkit:fault ErrNotFound` also gains a one-shot `FaultNotFound()` helper — see [Stub](../generators/stub.md#fault-helpers).

## See also

- [Method stub](method-stub.md) — where faults are configured in practice.
- [Rand](rand.md) — the source `NewProbabilityFault` draws from.
- [Clock](clock.md) — the clock `NewWindowedFault` reads.
