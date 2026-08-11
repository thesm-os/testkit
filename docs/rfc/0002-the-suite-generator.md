---
rfc: 0002
title: The suite generator
status: Draft
date: 2026-08-07
---

# RFC-0002: The suite generator

## Summary

`suite` reads a Go interface and emits a conformance harness into the consumer's
own package: everything the interface's shape and classifications imply about how
an implementation must behave, derived rather than configured, with a typed
extension point for what derivation cannot reach.

Two further generators read the same projection, each opted into per method
through its own directive. `bench` turns a `//testkit:bench` operation into a
measured loop with the budgets the directive declares; `fuzz` turns a
`//testkit:fuzz` operation into a target seeded from the same values the
assertions use.

[ADR-0018](../adr/0018-one-tier-owns-each-classification.md) fixes the floor —
every classification owes an assertion in exactly one tier, and this generator
owns the cheap one. This RFC is the design that discharges that share and goes
past it: a classification is not the unit of a conformance suite, it is one
contributor to a method's family of checks, and most of that family comes from
the signature.

## Problem

A generated conformance harness has to answer five questions, and getting any of
them wrong makes the output worse than nothing.

**What does it assert?** The vocabulary is seventy-two classifications on three
orthogonal axes, arriving as metadata on a callable. Nothing in a stamp says what
to assert about it.

**Is it dense enough to mean anything?** A file that asserts one property per
declared classification, and nothing about a method that declares none, is a
smoke test wearing the word *conformance*. Substitutability is a claim about the
interface as a whole, and most of the evidence for it comes from the signature
and from sequences across methods, not from directives.

**What does the consumer have to configure before it runs?** Every value a
harness makes the author supply is a value the annotator may already know. A
harness needing a page of wiring per interface is configured once, for the first
interface, and skipped thereafter.

**Can each check fail?** A check that cannot fail raises coverage and certifies
nothing, so the gap it leaves is invisible precisely where someone went looking
for it. Reading generated code does not settle this; only running it against a
subject that violates the property does.

**What does a consumer add, and what do they need to know first?** They write
implementations and extensions, never plugins. The shape of the extension point
therefore decides the whole ceiling on extensibility, and how much of the
generator's vocabulary has to be learned before writing one line.

**What belongs here rather than in the model tier?** `engine/model/law` already
implements seventy-one properties, and their names line up with the
classification vocabulary almost row for row. A design that reimplements them as
templates ships sixty-odd properties twice, in two languages, with nothing
relating the copies.

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
                                        shape.contract.<name>.param.<key>
         ──suite─────▶ per-method projection, check selection
         ──layout────▶ target directory, filename, package
         ──backend───▶ the rendered file
```

Placement is `sdk.GeneratorComposition`, one bucket after `stub`'s
`GeneratorFoundation`.

**The projection is generator-internal and stays that way.** Three modes read it,
which is exactly the shape that tempts a design into publishing an intermediate
representation so the modes can be split further. Nothing resembling an operation
table appears in a consumer's package: they receive concrete, typed, readable Go.
A mode needing something the projection cannot give grows its own local shape
rather than widening the shared one — because every field of a leaked IR that we
later regret is a breaking change in someone else's repository.

### The directive

`//testkit:suite` on an interface, taking no positional argument and denying
negation: a suite exists where one is declared, so deleting the line is the
suppression and a negated form has nothing to act on
([ADR-0016](../adr/0016-directives-are-positive-only.md)). It is the plugin's
only directive — the classification vocabulary that decides *what* is checked
belongs to the annotator, and re-declaring any of it here would be free to drift
from the registries the corpus gate measures against.

No keys. Benchmarks and fuzz targets are opted into per method, through
`//testkit:bench` and `//testkit:fuzz`, each declared by the generator that
reads it. A budget is a property of one hot path and a corpus is a cost
somebody chose to pay, so the method is the grain — an interface-level key
here could never say "fuzz Put but not Close" — and directive ownership stays
where the house rule puts it: the plugin that reads the stamps is the plugin
that declares the schema, with no other plugin's mode riding on this one's
line. An earlier draft had suppressive `bench=off` / `fuzz=off` keys here
instead; the fuzz section records why that lost.

Everything else the generator needs it reads: the shape stamps for what to check,
`//testkit:out` and `pkg=` for where the file lands, and the source signature for
the rest.

### The unit is the method, not the classification

A method's checks come from four sources, and only the third is a classification:

1. **The signature.** A context parameter earns cancellation, deadline and
   nil-context checks. An error return earns a zero-value-on-error check. Every
   method earns a smoke call. None of this needs a directive, and it is most of
   the volume.
