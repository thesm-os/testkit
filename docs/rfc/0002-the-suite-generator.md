---
rfc: 0002
title: The suite generator
status: Draft
date: 2026-08-07
---

# RFC-0002: The suite generator

## Summary

`suite` reads a Go interface and emits a tier-1 conformance harness into the
consumer's own package: one assertion per classification the annotator stamped,
a benchmark harness over the same projection, and a typed extension point per
method so a team builds the rest of its suite on top of the generated one rather
than beside it.

[ADR-0017](../adr/0017-every-classification-owes-an-assertion.md) fixes the
obligation — every classification owes an assertion. This RFC is the design that
discharges it: what the generator reads, how an assertion is selected, what the
emitted file contains, and how each assertion is shown to be capable of failing.

## Problem

A generated conformance harness has to answer four questions, and getting any of
them wrong makes the output worse than nothing.

**What does it assert?** The vocabulary is seventy-two classifications on three
orthogonal axes, arriving as metadata on a callable. Nothing in a stamp says what
to assert about it.

**Can each assertion fail?** An assertion that cannot fail raises coverage and
certifies nothing, so the gap it leaves is invisible precisely where someone went
looking for it. Reading generated code does not settle this; only running it
against a subject that violates the property does.

**What does a consumer add, and what do they need to know first?** The generated
assertions cover what a classification implies. Everything specific to the
subject is the consumer's, and the shape of the extension point decides how much
of the generator's internal vocabulary they have to learn before writing one
line.

**What does it cost to wire up?** A harness that needs a page of configuration
per interface before it asserts anything real will be configured once, for the
first interface, and skipped thereafter.

## Design

### What the generator reads

`suite` is a generator-phase plugin. It reads source through `ctx.Reader`, reads
the stamps eidos's shape annotator left on each callable, and appends emit
values. It computes no output paths — layout owns routing — and it declares no
classification vocabulary of its own
([ADR-0004](../adr/0004-consume-only-the-annotator-plugin.md)).

```
iface.go ──frontend──▶ node.Interface, node.Method, node.TypeRef
         ──annotator─▶ shape, shape.key_type, shape.value_type
                       shape.mixins, shape.mixin.<name>.<param>
                       shape.contracts, shape.contract.<name>.role
                                        shape.contract.<name>.partner.<role>
         ──suite─────▶ per-method projection, assertion selection
         ──layout────▶ target directory, filename, package
         ──backend───▶ the rendered file
```

Placement is `sdk.GeneratorComposition`, one bucket after `stub`'s
`GeneratorFoundation`, and it declares `Requires: [stub]`. That ordering is what
lets it read stub's queued emit values to learn the double's constructor and
option identifiers rather than re-deriving them; two derivations of one
identifier are two chances to name a symbol stub never emitted.

### The directive

`//testkit:suite` on an interface, taking no positional argument and denying
negation: a suite exists where one is declared, so deleting the line is the
suppression and a negated form has nothing to act on
([ADR-0016](../adr/0016-directives-are-positive-only.md)). It is the plugin's
only directive — the classification vocabulary that decides *what* is asserted
belongs to the annotator, and re-declaring any of it here would be free to drift
from the registries the corpus gate measures against.

One key. `bench=off` suppresses the benchmark harness for a subject whose cost is
not a contract, leaving the assertions; the benchmark half is otherwise emitted
because it derives from the same projection at no additional analysis.

A second directive, `//testkit:bench`, carries per-method budgets. It is
method-scoped because a budget is a property of one hot path, and it batches its
properties onto one line rather than spending a directive name on each
([RFC-0001](0001-testkit-as-a-generator-platform.md)):

```go
//testkit:bench allocs=0 p99=500us mean=100us mem=4KiB
```

| Key | Gates | Backed by |
|---|---|---|
| `allocs=N` | allocations per operation; `0` is an alloc-free hot path | `Contract.AllocsMax` |
| `p99=D` | 99th-percentile nanoseconds per operation | `Contract.LatencyMax` |
| `mean=D` | mean nanoseconds per operation | a new `Contract` method; reported today, not gated |
| `mem=B` | bytes allocated per operation | a new `Contract` method over `MemStats.TotalAlloc` |

Without the directive a method's benchmark measures and reports; with it, each
key present becomes a ceiling that fails the run. A budget nobody declared is a
number the generator invented, so there is no default ceiling.

