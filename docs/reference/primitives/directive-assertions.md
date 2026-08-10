# Directive assertions

```go
import "go.thesmos.sh/testkit"
```

Three helpers for invariants a signature cannot express: that a call does not panic, that it changes nothing observable, and that its result stays inside a range. Each corresponds to a [mixin](../generators/shapes.md#mixins) a generated suite would assert automatically — exported so a hand-written test can assert the same thing on code no generator has reached.

## AssertNilSafe

```go
func AssertNilSafe(tb testing.TB, fn func())
```

Calls `fn` and fails if it panics, reporting the recovered value and the stack.

```go
testkit.AssertNilSafe(t, func() {
    var s *store.Store
    _ = s.Len()
})
```

The invariant is that a method on a nil receiver, or one handed a nil argument, returns rather than panicking. It corresponds to `//testkit:mixin nilsafe`.

Reach for it where a nil is reachable from a caller — an optional dependency, a zero-value struct, a map lookup that missed. Not as a blanket wrapper: a panic that a test converts into a failure message is still a panic, and `AssertNilSafe` around everything hides which call was supposed to be safe.

## AssertPure

```go
func AssertPure[S any](tb testing.TB, observe func() S, fn func())
```

Snapshots observable state with `observe`, runs `fn`, snapshots again, and fails if the two differ under `cmp.Diff`.

```go
testkit.AssertPure(t,
    func() []store.Record { return db.All() },
    func() { _ = svc.Validate(rec) },
)
```

The invariant is that `fn` has no side effect on whatever `observe` can see. It corresponds to `//testkit:mixin pure`.

**The assertion is only as strong as `observe`.** A snapshot that reads one field proves nothing about the rest, and one that reads a map without ordering it will fail intermittently for a reason that has nothing to do with purity. Return a value with a stable rendering, and make it cover the state the method could plausibly touch.

## AssertBounded

```go
func AssertBounded[T cmp.Ordered](tb testing.TB, lower, upper T, fn func() T)
```

Calls `fn` and fails if the result falls outside `[lower, upper]`, inclusive.

```go
testkit.AssertBounded(t, 0, 100, func() int { return pool.Size() })
```

It corresponds to `//testkit:mixin bounded limit=100`. Both ends are checked, so it catches a counter that went negative as well as one that overran — and the negative case is the one a `<= limit` check written by hand usually misses.

## Choosing between these and a plain assertion

These wrap a call and assert something about the *manner* of the call. Where the property is about a value you already have, the [plain assertions](assertions.md) say it more directly:

```go
// Not this — the closure adds nothing.
testkit.AssertBounded(t, 0, 100, func() int { return n })

// This.
testkit.Assert(t, float64(n)).IsWithin(0, 100, "n must stay in range")
```

## See also

- [Context assertions](context.md) — the four context-shaped contract helpers.
- [Assertions](assertions.md) — the direct forms.
- [Shape classification](../generators/shapes.md) — the mixin vocabulary these mirror.