2. **The detector.** One stamp per callable, adding what is specific to the shape
   — a `reader` misses, a `streamreader` honours `break`, a `poisonaccessor` is
   clean when fresh.
3. **Each mixin and contract it carries.** The direct form of the declared law.
4. **The extension point**, typed, whether or not anything filled it.

And the interface as a whole earns **cross-method checks** — those of them a
fixed sequence can witness. Read-after-write, delete-removes and
lifecycle-after-close are all `engine/model/law` properties and belong to the
model tier; what stays here is the pairing the projection derives and hands to
that tier as a binding.

Making the classification the unit is what produced a first draft asserting four
things about a three-method interface. Source 1 alone owes ten of it, before
any directive is read — which is also why the tier boundary costs the suite far
less than it appears to. Handing every law-backed classification to the model
tier removes at most one check per declaration; the signature-derived family is
untouched.

### Derivation first

The design goal is that a consumer supplies nothing. Measured against the prior
art, that is achievable: every option in the previous generation's wiring is a
value the stamps already carry.

```go
// What v1 made the consumer write:
servicetest.ServiceOnGet(suite.AssertReturnsSentinel[Service, string, Item]("nonexistent", ErrNotFound)),
servicetest.ServiceOnDelete(suite.AssertDeleteSucceeds[Service, string]("seed")),
servicetest.ServiceOnCount(suite.AssertAggregatorConsistent[Service, int](3)),
servicetest.ServiceOnClose(suite.AssertLifecycleSucceeds[Service]()),
servicetest.ServiceOnList(suite.AssertStreamCompletes[Service, Item]()),
servicetest.ServiceOnErr(suite.AssertPoisonAccessorNilOnFresh[Service]()),
// …eleven in total
```

`ErrNotFound` is on the `errors` mixin stamp. `AssertLifecycleSucceeds` takes no
data at all — the `lifecycle` shape is its whole input, as it is for
`AssertStreamCompletes` and `AssertPoisonAccessorNilOnFresh`. `"nonexistent"` is
a miss key, derived to differ from the hit key. `3` is how many values the seed
wrote. Eleven lines of wiring, none of them information the run did not already
hold.

The type parameters are the second artefact of the same mistake. `[Service,
string, Item]` is erasure: the helper lived in a shipped package, could not name
the consumer's types, and made the consumer name them back. Generated beside the
subject there is nothing to erase
([ADR-0012](../adr/0012-generate-per-shape-helpers-into-the-consumer.md)), so the
same check reads:

```go
servicetest.AssertServiceGetReturnsSentinel(t, subject, "nonexistent")
```

and cannot be instantiated at the wrong types, because there are none.

### The interface seeds itself

Pre-population looks like the one thing derivation cannot reach — a reader over
an empty subject asserts nothing, and only the author knows how to fill it.

It is derivable for any interface that carries a writer. The stamps name it:
`Put(ctx, Item)` is a `writer` whose `shape.value_type` is `Item`, and
`golang.SampleRefFor` produces an `Item` to write. The suite seeds itself by
calling the interface's own writer, which is also what makes the cross-method
checks free: read-after-write is the writer followed by the reader, and
delete-removes is the writer, the deleter, then the reader.

All three writer detectors, not `writer` alone. They differ only in how many
non-context arguments they take — `writer` one, `compositewriter` two,
`multiargwriter` three or more — and the seed passes whatever the method
declares, so arity is not something it has to know. `mutator` is excluded despite
writing: it returns nothing, so a seed through one cannot report its own failure.

A read-only interface over external state has no writer and is the case that
needs a seed hook. The generated documentation says which of the two it is, so a
reader is never left wondering whether a check was omitted or merely unstated.

### Checks and extensions are the same type

Per method, one named type over the failure-recording interface and the method's
own concrete parameters:

```go
type MixedStoreCheck func(tb testing.TB, subject validates.Mixed, v validates.Payload)
```

Every generated check for that method is a value of it, and so is a consumer's —
so they compose, reorder, and each runs standalone. A method the generator has
nothing classification-specific to say about gets the same treatment: having no
law to assert does not make the signature unknown.

`testing.TB` rather than `*testing.T` is what makes a check drivable by a
stand-in, which is what the self-check rests on — and it applies to a consumer's
checks as much as to the generated ones.

### The option API has one required job and three escape hatches

The consumer cannot reach the generator, so the generated options are the entire
extension surface.

**Name the implementations.** `ServiceSubject(name, factory)`, repeated once per
implementation that claims to satisfy the interface. This is the one option a
consumer always writes, and it is not an escape hatch: which implementations
exist is the single fact no amount of derivation can recover from the source.
Plural and named because that is what a conformance suite is for — one statement
of the contract, run against every subject, so a check written once covers all of
them rather than being restated per implementation, which is where the statements
drift.

