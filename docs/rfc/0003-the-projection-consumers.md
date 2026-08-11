---
rfc: 0003
title: "The projection consumers: bench, fuzz, model"
status: Draft
date: 2026-08-11
---

# RFC-0003: The projection consumers: bench, fuzz, model

## Summary

Three generators read the projection `suite` queues and emit what the harness
cannot honestly carry. `bench` turns a `//testkit:bench` method into a measured
loop holding the budgets the directive declares. `fuzz` turns a `//testkit:fuzz`
method into a seeded target — the only mode that witnesses many values. `model`
binds the classifications [ADR-0018](../adr/0018-one-tier-owns-each-classification.md)
assigned to the model tier onto the shipped `engine/model` runtime: per-interface
law bindings, an action set, a derived reference, a linearizability
configuration, and a mutation self-check that proves the bound laws can kill
injected bugs.

[RFC-0002](0002-the-suite-generator.md) fixed their positions — separate plugins
one bucket after `suite`, reading its queued value, never the source it derived
from. This RFC is the design that makes each implementable: what the projection
guarantees its readers, what each plugin emits, and where each runtime surface
it binds to already exists. The consumer surface it lands on is one init-time
registration per interface — the generated packages carry their own
`Benchmark*` and `Fuzz*` entry points, and no consumer ever writes a
per-method shim.

## Problem

**The projection is a cross-plugin API in fact, with no stated rules.**
`generator/fault` reads `stub`'s queued value through `sdk.PendingByOrigin`, and
`suite` reads the same value for its double. Three more readers arrive with this
RFC. Nothing says what a reader may assume about `suite.Contract`'s fields, what
`suite` may change without walking three consumers, or which facts a satellite
must take from the projection versus derive itself. The failure mode is the one
RFC-0002 named: two derivations of one thing, disagreeing silently — a benchmark
seeded differently from the assertion it mirrors.

**The model runtime names a generator that does not exist.** `engine/model` is
eleven subpackages built against a generator contract nobody wrote down. The
docblocks are explicit: `Registry` is "populated by the generator with
auto-derived laws"; `model/action`'s package doc says "the generator emits one
call per detected method"; `model/domhint` describes "the model generator's
analysis" and a `//testkit:domain-gen` directive no plugin declares. The runtime
shipped; its counterpart is a docblock promise.

**ADR-0018's floor is unenforceable until the bindings exist.** The ADR assigns
every law-backed classification to the model tier, and the gate fails a
classification no tier asserts. `engine/model/law` implements the properties,
but nothing binds `law.ReadAfterWrite[T, K, V]` to a concrete interface. The
binding is per-interface, concrete, and derivable from exactly the stamps
`suite` already reads — generator work, unwritten. Until it is written, the
model tier's ownership of fifty-odd classifications is a table in RFC-0002, not
an assertion anywhere.

**The reference pages describe the previous architecture.**
`reference/generators/bench.md` documents `//testkit:latency` and
`//testkit:percentiles`, neither of which exists; `model.md` documents the
taxonomy the shape detectors replaced; no `fuzz.md` exists. A reviewer today
cannot tell the designed surface from the remembered one — which is the
condition this RFC exists to end.

## Design

### Three plugins, one bucket, one read path

`bench`, `fuzz` and `model` are generator-phase plugins at
`sdk.GeneratorCrossCutting`, one bucket after `suite`'s
`GeneratorComposition`. The cross-bucket ordering is what guarantees `suite`'s
value is queued before any of them runs; `Requires: [suite]` is declared for the
record and orders nothing, exactly as RFC-0002 warned.

All three read the projection the same way `fault` reads `stub`'s:

```go
for origin, c := range sdk.PendingByOrigin[*suite.Contract](ctx.Store.Emit()) {
    // c is the value suite queued for origin — or absent, and then
    // this plugin emits nothing for that interface.
}
```

The type assertion inside `PendingByOrigin` is the orphan-file guard: a
satellite emits only where `suite` produced, so no fuzz target ever hangs off an
interface with no harness. Each plugin owns its outputs — no shared suffixes,
so `-run`, `-bench` and `-fuzz` scoping falls out of the file layout instead of
being fought.

Each plugin also reads two things from the source, through `ctx.Reader` so the
reads land in its cache key: its **own directive stamps** (`//testkit:bench`,
`//testkit:fuzz`, `//testkit:domain-gen` — each declared by the plugin that
reads it), and the **shape stamps**, which are the annotator's public
vocabulary and every generator's to read
([ADR-0004](../adr/0004-consume-only-the-annotator-plugin.md)).

### The projection is a module-internal contract

RFC-0002 said the projection is generator-internal, and this RFC does not
reverse that: nothing resembling `suite.Contract` ever appears in a consumer's
repository. What changes is its standing *inside* the generator module. It is
now the shared derivation four plugins depend on, and it gets the rules of one.

```go
// generator/suite — the value queued once per interface. Field lists
// abbreviated; suite.go carries the full docblocks.
type Contract struct {
    sdk.BaseEmit
    Subject   Subject  // IfaceName, IfaceRef, Runtime, TypeParams, TypeArgs
    EntryName string   // Assert<Iface>Contract
    Fixture   Fixture  // TypeName, CtorName, Fields []FixtureField
    Seed      *Seed    // the writer the harness populates through; nil without one
    Double    *Double  // TypeName, CtorName, DelegateToName; nil without //testkit:stub
    Methods   []Method // *golang.Sig, CheckType, ArgFields, Checks
}

type FixtureField struct {
    Name           string        // exported field on <Iface>Fixture
    Type           sdk.Ref
    Sample, Other  golang.Sample // the derived pair; empty for a composed struct
    Parts          []FixturePart // per-field pair for a struct parameter
    Variadic       bool
    Companion      *sdk.Expr     // <Type>Defaults(), where the source declares one
}
```

