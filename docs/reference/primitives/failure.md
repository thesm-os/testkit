# Failure

```go
import "go.thesmos.sh/testkit/core/failure"
```

The envelope every generator reports a failure through. One shape, classified by kind, carrying the artifacts a reader needs to diagnose it — so CI ingestion routes on a field rather than parsing prose, and a failure from one generator reads like a failure from any other.

```go
func New(generator string, kind Kind, err error) *Failure

func (f *Failure) Error() string
func (f *Failure) Unwrap() error
```

`Failure` implements `error` and unwraps to the cause, so `errors.Is` and `errors.As` reach through it.

## Fields

| Field | Carries |
|---|---|
| `Kind` | The classification. Routes to a per-kind reporter. |
| `Generator` | Which generator produced it — `model`, `sim`, `chaos`, `diff-rollout`, `replay`. CI routes per-generator detail handlers from this. |
| `Subject` | The interface or subsystem the failure pertains to (`basic.Store`, `ledger.Subsystem`). Appears in `Error()` and in artifact names. |
| `REQID` | The primary requirement traced through this failure, when known. The REQ-to-law coverage matrix indexes on it. |
| `Pos` | The source position, when the failure ties to one. Zero-valued otherwise. |
| `Err` | The cause. |

## Kinds

```go
func ParseKind(s string) (Kind, error)
func (k Kind) String() string
```

| Kind | Means |
|---|---|
| `KindUnclassified` | No classification was assigned |
| `KindStructural` | The shape of the result is wrong |
| `KindSemantic` | The value is wrong |
| `KindInvariant` | A stated invariant was violated |
| `KindLiveness` | Something that must eventually happen did not |
| `KindDivergence` | Two implementations disagreed |
| `KindReplayMismatch` | A replayed trace did not reproduce |
| `KindChaosCrash` | A fault schedule produced a crash |
| `KindBudgetExceeded` | An allocation or latency budget was breached |

`KindUnclassified` is the zero value, so a `Failure` built without a kind is visibly unclassified rather than silently structural.

## Artifacts

```go
type Artifact struct { ... }

func (a Artifact) Open() (io.ReadCloser, error)
func (a Artifact) JSON() ([]byte, error)
func ParseArtifactKind(s string) (ArtifactKind, error)
```

An artifact is a file a failure produced — the thing a reader opens after reading the message.

| ArtifactKind | Holds |
|---|---|
| `ArtifactFailfile` | The minimal reproduction |
| `ArtifactTraceJSON` | The [trace](trace.md) |
| `ArtifactTimelineHTML` | The rendered [timeline](visualize.md) |
| `ArtifactClassifiedJSON` | The failure envelope itself |
| `ArtifactPorcupineHTML` | A linearisability check's visualisation |
| `ArtifactSnapshotJSON` | Component state at the point of failure |
| `ArtifactDivergenceReport` | Where two implementations differed |
| `ArtifactTLATrace` | A model-checker counterexample |
| `ArtifactReplayCapture` | The captured production trace |
| `ArtifactCertificationRecord` | The certification result |
| `ArtifactUnclassified` | Anything else |

`Open` returns a reader rather than bytes, so a large artifact does not have to be held in memory to be forwarded.

## Snapshot

```go
func (s *Snapshot) IsEmpty() bool
```

Component state captured at the failure point. `IsEmpty` distinguishes "nothing was captured" from "captured, and it was empty" — which matters when deciding whether the absence of state is itself the finding.

## Position

```go
func (p Position) IsZero() bool
func (p Position) String() string
```

A source location. `IsZero` is how a reporter decides whether to print one at all: a runtime failure with no source line should not render `:0:0`.

## JSON

`Failure`, `Kind`, `ArtifactKind` and `Artifact` all round-trip through JSON. That is the CI contract — a failure crosses a job boundary as data and is re-routed on the other side without re-parsing a message.

## Status

Consumed by `core/visualize` and the `engine/model` tree. The generators that report through it are not implemented.

## See also

- [Trace](trace.md) — usually the most useful artifact attached to a failure.
- [Visualize](visualize.md) — turns a failure's trace and snapshot into a timeline.
- [Coverage](coverage.md) — the REQ matrix `REQID` indexes into.