The three that follow are for what a run got wrong, and a suite that needs none
of them is the design working.

**Supply what derivation cannot reach.** A seed for a writer-less interface, an
authorised context for `scope`, a stale version for `cas`. One override struct
rather than one option per assertion, because the values are shared across checks
and a per-assertion option makes the consumer state each one twice.

**Add a check no classification implies.** A named function of the method's check
type. Named, so it reports beside the generated ones rather than as an anonymous
subtest.

**Drop or replace a generated check by name.** A check that is wrong for one
subject must be removable without abandoning the other twenty-six. Without this,
the consumer's only recourse is to stop running the suite — which is the failure
this whole design exists to avoid.

```go
servicetest.AssertServiceContract(t,
    servicetest.ServiceSubject("in-memory", newInMemory),
    servicetest.ServiceSubject("postgres", newPostgres),
    servicetest.ServiceSeed(func(ctx context.Context, s allshapes.Service) error { … }),
    servicetest.ServiceOnGet("second read is served from cache", myCheck),
    servicetest.ServiceWithout("Delete/deleteremoves"),
)
```

The seed returns an error rather than swallowing one: a seed that failed quietly
leaves every check after it running against an empty subject, passing and
asserting nothing.

`ServiceWithClock` is the fourth, and belongs to the first job rather than
being a fifth: it supplies something derivation cannot reach. Any check that
reads time measures on the run's clock, never the wall clock — generated code
calling `time.Now` has the machine it runs on as part of its subject, and fails
a correct implementation on a loaded box while passing a slow one on an idle
box. The default is `clock.RealClock()`, so an implementation taking no clock is
measured as it would have been. An implementation that does take one is built
with the same `clock.TestClock` in the consumer's own factory, and then a
duration budget is a claim about the time the implementation means to spend —
settled exactly, in no wall-clock time, which is also what makes the check's
failure direction provable at all.

### Worked example: source to output

The fixture is `conformance/corpus/iface/mixin/validates`. Everything below is in
the repo except the `//testkit:suite` line:

```go
type Payload struct{ Key, Body string }

//testkit:out validatestest/ pkg=validatestest
//testkit:stub
//testkit:suite
type Mixed interface {
    //testkit:mixin validates fn=Validate
    Store(ctx context.Context, v Payload) error

    Validate(v Payload) error

    Read(ctx context.Context, key string) (Payload, error)
}
```

What the annotator stamps, dumped from a run of the pipeline `conformance/gate`
assembles:

```
Store      shape=writer  key=        value=…/validates.Payload  mixins=[validates]
    mixin.validates.fn = "go.thesmos.sh/…/iface/mixin/validates.Mixed.Validate"
Validate   shape=writer  key=        value=…/validates.Payload  mixins=[]
Read       shape=reader  key=string  value=…/validates.Payload  mixins=[]
```

Two things there are load-bearing and neither is visible in the source.
`Validate` is classified `writer`, not left unclassified — the writer detector
matches `func(V) error` whether or not a context leads. And
`mixin.validates.fn` holds the **fully-qualified method name**, not the bare one
the author wrote: the shape resolver rewrites it, and the generator recovers
`Validate` through `golang.LocalName` exactly as `generator/stub` already does
for `orderafter`.

Selection is a function of the stamps and the signature, and nothing else:

| Method | Source | Check |
|---|---|---|
| `Store` | signature | smoke, cancellation, deadline, nil context |
| `Store` | mixin `validates` → `Validate` | refuses what the validator refuses |
| `Validate` | signature, no context | smoke only |
| `Read` | signature | smoke, cancellation, deadline, nil context |
| `Read` | detector `reader` | an error is accompanied by the zero value; a miss key reports not-found |
| interface | writer + reader | read-after-write |

Fourteen checks across three methods, of which exactly one came from a directive.

### Which tier owns which classification

[ADR-0018](../adr/0018-one-tier-owns-each-classification.md) assigns each
classification to exactly one tier, by a rule rather than a judgement:

> The suite tier implements no property `engine/model/law` already carries.
> Where a law exists, the classification is the model tier's. Where none does,
> the suite tier owns it — unless the claim cannot be stated against one subject
> making a fixed call, in which case neither tier covers it and the gate fails.

That rule is mechanical, so it needs a machine-readable mapping: one entry per
classification naming its law ID or `none`. A test asserts every name
`gate.Registered()` returns has an entry, and that every entry naming a law
names one `law` registers. Without it the assignment becomes the per-classification
opinion ADR-0017 correctly refused.

