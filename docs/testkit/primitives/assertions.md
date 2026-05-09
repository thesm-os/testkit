# Assertions

Type-safe assertion helpers with mandatory contract messages. Every assertion accepts `testing.TB` (works with both `*testing.T` and `*testing.B`) and produces a clear failure message without requiring `%v` formatting.

## Positional assertions

| Function | Description |
|----------|-------------|
| `Equal[T any](t, got, want, msg)` | go-cmp structural diff on mismatch |
| `NotEqual[T any](t, got, want, msg)` | Fails if `got` deep-equals `want` |
| `ErrorIs(t, err, target, msg)` | Fails if `!errors.Is(err, target)` |
| `ErrorIsNot(t, err, target, msg)` | Fails if `errors.Is(err, target)` — for distinguishing two errors |
| `ErrorAs[T](t, err, msg) T` | Fails if `!errors.As`; returns matched value |
| `NoError(t, err, msg)` | Fails if `err != nil` |
| `Error(t, err, msg)` | Fails if `err == nil` |
| `True(t, cond, msg)` / `False(t, cond, msg)` | Boolean assertions |
| `Len(t, obj, n, msg)` | Length check (slice, map, string, chan, array) |
| `Contains(t, haystack, needle, msg)` | Substring / element / map-key membership |
| `NotContains(t, haystack, needle, msg)` | Inverse of `Contains` |
| `HasPrefix(t, s, prefix, msg)` | Fails if string `s` doesn't start with `prefix` |
| `HasSuffix(t, s, suffix, msg)` | Fails if string `s` doesn't end with `suffix` |
| `ContainsInOrder(t, haystack, needles, msg)` | Fails unless all needles appear in order within haystack |
| `Panics(t, fn, msg) any` | Fails unless `fn` panics; returns recovered value |
| `Sequence[T](t, items, pred, msg)` | Fails unless `pred(items[i-1], items[i])` holds for every adjacent pair |

`Equal` uses `go-cmp` internally for structural comparison and produces field-level diff output on mismatch — dramatically better than `reflect.DeepEqual` for complex structs. For JSON comparison, unmarshal first and use `Equal`.

```go
testkit.Equal(t, got, want, "Get must return the stored item")
testkit.ErrorIs(t, err, store.ErrNotFound, "Get on missing key")
testkit.ErrorIsNot(t, err1, err2, "ErrA and ErrB must be distinct")
testkit.HasPrefix(t, path, "/api/", "path must start with /api/")
testkit.ContainsInOrder(t, log, []string{"init", "ready", "shutdown"},
    "lifecycle events must appear in order")
testkit.Sequence(t, timestamps,
    func(a, b time.Time) bool { return a.Before(b) },
    "timestamps must be strictly ordered")
```

## Fluent assertions

Fluent matcher chain over a single subject. Start with `Assert`; every matcher returns the same `*Assertion[T]` for chaining. Matchers fatal on failure (AND logic).

```go
testkit.Assert(t, got).
    Equals(want, "contract X").
    HasLen(3, "must carry 3 entries").
    Contains(required, "must include required")
```

| Matcher | Description |
|---------|-------------|
| `Equals(want, msg)` | go-cmp structural comparison |
| `NotEquals(want, msg)` | Inverse |
| `IsNil(msg)` / `IsNotNil(msg)` | Interface/pointer/slice/map/chan/func nil checks |
| `HasLen(n, msg)` | Length check |
| `IsEmpty(msg)` / `IsNotEmpty(msg)` | Length 0 / >0 |
| `Contains(needle, msg)` / `NotContains(needle, msg)` | Membership |
| `HasPrefix(prefix, msg)` / `HasSuffix(suffix, msg)` | String prefix/suffix checks |
| `ContainsInOrder(needles, msg)` | Ordered substring membership |
| `IsError(target, msg)` / `IsNotError(target, msg)` | `errors.Is` checks |
| `Matches(pattern, msg)` | Regexp match on string/[]byte |
| `IsApproximately(want, tol, msg)` | Numeric within tolerance |
| `IsWithin(lo, hi, msg)` | Numeric in range |
| `Panics(msg)` | Subject is `func()`; must panic |

Use whichever reads more clearly: positional for a single assertion, fluent for a sequence about the same subject.

## Directive-driven assertions

For semantic properties that match `//testkit:` directives, see [directive-assertions.md](directive-assertions.md) — 21 directive-driven assertions, cross-method invariants, HookRecorder, and suite options.