**One derivation per fact.** The annotator owns classifications. `suite` owns
the derived inputs, the seed choice, and every generated identifier. Each
satellite owns its own directive. A satellite that re-derives a fact the
projection carries has reintroduced the disagreement this rule exists to
prevent — the concrete case being a fuzz corpus seeded with values the checks
never used, so a corpus finding cannot be replayed against the harness.

| Reader | Takes from the projection | Takes from source |
|---|---|---|
| `bench` | `Subject`, `Fixture` (loop arguments), `Seed`, `Double`, `Method.Sig` | `//testkit:bench` stamps |
| `fuzz` | `Subject`, `Fixture` (`Sample`/`Other` → corpus), `Method.Sig` | `//testkit:fuzz` stamps, `validates`/`nilsafe`/`pure`/`bounded` stamps for the body |
| `model` | `Subject`, `Fixture`, `Seed`, `Methods` (naming, signatures) | shape stamps (law selection is the model tier's own read) |

`model` takes classifications from the source rather than the projection
deliberately: the projection's `Check` values carry only the suite tier's
checks, and a model-owned classification produces none of those — reading
`Methods[].Checks` for `readafterwrite` would find silence and conclude
correctly-nothing. The stamps are the annotator's vocabulary; the projection is
`suite`'s derivation. Each is read where it is owned.

**Stability.** All four plugins compile in one module, so renames and removals
are the compiler's problem and cost nothing. The hazard is semantic — a field's
emptiness changing meaning, `Fixture.Fields` gaining entries a check dropped.
The rule: field semantics live in docblocks on the emit types (they already do
— `fixture.go` and `suite.go` carry them), and a semantic change to a field
walks the three consumers in the same commit. `sdk.PendingByOrigin` reads
pending values within one run; there is no serialization, no versioning, and no
cross-version compatibility to maintain. That is the entire contract, and its
smallness is the argument for keeping the projection unexported rather than
promoting it to a published IR.

### One registration; the entry points are generated

The factory is the one underivable fact, and Go's discovery contract is rigid:
`Benchmark*` and `Fuzz*` entry points must live in a `_test.go` file with
exactly `func(*testing.B)` / `func(*testing.F)` — no parameters, no injection
point. One resolution keeps every generated helper parameterised on the
factory and makes the consumer write the discovery shims — correct, and a
page of ceremony per interface that grows by one function per fuzzed method.
This design moves the fact instead of the functions: name the implementations
**once, at init time**, and every entry point becomes generatable into the
`test`-tagged outputs.

```go
// Emitted by suite into _suite.gen.go.
type MixedRegistration struct {
    Name string
    New  func() validates.Mixed
}

// RegisterMixedSubject names an implementation once, for every mode at once
// — the harness, the benchmarks, the fuzz targets and the model laws all
// read the same registry. Registering a duplicate name panics at init.
func RegisterMixedSubject(name string, factory func() validates.Mixed)

// RegisteredMixedSubjects is the read side, exported because the generated
// entry points live in the external test package.
func RegisteredMixedSubjects() []MixedRegistration
```

The registry is package-level, written only at init, read only at test run —
init ordering guarantees every registration lands before any entry point
executes, and the duplicate-name panic is `domhint`'s always-err collision
policy applied one tier down. `AssertMixedContract` merges registered
subjects with any passed through the existing `MixedSubject` option (a name
collision between the two paths is an error, not a shadowing); the generated
`Benchmark*` and `Fuzz*` entries read the registry alone, and when it is
empty they skip with the one-line instruction that fills it. A skip is the
visible form of "not wired" — the invisible form is the defect.

One further step was rejected: *generating `TestMixedContract` too*. The
registry makes it possible, but the test entry is where run-scoped options,
`t.Parallel` policy and build-tag guards live — all consumer decisions — and
a generated entry beside a consumer-written one runs the contract twice
silently. One test function per interface is the floor Go leaves, and it is
where the consumer's intent is declared.

### bench

`//testkit:bench` is method-scoped — a budget is a property of one hot path —
with its properties batched on one line. The directive's presence is the
opt-in: a bare `//testkit:bench` measures and reports, and each key present
becomes a ceiling. A budget nobody declared is a number the generator invented,
so there is no default ceiling.

```go
//testkit:bench allocs=0 p99=500us
Read(ctx context.Context, key string) (Payload, error)
```

| Key | Gates | Backed by |
|---|---|---|
| `allocs=N` | allocations per operation | `Contract.AllocsMax(uint64)` — shipped |
| `p99=D` | 99th-percentile latency per operation | `Contract.LatencyMax(time.Duration)` — shipped |
| `mean=D` | mean latency per operation | `Contract.MeanMax(time.Duration)` — **commissioned by this RFC** |
| `mem=B` | bytes allocated per operation, over `MemStats.TotalAlloc / N` | `Contract.BytesMax(uint64)` — **commissioned by this RFC** |

The runtime is `testkit.Contract`: `StartContract(tb BenchTB) *Contract`, the
chained ceilings, `Loop()`, `End()`. `BenchTB` is the subset of `testing.B` the
contract needs, and `testkit.FailableTB` satisfies it — which is how the
generated bodies are tested without a real benchmark harness, the pattern
`contract_test.go` already uses. `MeanMax` and `BytesMax` are the build order,
not an open question: two methods over state `Contract` already tracks
(`durations`, `before runtime.MemStats`), landing first with `FailableTB`
tests, so the plugin ships all four keys at once. `mem=` stays despite
overlapping `allocs=` — one large allocation and many small ones are different
regressions, and each ceiling names one.

Per annotated method, one exported measured-loop body into `_bench.gen.go` —
the manual path, and the function the generated entry calls:

```go
// Generated. The subject is seeded through its own writer before the loop —
// the same seed the harness uses — so a reader measures a non-empty subject
// on the hit path. No zero values: Read(ctx, "") against an unseeded
// subject measures the miss path of an empty store, a real number about
// nothing. The loop arguments are fixture fields.
func BenchmarkMixedRead(b *testing.B, factory func() validates.Mixed) {
    subject := factory()
    fx := DefaultMixedFixture()
    if err := subject.Store(b.Context(), fx.V); err != nil {
        b.Fatalf("seed: %v", err)
    }
    c := testkit.StartContract(b).AllocsMax(0).LatencyMax(500 * time.Microsecond)
    for c.Loop() {
        _, _ = subject.Read(b.Context(), fx.Key)
    }
    c.End()
}
```

And into `_bench.gen_test.go` — the `test`-tagged output, external test
package — the entry point `go test -bench` discovers, ranging the registry:

```go
// Generated. One entry for the whole contract; subjects and methods are
// sub-benchmarks, so -bench 'MixedContract/in-memory/Read' scopes to one.
func BenchmarkMixedContract(b *testing.B) {
    subjects := validatestest.RegisteredMixedSubjects()
    if len(subjects) == 0 {
        b.Skip("no subjects registered: call validatestest.RegisterMixedSubject in an init")
    }
    for _, s := range subjects {
        b.Run(s.Name+"/Read", func(b *testing.B) {
            validatestest.BenchmarkMixedRead(b, s.New)
        })
    }
}
```

The loop arguments are the projection's fixture fields — `fx.Key` here is the
value `AssertMixedReadSmoke` was handed, so a budget regression and an
assertion failure point at the same call with the same input.

There is no per-method option surface. An entry point that owns the `b.Run`
tree invites per-method plug-in options over shipped generic contexts —
`bench.Reader[Service, string, Item]` and its dozen siblings — which
re-import the type-parameter erasure
[ADR-0012](../adr/0012-generate-per-shape-helpers-into-the-consumer.md)
removed. With the entry generated and the bodies exported, a custom
benchmark is a plain `Benchmark*` function in the consumer's own file: Go
already provides the extension point, and generated surface that duplicates
one is cut.

The measured subject is the consumer's implementation; a stub answers from a
table and benchmarking it prices nothing. The double enters only in delegate
mode — a real implementation wrapped in the recording stand-in — and there the
generated helper constructs it with `MixedStubBenchMode()`, the option `stub`
already emits to disable call recording, so the wrapped run prices the
delegation and not the ledger. The delta between the plain and wrapped runs is
the double's overhead, measured rather than asserted.

### fuzz

`//testkit:fuzz` is method-scoped, takes no keys, and denies negation — a
target exists where one is declared, and deleting the line is the suppression.
RFC-0002 records, at full strength, why the emit-everywhere-with-`fuzz=off`
alternative lost; this RFC inherits that decision and does not reopen it.

**Feasibility is Go's constraint, checked at codegen.** `f.Fuzz` accepts
`string`, `[]byte`, `bool`, the sized and unsized integer types, and
`float32`/`float64`. An annotated method qualifies iff every non-context
parameter is one of those, or a struct whose exported fields recursively
decompose to them — `Store(ctx, Payload{Key, Body string})` fuzzes as two
strings and reconstructs the struct. Recursion through a cycle, an unexported
field, a func, a channel or an interface is infeasibility — and on an
*annotated* method that is a **diagnostic at the directive's position**, never
a silent skip. The author asked; "asked and impossible" is an error with a
line number.

**The corpus is the fixture.** Two `f.Add` seeds per target — the projection's
`Sample` and `Other` for each parameter, a composed struct contributing its
`Parts` in declaration order. The same values the checks assert with, so a
crashing input found by the fuzzer replays against the harness with no
translation, and the corpus regenerates identically with the run.

**The stamps decide the body.** Absent any of them the target still calls the
method — Go's fuzzer finds panics, hangs and out-of-memory on any exported
method, which is worth having before a single property is stated.

| Stamp | The fuzzed property |
|---|---|
| *(none)* | the call returns; no panic, no hang |
| `nilsafe` | zero-valued inputs included; still no panic |
| `pure` | same input twice, same output twice |
| `bounded` | the result stays inside the declared range |
| `validates` | what the named validator refuses, the method refuses |

The per-input body is exported into `_fuzz.gen.go`; the discoverable target is
generated into `_fuzz.gen_test.go`, ranging the registry:

```go
// _fuzz.gen.go — the property for one input against one subject.
func FuzzMixedStoreInput(t *testing.T, s validates.Mixed, key, body string) {
    v := validates.Payload{Key: key, Body: body}
    if err := s.Validate(v); err != nil {
        testkit.Error(t, s.Store(t.Context(), v),
            "Store accepted what Validate refused")
    }
}

// _fuzz.gen_test.go — the target go test -fuzz discovers. Every registered
// subject sees every input; a fresh subject per input keeps state out of
// the corpus.
func FuzzMixedStore(f *testing.F) {
    subjects := validatestest.RegisteredMixedSubjects()
    if len(subjects) == 0 {
        f.Skip("no subjects registered: call validatestest.RegisterMixedSubject in an init")
    }
    f.Add("test-key", "test-body")
    f.Add("other-key", "other-body")
    f.Fuzz(func(t *testing.T, key, body string) {
        for _, s := range subjects {
            FuzzMixedStoreInput(t, s.New(), key, body)
        }
    })
}
```

The subjects are the consumer's real implementations: a fuzz target cannot be
driven through the double, because a stub returns configured answers
regardless of input and fuzzing it proves nothing. Each subject checks its own
property per input; cross-subject *agreement* is deliberately not asserted —
two conforming implementations may legitimately differ in error text or
iteration order, and substitutability is defined by the contract's laws, not
by bytewise agreement. The growing corpus lands where Go puts it,
`testdata/fuzz/FuzzMixedStore/` beside the target — in the consumer's tree,
under their ownership; `prune` matches generated suffixes and never enters
`testdata/`.

**Why targets stay per-method.** There is no `FuzzMixedContract` because Go
has no aggregate form: `testing.F` has no sub-targets, `f.Fuzz` may be called
once per target, the mutation vector and corpus entry format are defined by
the one callback's parameter tuple, and `go test -fuzz` refuses a pattern
matching more than one target in a package. The only construction left is a
dispatcher target — union the parameters, switch on a discriminator byte —
which un-names crashes, lets minimization wander across methods, widens the
mutation space so each CPU-hour finds less, and re-couples the per-method
corpus bills the directive design deliberately separates. Benchmarks
aggregate because `b.Run` exists; fuzzing is one engine per signature, and
the design follows the tool. (The *state-space* counterpart — one fuzz target
for the whole interface's action sequences — does exist, and belongs to the
model tier: see `FuzzMixedModel` below.)

### model

#### The runtime is shipped; the generator is its missing half

`engine/model` is a property-based state-machine engine on
[pgregory.net/rapid] and Porcupine, in eleven subpackages: the runner (`model`),
shape-typed action constructors (`action`), the law catalogue (`law`),
prebuilt linearizability models (`linearize`), reference oracles (`ref`),
mutation operators (`mutation`), opaque-type generator hints (`domhint`),
clock-shaped checkers (`timeaware`), a bounded model checker (`bmc`), failure
minimization (`shrinker`), and chain-partition tracing (`history`). The
surfaces the generator binds to, verbatim:

```go
// engine/model
func Assert[T any](t rapid.TB, sutFactory func() T, opts ...Option[T])
func Property[T any](sutFactory func() T, opts ...Option[T]) func(*rapid.T)
func Check(t TB, prop func(*T))
func MakeFuzz(prop func(*T)) func(*testing.T, []byte)
func WithReference[T any](factory func() T) Option[T]
func WithLaws[T any](r *Registry[T]) Option[T]
func WithConcurrent[T any](cfg ConcurrentConfig[T]) Option[T]
func WithCoverageSink[T any](sink *coverage.ComponentCoverage) Option[T]
func NewRegistry[T any]() *Registry[T]           // Add, SkipByID, CheckAll, Coverage

// engine/model/law
type Law[T any] interface {
    ID() string     // "AUTO-READ-AFTER-WRITE"
    REQID() string  // "REQ-PKG-FOO-001", or "" for auto-derived
    Check(rt *rapid.T, sut, ref T) error  // observational; never mutates
}

// engine/model/action — one constructor per detector and per contract
func Reader[T any, K comparable, V any](
    name string,
    keys *rapid.Generator[K],
    read func(context.Context, T, K) (V, error),
) model.Action[T]
```

The division of labour was designed into these signatures. `WithLaws` exists
"to pass the pre-built auto-law registry"; `action`'s doc says the generator
emits one call per detected method; `domhint`'s doc specifies the
`//testkit:domain-gen` directive and the emitted option's name. What follows is
the counterpart those docblocks assume.

#### Emission is derived, and running is the default

`model` declares no opt-in directive. Its trigger is the ADR-0018 mapping: it
emits for every interface where `suite` queued a projection **and** at least
one declared classification maps to a law — which, because the detector rows
map too (`writer` → `AUTO-WRITE-OBSERVABLE`, `aggregator` →
`AUTO-COUNT-EQUALS-REFERENCE`), is nearly every stateful interface. The
directive already exists: it is the classification line itself, and
[ADR-0017](../adr/0017-every-classification-owes-an-assertion.md) says what a
declared classification owes. An owed assertion that is generated but off by
default is owed on paper, so the bound laws run inside the ordinary contract
entry — `AssertMixedContract` — under rapid's default iteration count, and the
existing controls apply to them unchanged: `MixedWithout("model/AUTO-…")` drops
one law for one consumer, by the same path syntax every other check uses.

RFC-0002 sketched the attachment as `ServiceModelChecks(newReference)` —
consumer-gated, because at sketch time the reference was assumed to be theirs
to write. `model/ref` is why the gate moved: the reference is derivable for
exactly the shapes the oracles cover, and the design rule of the whole platform
is that a derivation not done is a cost shifted to every consumer. What
survives of the sketch is the override, not the gate.

#### What one interface gets

**The law registry.** One binding per (declared classification → law) row of
the ADR-0018 mapping, instantiated at the interface's concrete types with the
projection's method names:

```go
// Generated into _model.gen.go.
func mixedModelLaws() *model.Registry[validates.Mixed] {
    r := model.NewRegistry[validates.Mixed]()
    r.Add(law.ReadAfterWrite[validates.Mixed, string, validates.Payload]{ /* wired */ })
    // …one Add per mapped classification the interface declares
    return r
}
```

**The action set.** One `model.Action` per method, through the `action`
constructor matching its detector or contract — `action.Reader` for a
`reader`, `action.Writer` for a `writer`, `action.CompareAndSwap` for the
`cas` role, `action.Unknown` as the explicit escape hatch for a method no
detector claims. Input generators come from the stamps: `shape.key_type
string` becomes a string generator; an opaque domain type goes through
`domhint` (below).

**The generators, derived once.** Keys draw from a small sampled set and the
value type's key field — `shape.key_type` names it — is pinned to the *same*
generator, because collisions are what make the laws fire: read-after-write
needs a read that revisits a written key, and keys from an unbounded space
never do. The derivation is emitted once and every path closes over it —
sequential, concurrent, exhaustive. The hazard of not doing so is silent and
specific: a concurrent path with its own inline generators uncouples from
the key set, readers stop revisiting what writers wrote, and the
linearizability check runs green over a history with almost no conflicts in
it. The one-derivation rule applies *within* the generated file, not only
between plugins.

**The property, as a value.** The wiring above composes into one generated
builder:

```go
// _model.gen.go — one derivation, every harness.
func mixedModelProperty(factory func() validates.Mixed,
    opts ...MixedModelOption) func(*rapid.T) {
    // actions, laws, reference, generators — assembled here, once
    return model.Property(factory, /* assembled options */)
}
```

The contract extension runs it through `model.Check`; the state-space fuzz
target below runs it through `model.MakeFuzz`; the concurrent path shares
its generators. One derivation, every harness. The builder is also exported
as `MixedModelProperty`: it is the composition point a consumer with a
bespoke harness needs, and hiding it forces them to rebuild the assembly by
hand.

**The state-space fuzz target.** `_model.gen_test.go` carries one
`Fuzz<Iface>Model` entry per interface — `f.Fuzz(model.MakeFuzz(prop))` over
each registered subject. This is the second fuzz family, orthogonal to the
`fuzz` plugin's: `FuzzMixedStore` explores one method's *input* space through
decomposed parameters, while `FuzzMixedModel` lets the coverage-guided engine
drive rapid's choice stream and explore the interface's *action-sequence*
space — the fuzzer hunting for the ordering that breaks a law rather than the
input that breaks a method. One target per interface, so Go's
one-target-per-`-fuzz` constraint never bites.

**The reference.** Three tiers, matching the runtime's own adoption ladder —
and armed **per method**, not per interface. An oracle rarely models every
method an interface declares, and making one uncovered accessor force the
whole reference onto the consumer is the all-or-nothing this dissolves:

- *Derived.* Where the interface's shape maps to a shipped oracle, the
  generator emits an unexported adapter implementing the **full interface**
  over it: the reader/writer/deleter family delegates to
  `ref.MapStore[K, V]`, `cas` to `ref.AtomicCell`, `lease` to
  `ref.LeaseTracker`, `pool` to `ref.BalancedPool`, `appender` to
  `ref.AppendOnly`, `saga` to `ref.CompensatingSaga`, `singleflight` to
  `ref.Coalescer`. Methods the oracle does not model get typed inert bodies —
  and the laws that would compare the SUT against the reference *on those
  methods* are not bound, because a comparison against an inert body is a law
  that fails every correct implementation. Reference-free laws
  (`AUTO-PURE-DETERMINISTIC`, `AUTO-PREDICATE-CONSISTENT` — self-consistency
  claims) bind regardless. The oracle is shipped and tested once; the adapter
  is thirty generated lines of delegation, and the seeding move from RFC-0002
  applied one tier up.
- *Supplied.* `MixedModelReference(factory)` replaces the derived adapter and
  arms every reference-needing law — the escape hatch for an interface whose
  semantics outrun its shape.
- *Absent.* Where no oracle maps and none is supplied, every law needing a
  reference **skips visibly**, one skipped subtest per law naming its ID and
  the option that would arm it. A declared classification whose law never ran
  is a skip in the test output, counted and greppable — never silence. That is
  ADR-0017 discharged at runtime, not only at codegen.

**The concurrent path.** Where the shape matches a prebuilt Porcupine model —
`linearize.KV[K, V]` for the CRUD family, `AppendLog`, `CASCell`, `Counter`,
`Cursor`, `LeaseTable`, `Pool`, `Set` — the generator emits a
`WithConcurrent(model.ConcurrentConfig[T]{…})` wiring: the matching model, one
`ConcurrentAction` per method through the `linearize.ConcurrentReader` /
`ConcurrentWriter` / `ConcurrentDeleter` wrappers, `PartitionByDirective`
where the `partition` mixin is declared, and the remaining methods as
`StressActions` — unrecorded, present to give `-race` something to bite. The
runtime's defaults (4 workers, 50 ops, 10s Porcupine budget) stand;
`linearize.VisualizeOnFailure` writes the interactive trace on violation.

**Opaque types.** A parameter whose type `domhint.RequiresHint` reports
non-generatable resolves against the hint registry. `//testkit:domain-gen
<Type> <Func>` — declared by this plugin, the third and last directive this
RFC adds — emits the init-time `domhint.Register` call; the generator also
emits the typed per-run option `MixedModelWith<Type>Gen(gen)` the `domhint`
docs specify. An opaque parameter with neither is a **diagnostic at the
parameter**, naming the directive that fixes it. This mirrors fuzz's
feasibility rule: the generator refuses loudly where it cannot derive, because
a silently narrowed action set is a model test of a different interface.

**The chain trace.** An `appender`-shaped interface gets its actions through
the chain family — `action.ChainAppendRecording` records each successful
append into a partition-keyed `history.History`, `action.ChainVerify` and
`action.ChainReplay` cross-check the SUT against the per-partition replay —
with `WithHistoryReset` wired so the trace clears at iteration boundaries.
Partitioning follows the `partition` mixin; an unpartitioned chain uses
`K = struct{}`, which is the runtime's own no-special-casing convention.

**Saturation, measured with a derived hash.** The runner's
`WithStateHash(func(T) uint64)` and `WithSaturationThreshold` exist to detect
a run that stopped visiting new states, and the hash is derivable wherever
the interface can observe itself: fold the readers over the fixture's key
set. The generator emits that observational fold once and wires both options,
so the coverage sink reports state-space saturation instead of guessing at
it.

**The exhaustive short-sequence check.** `bmc` is bound, not shelved, because
its two consumer concerns both derive. Determinism: `bmc.Action.Apply` must
be a pure state step, and the fixture pair supplies a fixed action alphabet —
`Store(V)`, `Store(VOther)`, `Read(Key)`, `Delete(KeyOther)` — methods
crossed with derived values, no rapid draws. State identity:
`bmc.Config.StateHash` takes the same observational fold as saturation. The
generated `TestMixedModelExhaustive` in `_model.gen_test.go` threads a
`(subject, reference)` pair as the BMC state, applies each action to both,
and checks one invariant at every reached state — observable agreement — per
registered subject, at a modest default depth. Where the random path samples,
this proves absence within bounds and returns the shortest counterexample
sequence when it cannot.

**The report header.** The generated entry's docblock is a per-method table
of what the run derived — the detector or contract behind each action, every
bound law by ID, which methods the derived reference models and which run
inert, what was skipped and the exact option that arms it. The header is the
difference between "the generator did something" and "the generator did
*this*", and it is the first thing a reader checks before trusting a green
run.

**Leak checking is derived.** Where the interface carries `lifecycle` or
`voidlifecycle`, the run wraps in `model.CheckGoroutineLeaks` — an interface
declaring teardown claims cleanliness, and a leaked goroutine per iteration
is exactly what a state-machine run amplifies into visibility. An opt-in
option nobody discovers is coverage nobody gets; the shape stamp is the
opt-in, and `MixedWithout("model/goroutine-leaks")` is the exit.

**Vacuity is measured, and enforced where inputs are controlled.**
`Registry.Coverage()` returns per-law `ran` and `fired` counts, and the
generated runner reads them after every run. A law with `ran == 0` **fails**
— a law that never executed is a wiring defect, deterministically
reproducible. A law with `fired == 0` — it ran, but its precondition never
occurred — **logs** in a consumer's run, because rare preconditions are
legitimate there, and **fails in the conformance corpus**, where the
fixtures are ours and a never-firing law is a vacuous binding. The gate
reads the same counts through `WithCoverageSink`: enforcement lives where
the inputs are controlled, and the consumer gets the signal without the
flake.

#### The integration matrix

`engine/model` was built for this generator, and the test of that claim is a
row per subpackage — what the generator emits, on what trigger. Nothing is
left as a hand-assembled tool.

| Subpackage | The generator emits | Trigger |
|---|---|---|
| `model` (runner) | property builder, `Check` run, options assembly, coverage sink, artifact dir | any model-owned classification |
| `model/action` | one constructor call per method | the method's detector or contract stamp |
| `model/law` | one binding per mapped classification | the ADR-0018 mapping |
| `model/ref` | the full-interface adapter over the mapped oracle | the shape-to-oracle mapping |
| `model/linearize` | `ConcurrentConfig` with the matching prebuilt model, wrappers, stress set | shape matches a prebuilt model |
| `model/mutation` | the kill-them-all self-check | a derivable reference |
| `model/domhint` | `Register` calls and typed `With<Type>Gen` options | `//testkit:domain-gen`, opaque parameters |
| `model/history` | chain actions over a partition-keyed trace, `WithHistoryReset` | `appender` shape |
| `model/bmc` | the exhaustive short-sequence test | fixture alphabet + observational hash derivable |
| `model/shrinker` | nothing — fires runner-internally on failure | `WithArtifactDir`, always passed |
| `model/timeaware` | binding specified, blocked on one upstream stamp | the commissioned `clocked` mixin (below) |

Two runtime surfaces stay consumer-facing by design rather than omission:
`Pair`/`Triple`/`Lift*` compose *multiple interfaces* into one runner, and
which interfaces a system composes is a fact about the consumer's
architecture, not about any one declaration the generator reads; and raw
`bmc.Run` remains callable for state types that are not interfaces at all.

**`timeaware` is one stamp away.** The checkers are shipped — TTL expiry,
deadline propagation, scheduled firing, plus the `Barrier` the concurrent
path needs under clock advance (`action.AdvanceClockSynced` already takes
it). What is missing is upstream: no eidos classification names a clock
dependency, and testkit does not invent classification vocabulary
([ADR-0004](../adr/0004-consume-only-the-annotator-plugin.md)). This RFC
commissions the upstream request — a `clocked` mixin naming the clock
accessor and the time-bound parameter — and specifies the binding now:
`clocked` + a TTL-bearing store shape binds `TTLExpiryAfterAdvance`,
`clocked` + `lifecycle` binds `DeadlineRespecting`, `clocked` + a scheduled
shape binds `ScheduledFiresAfterAdvance`, and every clock-advancing action
goes through the `Barrier`. When the mixin lands, the binding is mechanical;
until then the generated header names it as the one unbound checker family.

#### How it attaches: one seam, generic, in suite's file

The model material lands in `_model.gen.go`, in the same generated package as
the harness. It reaches the entry point through one seam `suite` emits
unconditionally:

```go
// Emitted by suite, always — empty unless a sibling generated file registers.
type mixedContractExtension struct {
    name string
    run  func(t *testing.T, subjectName string, factory func() validates.Mixed,
        cfg *mixedConfig)
}

var mixedContractExtensions []mixedContractExtension
```

`_model.gen.go` appends its runner in `init()`; `AssertMixedContract` runs
every registered extension per subject, after the per-method checks. The seam
is generic — `suite` knows nothing about `model`, and a future satellite uses
the same slot. Model-specific options (`MixedModelReference`,
`MixedModelWith<Type>Gen`) are declared in model's file and store through the
opaque extension bag `mixedConfig` carries for exactly this purpose; same
package, so no export ceremony. `bench` and `fuzz` need no seam: their
generated entry points live in the `test`-tagged outputs and read the
registry directly. The seam exists for exactly one kind of material — what
must run *inside* the one contract entry, so that laws, checks and skips
report under a single `go test -run` tree.

The alternative — a standalone `AssertMixedModel` harness — is the "second
harness a team has to remember" RFC-0002 already rejected for the model tier,
and registering through suite's *file* (shared suffix, slot contribution) was
rejected there too: a dual-role plugin sorts ahead of every generator and its
contributions render above the host's. The init-time seam has neither problem
and keeps the two files independently deletable.

#### The self-check is mutation

The suite tier proves its checks can fail by driving them against a
misconfigured double. A law cannot be proven that way — its claim spans
sequences — so the model tier's proof is `engine/model/mutation`: inject a
synthetic bug, and the bound laws must kill it.

Where a reference is derivable, the generator emits `_model.gen_test.go`: the
derived adapter as the subject, a second clean adapter as the reference, and
one wrapped subject per compatible operator — `DropWrites` on the writer,
`MissDeletes` on the deleter, `ReturnWrongValue` on the reader,
`DuplicateAppends` on an appender, `LossyStream` on a stream reader,
`OffByOneIndex` on an aggregator, `RandomDelay` and `ReorderConcurrent` on the
concurrent path. `mutation.Run` drives the full law suite once per operator
through `testkit.NewFailableTB`, and the test asserts `Report.Unkilled()` is
empty. The generator wires only pairs the ADR-0018 mapping claims a bound law
witnesses, so an unkilled operator is one of exactly two defects: the binding
is wrong, or the law is weaker than the mapping says. Both are this
generator's bugs to surface, which is why the harness runs in the generated
test rather than in the consumer's suite — it certifies the emission, not
their implementation.

#### What needs no generator at all

Failure minimization is the runner's: on a failing sequence it composes
`shrinker.CausalHull` over the recorded steps, writes the
`witness-<seed>.txt` predicate, and extracts the minimal racing pair —
`history` feeds the chain laws their per-partition replays. The generator's
whole contribution is passing `WithArtifactDir`. `bmc` (exhaustive search
under a state hash) and `timeaware` (TTL, deadline and scheduled-task laws
over a controlled clock) are shipped and bindable but not bound by v1; the
open questions say why.

### Outputs

| Plugin | Tag | Suffix | Contents |
|---|---|---|---|
| `bench` | `""` | `_bench.gen.go` | per annotated method, the exported measured-loop body |
| `bench` | `test` | `_bench.gen_test.go` | the `Benchmark<Iface>Contract` entry over the registry |
| `fuzz` | `""` | `_fuzz.gen.go` | per annotated method, the exported per-input property |
| `fuzz` | `test` | `_fuzz.gen_test.go` | one `Fuzz<Iface><Method>` entry per annotated method, over the registry |
| `model` | `""` | `_model.gen.go` | property builder, law registry, action set, derived reference adapter, concurrent config, extension registration, model options |
| `model` | `test` | `_model.gen_test.go` | the mutation self-check (where a reference is derivable), `Fuzz<Iface>Model`, `Test<Iface>ModelExhaustive` |

All follow `<source-basename>_<generator>.gen.go`, so the repository's
existing patterns — licence excludes, coverage, `.gitattributes` — match with
no new entries. The `test`-tagged outputs take Layout's external-test-package
shift, which is why the registry's read side is exported; the primary outputs
hold every body, exported, so the manual path — driving one body from a
hand-written harness — costs nothing extra.

### What a consumer writes

The whole conformance wiring, for every mode at once:

```go
func init() {
    validatestest.RegisterMixedSubject("in-memory",
        func() validates.Mixed { return NewInMemory() })
}

func TestMixedContract(t *testing.T) {
    t.Parallel()
    validatestest.AssertMixedContract(t)
}
```

One registration naming the one fact the source does not hold, and one test
function declaring that the contract runs in this suite. Everything else
already exists and already discovers:

```console
go test                                    # checks, laws, mutation, exhaustive
go test -bench 'MixedContract/in-memory'   # every declared budget
go test -fuzz FuzzMixedStore               # one method's input space
go test -fuzz FuzzMixedModel               # the interface's sequence space
```

This file is two declarations, and it grows by zero as methods gain
annotations. RFC-0002's acceptance measure carries over sharpened: every
*further* line a consumer writes — a second subject, a
`MixedModelReference`, a `MixedWithout` — is either a fact the source cannot
state or a derivation not yet done, and the generated headers say which. The model tier added none of these lines: the laws run
inside `AssertMixedContract` against the derived reference, and the consumer
meets them only as subtests — or as the skips that name the option a
classification is waiting on.

**The bill, itemised.** Per interface per subject, the model tier adds one
rapid run (default iteration count) over the action set, one Porcupine check
where a linearize model matched (4×50 ops), and one bounded exhaustive
search whose cost is `|alphabet|^depth` before hash pruning — the derived
alphabet is small by construction and the default depth modest, and
`Outcome.Explored`/`Pruned` quantify what a run actually paid. Once per
generated package — against the derived adapter, not per consumer subject —
the mutation harness runs the law suite once per wired operator.
`FuzzMixedModel` costs nothing until someone points `-fuzz` at it; in plain
`go test` it replays seeds only. The generated test dependency set grows by
`pgregory.net/rapid` and `github.com/anishathalye/porcupine`, which
`engine/model` already carries. Consumers for whom any of that is too much
drop by path: `MixedWithout("model")` removes the tier per call site, and the
skip reporting keeps the removal visible in output rather than silent.

## What is decided

| ADR | Decision |
|---|---|
| [0004](../adr/0004-consume-only-the-annotator-plugin.md) | Classifications are read from the annotator's stamps, never re-declared |
| [0012](../adr/0012-generate-per-shape-helpers-into-the-consumer.md) | Bindings are generated into the consumer's package at concrete types |
| [0015](../adr/0015-subtest-names-carry-the-classification.md) | A law reports under its ID; a skipped law names what would arm it |
| [0016](../adr/0016-directives-are-positive-only.md) | `//testkit:bench`, `//testkit:fuzz`, `//testkit:domain-gen` — presence is the ask; deletion is the suppression |
| [0017](../adr/0017-every-classification-owes-an-assertion.md) | Model-owned classifications run by default and skip visibly, never silently |
| [0018](../adr/0018-one-tier-owns-each-classification.md) | The mapping in RFC-0002 is the emission trigger and the mutation-wiring source |

## What the design rests on

- **`sdk.PendingByOrigin` is proven in-repo.** `generator/fault/fault.go:465`
  and `generator/suite/suite.go:356` both read `stub`'s queued value through
  it today; the three new readers add no mechanism.
- **The runtime names this generator.** `model.Registry`: "The generator
  populates it with auto-derived laws"; `model/action`'s package doc: "the
  generator emits one call per detected method"; `model/domhint`'s package doc
  specifies `//testkit:domain-gen` and the `<Iface>ModelWith<Type>Gen` option
  shape this RFC adopts unchanged.
- **`stub` already emits `<Iface>StubBenchMode()`** — the recording-off option
  the delegate-mode benchmark constructs the double with.
- **`Contract`'s surface is `AllocsMax` and `LatencyMax`, driven through
  `BenchTB`** — which is what lets generated bench bodies be tested with
  `FailableTB`, and what makes `mean=`/`mem=` prerequisites rather than keys.
- **Law identity is stable and typed.** IDs are the `AUTO-*` strings the
  ADR-0018 tables in RFC-0002 already cite; `REQID` carries the requirement
  tag or empty for auto-derived; `Law.Check` is contractually observational,
  which is what makes running the registry inside the harness safe for the
  checks that follow it.
- **The suite's generated surface already has every hook the examples use.**
  `MixedOption`, `MixedSubject`, `MixedWithout`, `DefaultMixedFixture` and the
  fixture's `V`/`VOther`/`Key`/`KeyOther` pairing are shipped in the
  conformance corpus; the registry, the seam and the extension bag are the
  three additions `suite` owes this RFC.
- **The property pattern needs no runtime addition.** `Property`
  (`runner.go`), `Check` and `MakeFuzz` (`rapid.go`) are shipped, as are the
  chain and clock action families (`action/chain.go`, `action/clock.go`) and
  `WithStateHash`, `WithSaturationThreshold`, `WithHistoryReset`. Every
  binding in the integration matrix names surface that exists.
- **`bmc`'s two consumer concerns derive.** `Action.Apply` must be
  deterministic, and the fixture pair is a fixed action alphabet;
  `Config.StateHash` wants a fingerprint, and the observational read-back
  fold is one. `Outcome` carries `Explored`/`Pruned`, so the generated test
  can assert its bound was not vacuous.

## Questions, answered

Each question this design opened is closed in the section that owns it; this
table is the index, not the argument.

| Question | Decision | Where |
|---|---|---|
| `mean=` / `mem=` have no runtime backing | `Contract.MeanMax` and `Contract.BytesMax` are commissioned build-order: they land first, `FailableTB`-tested, and the plugin ships all four keys at once. `mem=` stays beside `allocs=` — one large allocation and many small ones are different regressions | bench |
| Where do the mappings live | Classification-to-law and shape-to-oracle are the same kind of data and live together in the conformance gate, each closed by a test: every model-owned classification names a law that registers, every derivable shape names an oracle that exists. The gate already computes the union both tests need | design; ADR-0018 |
| Can an aggregate fuzz target exist | No — no `f.Run`, one `f.Fuzz` per target, per-signature corpus, and `-fuzz` refuses multi-target patterns. Per-method targets for input space; one `Fuzz<Iface>Model` per interface for sequence space | fuzz; model |
| Does `bmc` get bound | Yes. The fixture pair is the deterministic action alphabet, the observational read-back fold is the state hash, and `Test<Iface>ModelExhaustive` proves absence within bounds per registered subject | model |
| Clock-shaped laws have no stamp | The binding is specified now and blocked on one commissioned upstream stamp — a `clocked` mixin on the annotator's side of the fence, per ADR-0004. Until it lands, the generated header names the unbound checker family | integration matrix |
| Where is vacuity enforced | `ran == 0` fails everywhere; `fired == 0` logs in consumer runs and fails in the conformance corpus, read through `WithCoverageSink`. Enforcement lives where the inputs are controlled | model |
| Do subjects get compared to each other | No. Substitutability is defined by the contract's laws, not bytewise agreement; each subject checks its own property per fuzz input | fuzz |
| Is the contract's test entry generated too | No. Run-scoped options, `t.Parallel` policy and build-tag guards are consumer decisions, and a generated entry beside a written one double-runs silently | registration |

## Deferred

- **Consumer-facing mutation runs.** The self-check kills operators against
  the derived adapter, certifying the emission. Running the same operators
  against a consumer's implementation is a different claim with an ambiguous
  failure reading — an unkilled operator there may be a weak law or a
  legitimate behavioural difference — and `mutation.EquivalenceClasses`
  exists to study exactly that. Deferred until the self-check has produced
  data worth grouping.
- **Multi-interface composition.** `Pair`/`Triple`/`Lift*` compose several
  interfaces into one runner. Which interfaces a system composes is a fact
  about the consumer's architecture, not any declaration the generator
  reads; the runtime surface stays consumer-facing. If contract partner
  stamps ever span interfaces, this is the section to reopen.
