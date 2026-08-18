# engine/model — what to keep, revise, and delete

A census of `engine/model` and its subpackages against the accepted suite
contract ([RFC-0004](../rfc/0004-the-suite-contract.md)), done with techne
type queries rather than grep. The question was whether the engine is still
needed once checks are plain data. It is — but two thirds of its exported
surface is not, and the parts that matter are not the parts that are large.

Measured 2026-08-14. One area is not covered: `engine/model/action`,
`timeaware`, `history`, and the `core/*` support packages. That survey did
not finish and is listed as open work at the end.

## The short answer

Keep the engine. Delete 64% of its exported surface. Move one seam. Wire
one instrument that is currently a gate reporting success over zero work.

| | Exported symbols | Share |
|---|---|---|
| Irreplaceable | 15 | 9% |
| Duplicated by the contract | 12 | 7% |
| Dead — no reference outside its own tests | **110** | **64%** |
| Live but never named directly | 5 | 3% |
| Members of the above | ~30 | 17% |

The irreplaceable part is small and sharp: **Porcupine linearizability**,
**the law census**, and **`LawsOnly`**. Everything else is either the
contract's job now or nobody's.

## What I got wrong before this census

Two corrections worth recording, because both were stated with confidence.

**The engine does not reimplement rapid's state machine.** `runner.go:445`
is `rt.Repeat(actionMap)`, with the law sweep installed under the `""` key —
the documented invariant slot. That is the textbook use of the API. The
thing that hand-rolled a loop instead of calling `Repeat` was the `gen/`
example, written here. The engine had it right first.

**`engine/model/rapid.go` already re-exports `SyncTest`.** The go1.25
`testing/synctest` integration is not a missing feature; it is present, with
zero consumers.

## Keep: the 15 that earn their place

**Porcupine linearizability** — `ConcurrentConfig`, `ConcurrentAction`,
`OpInput`, `OpOutput`, `TraceResult`, `WithConcurrent`. `runConcurrent` does
not use `Repeat` and cannot: it pre-draws a whole `[workers][ops]` schedule
in the single rapid goroutine (because `rapid.T` is not safe for concurrent
use), executes across workers with an atomic logical clock stamping call and
return, and hands the history to Porcupine. Rapid has no linearizability
checker. Nothing else in the tree provides this.

**The law census** — `Registry` with `ran`/`fired`/`vacuous`/`declined`.
Two live behaviours matter. `noteVacuous` reports a law that returned
`law.Vacuous` on every single check: *"the subject refuses every
precondition this run supplies, and the binding is asserting nothing; widen
the pools or supply accepted values."* `Declined` records a law that was
selected but could not be armed, naming the option that would arm it. The
contract's `Check` is plain data and has no notion of a check that ran and
engaged nothing. `unmetCaps` covers declination better — it fails instead of
logging — but nothing covers vacuity.

**`LawsOnly`** — 77 consumer files. It silences the action differential so a
prover can ask whether a law can fail. Its doc is the sharpest paragraph in
the codebase and it is aimed at a hole in RFC-0004; see below.

## The law catalogue: 83 laws, and where the value is

83 IDs, 83 implementations, zero orphans, zero unbound, two documented
unreachable — held total by gate tests in three directions. That is
unusually clean and is not the part in question.

The question is how much of it a plain-data check could restate:

| Bucket | Count | What it needs |
|---|---|---|
| EASY | 40 (48%) | call, call again, compare. Under 20 lines. |
| MODERATE | 22 (27%) | a helper, two fresh subjects, or a vacuity precondition |
| HARD | 21 (25%) | cross-call state, the trace, a fake clock, N replicas, cycle detection |

Three things the line count hides, all favouring the catalogue:

- **The `Vacuous` sentinel is the difference between a test and a lie.**
  `PoisonConsistent` asserts nothing if the poison did not take;
  `AtomicWrite` asserts nothing if the write succeeded; `TTLExpiry` aborts
  if `Put` errored. A naive generated check writes `if err == nil { return }`
  and reports a **pass**. `DeadlineRespecting`'s doc is a two-paragraph
  account of getting this wrong in both directions — discarding the error
  made it unfalsifiable, demanding it made it wrong about correct code.
  That reasoning does not survive a rewrite.
- **`CASAtomicOneWinner` is short but not easy.** The load-bearing line is
  `v1, v2 = Stamp(v1), Stamp(v2)` — *both stamped before either runs*. Stamp
  lazily and the second attempt gets a fresh version, both succeed, and the
  law can never fail.
- **17% of laws carry state** and would go non-deterministic under shrinking
  without the reset discipline. `Sticky` remembering a value from a store
  that no longer exists false-fails; a trace law that never gets bound
  panics on nil.