`p99` rather than `latency` names what `Contract.LatencyMax` actually gates —
`reference/generators/bench.md` describes it as a mean ceiling, and the code
compares the 99th percentile. `mem` reads `MemStats.TotalAlloc` beside the
`Mallocs` reading `allocs` already takes, so the two share one stop-the-world
read.

The directive is independent of `//testkit:stub`. Where both are present the
generated suite additionally runs the subject through the double, which is what
proves the double faithful; where only `suite` is present that half is not
emitted, and the header says so.

Everything else the generator needs it reads: the shape stamps for what to
assert, `//testkit:out` and `pkg=` for where the file lands, and the source
signature for the rest.

### Worked example: source to output

The fixture is `conformance/corpus/iface/mixin/validates`. Everything below is in
the repo except the `//testkit:suite` line, which is what this design adds to
every fixture that is to carry a suite:

```go
// Payload is what Validate accepts or refuses.
type Payload struct{ Key, Body string }

//testkit:out validatestest/ pkg=validatestest
//testkit:stub
//testkit:suite
type Mixed interface {
    // Store rejects what Validate refuses.
    //testkit:mixin validates fn=Validate
    Store(ctx context.Context, v Payload) error

    // Validate is the predicate fn names.
    Validate(v Payload) error

    // Read proves a rejected value was not stored.
    Read(ctx context.Context, key string) (Payload, error)
}
```

What the annotator stamps on it, as produced by a run of the pipeline
`conformance/gate` assembles:

| Method | `shape` | `key_type` | `value_type` | `mixins` | params |
|---|---|---|---|---|---|
| `Store` | `writer` | — | `…validates.Payload` | `[validates]` | `mixin.validates.fn = Validate` |
| `Validate` | `writer` | — | `…validates.Payload` | `[]` | — |
| `Read` | `reader` | `string` | `…validates.Payload` | `[]` | — |

Two things in that table are load-bearing and neither is obvious from the source.
`Validate` is classified `writer`, not left unclassified — the writer detector
matches `func(V) error` whether or not a context leads. And `mixin.validates.fn`
holds the bare source name, which the generator turns into a call on the same
subject.

Selection is a function of the stamp and the signature, and nothing else:

| Method | Rule | Assertion |
|---|---|---|
| `Store` | mixin `validates`, partner `Validate` | what the validator rejects, the method rejects |
| `Store` | `writer` with a context parameter and an error return | a cancelled context is reported |
| `Validate` | `writer`, but no context parameter | none — the shape's assertion needs a context to cancel |
| `Read` | `reader` with a context parameter and an error return | a cancelled context is reported |
| `Read` | `reader`, error return | an error is accompanied by the zero value |

The emitted primary file, `validatestest/iface_suite.gen.go`. Proposed:

```go
// MixedStoreCheck is one assertion about Store. Every check this file generates
// for Store is one, so a consumer's own composes with them, runs standalone, or
// replaces one.
type MixedStoreCheck func(tb testing.TB, subject validates.Mixed, v validates.Payload)

// AssertMixedStoreValidates asserts Store refuses what Validate refuses.
//
// Fails when: Store accepts a value its own validator rejects. Also fails when
// the value handed in is one Validate accepts — the assertion would otherwise
// pass by having nothing to test — so supply a rejected value via
// MixedFixtureOf.
//
// This is the direct form of the mixin. It witnesses one value; that every
// rejected value is refused is the model tier's claim, not this one's.
func AssertMixedStoreValidates(tb testing.TB, subject validates.Mixed, v validates.Payload) {
    tb.Helper()
    if err := subject.Validate(v); err == nil {
        tb.Fatalf("Validate accepted the value this check treats as invalid; " +
            "supply one it rejects via MixedFixtureOf")
    }
    if err := subject.Store(tb.Context(), v); err == nil {
        tb.Errorf("Store accepted a value Validate rejects")
    }
}

// AssertMixedContract runs every generated assertion against implementations
// produced by factory.
//
//    Checks:  4 across 2 of 3 methods
//    Shapes:  writer (Store, Validate), reader (Read)
//    Mixins:  validates (Store → Validate)
//    Silent:  Validate — writer, but no context parameter to cancel
//    Extend:  MixedOnStore, MixedOnValidate, MixedOnRead, MixedCheck
func AssertMixedContract(t *testing.T, factory func() validates.Mixed, opts ...MixedOption) {
    t.Helper()
    cfg := newMixedConfig(opts...)
    t.Run("Store", func(t *testing.T) {
        t.Parallel()
        t.Run("validates", func(t *testing.T) {
            t.Parallel()
            AssertMixedStoreValidates(t, factory(), cfg.fixture.Invalid)
        })
        t.Run("reports a cancelled context", func(t *testing.T) { /* … */ })
    })
    t.Run("Read", func(t *testing.T) { /* … */ })
    for _, c := range cfg.extra {
        t.Run(c.name, func(t *testing.T) { t.Parallel(); c.fn(t, factory()) })
    }
}
```