**Signature-derived checks are unaffected.** Smoke, cancellation, deadline,
nil-context and zero-on-error are the suite's for every method regardless of what
it declares, because no law carries them and none needs a reference.

**Detectors (20).** The suite owns the shape check for a detector `law` does not
reach; the rest add a model binding.

| Detector | Tier | Check, or law |
|---|---|---|
| `reader` | suite | an error is accompanied by the zero value |
| `readernoerror` | suite | an unknown key yields the zero value |
| `readerwithbool` | suite | `ok == false` is accompanied by the zero value |
| `lookup` | suite | `ok == false` is accompanied by zero values in both slots |
| `pointerreader` | suite | a nil pointer is accompanied by an error |
| `multireader` | suite | an error is accompanied by zero values in every slot |
| `batchreader` | suite | one result per key requested, in order |
| `mutator` | suite | a sample value does not panic |
| `streamconsumer` | suite | a consumed sequence is drained |
| `voidlifecycle` | suite | a call does not panic |
| `writer`, `compositewriter`, `multiargwriter` | model | `AUTO-WRITE-OBSERVABLE` |
| `aggregator`, `multiaggregator` | model | `AUTO-COUNT-EQUALS-REFERENCE` |
| `streamreader` | model | `AUTO-STREAM-COMPLETION` |
| `lifecycle` | model | `AUTO-LIFECYCLE-RESPECTS-CONTEXT` |
| `pure` | model | `AUTO-PURE-DETERMINISTIC` |
| `predicate` | model | `AUTO-PREDICATE-CONSISTENT` |
| `poisonaccessor` | model | `AUTO-POISON-NIL-ON-FRESH` |

**Mixins (28).**

| Mixin | Tier | Check, or law |
|---|---|---|
| `deprecated` | suite | the check logs the replacement and skips |
| `errors` | suite | the declared sentinel is returned for the miss input |
| `hooks` | suite | a registered callback fires |
| `integrationonly` | suite | checks sit behind the integration guard, so an unset environment yields no subtest rather than a passing one |
| `nilsafe` | suite | zero-value inputs do not panic |
| `orderafter` | suite | calling before the prerequisite fails |
| `partition` | suite | writes to two partitions do not interfere |
| `retrysucceeds` | suite | the call succeeds within the declared attempts |
| `sample` | suite | the named builder produces a value the method accepts |
| `scope` | suite | an unauthorised call is refused and an authorised one is not |
| `sideeffect` | suite | the named observation changes across the call |
| `timeout` | suite | the call completes within the declared budget |
| `validates` | suite | what the named validator rejects, the method rejects |
| `wrappedvia` | suite | the returned error wraps the named target |
| `concurrent`, `concurrentreaders` | suite | concurrent callers do not race — observable only under `-race` |
| `atomic` | model | `AUTO-ATOMIC-WRITE` |
| `bounded` | model | `AUTO-AGGREGATOR-BOUNDED` |
| `cacheable` | model | `AUTO-CACHEABLE` |
| `crdtmerge` | model | `AUTO-CRDT-MERGE` |
| `deleteremoves` | model | `AUTO-DELETE-RETURNS-NOT-FOUND` |
| `eventually` | model | `AUTO-EVENTUAL-CONVERGENCE` |
| `idempotent` | model | `AUTO-IDEMPOTENT-WRITE` |
| `lifecycleafterclose` | model | `AUTO-LIFECYCLE-AFTER-CLOSE` |
| `monotonic` | model | `AUTO-MONOTONIC-NON-DECREASING` |
| `pure` | model | `AUTO-PURE-DETERMINISTIC` |
| `readafterwrite` | model | `AUTO-READ-AFTER-WRITE` |
| `streamreflectsmutations` | model | `AUTO-STREAM-REFLECTS-MUTATIONS` |

**Contracts (24).**