**And one thing that favours narrowing:** roughly 32 of 83 laws carry
`Mirrored` or `Isolated` complexity that exists **only because the engine
interleaves laws with a differential action stream over one shared pair**.
Lines like `_ = l.CAS(rt, ref, v1)` are engine tax, not property content.

## Revise: three concrete changes

### 1. The 11 self-contained laws should be plain checks

Six laws carry the `Isolated` marker and five more carry a `Factory` field.
All eleven build their own subjects and never touch the shared pair — the
runner already runs them once per iteration against throwaway pairs.

They do not need the action stream, so they do not need the engine's
execution model. Moved to `suite.Check` values they become ordinary checks,
and the `Isolated` mechanism — plus the `taggedIsolatedLaw` wrapper trick it
forces — can be deleted from the runner.

The laws that must stay are the ones that observe *accumulated* history:
those genuinely need the shared pair, and that is what `Mirrored` pays for.

### 2. Move subject construction to the contract

`SUTFactory`, `RefFactory`, and `Cleanup` are duplicated, and the contract's
versions are better:

- `Subject.New(tb)` takes `tb`, so a container-backed subject registers its
  own teardown. `Cleanup` fires per iteration via `defer` and misses the
  isolated-law and concurrent paths.
- **`RefFactory` is a generation-time decision — that is what forces the
  twin floor.** The generator must decide the reference when it writes the
  file. `Subject.Oracle` is a run-time consumer decision and can express
  "compare Postgres against the in-memory implementation", which the engine
  cannot say at all.

Keep `Actions`: it is the comparison, trace, and classification wrapper
around each `Repeat` entry, not a reimplementation of the driver.

### 3. Wire `FireRate` — it is one function away

`coverage.WeakLaws` exists to name laws that never fire. It reads
`ComponentCoverage.FireRate`. **Nothing in production writes that map**, so
`WeakLaws` returns "no weak laws" unconditionally and reads as a pass.

`Registry.Coverage()` returns the `(ran, fired)` maps that would compute it,
and has no caller outside `runner_test.go`. The engine's own comment says
so: *"the runtime fire-rate fill is a separate, not-yet-wired concern."*

This is the vacuity instrument the depth program needs, and the counters
already exist. Wiring it is small; the fallout will not be.

## Delete: the 110

**81 of them are one file.** `engine/model/rapid.go` is a pass-through shim
over `pgregory.net/rapid` — `Bool`, `Int64Range`, `SliceOfDistinct`,
`Permutation`, `MapOfN`, and 76 more, each a one-line delegation, each
referenced only by `rapid_test.go`. It exists so consumers never type
`rapid.`. The `gen/` example types `rapid.` and is shorter for it.

Keep from that file: the type aliases consumers do use (`T`, `TB`,
`Generator`), the nine re-exports with real consumers, and
`AdversarialStrings` — which is genuine content sitting in the wrong
package.

**The rest:**

- `reference.go`'s composition helpers — `Pair`, `Triple`, `LiftA`, `LiftB`,
  `LiftTripleA/B/C`. Seven, dead as a cluster.
- Options with no consumer: `WithStateHash`, `WithSaturationThreshold`,
  `WithCoverageSink`, `WithCleanup`, `WithoutTrace`, `WithArtifactDir`,
  `WithLawREQ`, `SkipLaw`.
- `RaceEnabled` — two build-tagged files defining a constant nothing reads.
- Six dead reference oracles: `PartitionedAppendOnly`, `FoldMachine`,
  `BootOnlyRegistry`, `CompensatingSaga`, `PureScheduler`, `GuardedStates`.
  Three are sketches; `PureScheduler` holds no state and is not a
  differential oracle at all.

**Do not delete, but move:** eleven more `ref` types are unreached by any
generator path and exist as testkit's own law-test fixtures
(`MonotonicLog`, `AtomicCell`, `BoundedCursor`, `CursorTable`,
`BalancedPool`, `AtLeastOnce`, `AtMostOnce`, `ExactlyOnce`, `Coalescer`,
`SnapshotIsolation`, `RollingCounter`). They are fixtures wearing public API
clothes. Behind an internal package they cost nothing; exported they are
surface a consumer reads and a tag freezes.

## The reference oracles that do earn their place

Only 8 of 25 `ref` types are reachable from generated code, and the split
matters for the differential-oracle question.

**Keep unconditionally — four.** `MapStore`, `KeyedStore`, `Collection`,
`StickyStore` fire from shape classification with **zero consumer input**:
no directive, no second implementation, no `.Oracle()` call. They account
for 20 of the corpus's 24 derived references. This is a different product
from the differential oracle, not a redundant one — `.Oracle()` costs the
consumer a second implementation; these cost nothing.

**Keep, narrowly — three.** `AppendOnly`, `LeaseTracker`, `VersionedCell`,
one contract each, one corpus fixture each. A consumer with two
implementations would rather mark `.Oracle()`. A consumer with one has
nothing else.

