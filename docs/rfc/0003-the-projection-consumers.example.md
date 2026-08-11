# RFC-0003 worked example: one interface, every output

Companion to [RFC-0003](0003-the-projection-consumers.md). It walks one
interface — the conformance fixture `validates.Mixed` — through the whole
pipeline: what the annotator stamps, what `suite` queues, and what each of
the three consumers emits from that value.

Provenance is marked throughout: excerpts under **(shipped)** exist in
`conformance/corpus/iface/mixin/validates/validatestest` today; everything
else is the proposal rendered by hand, in exactly the form the corpus will
hold once the plugins land. The two `//testkit:bench` / `//testkit:fuzz`
lines on the source are example additions — the shipped fixture does not
carry them yet.

## The source

```go
type Payload struct{ Key, Body string }

var ErrInvalid = errors.New("validates: invalid payload")

//testkit:out validatestest/ pkg=validatestest
//testkit:stub
//testkit:suite
type Mixed interface {
    //testkit:mixin validates fn=Validate
    //testkit:fuzz
    Store(ctx context.Context, v Payload) error

    Validate(v Payload) error

    //testkit:bench allocs=0 p99=500us
    Read(ctx context.Context, key string) (Payload, error)
}
```

## What the annotator stamps

Dumped from a pipeline run — this is the input every generator shares, and
none of them re-derives it:

```text
Store      shape=writer  key=        value=…/validates.Payload  mixins=[validates]
    mixin.validates.fn = "go.thesmos.sh/…/iface/mixin/validates.Mixed.Validate"
Validate   shape=writer  key=        value=…/validates.Payload  mixins=[]
Read       shape=reader  key=string  value=…/validates.Payload  mixins=[]
```

Two stamps are load-bearing below. `Validate` is classified `writer` — the
detector matches `func(V) error` whether or not a context leads — but it is
also *named* by `mixin.validates.fn`, and that partner reference is what the
model tier's cluster rule acts on. `Read`'s `key=string` plus the shared
`value=Payload` is what places `Store` and `Read` in one state cluster.

## What suite queues: the projection

`suite` derives once and queues one value per interface. The satellites read
this — never the source it came from. Rendered as a literal with the values
this fixture produces:

```go
suite.Contract{
    Subject: suite.Subject{
        IfaceName: "Mixed",
        IfaceRef:  /* …/iface/mixin/validates.Mixed */,
        Runtime:   "go.thesmos.sh/testkit",
    },
    EntryName: "AssertMixedContract",
    Fixture: suite.Fixture{
        TypeName: "MixedFixture",
        CtorName: "DefaultMixedFixture",
        Fields: []suite.FixtureField{
            {Name: "V", Parts: /* Key: "test-key" / "other-key",
                                  Body: "test-body" / "other-body" */},
            {Name: "Key", Sample: /* "test-key" */, Other: /* "other-key" */},
        },
    },
    Seed:   &suite.Seed{Method: /* Store */, Args: []string{"V"}},
    Double: &suite.Double{TypeName: "MixedStub", CtorName: "NewMixedStub",
        DelegateToName: "MixedStubDelegateTo"},
    Methods: []suite.Method{ /* Store, Validate, Read — sig, check type,
        arg-field names, generated checks */ },
}
```

Note the coupling already present in the derivation: `V.Key` and `Key` hold
the same `"test-key"`, so a value written through the seed is reachable
through the reader's fixture key. Every mode below inherits that hit-path
guarantee for free.

## What suite emits (shipped)

The harness itself is prior art here; three excerpts locate the hooks the
satellites use.

```go
type MixedReadCheck func(tb testing.TB, subject validates.Mixed, key string)

type MixedFixture struct {
    V        validates.Payload
    VOther   validates.Payload
    Key      string
    KeyOther string
}

func DefaultMixedFixture() MixedFixture { /* the derived pair per field */ }

func AssertMixedContract(t *testing.T, opts ...MixedOption)
```

This RFC adds the registry beside them:

```go
type MixedRegistration struct {
    Name string
    New  func() validates.Mixed
}

func RegisterMixedSubject(name string, factory func() validates.Mixed)
func RegisteredMixedSubjects() []MixedRegistration
```

## bench

**Reads:** the `//testkit:bench` stamp on `Read` (its own directive, from
source), plus `Subject`, `Fixture`, `Seed` and `Read`'s signature from the
projection.

**Emits** `iface_bench.gen.go` — the exported body:

```go
// BenchmarkMixedRead measures Read against a seeded subject and enforces
// the declared ceilings: allocs=0, p99=500us.
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

The loop argument is `fx.Key` — the value the harness's own `Read` checks
were handed, and, via the seed, a guaranteed hit. Nothing is zero-valued and
nothing is unseeded, so the number is about the hot path it claims to be
about.

**And** `iface_bench.gen_test.go` — the discoverable entry, external test
package:

```go
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

## fuzz

