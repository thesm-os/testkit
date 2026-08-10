# Coverage

```go
import "go.thesmos.sh/testkit/core/coverage"
```

Statement coverage says which lines ran. This says which **laws** were exercised, which requirements were reached, and how much of a state space a run explored — the questions a property-based or simulation run raises and `go test -cover` cannot answer.

A law that never fired is a law that proved nothing, and it is invisible in a green run.

## Aggregator

```go
func NewAggregator() *Aggregator

func (a *Aggregator) SetComponent(name string, c *ComponentCoverage)
func (a *Aggregator) ComponentNames() []string
func (a *Aggregator) Report() Report
func (a *Aggregator) DiffSince(prior *Aggregator) DiffReport
```

Collects per-component coverage and renders it as a report. `ComponentNames` is sorted, so a rendered report is stable across runs.

`DiffSince` answers the question that matters in CI: **did this change exercise less than the last one did?** A run that holds its coverage number while quietly stopping to exercise two laws has regressed, and only a diff shows it.

## ComponentCoverage

```go
func (c *ComponentCoverage) ActiveLawCount() int
func (c *ComponentCoverage) LawIDs() []string
func (c *ComponentCoverage) REQIDs() []string
func (c *ComponentCoverage) WeakLaws(threshold float64) []string
```

`WeakLaws(threshold)` returns the laws that fired but rarely — below the given proportion of runs. These are the ones to look at first: a law that fires once in ten thousand iterations is technically covered and practically untested, and it will not be the thing that catches a regression.

## SubsystemCoverage

```go
func (s *SubsystemCoverage) ActiveInvariantCount() int
func (s *SubsystemCoverage) InvariantIDs() []string
func (s *SubsystemCoverage) REQIDs() []string
```

The subsystem-level equivalent, for a simulation spanning several components. A per-interface run has none, which is why `Report.Subsystem` is a pointer and nil is meaningful.

## Report

```go
type Report struct {
    Components []ComponentSummary  // one per component, sorted by name
    Subsystem  *SubsystemSummary   // nil for per-interface harnesses
}
```

Marshals to JSON for CI ingestion. Sorted by name so two runs of the same suite produce a diffable document.

## Metrics

Three metric types travel inside the summaries:

| Type | Measures |
|---|---|
| `BranchCoverageMetrics` | Branches taken. `Ratio()` gives taken over total. |
| `StateSpaceMetrics` | How much of a model's reachable state space a run visited |
| `CausalityMetrics` | Concurrency actually achieved — a "concurrent" run that serialised proves nothing about concurrency |

`CausalityMetrics` is worth calling out. A stress run that happens to interleave nothing has exercised the code single-threaded, passes, and reports nothing unusual. The metric is what makes that visible.

## Diffing

```go
type DiffReport struct { ... }   // ComponentDiff, SubsystemDiff inside
```

`DiffSince` produces this. Read it as a gate rather than a report: a component that lost a law or a REQ between runs is the signal.

## Status

Consumed by the `engine/model` tree. The generators that would populate an aggregator from a live run are not implemented — see the [generator index](../generators/README.md).

Repository-wide statement, branch and mutation gates are a different thing entirely and are ergon's job (`ergon check coverage`, `check branch`, `check mutation`). This package measures what a *generated run* explored, not what the test suite touched.

## See also

- [Failure](failure.md) — `REQID` is the key both index on.
- [Trace](trace.md) — the observations coverage is computed from.