| Contract | Tier | Check, or law |
|---|---|---|
| `if-absent` | suite | a second write for one key is refused |
| `if-match` | suite | a non-matching predicate refuses the write |
| `outbox` | suite | an appended message reaches the subscriber |
| `appender` | model | `AUTO-APPEND-ONLY-GROWS` |
| `batch-writer` | model | `AUTO-ATOMIC-WRITE` |
| `cache` | model | `AUTO-CACHEABLE` |
| `cas` | model | `AUTO-CAS-ATOMIC-ONE-WINNER` |
| `cursor` | model | `AUTO-CURSOR-NEXT-AFTER-CLOSE` |
| `lease` | model | `AUTO-LEASE-DOUBLE-ACQUIRE-BLOCKS` |
| `pagination` | model | `AUTO-PAGINATOR-NO-DUPLICATES` |
| `persister` | model | `AUTO-PERSISTER-RETRIEVABLE` |
| `pool` | model | `AUTO-POOL-BALANCED` |
| `publisher` | model | `AUTO-PUBLISHER-DELIVERS` |
| `saga` | model | `AUTO-SAGA-FULL-COMPENSATION` |
| `singleflight` | model | `AUTO-SINGLEFLIGHT-COALESCES` |
| `transaction` | model | `AUTO-TRANSACTION-ROLLBACK` |
| `updater` | model | `AUTO-UPDATER-REPLACES` |
| `upserter` | model | `AUTO-UPSERTER-IDEMPOTENT` |
| `watcher` | model | `AUTO-WATCHER-RETURNS-ON-CHANGE` |
| `workflow` | model | `AUTO-VALID-TRANSITION` |
| `circuit-breaker` | **neither** | needs a call that fails on demand; no law |
| `leader-election` | **neither** | needs two subjects; no law |
| `rate-limit` | **neither** | needs controlled time; no law |
| `tx` | **neither** | needs accumulated begin/commit/rollback state; no law |

`batch-writer` reads as this tier's until the rule is applied to it. `mode=atomic`
is the claim that an error leaves observable state unchanged, and
`AUTO-ATOMIC-WRITE` already implements exactly that — snapshot, write, and on
failure compare the snapshot back. Discharging it needs an observation of the
state, which the contract declares no reader role for, and a write that fails on
demand. So it is the model tier's, by the same rule that placed `atomic` there.

Four classifications are owned by no tier, and under ADR-0018 that fails the
gate — correctly. Each is a law to write, not a suite check to invent: a
one-shot `circuit-breaker` check with no way to induce a failure passes against
every implementation, including a broken one.

Fifteen of the twenty-four contracts name a partner callable, which the resolver
rewrites into a qualified name and back-stamps onto the partner. Both halves are
prerequisites for contract checks in either tier, and both are available.

### One template file per classification

Seventy-two checks cannot live in one template. The backend supports the split,
and the mechanism decides the emit model rather than the other way round.

**Every `.tmpl` under the tree is loaded, at any depth.** `collectTmplFiles`
walks the plugin's filesystem with `fs.WalkDir(fsys, ".")`, and `sdk/golang`
hands the backend an FS already rooted at `templates/golang`. Subdirectories
work — but the embed pattern has to be the recursive directory form, because
`//go:embed templates/golang/*.tmpl` reaches one level only:

```go
//go:embed templates/golang
var goTemplatesFS embed.FS
```

**Every file's defines merge into one tree per run**, so a define in one file is
reachable from any other with no import and no registration. Only the emit kinds
need a define named exactly `Kind()`.

**But a template reference is a string literal.** `{{ template "x" . }}` cannot
take an expression, so seventy-two classifications cannot be dispatched by
composing a name in the template. `{{ render . }}` can, because it looks the name
up from the item's `Kind()`:

```go
func (s *renderState) renderInto(w io.Writer, n emit.Node) error {
    kind := string(n.Kind())
    if s.tmpl.Lookup(kind) == nil {
        return fmt.Errorf("%w: %s", ErrTemplateMissing, kind)
    }
    return s.tmpl.ExecuteTemplate(w, kind, n)
}
```

So **each check is its own emit kind**, and the entry template ranges over a
method's checks calling `render`. They need no separate queueing: they are a
`[]sdk.Node` field on the one value queued per interface, and `render` is
re-entrant, so a check may render sub-nodes of its own.

```
generator/suite/templates/golang/
  suite.file.tmpl                    define "suite.file"    ← emit kind
  suite.tests.tmpl                   define "suite.tests"   ← emit kind
  suite.entry.tmpl                   suite.fixture.tmpl
  suite.options.tmpl                 suite.signature.tmpl
  detector/suite.check.reader.tmpl   define "suite.check.reader"
  mixin/suite.check.validates.tmpl   define "suite.check.mixin.validates"
  contract/suite.check.tx.tmpl       define "suite.check.contract.tx"
```

Uniqueness is **run-wide across plugins**, over both the define names and each
file's path relative to `templates/golang` — two plugins shipping `entry.tmpl`
collide at merge and the run writes nothing — so every filename and every define
carries the plugin's name. `fragment.` is the one reserved prefix.

The kind string is composed once, in Go, from the classification name the
projection already read. A test asserts that for every name `gate.Registered()`
returns, the merged tree holds a template of the composed kind — which fails at
build rather than midway through a consumer's render.