The subtest for a classification is named for the classification —
[ADR-0015](../adr/0015-subtest-names-carry-the-classification.md) — and the one
for a structural property is named descriptively, which that record carves out.

And the wiring a developer writes:

```go
func TestMixedContract(t *testing.T) {
    t.Parallel()
    validatestest.AssertMixedContract(t, validates.NewInMemory)
}
```

### The assertion catalogue

Seventy-two classifications, one corpus directory each. `‡` marks an assertion
materially weaker than the model tier's form for the same classification, which
obliges the generated documentation to say so. `†` marks one whose failure is
only observable under `-race`.

**Detectors (20).** The assertion follows from the signature. Every shape with a
context parameter and an error return carries *reports a cancelled context*; the
rows below add what is specific to the shape.

| Detector | Specific assertion |
|---|---|
| `reader` | an error is accompanied by the zero value |
| `readernoerror` | an unknown key yields the zero value |
| `readerwithbool` | `ok == false` is accompanied by the zero value |
| `lookup` | `ok == false` is accompanied by zero values in both slots |
| `pointerreader` | a nil pointer is accompanied by an error |
| `multireader` | an error is accompanied by zero values in every slot |
| `batchreader` | one result per key requested, in order |
| `writer`, `compositewriter`, `multiargwriter` | cancellation only |
| `mutator` | a sample value does not panic ‡ |
| `deleter` | cancellation only |
| `aggregator`, `multiaggregator` | cancellation only |
| `streamreader` | the sequence honours `break`, and drains without yielding an error |
| `streamconsumer` | cancellation only |
| `lifecycle` | cancellation only |
| `voidlifecycle` | a call does not panic ‡ |
| `pure` | a call does not panic ‡ |
| `predicate` | a call does not panic ‡ |
| `poisonaccessor` | a freshly constructed subject reports no error |

**Mixins (28).** The directive states the law; the assertion is its direct form.

| Mixin | Assertion |
|---|---|
| `atomic` | after a failing call, observable state equals a fresh subject's |
| `bounded` | the result lies within the declared bound |
| `cacheable` | two reads of one key agree ‡ |
| `concurrent` | concurrent callers do not race † |
| `concurrentreaders` | concurrent readers do not race † |
| `crdtmerge` | two replicas merged in opposite orders converge |
| `deleteremoves` | a read after delete reports not-found |
| `deprecated` | the call still works — deprecation is not removal |
| `errors` | the declared sentinel is returned for the miss input |
| `eventually` | the observation converges within the deadline |
| `hooks` | a registered callback fires |
| `idempotent` | the second call does not newly fail ‡ |
| `integrationonly` | assertions are emitted behind the integration guard, so an unset environment yields no subtest rather than a passing one |
| `lifecycleafterclose` | a read after close reports closed |
| `monotonic` | two samples do not decrease ‡ |
| `nilsafe` | zero-value inputs do not panic |
| `orderafter` | calling before the prerequisite fails |
| `partition` | writes to two partitions do not interfere ‡ |
| `pure` | two independently constructed subjects agree |
| `readafterwrite` | a write is visible to the named reader |
| `retrysucceeds` | the call succeeds within the declared attempts |
| `sample` | the named builder produces a value the method accepts |
| `scope` | an unauthorised call is refused and an authorised one is not |
| `sideeffect` | the named observation changes across the call |
| `streamreflectsmutations` | a written value appears in the stream |
| `timeout` | the call completes within the declared budget |
| `validates` | what the named validator rejects, the method rejects |
| `wrappedvia` | the returned error wraps the named target |

**Contracts (24).** Roles and partners come from the stamps; the assertion is the
protocol's minimal round trip.

