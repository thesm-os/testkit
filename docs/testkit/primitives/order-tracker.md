# OrderTracker

`OrderTracker` enforces call-order constraints across methods on a stub. It is the runtime substrate for the `//testkit:order-after` directive — generated stubs embed an `OrderTracker` when any method has the directive and call into it from generated dispatch.

## Use

Most consumers don't construct `OrderTracker` directly — generated stub code does. The interaction is through directives:

```go
type Migrator interface {
    //testkit:order-after Connect
    Apply(ctx context.Context) error

    Connect(ctx context.Context) error
}
```

The `stub` generator emits an `OrderTracker` inside the aggregate stub, and `Apply`'s dispatch calls `AssertAfter("Connect")` before proceeding. Calling `Apply` before `Connect` fatals the test (in strict mode) or no-ops (in lenient mode).

## API

```go
ot := testkit.NewOrderTracker(strict bool)

ot.Record("Connect")                    // mark as called
ot.AssertAfter(tb, "Apply", "Connect")  // checks Connect was called
ot.Called("Connect")                    // returns true
ot.Reset()                              // clear history
ot.String()                             // debug snapshot
```

| Method | Purpose |
|--------|---------|
| `NewOrderTracker(strict)` | Construct; strict=true fatals on violation |
| `Record(method)` | Mark method as called |
| `AssertAfter(tb, method, prereq)` | Fail if `prereq` hasn't been called yet |
| `Called(method)` | Has the method been called? |
| `Reset()` | Clear history |
| `String()` | Debug rendering of called set |

## Strict vs lenient

```go
ot := testkit.NewOrderTracker(true)  // strict — fatal on violation
ot := testkit.NewOrderTracker(false) // lenient — Record and AssertAfter no-op
```

Generated stubs default to strict. Lenient mode is for benchmarks and stress tests where ordering enforcement is overhead and the test isn't asserting ordering.

## Concurrency

`Record` and `AssertAfter` are safe under concurrent dispatch — the tracker uses a mutex.

## See also

- [`order-after` directive](../../testkit/configuration.md) — produces OrderTracker integration
- [MethodStub](method-stub.md) — generated stubs that embed an OrderTracker