### How a check is shown to fail

A check is emitted as a body over `testing.TB` with its `t.Run` wrapper separate.
The interface is what a stand-in can satisfy; `*testing.T` is not.

So the generated test file drives every check twice — against a subject
configured to comply and one configured to violate — and asserts the second
fails:

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
hand-written twin, so the proof accompanies the check instead of trailing it
across seventy fixtures, and it is one property away from the complying subject.
Where the interface declares no `//testkit:stub`, this half is not emitted and
the header says so.

This is the mechanical form of
[ADR-0015](../adr/0015-subtest-names-carry-the-classification.md)'s requirement
that a broken fixture fail the classification it claims to: calling one exported
check directly is failure identity, with no name matching in between.

### Benchmarks and fuzzing are separate plugins

`bench` and `fuzz` are their own generators at `sdk.GeneratorCrossCutting`, one
bucket after `suite`.

Separate, because each is deletable in six months without taking the others down,
each declares its own capabilities and runs its own conformance suite, and a
consumer registers only what they want. The two grains of opting in are
different answers to different questions: dropping `fuzz.New()` from `All()`
decides whether the mode exists in a build at all, and `//testkit:fuzz` on a
method decides where a registered mode applies.

**The rule that makes it safe: `suite` computes the projection and queues it;
`bench` and `fuzz` read that value, never the source.** Two derivations of one
thing are two chances to disagree, and a benchmark seeded differently from the
assertion it mirrors is a disagreement nothing reports. `generator/fault` already
does exactly this against `stub`, through `sdk.PendingByOrigin`.

Two hazards ride along:

- **The orphan file.** `FileFor` is lookup-or-create, so a contributor emitting
  where `suite` did not creates a file alone — fuzz targets hanging off an
  interface with no suite. The type assertion on suite's pending value *is* the
  guard.
- **`Requires` resolves within a bucket only.** eidos's sorter, verbatim:
  `// Cross-bucket or simply absent — silently ignored at this layer.` The
  dependency is expressed by choosing the later bucket. `Requires: [suite]`
  documents the intent and orders nothing.

Each owns its outputs rather than sharing suite's suffix. Sharing means two
constants that must move in lockstep, and the three have wildly different run
costs — separate files make `-run`, `-bench` and `-fuzz` scoping fall out instead
of being fought.

`//testkit:bench` is `bench`'s directive, method-scoped because a budget is a
property of one hot path, batching its properties onto one line:

```go
//testkit:bench allocs=0 p99=500us mean=100us mem=4KiB
```

| Key | Gates | Backed by |
|---|---|---|
| `allocs=N` | allocations per operation; `0` is an alloc-free hot path | `Contract.AllocsMax` |
| `p99=D` | 99th-percentile nanoseconds per operation | `Contract.LatencyMax` |
| `mean=D` | mean nanoseconds per operation | a new `Contract` method |
| `mem=B` | bytes allocated per operation | a new `Contract` method over `MemStats.TotalAlloc` |

The directive's presence is the opt-in: a bare `//testkit:bench` measures and
reports, and each key present becomes a ceiling. A budget nobody declared is a
number the generator invented, so there is no default. Where a double exists
the benchmark drives `MixedStubBenchMode`, which `stub` already emits.

### Which methods earn a fuzz target

The ones that ask and can: `//testkit:fuzz` on the method is the ask, and Go's
constraint decides the can.

**The directive decides existence.** Method-scoped, no keys, negation denied —
a target exists where one is declared, and deleting the line is the
suppression, exactly as for the suite itself.

**Go's constraint decides feasibility.** `f.Fuzz` accepts basic types only, so
an annotated method qualifies iff every non-context parameter is one of those,
or is a struct whose exported fields decompose to them — `Store(ctx,
Payload{Key, Body string})` fuzzes as two strings and reconstructs. An
annotated method that does not decompose — a func, a channel, an interface in
a parameter — is a **diagnostic at the directive**, not a silent skip: the
author asked, so "asked and impossible" is an error with a position, where the
earlier default-on form could only leave a comment in a file nobody was
reading for one.

**The stamps decide the body.** `nilsafe` becomes "no input panics"; `bounded`,
"the result stays in range"; `pure`, "same input, same output"; `validates`, the
implication below. Absent any of them the target still runs the method, and Go's
fuzzer still finds panics, hangs and OOMs — worth having on any exported method.

```go
func FuzzMixedStore(f *testing.F, factory func() validates.Mixed) {
    f.Add("test-key", "test-body")   // seeds from the same fixture the checks use
    f.Fuzz(func(t *testing.T, key, body string) {
        s := factory()
        v := validates.Payload{Key: key, Body: body}
        if err := s.Validate(v); err != nil {
            testkit.Error(t, s.Store(t.Context(), v),
                "Store accepted what Validate refused")
        }
    })
}
```

