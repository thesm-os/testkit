# Order tracker

```go
import "go.thesmos.sh/testkit/stub"
```

Some methods may only be called after others. `Commit` after `Begin`, `Write` after `Open`, `Release` after `Acquire` — a contract the compiler cannot express and a double will happily violate.

`OrderTracker` records method names in call order and asserts the ordering held.

```go
func NewOrderTracker(tb testing.TB, strict bool) *OrderTracker

func (o *OrderTracker) Record(method string)
func (o *OrderTracker) AssertAfter(method, prerequisite string)
func (o *OrderTracker) Called(method string) bool
func (o *OrderTracker) Reset()
func (o *OrderTracker) String() string
```

## Strict

The `strict` flag decides what happens when the prerequisite was never called at all.

**Strict** fails: `Commit` without any `Begin` is a violation.

**Non-strict** passes: the ordering claim is conditional — *if* both were called, the order must have held. Use it for an optional prerequisite, where the absence of both is a legitimate path.

The default worth reaching for is strict. A conditional assertion that passes when neither method ran is a check that a broken implementation can satisfy by doing nothing.

## Using it

```go
ot := stub.NewOrderTracker(t, true)

s := txtest.NewTxStub(t)
s.OnBegin.OnRecord(func(BeginCall) { ot.Record("Begin") })
s.OnCommit.OnRecord(func(CommitCall) { ot.Record("Commit") })

svc.Transfer(ctx, from, to, amount)

ot.AssertAfter("Commit", "Begin")
```

`OnRecord` on each [recorder](recording.md) is the wiring: the tracker sees a name whenever that method is called, without the double knowing the tracker exists.

`AssertAfter(method, prerequisite)` checks every occurrence of `method`, not just the first. A `Commit` before its `Begin` fails even if a later pair is correctly ordered.

## Inspecting

`Called(method)` reports whether a name was recorded at all — useful for asserting a teardown ran without claiming anything about its position.

`String()` renders the recorded sequence, which is what makes a failure diagnosable. Log it when an ordering assertion fails and the reason is not obvious:

```go
t.Logf("call order was: %s", ot)
```

`Reset()` clears the recorded sequence, for a test that drives several phases through one tracker.

## Where the ordering comes from

A method that declares `//testkit:mixin orderafter fn=Prepare` states this contract in the source, and a generated conformance suite asserts it. The `stub` generator reads the same stamp to expose the prerequisite on the generated method — see [Shape classification](../generators/shapes.md#mixins).

Using `OrderTracker` by hand is for the case where the contract is real but the declaration has not been written yet, or where the ordering spans two interfaces a single directive cannot relate.

## See also

- [Recording](recording.md) — `OnRecord`, the hook that feeds the tracker.
- [Method stub](method-stub.md) — where the per-method recorder lives.
- [Shape classification](../generators/shapes.md) — the `orderafter` mixin.