| Contract | Assertion |
|---|---|
| `appender` | appending grows the sequence |
| `batch-writer` | a batch write succeeds and reports per-item outcomes |
| `cache` | a miss populates the backing store |
| `cas` | a stale version is refused |
| `circuit-breaker` | the breaker opens after the declared failures |
| `cursor` | advancing after close fails |
| `if-absent` | a second write for one key is refused |
| `if-match` | a non-matching predicate refuses the write |
| `leader-election` | after campaign the subject leads; after resign it does not |
| `lease` | acquire, release, acquire succeeds; acquire twice fails |
| `outbox` | an appended message reaches the subscriber |
| `pagination` | the cursor advances and terminates |
| `persister` | a written value is readable |
| `pool` | a get after a put yields an item |
| `publisher` | a published message reaches a subscriber |
| `rate-limit` | exceeding the declared burst is refused |
| `saga` | a failed step runs its compensation |
| `singleflight` | concurrent identical calls collapse to one † |
| `transaction` | the call leaves no partial effect on failure |
| `tx` | begin then commit persists; begin then rollback discards |
| `updater` | an update is visible to the named reader |
| `upserter` | a second upsert wins |
| `watcher` | a trigger reaches the watcher |
| `workflow` | a declared transition is permitted and an undeclared one refused |

Fifteen of the twenty-four contracts name a partner callable, which the shape
resolver rewrites from a source name into a qualified one and back-stamps onto
the partner. Both halves are prerequisites for contract assertions and neither is
available today — see Open questions.

### Assertions and extensions are the same type

Per method, one named type over the failure-recording interface and the method's
own concrete parameters:

```go
type MixedStoreCheck func(tb testing.TB, subject validates.Mixed, v validates.Payload)
```

Every generated assertion for that method is a value of it, and so is a
consumer's, so they compose and reorder and each can run standalone. The
parameters are concrete — `validates.Payload`, not a type parameter — which is
what generating into the consumer's package buys
([ADR-0012](../adr/0012-generate-per-shape-helpers-into-the-consumer.md)). An
assertion in a shipped package cannot name the method it calls and has to erase
it behind a closure and type parameters; one generated beside the subject does
not.

A method the generator has nothing to assert about gets the same treatment.
Having no property to assert does not make the signature unknown:

```go
type MixedValidateCheck func(tb testing.TB, subject validates.Mixed, v validates.Payload)
```

`testing.TB` rather than `*testing.T` is what makes an assertion drivable by a
stand-in, which is what the next section rests on — and it applies to the
consumer's assertions as much as the generated ones.

### One fixture, derived, overridable

Every assertion needs inputs. They are derived from the parameter's name and type
through `generator/internal/samples`, collected in one struct, and threaded
through assertions and benchmarks alike:

```go
type MixedFixture struct {
    Key     string           // "test-key"
    Missing string           // "other-key" — derived to differ from Key
    Payload validates.Payload
    Invalid validates.Payload // the zero value
}
```

One override point rather than one per assertion, and the miss key is derived to
differ from the hit key rather than trusted to. A suite runs with no options at
all; `MixedFixtureOf` replaces what the derivation cannot know.

### Outputs

| Tag | Suffix | Contents |
|---|---|---|
| `""` | `_suite.gen.go` | contract entry point, benchmark harness, per-method assertions and check types, fixture, options |
| `test` | `_suite.gen_test.go` | per assertion, a complying subject and a violating one |

Suffixes follow `<source-basename>_<generator>.gen.go` and
`<source-basename>_<generator>.gen_test.go`, which `stub` and `builder` use and
`reference/layout.md` documents. `enum` and `sentinel` deviate from it.

Benchmarks share the primary file because they share the projection: one plugin,
one analysis, `AssertMixedContract` and `BenchmarkMixedContract` as siblings.
They assert nothing until a ceiling is supplied, because a budget nobody declared
is a number the generator invented.

### How an assertion is shown to fail

An assertion is emitted as a body over `testing.TB` and a benchmark as a body
over `testkit.BenchTB`, each with its `t.Run` or `b.Run` wrapper separate and
taking the concrete type. The interfaces are what a stand-in can satisfy;
`*testing.T` and `*testing.B` are not.

So the generated test file drives every assertion twice — against a subject
configured to comply, and one configured to violate — and asserts the second
fails. Proposed:

```go
func TestMixedStoreValidatesFails(t *testing.T) {
    t.Parallel()
    s := validatestest.NewMixedStub(t)
    s.OnValidate.Returns(testkit.TestError("invalid")) // rejects…
    s.OnStore.Returns(nil)                             // …and stores it anyway

    f := testkit.NewFailableTB()
    validatestest.AssertMixedStoreValidates(f, s, validates.Payload{})
    testkit.True(t, f.Failed(),
        "the validates check must reject a Store that accepts what Validate refuses")
}
```