**The rejected alternative: emit for every decomposable method, `fuzz=off` to
decline.** Its argument was ADR-0017's, and at full strength: whether a method
is a trust boundary is the strongest signal there is and it has no owner, so
coverage that depends on a remembered directive reintroduces the silent
omission that record exists to prevent. That argument is real, and it lost on
the costs a generated assertion does not have. A check runs in microseconds
per suite; a fuzz target is CI minutes and a corpus **per method**, which
makes unasked-for targets on all six methods a bill nobody itemised, and the
interface-scoped `fuzz=off` could not spell the case teams actually have —
fuzz the two parsers, skip the four accessors. What survives of the omission
concern is the feasibility diagnostic above — opt-in converts "silently not
fuzzable" into an error the author sees — and the residue is accepted as a
cost: an unannotated trust boundary gets no target, and this paragraph is the
record that the trade was made knowingly rather than by default.

Fuzzing is the only mode that witnesses **many values**. Checks witness one; the
model tier witnesses many sequences but costs a reference implementation most
subjects will never have. That is most of the distance between "some assertions"
and a conformance suite, at a fraction of the model tier's price.

One consequence: **a fuzz target cannot be driven through the double.** A stub
returns configured answers regardless of input, so fuzzing it proves nothing. Its
factory parameter takes the consumer's real implementation and says so.

### Outputs

| Plugin | Tag | Suffix | Contents |
|---|---|---|---|
| `suite` | `""` | `_suite.gen.go` | entry point, per-method checks and check types, fixture, options |
| `suite` | `test` | `_suite.gen_test.go` | per check, a complying subject and a violating one |
| `bench` | `""` | `_bench.gen.go` | per-method and whole-contract benchmark helpers |
| `fuzz` | `""` | `_fuzz.gen.go` | per annotated method, a seeded target helper |

Suffixes follow `<source-basename>_<generator>.gen.go`, which `stub` and
`builder` use and `reference/layout.md` documents. `enum` and `sentinel` deviate.

Benchmark and fuzz helpers are exported functions taking `*testing.B` and
`*testing.F`, called from the consumer's own `_test.go` — Go requires the target
itself to live there, and it is the same pattern `stub` already uses.

### What a consumer writes

```go
func inmemory() allshapes.Service { return allshapes.NewInMemoryService() }

func TestServiceContract(t *testing.T) {
    t.Parallel()
    servicetest.AssertServiceContract(t,
        servicetest.ServiceSubject("in-memory", inmemory))
}

// Present because Put carries //testkit:bench and //testkit:fuzz in the
// source; a method without the directive gets no helper to call.
func BenchmarkServiceContract(b *testing.B) { servicetest.BenchmarkServiceContract(b, inmemory) }
func FuzzServicePut(f *testing.F)           { servicetest.FuzzServicePut(f, inmemory) }
```

One option, naming the one thing the source does not say. That is the acceptance
test for this design and the measure it is held to: every *other* option a
consumer writes is a derivation that has not been done.

The double is not in that file and should not be. `AssertServiceContract` runs
the whole contract a second time against each subject wrapped in
`ServiceStub` — derived from the `//testkit:stub` on the same interface, read off
the double's queued emit value rather than its directive, because a directive
says what was asked for and the queue says what was produced. Anything the
wrapper fails that the subject passes is the double lying, which is what makes a
generated double trustworthy. `ServiceWithoutDouble()` declines the second pass
where the double is not used; an interface carrying no `//testkit:stub` generates
neither the run nor the option.

### Composition and generics

The axes are orthogonal, so a method carrying a detector, a mixin and a contract
owes all three families in one file without collision. The four fixtures under
`corpus/iface/composite` — `batched-mixins`, `leased-idempotent-writer`,
`paginated-reader`, `tx-with-retry` — are where that is proven.

The ten fixtures under `corpus/iface/lang` prove the emission survives Go's type
system rather than any classification: embedding, foreign embedding, generics,
opaque constraints, a function-typed parameter, named returns, multiple returns,
variadics, a method with no context, and a parameter named for the identifier a
generated receiver binds. A generic interface's entry point carries the type
parameters; the generated self-check instantiates at the witnesses the source
names, because a Go test function cannot take type parameters.

Foreign embedding is generated rather than refused. eidos carries the embed's
type-checked method set on the embed itself, so `io.Closer`'s `Close` reaches the
double and the harness with no declaration in the run — and a loaded declaration
still wins where there is one, which is what makes the local and foreign fixtures
have to agree.

