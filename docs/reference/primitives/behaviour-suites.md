# Behaviour suites

```go
import "go.thesmos.sh/testkit/stub"
```

Every generated double owes the same contract: a call is recorded, a reset clears it, a fault takes precedence over an override, an unmet `Times` is reported. That contract does not depend on any method's signature — so asserting it in every generated companion would restate the same checks once per interface, and these are the assertions most likely to be subtly wrong.

`Behaviour` and `DoubleBehaviour` assert it once, in Go. A generated companion calls them and adds only the checks that genuinely need the method's types.

## Behaviour — the per-method contract

```go
func Behaviour[C, R any](t *testing.T, name string, newSubject func(tb testing.TB) Subject[C, R])
```

`newSubject` must build a **fresh** double bound to the supplied `TB` and return one method's subject. It is called once per check: several assert on failure, which needs a double bound to a [`testkit.FailableTB`](helpers.md#failabletb) rather than to the running test, and a shared instance would carry recorded calls between checks.

```go
type Subject[C, R any] struct {
    Stub     *MethodStub[C]
    Call     func()            // invoke with zero-valued arguments
    Fails    func() error      // invoke and return the error; nil when the method cannot fail
    Result   func() R          // invoke and return the boxed return tuple; nil when it returns nothing
    Override func(mark func()) // install a Func override that calls mark
}
```

The argument values are incidental. What the checks assert is that a call was recorded, cleared, faulted or refused — not what came back.

`Override` is a closure because installing one needs the method's signature. What it proves — that an override beats the zero value — is the same for every method.

The checks:

| Subtest |
|---|
| `records the call` |
| `clears the record on reset` |
| `answers with the zero value when unconfigured` |
| `dispatches to the Func override` |
| `returns an injected fault` |
| `records the injected fault on the call` |
| `fires a counted fault on the nth call` |
| `refuses an unconfigured call in strict mode` |
| `reports an unmet call count` |
| `reports an unmet minimum call count` |

**What is deliberately absent:** any check needing a value distinguishable from a zero one. "Returns what `Returns` pinned" is that case — pinning a zero result is indistinguishable from configuring nothing, and building a distinguishable value of an arbitrary type needs the type. That one stays generated.

## DoubleBehaviour — the whole-double contract

```go
func DoubleBehaviour[C any](t *testing.T, d Double[C])
```

Every check here concerns a setting supposed to *reach* the double rather than one method of it, or a property of the recording itself.

```go
type Double[C any] struct {
    New            func(tb testing.TB) Instance[C]
    WithClock      func(tb testing.TB, clk clock.Clock) Instance[C]
    WithRandSource func(tb testing.TB, src rand.Source) Instance[C]
    BenchMode      func(tb testing.TB) Instance[C]
    Strict         func(tb testing.TB) Instance[C]
}

type Instance[C any] struct {
    Stub  *MethodStub[C]
    Call  func()
    Reset func()
}
```

Any method will do for `Instance.Stub` — what is under test is that a setting reached the double at all, and a generated companion picks the first.

The checks:

| Subtest |
|---|
| `sleeps latency against the injected clock` |
| `records a call only once its latency has elapsed` |
| `stops a time-windowed fault once its window closes` |
| `draws probabilistic faults from the injected source` |
| `stops recording in bench mode` |
| `refuses an unconfigured call when built strict` |
| `clears timestamps on reset` |

These are the ones most likely to be quietly broken: a clock that never reached a method, a latency slept *after* the call was already recorded.

## Two hazards worth knowing

**Failures report against `behaviour.go`, not against generated code.** The subtests are scoped by the caller's test function, which names the method, and the name is carried into every failure message — but that is a weaker signal than an inlined assertion. It is the price of testing this logic once instead of trusting it everywhere.

**The latency checks are bounded by a real-time budget.** They drive a virtual clock from another goroutine and would block rather than fail if the double slept on wall time. A failure there means latency is not clock-driven; it does not mean the machine is slow.

## See also

- [Method stub](method-stub.md) — the engine whose contract these assert.
- [Stub](../generators/stub.md) — the generator whose companions call them.
- [Helpers](helpers.md) — `FailableTB`, which the failure-asserting checks bind to.
