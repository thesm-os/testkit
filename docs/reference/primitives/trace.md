# Trace

```go
import "go.thesmos.sh/testkit/core/trace"
```

A record of method-call observations across a run, queryable after the fact. Where [`Recorder`](recording.md) captures one method's calls for a test to assert on, a `Trace` captures a whole run's calls for a *failure* to be explained from — which lane, which client, which fault, in what causal order.

## Recording

```go
func New() *Trace

func (t *Trace) Record(e Event) EventID
func (t *Trace) RecordOp(start time.Time, method string, inputs []any, output any, err error) EventID
func (t *Trace) Len() int
func (t *Trace) Reset()
func (t *Trace) Snapshot() []Event
```

`Record` assigns the ID and returns it; a caller-supplied `Event.ID` is overwritten. IDs are monotonic from 1, and **zero is the no-event sentinel** — which is what makes an unset causal reference detectable rather than pointing at the first event.

`RecordOp` is the shorthand for the common case: a method call with inputs, one output and an error.

`Snapshot` returns an independent copy, so a caller can hold it while recording continues.

## Event

The fields worth knowing, because several are easy to misread:

| Field | Carries |
|---|---|
| `ID` | Assigned by `Record`. Monotonic from 1. |
| `Tick` | The engine tick. **Zero for traces produced outside an engine context** — a per-interface model trace uses only the timestamps. |
| `StartNs`, `EndNs` | Nanoseconds. Engine-clock-relative for engine traces, wall-clock-relative otherwise. `EndNs >= StartNs` is invariant. |
| `Goroutine` | The **engine-assigned worker ID, not the OS goroutine ID**. Zero means single-worker or no partition; engine values start at 1. |
| `ClientID` | Partitions multi-client traces. Zero is the default partition. |
| `Component` | The subsystem component that produced the call. Empty for per-interface traces. |

`Goroutine` is the one that catches people. It will not match anything `runtime` reports — see [`concurrency.CaptureGoroutineIDs`](concurrency.md#inspecting-the-goroutine-set-directly) for real goroutine IDs.

### Fault context

An event carries what was being done to it, not only what it did.

```go
type FaultContext struct {
    Active   []FaultActivation
    Affected bool
}

type FaultActivation struct {
    Name      string          // "NetworkPartition", "ClockSkew"
    Component string          // empty for a global fault
    StartedAt int             // engine tick
    EndsAt    int             // engine tick; -1 means unbounded
    Args      map[string]any  // per-activation parameters
}
```

`Active` is a list because a call can be touched by several faults at once — a network partition *and* a clock skew.

**`Affected` is the field that matters.** A fault can be active without affecting a particular call: a partition on component A does not touch calls to component B. Filtering on "a fault was active" over-reports; `Affected` is what the chaos runtime sets from per-fault routing rules, and it is what `FaultEvents()` keys on.

`EndsAt` of `-1` means the fault heals only on explicit removal, not on a tick.

## Querying

Every filter returns a new `*Trace`, so they compose:

```go
suspects := tr.FilterByComponent("ledger").FilterByMethod("Commit").FaultEvents()
```

| Method | Returns |
|---|---|
| `FilterByComponent(name)` | events from one component |
| `FilterByMethod(name)` | events for one method |
| `FilterByGoroutine(gid)` | events from one worker lane |
| `FilterByClient(id)` | events from one client partition |
| `FilterByREQ(reqID)` | events tagged with a requirement ID |
| `FilterByPredicate(pred)` | anything else |
| `FaultEvents()` | events where a fault activated |
| `CausalSlice(id)` | the events causally reachable from `id` |

`CausalSlice` is the one that turns a trace into an explanation: given the failing event, it returns what led to it and drops everything concurrent but unrelated.

## Causality validation

```go
func (t *Trace) ValidateCausality() []DanglingRef
```

Returns every causal reference pointing at an event that is not in the trace. An empty result means the causal graph closes.

A dangling reference usually means a filter dropped an event another event still points at, or a trace was truncated. Either way the graph no longer explains itself, and a visualisation built from it will draw arrows into nothing.

## Determinism

```go
func EqualForDeterminism(a, b *Trace) bool
func DiffForDeterminism(a, b *Trace) string
```

Compare two traces ignoring the fields that legitimately vary between runs, and report the difference as text.

This is the assertion behind "the same seed produces the same run". A plain `cmp.Diff` would fail on wall-clock timings that carry no meaning; these compare the part that is supposed to be reproducible.

## JSON

`Trace` and `Event` both marshal and unmarshal, so a trace survives a CI artifact boundary and can be re-queried later against the run that produced it.

## Status

Consumed by `core/failure`, `core/visualize`, and the `engine/model` tree. The generators that would populate a trace from a live run — `model`, `sim`, `chaos` — are not implemented.

## See also

- [Failure](failure.md) — the envelope a trace is attached to.
- [Visualize](visualize.md) — renders a trace as an HTML timeline.
- [Recording](recording.md) — the per-method recorder, which is the per-test equivalent.