The violating subject is a configuration of the generated double rather than a
hand-written twin, so the proof accompanies the assertion instead of trailing it
across seventy fixtures, and it is one property away from the complying subject.

This is the mechanical form of ADR-0015's requirement that a broken fixture fail
the classification it claims to: calling one exported assertion directly is
failure identity, with no name matching in between.

One stand-in serves both halves. `testkit.FailableTB` satisfies `testing.TB` and
`testkit.BenchTB`: `Loop` returns true a bounded number of times, `ReportMetric`
records rather than prints, and a violated ceiling lands in `Msg` instead of
failing the run. `contract.go` states that `BenchTB` exists so the machinery can
be tested without a real benchmark harness, and this is what makes that true
from outside the package as well as inside it.

Two assertions in a benchmark's shape are not symmetric. That an allocating body
violates `AllocsMax(0)` is robust; that a non-allocating one satisfies it is not,
because `runtime.ReadMemStats` reads process-wide counters and a parallel test
contributes to them. The generated demonstration asserts the first direction
only.

### Composition and generics

The axes are orthogonal, so a method carrying a detector, a mixin and a contract
owes all three assertions in one file without collision. The four fixtures under
`corpus/iface/composite` — `batched-mixins`, `leased-idempotent-writer`,
`paginated-reader`, `tx-with-retry` — are where that is proven.

The nine fixtures under `corpus/iface/lang` prove the emission survives Go's type
system rather than any classification: embedding, foreign embedding, generics,
opaque constraints, named returns, multiple returns, variadics, and a method with
no context. A generic interface's entry point carries the type parameters; the
generated self-check instantiates at the witnesses the source names, because a Go
test function cannot take type parameters.

### The model tier attaches to the same entry point

Tiers 2–3 attach as an option rather than as a second harness, so a team runs one
conformance suite:

```go
validatestest.AssertMixedContract(t, factory,
    validatestest.MixedModelChecks(newReference),
    validatestest.MixedOnStore(myOwnCheck),
)
```

The alternative — the model generator declaring suite's output suffix so both
compose the same target and render into one file — works, but couples two plugins
through a constant that must move in lockstep, and a plugin holding two roles
sorts ahead of every generator regardless of what it declares it requires, so its
contributions render above suite's. The option form has neither problem and stays
reversible.

## What is decided

| ADR | Decision |
|---|---|
| [0004](../adr/0004-consume-only-the-annotator-plugin.md) | Consume only eidos's annotator plugin |
| [0012](../adr/0012-generate-per-shape-helpers-into-the-consumer.md) | Generate per-shape helpers into the consumer |
| [0015](../adr/0015-subtest-names-carry-the-classification.md) | Subtest names carry the classification |
| [0017](../adr/0017-every-classification-owes-an-assertion.md) | Every classification owes an assertion |

## Open questions

- **The shape resolver is not registered.** `generator.Annotators()` registers
  `shape.New()` but not `shape.Resolver()`, which runs one priority bucket later.
  Without it, contract partner references keep their raw source names, no
  back-stamping occurs — so only the declaring side of a contract carries the
  membership — and unknown roles and unresolvable partners are never diagnosed.
  Every contract assertion depends on both. `gate.Compare` cannot see the gap,
  because it measures which classifications were stamped and the declaring side
  stamps one. Registering the resolver also changes what `generator/stub` reads
  from `mixin.orderafter.fn`, from a bare name to a qualified one.
- **Concurrency assertions need `-race` to fail.** Three classifications carry
  `†`. `make check` runs `mod`, `lint`, `test`, `coverage` and `branch`; `test
  race` is a separate target. Either it joins the check stages or those three
  assertions are decoration in the default gate.
- **Whether allocation budgets should be a directive.** Benchmarks measure until
  a ceiling is supplied as an option. A directive carrying budgets the way
  `bounded` carries value bounds would make them part of the source contract, and
  is what separates tier 4 as measurement from tier 4 as a gate.
- **Several assertions need inputs no stamp describes.** `scope` needs an
  authorised context and an unauthorised sentinel; `cas` needs a stale version;
  `if-match` needs a failing predicate. The fixture struct is assumed sufficient.
  Where it is not, the assertion has to fail loudly on an unsupplied input rather
  than pass vacuously, which makes it a check on the wiring as much as on the
  subject.
