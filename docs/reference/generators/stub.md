# Stub

A hand-written test double answers calls. It rarely records them — and most of what a conformance test needs to assert is the interaction, not the return value. Whether `Get` was called twice, whether `Close` came after `Open`, what key the caller actually passed: none of that survives a double that only returns.

The `stub` generator writes a recording double for every annotated interface, plus a companion test proving the double satisfies the interface it stands in for. The double is the substrate every other conformance tier composes against — a generated suite drives it, a benchmark measures through it, a model runs it against a reference implementation.

## The directive

```go
//testkit:stub
type Store interface { ... }
```

The directive takes no positional argument and denies the negated form. A double exists exactly where one is declared, so deleting the line is the suppression.

| Key | Value | Effect |
|---|---|---|
| `witness` | comma-separated type names | Concrete types a generic double's companion test is instantiated at. See [Generic interfaces](#generic-interfaces). |

## Where the output goes

Two files flow from one annotated interface, and the tag is what makes them independently routable.

| Tag | Suffix | Contents |
|---|---|---|
| *(primary)* | `_stub.gen.go` | The double. Declares the source package, so other packages' tests can import it. |
| `test` | `_stub.gen_test.go` | The companion checks. The `_test.go` ending triggers the external test package shift. |

By default both land beside the source. Route them with `//testkit:out`, which is usually written once at package scope rather than repeated per interface:

```go
//testkit:out readertest/ pkg=readertest
package reader
```

Every double in the package then lands in `readertest`. A per-interface directive says the same thing N times, and the Nth copy is the one that gets forgotten.

To move one output without the other, scope the override by tag:

```go
//testkit:out tag=test ./elsewhere/
```

## What it generates

Given this interface:

```go
//testkit:out readertest/ pkg=readertest
package reader

//testkit:stub
type Reader interface {
    Get(ctx context.Context, key string) (Value, error)
}
```

the generator writes `readertest/iface_stub.gen.go`. Six pieces, in order.

### The recorded call

```go
type ReaderGetCall struct {
    Ctx    context.Context
    Key    string
    Result reader.Value
    Err    error
}
```

Field names come from the source signature — parameters and named returns alike. A signature written `(item User, err error)` documents what its returns mean, and the recorded-call struct is the main consumer of that documentation: it is what a reader sees in a failure message. A slot the source left unnamed, or named `_`, falls back to `Result0`, `Result1`, positionally.

### The per-method configuration point

```go
type ReaderGetStub struct {
    *stub.MethodStub[ReaderGetCall]

    fn       func(context.Context, string) (reader.Value, error)
    fallback *ReaderGetReturn
}

func (s *ReaderGetStub) Returns(result reader.Value, err error) *ReaderGetStub
func (s *ReaderGetStub) Func(fn func(context.Context, string) (reader.Value, error)) *ReaderGetStub
```

The embedded [`stub.MethodStub`](../primitives/method-stub.md) supplies everything that does not depend on the signature: call recording, fault injection, latency against a virtual clock, gates, call-count expectations, strict mode.

### The double

```go
type ReaderStub struct {
    OnGet *ReaderGetStub
    // ...
}

var _ reader.Reader = (*ReaderStub)(nil)

func NewReaderStub(tb testing.TB, opts ...ReaderStubOption) *ReaderStub
func (s *ReaderStub) ResetCalls()
```

Each `On<Method>` field is that method's configuration point. Left alone, a method returns its zero value and records the call.

The compile-time assertion lives in the double's own file rather than in the companion, so a drifted signature fails `go build` instead of waiting for a test run.

Passing `tb` to the constructor registers a cleanup that verifies every method's call-count expectations when the test ends, so an unmet `Times(2)` is reported without the caller remembering to check. A `nil` tb skips that, which is what benchmarks and non-test callers want. `*testing.F` is skipped too: a fuzz target reruns its body many times and registers a cleanup per run, so verifying there would report against the wrong iteration.

`ResetCalls` clears recorded calls, fault counters and call-count expectations. `Func` and `Returns` configuration is deliberately preserved, so one double can carry across test phases without being rebuilt.

### Construction options

```go
func ReaderStubStrict() ReaderStubOption
func ReaderStubDelegateTo(impl reader.Reader) ReaderStubOption
func ReaderStubWithClock(clk clock.Clock) ReaderStubOption
func ReaderStubWithRandSource(src rand.Source) ReaderStubOption
func ReaderStubBenchMode() ReaderStubOption
func WithReaderGet(fn func(context.Context, string) (reader.Value, error)) ReaderStubOption
```

| Option | Effect |
|---|---|
| `Strict` | An unconfigured method fails the test rather than returning its zero value, turning a call nobody planned for into a failure at the call site instead of a puzzling zero downstream. |
| `DelegateTo` | Every method forwards to a real implementation and is recorded on the way through. |
| `WithClock` | Latency and time-windowed faults run on a [virtual clock](../primitives/clock.md), so a test asserting on a five-second timeout does not take five seconds. |
| `WithRandSource` | Probabilistic fault injection becomes reproducible. A failure nobody can replay is not a finding. |
| `BenchMode` | Disables call recording. Dispatch still works; only the accumulating call log is dropped, because its allocations are what a benchmark would otherwise measure. |
| `With<Iface><Method>` | Sets one method's body at construction, for the common case of configuring one method and taking defaults for the rest. |

