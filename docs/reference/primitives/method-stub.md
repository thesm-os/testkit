# Method stub

```go
import "go.thesmos.sh/testkit/stub"
```

`MethodStub[C]` is the per-method engine a generated double embeds. It owns everything that does not depend on the signature: fault injection, latency, call recording, call-count expectations, strict mode. `C` is the recorded-call type.

Most tests reach it through a generated `On<Method>` field rather than constructing one:

```go
s := readertest.NewReaderStub(t)
s.OnGet.Faults(store.ErrUnavailable, 2).Times(4)
```

```go
func NewMethodStub[C any](tb testing.TB, name string) *MethodStub[C]
```

The name (`"Reader.Get"`) is what failure messages identify the method by.

## Answer decides which arm wins

```go
func Answer[C, R any](s *MethodStub[C], call *C, arms Arms[C, R]) R
```

Generated methods do not implement precedence themselves — they hand the arms to `Answer`, so every double resolves a call the same way and the ordering is tested once rather than restated per method.

The order is:

1. **An injected fault**, if one fires for this call.
2. **The `Func` override**, if one is set.
3. **The `Returns` fallback**, if one is set.
4. **The zero value** — or, under [strict mode](#strict-mode), a test failure.

`Answer` records the call and stamps the outcome onto the record on the way out, so the recorded call carries what the method actually returned rather than only what it was asked.

## Faults

| Method | Fires |
|---|---|
| `Faults(err, failEveryN)` | every `failEveryN`th call |
| `FaultsWhen(pred func(C) bool, err, n)` | when the call matches `pred` |
| `FaultsWithProbability(p float64, err)` | with probability `p` |
| `FaultsUntil(deadline time.Time, err)` | until `deadline` |
| `FaultsFor(d time.Duration, err)` | for `d` from now |
| `SetFault(f fault.Fault[C])` | whenever the supplied strategy says so |
| `ShouldFaultFor(call C) (bool, error)` | — asks the current strategy directly |

All of them return the stub, so they chain. See [Fault injection](fault-injection.md) for the strategies underneath and how they compose.

## Latency

```go
func (s *MethodStub[C]) Latency(d time.Duration) *MethodStub[C]
```

Sleeps `d` on every call, against the stub's clock. Under a [`TestClock`](clock.md) that costs no wall-clock time — the test advances the clock and the sleeping call resumes.

`SleepLatency()` applies the configured delay once, for a generated method that needs to control when in its body the delay lands.

## Determinism

```go
func (s *MethodStub[C]) WithClock(clk clock.Clock) *MethodStub[C]
func (s *MethodStub[C]) WithRandSource(src rand.Source) *MethodStub[C]
func (s *MethodStub[C]) Clock() clock.Clock
```

`WithClock` binds the clock latency and time-windowed faults read. `WithRandSource` binds the source probabilistic faults draw from, which is what makes a failing run replayable.

A generated double sets both across every method at once through `ReaderStubWithClock` and `ReaderStubWithRandSource`.

## Call-count expectations

```go
func (s *MethodStub[C]) Times(n int) *MethodStub[C]
func (s *MethodStub[C]) TimesAtLeast(n int) *MethodStub[C]
func (s *MethodStub[C]) Verify()
```

`Times(n)` requires exactly `n` calls; `TimesAtLeast(n)` requires at least that many. Neither fails at the moment the count is exceeded — `Verify` checks them.

A generated constructor given a non-nil `tb` registers `Verify` as a cleanup, so an unmet expectation is reported without the test remembering to ask. That is the whole reason to pass `tb`.

## Strict mode

```go
func (s *MethodStub[C]) Strict()
func (s *MethodStub[C]) IsStrict() bool
func (s *MethodStub[C]) FailUnexpectedCall(call C)
```

Under strict mode a call that reaches the zero-value arm fails the test instead. A method nobody configured was probably not meant to be called, and the alternative is a zero value flowing downstream until it causes a confusing failure somewhere else.

`FailUnexpectedCall` is what the zero-value arm calls; it reports the method name and the call.

## Reset

```go
func (s *MethodStub[C]) Reset()
```

Clears recorded calls, fault counters and call-count expectations. `Func` and `Returns` configuration is preserved, so one double carries across test phases without being rebuilt.

## Configurable

```go
type Configurable interface {
    Strict()
    Reset()
    Verify()
    BenchMode()
    SetClock(clk clock.Clock)
    SetRandSource(src rand.Source)
}
```

The signature-independent surface. A generated double holds every method's stub as a `[]Configurable`, which is what lets a construction option apply to the whole double in a loop rather than in one line per method.

## See also

- [Recording](recording.md) — the `Recorder` that backs call inspection.
- [Fault injection](fault-injection.md) — the strategies behind the `Faults*` methods.
- [Clock](clock.md) — the virtual clock latency runs on.
- [Stub](../generators/stub.md) — the generator that produces the code embedding this.