The generator's own file records why several others can never be derived:
*"a pool needing a resource constructor, a saga needing its steps, a
coalescer needing the function it coalesces — stay on the twin floor."*

## Porcupine: keep, but narrow and bound it

It belongs in conformance, not simulation. The argument is what the code
needs to run: `concurrent.go` imports no clock control, no scheduler hook,
no fault injector, no crash or partition machinery. `time` appears once, for
a default timeout. Its inputs are ordinary method signatures; its failures
ride the same `Failure` pipeline as every deterministic check.

But it is over-built relative to demand and under-bounded relative to cost.

**Demand:** 8 hand-written models, and **four have zero corpus reach** —
`Counter`, `Set`, `Pool`, `Cursor`. `KV` alone accounts for 7 of the 10
wirings. Model #9 costs roughly 300–400 lines across 4–5 files plus a
generator row and a fixture. Do not build it until models five through eight
earn a single call site.

**A general mechanism exists and is declined.** `ModelBuilder` was written
to spare authors the untyped Porcupine wrapping. **None of the eight models
use it** — every one hand-rolls a full `porcupine.Model` literal. `kv.go`
even inlines its own copy of `partitionByKey` verbatim.

**Four defects to fix in place:**

- **Nothing bounds load against the budget.** 4 workers × 50 ops = 200
  operations per iteration against a fixed 10s. Porcupine's search is
  worst-case exponential in per-partition history length, and the only thing
  keeping it tractable is partitioning — which the four unreached models do
  not do (`singleHistoryPartition` puts all 200 in one partition). Under CPU
  pressure the same correct subject can move from pass to a hard failure.
  Either bound `Workers × OpsPerWorker` against `Timeout`, or shrink on
  `Unknown` rather than failing.
- **No visualization on `Unknown`.** `writeVisualization` runs only on the
  `Illegal` branch, so the verdict most needing diagnosis produces the
  fewest artifacts.
- **Doc contradicts code.** `runner.go:100` says *"Zero means unlimited"*;
  `concurrent.go:38` sets zero to 10s. Unlimited is unreachable except by
  passing a negative duration.
- **A failure is not reproducible from its seed.** The seed fixes the
  pre-drawn inputs; the interleaving comes from the Go scheduler. Replaying
  reproduces the operation schedule, not the goroutine schedule.

## Use rapid harder

The engine uses rapid well. The generated code does not use these:

- **`Generator.Example(seed)`** — this *is* the canonical draw RFC-0004
  invented. It works on any generator, including reflected ones, so it
  derives a fixed value for types the generator cannot write a literal for.
  That is the direct fix for zero-struct fixtures.
- **`Make[V]()` and `MakeCustom{Fields:...}`** — reflection-based generation
  with per-field overrides, which maps exactly onto the role partition:
  reflect the struct, override the role fields with their pools.
- **`SyncTest`** — already re-exported, zero consumers. Deterministic
  goroutine scheduling and fake time, and a cheaper route than Porcupine for
  concurrency claims that do not need a linearizability verdict.
- **`Filter`, `SliceOfDistinct`, `Permutation`** — richer pools the corpus
  visibly needs.

One gap to know about: rapid has no statistics or classification API. It
will not tell you what fraction of draws reached the interesting branch.
That is why `FireRate` has to be wired rather than borrowed.

## An amendment RFC-0004 needs

`LawsOnly`'s doc states a problem the RFC does not account for:

> The two oracles compete: the actions compare every call against the
> reference and abort on divergence, so a subject broken on purpose almost
> always dies of the differential at step 0 and the laws are never reached.

RFC-0004 sells `.Oracle()` as raising the floor. It does — and it also
**masks every law behind it**, because the differential fires first. The
engine already knew this and has a flag. The contract needs the same
separation, or laws become decorative exactly when a real reference exists.

## The fragile thing to leave alone

`taggedLaw` forwards `Reset`, `BindTrace`, and `CheckWithStep`; `Isolated`
is a marker rather than a behaviour, so it needs a *second* wrapper type
chosen at tag time — otherwise a forwarding method would make every tagged
law isolated. Roughly 40 lines of bug-derived design, guarded by a
reflective gate test.

Any future decorator — a metrics wrapper, a skip filter, a retry shim —
silently drops all four unless it repeats this dance. If the eleven
self-contained laws move out (revision 1 above), the `Isolated` half of this
problem disappears with them.

## Open work

The support-layer survey did not finish: `engine/model/action`, `timeaware`,
`history`, and `core/{trace,failure,coverage,equivalence,factory,visualize}`.
The specific questions still unanswered are whether `action`'s constructors
add anything over a plain `Repeat` map entry beyond the comparison wrapper,
and whether `core/equivalence` and `core/factory` have any consumer at all.