### The interface methods

```go
func (s *ReaderStub) Get(ctx context.Context, key string) (reader.Value, error) {
    call := ReaderGetCall{Ctx: ctx, Key: key}
    r := stub.Answer(s.OnGet.MethodStub, &call, stub.Arms[ReaderGetCall, ReaderGetReturn]{
        Invoke:   s.OnGet.invoke(ctx, key),
        Fallback: s.OnGet.fallback,
        Fault:    func(err error) ReaderGetReturn { return ReaderGetReturn{Err: err} },
        Stamp: func(c *ReaderGetCall, r ReaderGetReturn) {
            c.Result = r.Result
            c.Err = r.Err
        },
    })
    return r.Result, r.Err
}
```

Which arm answers is [`stub.Answer`](../primitives/method-stub.md)'s decision, not the template's, so every generated double resolves a call the same way and the precedence is tested once rather than restated per method. The order is **injected fault, then `Func` override, then `Returns` fallback, then the zero value** — and under `Strict`, the zero-value arm fails the test instead.

## Fault helpers

A method that declares which errors it can be made to fail with gains one-shot helpers, contributed into the same file by the `fault` plugin:

```go
type Mixed interface {
    //testkit:mixin errors
    //testkit:fault ErrNotFound ErrGone
    Get(ctx context.Context, key string) (string, error)
}
```

```go
// FaultNotFound makes the next call to Get fail with ErrNotFound.
func (s *MixedGetStub) FaultNotFound() *MixedGetStub {
    s.Faults(errors.ErrNotFound, 1)
    return s
}
```

The `Err` prefix is stripped from the helper name. These are one-shot — the fault fires once and clears, which is what a test asserting on that error's handling wants. For anything else, `Faults(err, n)` on the embedded `MethodStub` takes a count, and [`FaultsWhen`, `FaultsUntil`, `FaultsFor` and `FaultsWithProbability`](../primitives/fault-injection.md) cover the rest.

The helpers arrive as a block at the end of the file rather than interleaved with each method's configuration. Slot ordering is per-plugin, not per-item, so there is no interleaving to be had — and the block is attributed in the file header, which an interleaved version would not be.

## Using the double

```go
func TestCacheMissesFallThrough(t *testing.T) {
    s := readertest.NewReaderStub(t, readertest.ReaderStubStrict())
    s.OnGet.Returns(reader.Value{}, reader.ErrNotFound).Times(1)

    c := cache.New(s)
    _, err := c.Lookup(t.Context(), "absent")

    testkit.ErrorIs(t, err, reader.ErrNotFound, "a miss must propagate")
    testkit.Equal(t, s.OnGet.LastCall().Key, "absent", "the key must reach the reader")
}
```

`Times(1)` is verified by the cleanup the constructor registered — nothing in the test body checks it.

### Delegating to a real implementation

`DelegateTo` wraps a production type so every call forwards to it and is recorded on the way through. This is what lets one generated suite run against both the double and the real thing, which is the point of conformance testing.

```go
real := memstore.New()
s := readertest.NewReaderStub(t, readertest.ReaderStubDelegateTo(real))

_, _ = s.Get(t.Context(), "k")

testkit.Equal(t, s.OnGet.CallCount(), 1, "the call must be recorded on the way through")
```

Per-method overrides still apply on top: set `s.OnGet.Func(...)` after construction and that method stops delegating while the rest continue. A fault takes precedence over both.

## Generic interfaces

A generic interface produces a generic double, and the double itself needs no help. The *companion test* does — Go cannot instantiate a generic type without concrete arguments, and the generator has no way to choose them.

```go
//testkit:stub witness=int,Score
type Store[K comparable, V any] interface { ... }
```

The witness names the concrete types the companion instantiates at. Without it the double is still generated; the companion test that would have proved it satisfies the interface is skipped, because there is no type to prove it at.

## Layout conventions

| File | Owner | Contents |
|---|---|---|
| `iface.go` | Developer | The interface, its directives, and the package-scope routing |
| `<pkg>test/iface_stub.gen.go` | Generator | The recording double. Do not edit. |
| `<pkg>test/iface_stub.gen_test.go` | Generator | The companion checks for the double itself. Do not edit. |

The double lives in a normal (non-test) file so tests in other packages can import it. The companion lives in a `_test.go` file so it stays out of the binary.

## See also

- [`stub.MethodStub`](../primitives/method-stub.md) — the per-method dispatch engine the generated code embeds.
- [Recording](../primitives/recording.md) — `Recorder`, gates, and the call-inspection API.
- [Fault injection](../primitives/fault-injection.md) — the strategies behind `Faults` and its variants.
- [Clock](../primitives/clock.md) — the virtual clock `WithClock` binds.
- [Builder](builder.md) — for constructing the values a double returns.
