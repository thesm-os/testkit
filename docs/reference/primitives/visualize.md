# Visualize

```go
import "go.thesmos.sh/testkit/core/visualize"
```

A failing property or simulation run produces a [trace](trace.md) of hundreds of events across several lanes. Reading that as JSON is possible and nobody does it. `visualize.Emit` renders one run as a self-contained HTML timeline.

```go
func Emit(w io.Writer, t Timeline) error
```

One HTML document with embedded CSS. **No external dependencies and no JavaScript** — it opens from a CI artifact directory in any browser, including one with no network. Tooltips are SVG `<title>` elements, and an event-detail table sits alongside the timeline for full-fidelity reading, because a tooltip cannot be copied out of.

## Timeline

```go
type Timeline struct {
    Subject   string        // "basic.Store", "ledger.Subsystem" — header
    Generator string        // "model" / "sim" / "chaos" / "diff-rollout" / "replay"
    Seed      int64         // rendered as 0x hex so a rerun is copy-pasteable
    Trace     *trace.Trace  // the event source
    Overlays  []Overlay
    // ...
}
```

Each event becomes a block, grouped into lanes by `trace.Event.Component` — or by `Goroutine` when no component is populated, which is the per-interface case.

The seed renders as hex in the header for one reason: the first thing anyone does with a failing run is try to reproduce it, and a decimal seed invites a transcription error.

## Overlays

An overlay adds markers on top of the timeline without the renderer knowing what it means.

```go
type Overlay interface {
    Render(*trace.Trace) []Marker
}
```

| Overlay | Marks |
|---|---|
| `CausalityOverlay()` | causal arrows between events |
| `FaultOverlay()` | fault activation windows |
| `REQOverlay()` | requirement tags |
| `DivergenceOverlay(markers)` | points where implementations disagreed |
| `ReplayMarkerOverlay()` | positions in a replayed capture |
| `SnapshotOverlay(snap)` | component state captured at failure |

```go
tl := visualize.Timeline{
    Subject:   "basic.Store",
    Generator: "model",
    Seed:      seed,
    Trace:     tr,
    Overlays: []visualize.Overlay{
        visualize.CausalityOverlay(),
        visualize.FaultOverlay(),
        visualize.SnapshotOverlay(snap),
    },
}
_ = visualize.Emit(out, tl)
```

`DivergenceOverlay` and `SnapshotOverlay` take their data as arguments because it comes from outside the trace — a comparison result and a [`failure.Snapshot`](failure.md#snapshot) respectively. The rest derive everything from the trace.

```go
type DivergenceMarker struct {
    EventID trace.EventID    // where divergence first appeared
    Lanes   map[string]any   // impl name to the value it produced
    Diff    string           // human-readable diff between lanes
}
```

`Lanes` is keyed by implementation name, which is where [`factory.Named`](factory.md#named) pays off — the tooltip says `redis` and `in-memory` rather than `0` and `1`.

## Style

```go
type Style struct {
    Theme           string             // "light" or "dark"; empty means light
    ComponentColors map[string]string  // component name to CSS colour
    Title           string             // overrides "<Subject> — <Generator> timeline"
}
```

A component with no entry in `ComponentColors` gets a deterministic colour from a fixed palette, indexed by a hash of its name. That is what keeps a component the same colour across runs and across documents — a legend nobody has to re-learn per artifact.

Implement `Overlay` for anything else. The renderer composes whatever markers come back.

## Determinism

Output is byte-identical for byte-identical input. Map iteration is normalised, there are no `time.Now()` calls, and no random IDs are generated.

That is what makes a timeline diffable: two runs on the same seed produce the same document, so a change in the rendering is a change in the run. Without it the artifact would be a picture, and pictures cannot be reviewed.

## Status

Nothing in this repository imports it. Its package doc names `sim`, `chaos`, `differential-rollout` and `replay` as the overlay contributors, and **none of those generators is implemented** — see the [generator index](../generators/README.md).

`Emit` works standalone. Any code holding a `*trace.Trace` can render one today.

## See also

- [Trace](trace.md) — the event source, and the filters that narrow one before rendering.
- [Failure](failure.md) — `ArtifactTimelineHTML` is what an emitted timeline is attached to.
