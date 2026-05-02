# Assertions

Type-safe assertion helpers with mandatory context strings.
Every assertion accepts `testing.TB` (works with both
`*testing.T` and `*testing.B`) and produces a clear
failure message without requiring `%v` formatting.

## Positional assertions

### Core assertions

| Function | Description |
|----------|-------------|
| `Equal[T any](t, got, want, msg)` | go-cmp structural diff on mismatch |
| `NotEqual[T any](t, got, want, msg)` | Fails if got deep-equals want |
| `ErrorIs(t, err, target, msg)` | Fails if !errors.Is(err, target) |
| `ErrorAs[T](t, err, msg) T` | Fails if !errors.As; returns matched value |
| `NoError(t, err, msg)` | Fails if err != nil |
| `Error(t, err, msg)` | Fails if err == nil |
| `True(t, cond, msg)` | Fails if !cond |
| `False(t, cond, msg)` | Fails if cond |
| `Len(t, obj, n, msg)` | Fails if len(obj) != n (slice/map/string/chan/array) |
| `Panics(t, fn, msg) any` | Fails unless fn panics; returns recovered value |

`Equal` uses `go-cmp` internally for structural comparison
and produces field-level diff output on mismatch —
dramatically better than `reflect.DeepEqual` for complex
structs. For JSON comparison, unmarshal first and use
`Equal`.

### Sequence assertion

| Function | Description |
|----------|-------------|
| `AssertSequence[T](t, items, pred, msg)` | Fails unless pred(items[i-1], items[i]) holds for every adjacent pair |

```go
testkit.AssertSequence(t, timestamps, func(a, b time.Time) bool {
    return a.Before(b)
}, "timestamps must be strictly ordered")
```

## Fluent assertions

Fluent matcher chain over a test subject. Start with
`Assert`; every matcher returns the same `*Assertion[T]`
for chaining. Matchers fatal on failure (AND logic).

```go
testkit.Assert(t, got).
    Equals(want, "contract X").
    HasLen(3, "must carry 3 entries").
    Contains(required, "must include required")
```

| Matcher | Description |
|---------|-------------|
| `Equals(want, msg)` | go-cmp structural comparison |
| `NotEquals(want, msg)` | Inverse of Equals |
| `IsNil(msg)` | Interface/pointer/slice/map/chan/func is nil |
| `IsNotNil(msg)` | Inverse of IsNil |
| `HasLen(n, msg)` | len(got) == n |
| `IsEmpty(msg)` | len(got) == 0 |
| `IsNotEmpty(msg)` | len(got) > 0 |
| `Contains(needle, msg)` | String contains substring, slice/array contains element, map contains key |
| `NotContains(needle, msg)` | Inverse of Contains |
| `IsError(target, msg)` | errors.Is(got, target) |
| `IsErrorAs(target, msg)` | errors.As(got, target) |
| `Matches(pattern, msg)` | Regexp match on string/[]byte |
| `Approximately(want, tol, msg)` | \|got - want\| <= tolerance (numeric) |
| `Within(lo, hi, msg)` | lo <= got <= hi (numeric) |
| `Panics(msg)` | got is func(); must panic |

The fluent form complements the positional helpers. Use
whichever reads more clearly: positional for a single
assertion, fluent for a sequence about the same subject.