A variadic method's checks witness exactly one element, because a fixture holds
one value per parameter. That is a narrowing rather than a defect — everything
about *several*, including the empty call, is a claim only the author can make —
but it is invisible in a check's signature, where `keys string` reads like an
ordinary parameter. The generated check type and fixture field both say so, so
the limit is legible where a consumer meets it rather than recorded here alone.

### The model tier is a sibling generator, not a second harness

`engine/model` is the runtime and `engine/model/law` the property library;
nothing binds `law.ReadAfterWrite[Mixed, string, Payload]{Read: …, Keys: …}` to a
concrete interface. That binding is per-interface, concrete, and derivable from
exactly the stamps read here — which is why `suite` was drifting into owning it.

A `model` generator reads suite's queued projection the way `bench` and `fuzz`
do, and emits the bindings plus the runner call. It is gated on a reference
implementation the consumer supplies, so it attaches as an option on the same
entry point rather than as a harness a team has to remember separately:

```go
servicetest.AssertServiceContract(t,
    servicetest.ServiceSubject("in-memory", inmemory),
    servicetest.ServiceModelChecks(newReference))
```

The alternative — the model generator declaring suite's output suffix so both
compose the same target — works, but couples two plugins through a constant that
must move in lockstep, and a plugin holding two roles sorts ahead of every
generator regardless of what it requires, so its contributions render above
suite's. The option form has neither problem and stays reversible.

## What is decided

| ADR | Decision |
|---|---|
| [0004](../adr/0004-consume-only-the-annotator-plugin.md) | Consume only eidos's annotator plugin |
| [0012](../adr/0012-generate-per-shape-helpers-into-the-consumer.md) | Generate per-shape helpers into the consumer |
| [0015](../adr/0015-subtest-names-carry-the-classification.md) | Subtest names carry the classification |
| [0018](../adr/0018-one-tier-owns-each-classification.md) | One tier owns each classification |

## What the design rests on

- **The shape resolver and validator are registered.** All 24 contract checks
  need partner references rewritten into qualified names and back-stamped onto
  the partner, and `shape.Plugin` is three annotators rather than one.
  `generator.Annotators()` registers every companion, so both hold today. The
  same change moved `mixin.orderafter.fn` from a bare name to a qualified one,
  which `generator/stub` already reads through `golang.LocalName`.
- **Consumers never write plugins.** testkit generates; consumers write
  implementations and extensions. The generated option API is therefore the whole
  extensibility ceiling, which is why it has three jobs rather than one.
- **`Contract` gates p99, not the mean.** `//testkit:bench p99=` names what
  `Contract.LatencyMax` compares. `reference/generators/bench.md` describes a
  `//testkit:latency` directive as a mean ceiling and a separate
  `//testkit:percentiles`; neither exists, and one directive naming the statistic
  it gates replaces both.
- **`mean=` and `mem=` need `Contract` methods that do not exist.** `AllocsMax`
  and `LatencyMax` are the whole surface today.

## Open questions

- **Four classifications are owned by no tier.** `circuit-breaker`,
  `leader-election`, `rate-limit` and `tx` have no law and no honest single-call
  form. Under ADR-0018 the gate fails until each has one, and the work is a law
  rather than a suite check — but nothing yet says which of the four is worth
  writing first.
- **The classification-to-law mapping has no home.** ADR-0018's rule is
  mechanical only if the mapping is data a test can check. It could live beside
  the suite generator, beside `law`, or in the gate that measures both. The
  third is where the union is already computed.
- **Concurrency checks need `-race` to fail.** `concurrent` and
  `concurrentreaders` are the suite's, and both are observable only under the
  race detector. `make check` runs `mod`, `lint`, `test`, `coverage` and
  `branch`; `test race` is a separate target. Either it joins the check stages
  or those two are decoration in the default gate.
- **A writer-less interface cannot seed itself.** The seed hook covers it, and a
  hook is an option — which the acceptance test above says is a derivation not
  done. Whether that is a genuine exception or a signal that seeding should come
  from a sibling interface in the same package is unresolved.
- **Some suite checks need inputs no stamp describes.** `scope` needs an
  authorised context and an unauthorised sentinel; `if-match` needs a failing
  predicate. `shape.contract.<name>.param.<key>` carries what the directive
  declared and is the first place to look, but it cannot carry a value the author
  did not write. Where the override struct is not enough either, the check has to
  fail loudly on an unsupplied input rather than pass vacuously — and if it
  cannot, the classification belongs in the "owned by no tier" list above.