**Reads:** the `//testkit:fuzz` stamp on `Store`; from the projection, the
fixture pair for the corpus and `Store`'s signature; the `validates` stamp
selects the body.

**Feasibility:** `Store`'s one non-context parameter is `Payload`, whose
exported fields are two strings — it decomposes, so the target is emitted.
Had it taken a channel, the directive itself would carry a codegen
diagnostic instead.

**Emits** `iface_fuzz.gen.go`:

```go
// FuzzMixedStoreInput is the property for one input: what Validate refuses,
// Store must refuse.
func FuzzMixedStoreInput(t *testing.T, s validates.Mixed, key, body string) {
    v := validates.Payload{Key: key, Body: body}
    if err := s.Validate(v); err != nil {
        testkit.Error(t, s.Store(t.Context(), v),
            "Store accepted what Validate refused")
    }
}
```

**And** `iface_fuzz.gen_test.go`:

```go
func FuzzMixedStore(f *testing.F) {
    subjects := validatestest.RegisteredMixedSubjects()
    if len(subjects) == 0 {
        f.Skip("no subjects registered: call validatestest.RegisterMixedSubject in an init")
    }
    f.Add("test-key", "test-body")    // fixture:sample (fx.V, decomposed)
    f.Add("other-key", "other-body")  // fixture:other (fx.VOther, decomposed)
    f.Add("", "")                     // boundary:zero-tuple — Validate's reject side
    f.Add("", "test-body")            // boundary:empty (Key)
    f.Add("test-key", "")             // boundary:empty (Body)
    f.Add("k\x00ey", "test-body")     // boundary:format-hostile (Key)
    // …one seed per boundary value per parameter, varied from the sample
    // base one at a time; enum and bounded seeds would follow here if the
    // parameter types carried those classifications.
    f.Fuzz(func(t *testing.T, key, body string) {
        for _, s := range subjects {
            validatestest.FuzzMixedStoreInput(t, s.New(), key, body)
        }
    })
}
```

The first two seeds are the fixture pair decomposed in declaration order, so
a crash found here replays against the harness with no translation; the rest
are the derived boundary alphabet, each line naming the rule that produced
it. The growing corpus lands in `testdata/fuzz/FuzzMixedStore/`, in the
consumer's tree, and `make fuzz-promote` commits what the cache corpus
learns between runs.

## model

