# Assertions

```go
import "go.thesmos.sh/testkit"
```

Every assertion takes `testing.TB` first and a message last, and calls `tb.Fatalf` on failure. Structural comparison goes through [`go-cmp`](https://pkg.go.dev/github.com/google/go-cmp/cmp), so a failure reports a diff rather than two rendered values you have to compare by eye.

The message says what was being checked. `"counts match"` is worth more than the diff below it, because the diff shows what differed and only the message shows what was supposed to be true.

## Functions

| Function | Fails when |
|---|---|
| `Equal[T](tb, got, want T, msg)` | `cmp.Diff(got, want)` is non-empty |
| `NotEqual[T](tb, got, want T, msg)` | `got` and `want` are structurally equal |
| `True(tb, cond bool, msg)` | `cond` is false |
| `False(tb, cond bool, msg)` | `cond` is true |
| `Len(tb, obj any, want int, msg)` | the length of `obj` is not `want` |
| `Contains(tb, haystack, needle any, msg)` | `haystack` does not contain `needle` |
| `NotContains(tb, haystack, needle any, msg)` | `haystack` contains `needle` |
| `HasPrefix(tb, s, prefix, msg)` | `s` does not start with `prefix` |
| `HasSuffix(tb, s, suffix, msg)` | `s` does not end with `suffix` |
| `ContainsInOrder(tb, haystack string, needles []string, msg)` | the needles do not all appear, in order |
| `Panics(tb, fn func(), msg) (recovered any)` | `fn` does not panic. Returns the recovered value. |
| `Sequence[T](tb, items []T, pred func(earlier, later T) bool, msg)` | any adjacent pair fails `pred` |

### Errors

| Function | Fails when |
|---|---|
| `NoError(tb, err, msg)` | `err` is non-nil |
| `Error(tb, err, msg)` | `err` is nil |
| `ErrorIs(tb, err, target error, msg)` | `errors.Is(err, target)` is false |
| `ErrorIsNot(tb, err, target error, msg)` | `errors.Is(err, target)` is true |
| `ErrorAs[T](tb, err error, msg) T` | `errors.As` cannot unwrap `err` into `T`. Returns the extracted value. |

`ErrorAs` returns what it extracted, so the assertion and the access are one statement:

```go
ve := testkit.ErrorAs[*store.ValidationError](t, err, "a bad field must surface as a ValidationError")
testkit.Equal(t, ve.Field, "email", "the offending field must be named")
```

`ErrorIsNot` exists because "this is not that error" is a real assertion and `!errors.Is` inside `True` loses the message. Generated sentinel checks rely on it.

## Sequence

`Sequence` checks a relation between every adjacent pair, which is how a monotonic or sorted property is stated without a loop:

```go
testkit.Sequence(t, timestamps, func(earlier, later time.Time) bool {
    return !later.Before(earlier)
}, "timestamps must not go backwards")
```

The predicate receives the pair in order. A failure names the index, so a long slice still points at the offending element.

## The fluent chain

```go
func Assert[T any](tb testing.TB, got T) *Assertion[T]
```

`Assert` starts a chain on one subject and returns an `*Assertion[T]`. Every method returns the same chain, and the chain stops at the first failure.

```go
testkit.Assert(t, resp.Body).
    IsNotEmpty("a response must carry a body").
    HasPrefix("{", "the body must be JSON").
    Matches(`"id":\s*"\w+"`, "the body must carry an id")
```

Use it when several properties of one value are worth stating together. Use the plain functions when the subjects differ — a chain over three different values reads as one assertion and is not.

| Method | Fails when |
|---|---|
| `Equals(want T, msg)` | `cmp.Diff` is non-empty |
| `NotEquals(want T, msg)` | structurally equal |
| `IsNil(msg)` / `IsNotNil(msg)` | the subject is non-nil / nil |
| `IsEmpty(msg)` / `IsNotEmpty(msg)` | the subject is non-empty / empty |
| `HasLen(want int, msg)` | the length is not `want` |
| `Contains(needle any, msg)` / `NotContains(needle any, msg)` | the subject does not contain / contains `needle` |
| `HasPrefix(prefix, msg)` / `HasSuffix(suffix, msg)` | the subject (string or `[]byte`) does not start / end with it |
| `ContainsInOrder(needles []string, msg)` | the needles do not all appear, in order |
| `Matches(pattern, msg)` | the subject does not match the regular expression |
| `IsError(target error, msg)` / `IsNotError(target error, msg)` | `errors.Is` against `target` is false / true |
| `IsApproximately(want, tolerance float64, msg)` | the subject is further than `tolerance` from `want` |
| `IsWithin(lo, hi float64, msg)` | the subject falls outside `[lo, hi]` |
| `Panics(msg)` | the subject (a `func()`) does not panic |

`IsError` and `Panics` require the subject to be an `error` and a `func()` respectively; anything else fails the assertion rather than compiling away.

## Asserting that an assertion can fail

A check whose every statement is `NoError` passes against a subject whose
methods return nil and do nothing. It reads as coverage and asserts nothing —
the same defect as a check that cannot fail, arriving from the other side. No
gate reports it, because a vacuous check passes.

`Rejects` names the wrong implementation, drives the check against it, and
observes the rejection:

```go
got := testkit.Rejects(t, "a pool with no bound", func(tb testing.TB) {
    handsOutWhatItHolds(tb, unboundedPool{})
})
testkit.Assert(t, got).Contains("the pool it came from is then empty",
    "and rejects it for the reason the check is about")
```

It returns the failure message rather than a bool, and that is worth using. A
stand-in that panics before the check's own assertion runs would satisfy a
boolean guard while proving nothing — which is the defect it exists to catch,
one level up.

The check runs on a goroutine of its own and is joined before the call returns:
a `FailableTB` in Goexit mode implements `Fatal` as `runtime.Goexit` does, which
is what stops a check running past an assertion it already failed, and Goexit
needs a goroutine to exit. A panic is deliberately not recovered — a check that
panics is a defect in the check or in the stand-in, and reporting it as a
rejection would hide it.

The [suite generator](../generators/suite.md) emits one of these per generated
check. `Rejects` is the same primitive for a check you write yourself.

## See also

- [Directive assertions](directive-assertions.md) — the contract-shaped helpers: nil-safety, purity, bounds.
- [Context assertions](context.md) — cancellation, deadline and timeout behaviour.
- [Helpers](helpers.md) — `FailableTB`, for testing an assertion's failing path.
