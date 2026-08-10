# Rand

```go
import "go.thesmos.sh/testkit/rand"
```

Probabilistic fault injection needs randomness, and a failure nobody can replay is not a finding. `rand.Source` is the seam that makes the draw controllable.

```go
type Source interface {
    Float64() float64
}
```

One method, because one is all a probability check needs. A narrower interface is a smaller thing to fake.

## The two implementations

```go
func DefaultRandSource() Source
func FixedRandSource(v float64) Source
```

`DefaultRandSource` draws from the standard library.

`FixedRandSource(v)` returns `v` from every call. That turns a probabilistic fault into a deterministic one and lets a test drive both sides of the threshold exactly:

```go
// p = 0.3; a draw of 0.1 is below it, so the fault fires every time.
s.OnGet.WithRandSource(rand.FixedRandSource(0.1))
s.OnGet.FaultsWithProbability(0.3, store.ErrUnavailable)

_, err := s.Get(ctx, "k")
testkit.ErrorIs(t, err, store.ErrUnavailable, "a draw below p must fire the fault")
```

Flip to `FixedRandSource(0.9)` for the other arm. Both branches of the comparison get covered, which a real source reaches only by luck.

## The boundary

The comparison is `draw < p`. `FixedRandSource(p)` for the exact `p` therefore does **not** fire — the boundary belongs to the non-firing side. Worth pinning explicitly if the threshold matters:

```go
s.OnGet.WithRandSource(rand.FixedRandSource(0.3))
s.OnGet.FaultsWithProbability(0.3, store.ErrUnavailable)

_, err := s.Get(ctx, "k")
testkit.NoError(t, err, "a draw equal to p must not fire")
```

## Seeded randomness for fixtures

`rand.Source` is for fault injection. Where a test wants varied but reproducible *data*, [`testkit.SeededRand`](helpers.md#fixtures) returns a `*math/rand.Rand` seeded from the test's name — deterministic within a test, different between tests.

```go
r := testkit.SeededRand(t)
for range 100 {
    _ = store.Put(fmt.Sprintf("k%d", r.IntN(1000)))
}
```

## Wiring a source

Per method:

```go
s.OnGet.WithRandSource(rand.FixedRandSource(0.1))
```

Across every method of a generated double:

```go
s := readertest.NewReaderStub(t, readertest.ReaderStubWithRandSource(rand.FixedRandSource(0.1)))
```

## See also

- [Fault injection](fault-injection.md) — `NewProbabilityFault`, the one consumer.
- [Method stub](method-stub.md) — `WithRandSource` and `FaultsWithProbability`.
- [Helpers](helpers.md) — `SeededRand` for fixture data.
