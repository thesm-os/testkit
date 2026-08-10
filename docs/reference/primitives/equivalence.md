# Equivalence

```go
import "go.thesmos.sh/testkit/core/equivalence"
```

Comparing two responses from two implementations, where "equal" is not `reflect.DeepEqual`. A generated ID differs by design. A timestamp differs by microseconds. A retry count differs because one side retried. None of those is a divergence, and a comparison that reports them buries the one that is.

A `Relation` names one such allowance. A `Chain` composes several into one comparison.

## The interface

```go
type Relation interface {
    Name() string
    Options() []cmp.Option
}
```

Relations layer on top of [`go-cmp`](https://pkg.go.dev/github.com/google/go-cmp/cmp): each contributes `cmp.Option` values and the chain composes them into one run. go-cmp's `FilterPath` handles "this relation applies to these paths only" natively, so the chain does not reimplement applies-versus-not routing.

`Name()` is what a divergence report cites, which is why every built-in composes a descriptive one — `id-field:store.Record:ID` rather than `relation 3`.

## Chain

```go
func NewChain() *Chain

func (c *Chain) Add(r Relation) *Chain
func (c *Chain) Equal(a, b any) bool
func (c *Chain) Diff(a, b any) string
func (c *Chain) Relations() []Relation
```

```go
eq := equivalence.NewChain().
    Add(equivalence.IDField(recordType, "ID")).
    Add(equivalence.Timestamp(recordType, "CreatedAt", time.Second)).
    Add(equivalence.OrderInvariant(recordType, "Tags"))

if !eq.Equal(fromOld, fromNew) {
    t.Fatal(eq.Diff(fromOld, fromNew))
}
```

`Diff` returns the rendered difference, empty when they match. Use it as the failure message — it names which relation admitted what, which is the difference between a report someone acts on and one they escalate.

## Built-ins

Twelve, covering the canonical migration-grade cases. Each takes a `reflect.Type` so it applies to one field of one type rather than to every field with that name anywhere in the tree.

| Relation | Admits |
|---|---|
| `Strict()` | nothing — exact equality |
| `IDField(typ, field)` | any difference in a generated identifier |
| `Timestamp(typ, field, tolerance)` | timestamps within `tolerance` |
| `Approximate(typ, field, tolerance)` | floats within `tolerance` |
| `RetryCount(typ, field)` | any difference in a retry counter |
| `OrderInvariant(typ, field)` | a collection reordered |
| `Cardinality(typ, field, lower, upper)` | a count inside `[lower, upper]` |
| `ErrorClass(errType)` | two errors of the same class |
| `RegexFields(typ, fields, pattern)` | fields both matching `pattern` |
| `IgnoreFields(typ, fields...)` | the named fields, whatever they hold |
| `IgnoreMapKeys(typ, keys...)` | the named map keys |
| `Custom(name, fn)` | whatever `fn` says |

Reach for the specific relation over `IgnoreFields`. `IDField` says "this is a generated ID and the two differ, as expected"; `IgnoreFields` says "do not look here" and would hide a genuine divergence in the same field. The names end up in the report, and the first one tells a reader why.

`Custom` takes the comparison as a closure for the case none of the built-ins expresses. Name it — the string is what appears when it admits something a reader disagrees with.

## Status

The package documents itself as consumed by `differential-rollout`'s response comparison and `replay`'s tolerance configuration. **Neither generator is implemented**, and nothing else in the repository imports it — see the [generator index](../generators/README.md).

The chain works standalone today. A hand-written differential test comparing an old and a new implementation can use it now, and that is the shape the generators will eventually emit.

The package doc also refers to a `testkit/registry/` package wrapping the preset registry for blank-import plug-ins. **That package does not exist in this repository.**

## See also

- [Assertions](assertions.md) — `Equal` uses go-cmp directly, for the case where exact equality is the assertion.
- [Failure](failure.md) — `KindDivergence` is what a failed chain comparison reports as.