**Reads:** the shape stamps (its own read — law selection is the model
tier's), and the projection's fixture, seed and naming.

**The cluster map**, derived first. Edges: `Store` and `Read` agree on
(`key=string` via the reader, `value=Payload`) — one CRUD cluster.
`Validate` is referenced by `mixin.validates.fn`, so the partner role wins
over its `writer` detector stamp: inert in the reference, out of the
alphabet.

```text
cluster crud → ref.MapStore[string, validates.Payload]
    Store     writer   modelled (write)
    Read      reader   modelled (read)
    Validate  partner  (validates.fn of Store) — inert
```

**Emits** `iface_model.gen.go`. The file-level docblock is the report
header — the centrepiece of what "derived" means here:

```go
// Code generated by testkit model. DO NOT EDIT.
//
// Model bindings for validates.Mixed.
//
//    Cluster:     crud → ref.MapStore[string, validates.Payload]
//                 Store    writer   modelled (write)
//                 Read     reader   modelled (read)
//                 Validate partner  (validates.fn of Store) — inert
//    Laws:        AUTO-WRITE-OBSERVABLE   (Store)
//                 AUTO-READ-AFTER-WRITE   (Store → Read)
//    Concurrent:  not derived — the KV model needs a miss sentinel and no
//                 errors mixin is declared; declare one to arm it
//    Exhaustive:  alphabet {Store(V), Store(VOther), Read(Key), Read(KeyOther)},
//                 default depth, determinism probe first
//    Mutation:    DropWrites(Store), ReturnWrongValue(Read) — all must be killed
//    Leak check:  not derived — no lifecycle shape
//    Arm / drop:  MixedModelReference, MixedWithout("model/…")
package validatestest
```

The generators, the adapter, the laws and the property — one derivation
each:

```go
type mixedModelGens struct {
    keys   *rapid.Generator[string]           // sampled: fx.Key, fx.KeyOther + drawn
    values *rapid.Generator[validates.Payload] // Key field pinned to keys
}

// mixedModelRef adapts the crud cluster's oracle to the full interface.
type mixedModelRef struct{ crud *ref.MapStore[string, validates.Payload] }

func (r *mixedModelRef) Store(_ context.Context, v validates.Payload) error {
    r.crud.Put(v.Key, v)
    return nil
}
func (r *mixedModelRef) Read(_ context.Context, k string) (validates.Payload, error) {
    return r.crud.Get(k)
}
func (r *mixedModelRef) Validate(validates.Payload) error { return nil } // partner: inert

func mixedModelLaws(gens mixedModelGens) *model.Registry[validates.Mixed] {
    r := model.NewRegistry[validates.Mixed]()
    r.Add(law.ReadAfterWrite[validates.Mixed, string, validates.Payload]{
        Read: func(rt *rapid.T, s validates.Mixed, k string) (validates.Payload, error) {
            return s.Read(rt.Context(), k)
        },
        Keys: gens.keys,
    })
    // AUTO-WRITE-OBSERVABLE wired the same way, from the Store closure.
    return r
}

// MixedModelProperty is the composition point: actions, laws, reference and
// generators assembled once. Check, MakeFuzz and the exhaustive test all
// consume it or its parts.
func MixedModelProperty(factory func() validates.Mixed,
    opts ...MixedModelOption) func(*rapid.T)
```

The extension registration is what threads it into the one entry point:

```go
func init() {
    mixedContractExtensions = append(mixedContractExtensions, mixedContractExtension{
        name: "model",
        run:  runMixedModel, // model.Check(t, MixedModelProperty(s.New, cfg.modelOpts...))
    })
}
```

**And** `iface_model.gen_test.go` — three residents:

```go
// FuzzMixedModel lets the fuzzer drive rapid's choice stream: the
// sequence-space counterpart of FuzzMixedStore's input space.
func FuzzMixedModel(f *testing.F) {
    subjects := validatestest.RegisteredMixedSubjects()
    if len(subjects) == 0 {
        f.Skip("no subjects registered: call validatestest.RegisterMixedSubject in an init")
    }
    f.Fuzz(model.MakeFuzz(func(rt *rapid.T) {
        for _, s := range subjects {
            validatestest.MixedModelProperty(s.New)(rt)
        }
    }))
}

// TestMixedModelExhaustive proves law absence within bounds: every
// reachable state under the fixture alphabet, subject and reference in
// lockstep, observable agreement checked at each.
func TestMixedModelExhaustive(t *testing.T) { /* determinism probe, then bmc.Run */ }

// TestMixedModelMutationKill wraps the derived adapter in each compatible
// operator and asserts the law suite kills every one. An unkilled operator
// is a hole in the bindings, not in any consumer's implementation.
func TestMixedModelMutationKill(t *testing.T) {
    report := mutation.Run(t,
        []mutation.Operator{
            mutation.DropWrites{ /* rate */ },
            mutation.ReturnWrongValue[string]{ /* keys */ },
        },
        func(tb testing.TB, op mutation.Operator) { /* law suite vs wrapped adapter */ },
    )
    testkit.Empty(t, report.Unkilled(), "every wired operator must be killed")
}
```

## What the consumer writes

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

```console
go test                                    # checks, laws, mutation, exhaustive
go test -bench 'MixedContract/in-memory'   # the declared budgets on Read
go test -fuzz FuzzMixedStore               # Store's input space
go test -fuzz FuzzMixedModel               # the interface's sequence space
```

## How it is generated

Each consumer is an ordinary eidos generator plugin with the standard
layout — `doc.go`, `<plugin>.go`, `<plugin>_go.go`,
`templates/golang/*.tmpl` — registered at `sdk.GeneratorCrossCutting`, one
bucket after `suite`.

**The read.** Each plugin walks `sdk.PendingByOrigin[*suite.Contract]` and
emits only for origins where `suite` queued — the orphan-file guard — then
reads its own directive stamps and (for `model`) the shape stamps through
`ctx.Reader`, so both reads land in its cache key.

**The emit.** One emit value per output file, queued against the source
interface; sub-parts (a law binding, a fuzz body) are child nodes the
backend renders through `render` dispatch on their kind, exactly the
one-template-per-kind mechanism the suite generator uses.

| Plugin | Emit kind | Renders | Output |
|---|---|---|---|
| `bench` | `bench.body` | the measured loop per annotated method | `_bench.gen.go` |
| `bench` | `bench.entry` | the registry-ranging `Benchmark*` entry | `_bench.gen_test.go` |
| `fuzz` | `fuzz.input` | the per-input property | `_fuzz.gen.go` |
| `fuzz` | `fuzz.entry` | one `Fuzz*` target per annotated method | `_fuzz.gen_test.go` |
| `model` | `model.bindings` | header, gens, adapter, laws, property, options, extension | `_model.gen.go` |
| `model` | `model.selfcheck` | model fuzz entry, exhaustive test, mutation kill test | `_model.gen_test.go` |

**The routing.** Layout owns every path: suffixes compose with the source
basename (`iface.go` → `iface_model.gen.go`), the `_test.go` suffixes take
the external-test-package shift (which is why the registry's read side is
exported), and `//testkit:out` / `pkg=` on the source move all eight files
together. No plugin computes a path.

**The proof obligations.** Before any of this merges, the corpus holds: the
mutation kill test green (the laws can fail), `fired > 0` for every bound
law across the fixtures (no vacuous binding), and a deliberately broken
subject failing exactly the laws it violates (failure identity) — the same
prove-can-fail gate the suite generator passed.
