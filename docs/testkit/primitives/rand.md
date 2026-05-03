# RandSource

`RandSource` is a pluggable random-number-generator interface used by probabilistic fault injection. testkit defines the interface; consumers inject their own deterministic RNG (e.g., a simulation engine's seeded PCG) when reproducibility matters.

## Interface

```go
type RandSource interface {
    Float64() float64 // returns a value in [0.0, 1.0)
}
```

Single method. Implementations must be safe for concurrent use — testkit calls `Float64` from generated stub dispatch code that may run across goroutines.

## Implementations

### DefaultRandSource

```go
src := testkit.DefaultRandSource()
```

Backed by `math/rand/v2` (the package-level global). Thread-safe. Used by every probabilistic fault when no source is explicitly configured.

### FixedRandSource

```go
src := testkit.FixedRandSource(0.0) // every call returns 0.0
```

Returns a constant. Use in tests to force deterministic outcomes from probabilistic faults: `0.0` makes every probabilistic fault fire; `1.0` makes none fire.

## Wiring into stubs

```go
stub.OnGet.WithRandSource(testkit.FixedRandSource(0.0))
stub.OnGet.FaultsWithProbability(errBoom, 0.05) // fires every call (because 0.0 < 0.05)
```

In a generated stub, the constructor option `<Stub>RandSource(src)` propagates the source to every method's `MethodStub`.

## Why not just use math/rand directly?

Two reasons:

1. **Reproducibility.** Simulation engines manage seeds carefully — a single seed reproduces an entire run. testkit can't manage seeds for them; the engine plugs in its own source so the seed lifecycle stays in one place.
2. **Determinism in tests.** `FixedRandSource` lets tests assert "this probabilistic fault will fire" or "won't fire" without flake.

## Concurrency

Implementations must be thread-safe — `Float64` is called from generated stub dispatch and may run concurrently across goroutines. `DefaultRandSource` is thread-safe (math/rand/v2 global). `FixedRandSource` returns a constant and is trivially thread-safe.

## See also

- [Fault injection](fault-injection.md) — `ProbabilityFault` consumes RandSource
- [MethodStub](method-stub.md) — `WithRandSource`
