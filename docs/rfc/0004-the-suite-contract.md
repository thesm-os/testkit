---
rfc: 0004
title: The suite contract
status: Accepted
date: 2026-08-13
---

# RFC-0004: The suite contract

## Summary

The generated conformance suite gets a consumer contract: a small public data
model (`go.thesmos.sh/testkit/suite`) that every generated package targets, an
ergonomic generated veneer that constructs it, generator-backed fixtures that
unify the suite tier's fixed values with the model tier's rapid pools, and a
placement doctrine that gives every claim exactly one home — on the role
interface that carries its semantics, inferred from types where inference is
decidable.

This RFC replaces the consumer-facing surface of [RFC-0002](0002-the-suite-generator.md)
(the per-interface functional-options funnel). It does not touch RFC-0002's
derivation pipeline, check families, or falsification layer, and it does not
merge the suite and model generators — both plugins contribute checks to one
composed surface through slots (argued, not assumed: see Alternatives).

The contract is designed around four invariants the current surface cannot
hold: an unknown drop must not compile, an unarmed capability must fail red
with its fix rather than skip, every knob must be data something provably
reads, and the set of emitted checks must be diffable in review.

## Problem

The audit of 2026-08-13 measured the consumer surface directly, twice — once
against the corpus, once against twelve real interfaces transcribed from the
standard library and ecosystem drivers. (The audit lives in the local
register at `docs/superpowers/platform-audit.md`; until it is committed, the
[Evidence](#evidence) appendix carries the derivation command behind every
number used here.) Five defect classes recur, and all five are structural
properties of the options funnel:

**Stringly identity.** `XWithout("GetAll/reports a miss")` was a bare map
lookup; the corpus carried three phantom drops, one naming a check the
generator provably cannot emit (`generator/suite/suite.go:926` requires one
call arg). Run-start validation landed the day after this RFC's first draft
(`b5c24be0`) and cleared the corpus — but the invariant this RFC demands is
**compile-time** drops, which no runtime validation delivers: a typo'd drop
must not build, not fail at first run.

**Invisible knobs.** `XWithClock` writes a config field nothing reads — ten
lines of prose per file promising time-budget checks that do not exist.
`ContractModelConcurrent` accepts `opts` and ignores them. Both are possible
only because the config struct is private: there is no data a reader or a
test can hold the option against.

**Unarmed capability = silent green.** A `//testkit:mixin ttl` annotation
binds `AUTO-TTL-EXPIRY (clocked)`, lists it in the header, and never runs it
unless the consumer discovers `XModelClocked`. Deleting TTL expiry from a
correct-looking implementation left 122 checks green.

**Per-interface vocabulary.** ~30 exported `X`-prefixed symbols per generated
package, relearned per interface. At 20 methods the flat ID space alone is
~100 constants with no navigation.

**Request-struct blindness.** On interfaces shaped `M(ctx, req Struct)
(Resp, error)` — the dominant idiom for evolving APIs — the derivation binds
nothing: 2.3% of generated checks on real interfaces were claim-derived, and
the engine's own law catalogue (per-stream chains, CAS, poison) sat unused
against an interface made of exactly those semantics.

The scale that bites: a composed interface of nine roles, eighteen methods,
three type parameters. The current surface generates quartet noise, smoke
checks for every ctx-less observer, and an annotation burden that grows with
method count while the assertion power does not.

## Design

### The data model: `go.thesmos.sh/testkit/suite`

One package, in the root module. It imports `testing` and
`go.thesmos.sh/testkit/clock` and **adds zero dependencies** — the root
module requires exactly one module today (go-cmp) and this package moves that
number nowhere. It never imports `engine/model`; model-tier checks close over
the engine themselves.

```go
// Package suite is the vocabulary every generated conformance package
// targets and every consumer composes with. Generated code constructs
// these values; nothing in this package is generated.
package suite

// ID names one check, stable across regenerations.
//
// Grammar (gated at generation; hand-built IDs are validated by Run):
//
// id            = scope-segment 1*( "/" sub-segment )
// scope-segment = method / family
// method        = Go exported identifier   ; begins uppercase: "Append"
// family        = reserved lowercase word  ; "chain", "model", "poison",
//                                          ; "cross-role", "sim", "x"
// sub-segment   = law-id / label
// law-id        = "AUTO-" 1*( ALPHA / "-" )
// label         = printable ASCII without "/", tab, newline
//
// The case split makes namespace collision impossible: an exported Go
// method cannot begin lowercase, so a method named Chain never collides
// with the "chain" family. The family registry is declared here and
// extended only by RFC. Hand checks live under their method's scope or
// under "x/" (extension). A bare scope segment is not a valid ID —
// every check names at least one sub-segment, so a scope is always a
// group and never a check. Uniqueness scope: one generated package.
type ID string

// Class groups checks by the claim family they discharge, axis-qualified
// so lock files never collide across vocabularies: "signature/…" here,
// "mixin/…", "contract/…", "cross-role/…", "sim/…" for derived families,
// whose strings are the axis-qualified eidos classification names and
// whose typed constants are generated into the veneer.
type Class string

const (
 ClassSmoke      Class = "signature/smoke"
 ClassCancel     Class = "signature/cancel"
 ClassDeadline   Class = "signature/deadline"
 ClassNilContext Class = "signature/nilcontext"
 ClassZeroValue  Class = "signature/zero-on-error"
)

// Caps declares what a check needs from the subject beyond construction.
// Zero value: nothing. A required capability the subject does not provide
// FAILS the check with the arming instruction — it never skips.
type Caps struct {
 Clock  bool  // subject must be constructible on the run's TestClock
 Induce error // subject must be able to induce this sentinel on demand
}

// Check is one assertion about a subject. Generated checks and
// hand-written checks are the same type; there is no second-class
// extension mechanism and no private registry behind this one.
type Check[S any] struct {
 ID     ID
 Method string // "" for whole-subject checks
 Class  Class
 Claim  string // one sentence; rendered in the report and the manifest
 Needs  Caps

 // Exactly one of Run / RunWith is set; Run's validate phase enforces
 // it before anything executes. Run receives one fresh instance — the
 // shape of nearly every check. RunWith is for ORCHESTRATING checks —
 // clocked laws, recovery scenarios, simulation storms — that build,
 // kill, and rebuild subjects themselves through the Subject's
 // constructors. Generated model extensions already receive the
 // factory today; RunWith is that precedent carried into the contract
 // instead of a check ignoring the instance it was handed.
 Run     func(tb testing.TB, s S)
 RunWith func(tb testing.TB, sub Subject[S])
}

// Subject is one implementation under test. New builds a fresh,
// unseeded instance per check; tb carries cleanup for real backends.
type Subject[S any] struct {
 Name    string
 New     func(tb testing.TB) S
 OnClock func(tb testing.TB, clk *clock.TestClock) S // nil = clocked checks FAIL, naming the fix
 Induces map[error]func(tb testing.TB, s S)          // sentinel → inducer; missing = those checks FAIL

 // Recover rebuilds a subject over the durable state prior instances
 // left behind — the closure owns the durable medium (the dir, the
 // container, the pool). nil = recovery and simulation checks FAIL,
 // naming this field.
 //
 // The one commitment the name carries today: Recover makes no claim
 // about the prior instance's shutdown state, and returns a subject
 // observing at least the effects of operations the prior instance
 // ACKNOWLEDGED; the frontier for in-flight operations is
 // implementation-defined until the sim RFC narrows it.
 //
 // RESERVED SEMANTICS beyond that: committed-versus-in-flight, and
 // whether crash is a first-class act, are the sim RFC's to define —
 // as a law pair (recover-after-clean-Close ≡ identity;
 // recover-after-abandon ⊇ acknowledged writes), not prose. The field
 // ships now because it is an additive nil-able capability today and
 // a v2 conversation after the freeze; no generated check uses it
 // until the sim RFC lands.
 Recover func(tb testing.TB) S

 // Oracle marks this subject as the differential reference: every
 // other subject's model leg runs against it instead of against a
 // twin. At most one per Run — two is a validation failure. The
 // oracle itself still runs the full deterministic suite and its own
 // twin leg.
 Oracle bool

 // Serial excludes this subject from parallel execution AT BOTH
 // LEVELS — the subject runs after the parallel group, and its checks
 // run sequentially within it. The field's motive is construction and
 // resource pressure, which per-check parallelism would reintroduce.
 // Reported in the run report.
 Serial bool
}

// Suite is checks as data. Every combinator returns a copy; the zero
// value is an empty suite Run refuses.
type Suite[S any] struct {
 Name   string
 Checks []Check[S]

 dropped []ID // recorded by Without; Run fails on any not present
}

func (s Suite[S]) With(extra ...Check[S]) Suite[S]
func (s Suite[S]) Without(ids ...ID) Suite[S]
func Merge[S any](suites ...Suite[S]) Suite[S]

// Ctor adapts a plain constructor to the tb-form every constructor
// position takes. Shape-generic, so it lives here rather than in every
// veneer: Subject("in-memory", suite.Ctor(kv.NewInMemory)).
func Ctor[T any](new func() T) func(testing.TB) T

// Run executes every check against every subject as
// t.Run(subject.Name) → t.Run(string(check.ID)), then emits the report.
//
// Run fails, before any check executes, on: zero subjects; duplicate
// subject names; a Without ID matching no check (the known set is
// printed); a hand-built ID violating the grammar; a Check setting
// neither or both of Run/RunWith; more than one Oracle. Per check it
// fails, not skips, a Needs the subject cannot meet, with the one-line
// fix in the message.
func Run[S any](t *testing.T, s Suite[S], subjects ...Subject[S])

// Report is the run's machine-readable account — checks × subjects by
// class, every drop, every skip with its reason, the reference tier
// each model leg ran on (differential / derived / twin, with the
// generated TwinWhy sentence on the twin floor), and each capability's
// armed state per subject. The text rendering in the test log is a
// VIEW of this struct, never the source: the JSON encoding is
// versioned ("testkit-report v1", additive-only) and is what the
// conformance statement and any tooling consume.
type Report struct { /* … versioned; fields additive-only after v1 */ }
```

The `Report` encoding, sampled (the full schema is step 1's docs-owed):

```json
{"format": "testkit-report v1",
 "suite": "ledgertest", "checks": 64, "subjects": 2,
 "legs": [{"check": "model/twin", "subject": "pebble",
   "tier": "differential", "oracle": "in-memory"}],
 "dropped": ["ShredAttester/AttestCryptoShred/smoke"], "…": null}
```

Combinator misuse is specified, not implementation-defined:

| Combinator | Misuse | Outcome |
|---|---|---|
| `With` | duplicate ID against the suite | validation failure naming both sources |
| `Without` | ID matching no check | validation failure printing the known set |
| `Merge` | duplicate ID across suites | validation failure naming both suites |
| any | grammar-violating hand ID | validation failure quoting the grammar |

Design consequences, stated once:

- **Phantom drops are impossible twice over.** Against generated checks the
  drop is a typed constant — regeneration that removes the check breaks the
  build at the drop site. Against the residual case (`suite.ID("typo")` by
  hand), `Run` fails naming the known set.
- **Write-only knobs are impossible.** There is no private config. Every
  option below is a function that constructs or transforms these public
  values, so "does anything read it" is answerable by grep and enforceable
  by the vacuity detector.
- **Skip-as-escape is inverted for capabilities.** The bound-but-unarmed
  class — advertised in the header, never run, green — becomes red with the
  fix in the message. Opting out is `Without`, which is visible in the diff.

Deliberately absent from v1, each additive later at zero break cost under
this semver posture: `Wrap` (no named consumer until the fault-injection
story lands), `Only` (its consumer, the fast tier, is Deferred), and
`WithoutClass` (sugar over `Without` + the index). Shipping unused surface
into a freeze is how the current tool got write-only knobs.

### The generated veneer

The data model is the contract; nobody types struct literals against it in a
test file. Each generated package emits a thin veneer that constructs it.

The subject builder is the veneer's one non-trivial type, and it exists
because Go has no generic methods: once a concrete `*ledgermem.Ledger` is
erased into `suite.Subject[Ledger]`, no later method can accept a
concrete-typed constructor or method expression again. The builder holds the
concrete type parameter for the whole chain and lowers once:

```go
// Ledger is the witnessed instantiation every generated symbol speaks.
// The consumer never writes the type arguments.
type Ledger = ledger.Ledger[patchtest.Text, string, string]

// Subject takes the tb-form constructor — the one real backends need.
// A plain constructor adapts with suite.Ctor, still closure-free at
// the call site:
//
// Subject("in-memory", suite.Ctor(ledgermem.New))
// Subject("pebble", pebbletest.Start)
//
// (A union constraint over both function shapes was the first design;
// it does not infer — constraint type inference unifies against the
// constraint's core type, and a union of two function shapes has
// none. Settled by compile on go1.26.5; spec-level, so no toolchain
// bump changes it.)
func Subject[T Ledger](name string, new func(testing.TB) T) SubjectBuilder[T]

// SubjectBuilder keeps the concrete type T for the whole chain. Every
// method is typed against T, so method expressions and concrete
// constructors pass directly; Build lowers into suite.Subject[Ledger]
// exactly once, wrapping each closure across the erasure boundary. The
// lowered inducer's type assertion back to T is safe by construction —
// every instance it can receive originated from this builder's own
// constructors — and a veneer gate test pins that.
type SubjectBuilder[T Ledger] struct{ /* … */ }

func (b SubjectBuilder[T]) OnClock(new func(testing.TB, *clock.TestClock) T) SubjectBuilder[T]

// Induce takes a bare trigger so method expressions fit —
// (*ledgermem.Ledger).RevokeFence — which deliberately omits tb: a
// trigger cannot fail the test directly. A trigger that can fail must
// surface it through the subject; the check reds downstream.
func (b SubjectBuilder[T]) Induce(sentinel error, trigger func(T)) SubjectBuilder[T]

func (b SubjectBuilder[T]) Recover(new func(testing.TB) T) SubjectBuilder[T]
func (b SubjectBuilder[T]) Oracle() SubjectBuilder[T]
func (b SubjectBuilder[T]) Serial() SubjectBuilder[T]
func (b SubjectBuilder[T]) Build() suite.Subject[Ledger] // implicit via RunOpt

// Run composes the full suite and runs it. Subjects, hand checks,
// drops, and Config may appear in any order; validation is Run's,
// loudly, before anything executes.
func Run(t *testing.T, opts ...RunOpt)

// Suite returns the composed data — the primary documented entry for
// anything beyond a wiring file: tooling, custom runners, subset
// backends, CI profiles. The variadic Run above is sugar over
// suite.Run(t, Suite(cfg), subjects…), and the documentation teaches
// the typed form first.
func Suite(cfg Config) suite.Suite[Ledger]
```

**The check index** replaces the flat constant space. Generated named types,
navigated by autocomplete, mirroring the role decomposition the interface
already declares:

```go
ledgertest.Checks.Appender.Append.Smoke     // one check: suite.ID
ledgertest.Checks.Appender.Append.All()     // one method: []suite.ID
ledgertest.Checks.Chain.Links               // one law
ledgertest.Checks.CrossRole.FoldFromCheckpoint
```

**Hand checks** use per-method hooks that fill `Method`, `Class`, and the ID
namespace, so writing a check feels like writing a test:

```go
ledgertest.OnRedact("a redacted entry keeps its receipt resolvable",
 func(tb testing.TB, l ledgertest.Ledger) { … })
```

Per-role runners (`RunAppender`, …) are **deferred to the harness train**:
a subset backend cannot be a `Subject[Ledger]` at all, so each role runner
needs its own subject constructors, config projection, and index scope —
unpriced surface that belongs where subset conformance actually lives.
`Suite(cfg)` plus the role-scoped index covers the in-repo need meanwhile.

### Two generators, one surface

The suite and model plugins remain separate. Each contributes its checks and
its `Config` fields to the shared veneer through eidos slots — the mechanism
built for plugins co-authoring one output. Today's ad-hoc version of this
seam is the model plugin appending extension closures into the suite's
*private* config, which is exactly where the dropped-`opts` and invisible-
clock defects lived; a slot over a public data model makes the seam
inspectable and testable.

Unchanged in role: `ModelProperty` stays exported for bespoke harnesses and
`model.MakeFuzz`; the falsification companions and the saturation prover stay
in `gen_test.go` as self-proof of the generated checks. They prove check
functions; the manifest (below) governs check *presence*.

### Fixtures are generators

Three value systems exist today — the suite tier's derived fixed pairs, the
builder generator's structs, the model tier's rapid pools — and the audit
caught each failing alone: derived seeds violating the carrier's own mixin,
zero-struct fixtures for generics, probe pools disjoint from write pools.
One principle replaces them:

**A fixture field is a generator. A fixed value is the degenerate case.**

```go
model.Just(v)             // pinned — explicit, see below
model.SampledFrom(a, b)   // pool — collision density made visible
patchtest.Colliding()     // arbitrary rapid generator, shrinkable
```

Two consumption modes, one source: deterministic checks take the **canonical
draw** — the first sample at a pinned seed, recorded with the rapid version
in the generated file's provenance header so "fixed" values are reproducible
across machines and honest across upgrades — and model checks take the
distribution. The disjoint-pool class dies structurally: laws, actions, and
checks draw from the same generator object per role.

**Role bindings partition request structs.** The field roles (next section)
split every request struct three ways, and the split is enforced by the type
system, not by documentation:

| Fields | Kind | Supplied by |
|---|---|---|
| stream, payload | drawn | the pool, collision-dense by default |
| seq, prev | stamped | the harness, from chain state — never drawn, so a derived fixture cannot violate the contract it tests |
| everything unbound | pinned | `Config` or a capability |

**Request builders** are the fixture surface — the builder generator's output,
grown generator-valued:

```go
ledgertest.Config{
 Append: ledgertest.AppendReq().
  Stream("s-a", "s-b").             // variadic = SampledFrom: a pool, visibly
  PatchFrom(patchtest.Colliding()). // …From = any rapid generator
  FencePin(fence1),                 // pinned — explicit by name
 // Seq and PrevHash have no method in this position: stamped fields
 // are the harness's to write. Compile-time, not doc-time.
}
```

**Single-value pools on role fields are refused**, not warned about: on a
drawn role field, `Field(v)` with one value is a generation-time red — a
one-value pool vacates every collision-dependent comparison downstream,
which is the unarmed-capability class reborn one layer down, and the audit's
evidence is that soft guards go unread. Pinning is a *different intent* and
carries a different name (`FieldPin`), so the choice is visible at the call
site and in review.

Hand checks get the inverse: `AppendReqRaw()` unlocks stamped fields, so a
deliberately invalid request — the negative test the current surface cannot
express — is three chained calls:

```go
req := ledgertest.AppendReqRaw().
 Stream("s-a").Seq(3).PrevHash(types.Digest{}). // forged on purpose
 Patch(patchtest.One()).Build()
```

A field with no witness and no generator is a **named red** at generation
time — `AppendRequest.Patch: no generator for patchtest.Text; supply
PatchFrom(...)` — never a silent zero struct.

### Placement doctrine and inference

Every claim has exactly one home:

| Fact | Home |
|---|---|
| A role's semantics | the role interface declaration |
| A single method's claim | the method |
| A field's meaning | the request-struct field — usually inferred |
| A property of the composition | the composed interface |
| Emission (`out`, `witness`, `stub`, `suite`, `model`) | the composed interface |
| Embedding lines | **nothing, ever** — inheritance is structural |

A claim on a role interface is written once and inherited by every embedder
and every standalone use; a backend implementing `Appender + Reader` alone
gets those role suites with zero additional annotation.

**Inference before annotation.** Field roles bind by type identity and
naming convention — `types.StreamID` is the stream, `types.Seq` the
sequence, `PrevHash types.Digest` the link, a type-parameter payload the
payload — and directives are the override, not the requirement. Inference is
tolerable only because it is visible everywhere it can bite:

- `testkit explain <type>` prints every binding with its evidence
  (`inferred: type identity` / `declared`), and the run report names
  inferred bindings.
- **A law failure whose fields were inferred carries the inference in the
  failure message** — `stream ← StreamID (inferred: type identity);
  override with //testkit:role` — so the consumer meeting a mis-inference
  red sees the diagnosis at the point of failure, not in a tool the failure
  never mentions. The resolver returns evidence or nothing; this threads
  that evidence to where the stakes are.
- An explicit directive always wins.
- `testkit advise ./...` reads the interfaces and *proposes* candidate
  claims with the law each would arm. Advise never applies anything: a
  wrong claim reds correct code, so a human accepts each line.

### Capabilities: red, with the fix

`Caps` requirements a subject cannot meet fail with the arming instruction:

```text
--- FAIL: TestLedgerConformance/pebble/poison/AUTO-POISON-STICKY
    poison laws need an inducer for ErrFenceRevoked; add
    .Induce(types.ErrFenceRevoked, <trigger>) to subject "pebble",
    or drop with Without(ledgertest.Checks.Poison.Sticky)
```

This is the failure-message bar the generated law failures already meet,
applied to wiring mistakes — where a newcomer actually lives.

### Evolution: the check manifest

Regeneration writes `<pkg>/checks.lock`. The format is versioned and its
escaping rule is enforced at the source:

```text
testkit-checks v1
Append/smoke<TAB>signature/smoke<TAB>Append survives a call with derived inputs
```

One header line, then one line per check — ID, class, claim, tab-separated,
sorted by ID. A claim containing a tab or newline is a generation-time
named red (claims are generator-authored prose; refusal is cheaper and
sturdier than escaping). Adding a column is a `v2` header, never a silent
reshape.

`testkit check` fails a regeneration that **drops** a line unless the lock
file changed **in the same run's write set** — the gate re-runs the pipeline
in memory and knows nothing of commits; keeping the lock update in the same
commit as the regen is review discipline, recorded here as convention and
enforced by the gate one level down. Consequences:

- Silent check-weakening across generator upgrades — a coverage loss no
  consumer can currently see — becomes a red the consumer must acknowledge
  by updating the lock.
- The PR reviewer reads the assertion diff, not the generated-code diff.
- Adding a method is additive everywhere; removing one breaks every typed
  drop that referenced it, which is the loud direction.

**The version-bump runbook** (a rapid or generator bump invalidates every
canonical draw and every lock across a corpus in one regeneration — the
design's first foreseeable incident): `testkit check` distinguishes
**value drift under a version bump** — reported as one summary line per
package, with the old/new versions named — from **check-set drift**,
reported line-level. The reviewer promised "the diff is the assertion diff"
gets exactly that; the bulk fixture churn is folded behind the version
line. The operator procedure lives in the implementation architecture note.

### Worked example: the ledger

The stress case: `Ledger[P patch.Patch, S any, K comparable]`, nine embedded
roles, eighteen methods, request structs throughout, per-stream hash chains,
optimistic sequence CAS, poison-on-fence semantics, epochs. (The example's
types exist nowhere in this workspace yet; landing the interface plus an
in-memory implementation in the corpus is the acceptance fixture for this
RFC — see the implementation architecture's merge order.)

**The complete annotation surface** — eight lines across eleven declarations,
zero on request structs (inferred), zero on embeddings:

```go
//testkit:contract chain            → on Appender, Reader, Verifier (their own files)
//testkit:contract fold             → on Folder
//testkit:mixin monotonic observe=CurrentEpoch   → on EpochController
//testkit:mixin idempotent          → on Close
//testkit:mixin poisonable via=ErrFenceRevoked,ErrIntegrity wraps=ErrPoisoned  → on Ledger
//testkit:out ledgertest/ witness=(P=patchtest.Text, S=string, K=string) stub suite model
```

**The complete wiring file** (under the builder type model above):

```go
func TestLedgerConformance(t *testing.T) {
 ledgertest.Run(t,
  ledgertest.Subject("in-memory", suite.Ctor(ledgermem.New)).
   OnClock(ledgermem.NewOn).
   Induce(types.ErrFenceRevoked, (*ledgermem.Ledger).RevokeFence).
   Oracle(),
  ledgertest.Subject("pebble", pebbletest.Start).Serial(),

  ledgertest.Config{
   Streams: model.SampledFrom([]types.StreamID{"s-a", "s-b"}),
   Patches: patchtest.Colliding(),
  },

  ledgertest.Without(ledgertest.Checks.ShredAttester.AttestCryptoShred.Smoke), // needs HSM

  ledgertest.OnRedact("a redacted entry keeps its receipt resolvable",
   func(tb testing.TB, l ledgertest.Ledger) { /* … */ }),
 )
}
```

**What runs**, by family — the part that discharges the 2.3% problem:

| Family | Laws | Oracle |
|---|---|---|
| chain | per-stream append-only grows / no-drops; PrevHash links; Range replays admission order | `ref.PartitionedAppendOnly`, keyed by the stream role |
| session | ExpectedSeq CAS: one winner per (stream, seq); stale seq refused, state unchanged | `CASAtomicOneWinner` stamped at the cell |
| cross-role | VerifyStreamChain green after any accepted appends; Fold(FromCheckpoint) = Fold(genesis); Redact preserves VerifyStreamChain; entries verify across RotateEpoch | **none** — two-paths-one-answer identities; the subject is its own oracle, off the twin floor by construction |
| poison | after induced fence revocation: every op `errors.Is(_, ErrPoisoned)`, cause preserved, first poison wins | the `Induce` capability |
| observers | CurrentEpoch monotonic under rotation; RootHash moves iff an append lands | model observations — ctx-less methods stop being orphan smoke checks |
| lifecycle | Close idempotent; ops after close fail closed | existing |
| differential | pebble vs the in-memory `Oracle()` | the consumer's own second implementation |

The cross-role row is the depth argument: identities across roles need no
reference implementation, and they only become derivable when the generator
understands the composed interface as roles — which the embedding already
declares.

### The harness layer — a deferred shape sketch

**Status: sketch, not v1 scope.** The Alternatives section rejects designing
the public-conformance API before its first consumer exists; that rejection
stands. This section records the *shape* the deferral is priced against —
merge steps 8–9 in the architecture — so both readers of this document exit
with the same belief: the harness is designed enough to know it fits, and
built only when a third-party implementor exists.

The design principle: the generator knows exactly which capabilities the
suite carries, so it emits a **specialized** harness interface — named
methods, no generics on the implementor, no options:

```go
// Harness is what a third-party backend implements to claim conformance.
type Harness interface {
 // Subject returns a fresh, isolated instance per check. Shared
 // expensive state lives on the harness value the implementor built.
 // Isolation between calls is the implementor's contract — and is
 // itself checked: two Subject() instances must not see each other's
 // writes, derivable wherever a reader exists.
 Subject(tb testing.TB) Ledger
}

// One extension per capability the suite carries, named from the claim.
// Absence is loud: checks needing one fail red naming the interface to
// implement. Never a silent skip.
type ClockedHarness interface { Harness; OnClock(tb testing.TB, clk *clock.TestClock) Ledger }
type FenceHarness   interface { Harness; InduceFenceRevoked(tb testing.TB, l Ledger) }

func RunConformance(t *testing.T, h Harness, opts ...ConformanceOpt)
func Unsupported(f ...Feature) ConformanceOpt
```

Four rules give a conformance claim integrity: **no `Without` in conformance
mode** — the signature has none, so an implementor cannot strip checks and
still claim conformance; **optionality is the publisher's vocabulary** — only
features the publisher marked optional are `Unsupported`-able, skipping with
reason in the report, while an undeclared gap is red; **capability absence is
red with the fix**; and **the run emits a conformance statement**.

The statement is a versioned artifact, owned and versioned by the `suite`
package alongside `Report`, additive-only after first release:

```json
{"format": "testkit-conformance v1",
 "suite": "go.thesmos.sh/ledger/ledgertest",
 "suiteVersion": "v0.12.0",
 "lockDigest": "sha256:…",
 "subjects": [{"name": "postgres", "outcomes": {"passed": 61, "failed": 0,
   "unsupported": ["shred"]}}]}
```

It is **derived from `Report` and the manifest** — the JSON structs, never
the log text. Per-role subset conformance lives here too (M6): a subset
backend implements the role's harness, not a `Subject[Ledger]`.

### Shape independence

The contract splits into two halves with different generality. The
**surface** — `Check[S]`, `Subject[S]`, `Run`, index, hooks, manifest,
report, capabilities — is shape-agnostic: nothing in it knows what a stream
is. The **derivation** is vocabulary-bound. The bound is stated per idiom,
with the mechanism that lifts each row above today's tool (next section):

| Idiom | Surface | Assertion power | Lifted above today by |
|---|---|---|---|
| Request-struct + distinctive types | full | chain, CAS, poison, cross-role | roles (this RFC) |
| Scalar keyed (today's corpus shape) | full | existing derivation, fixtures unified | differential + properties |
| Flat interface, no role embedding | full — index by method; role claims never required embedding | per-method claims | equivalence + differential |
| Generic | full via witness; missing generator = named red | as instantiated | witnesses (this RFC) |
| ctx-less observers | full | observer bindings | this RFC |
| Byte streams, iterators, comparators | full | smoke + hand checks | vocabulary batch (RFC-0005) |
| Bare-typed codebases (`string`/`int64`) | full | zero inference — explicit directives required | differential + properties, which need no roles at all |
| Cross-package protocols | harness bundles the surface | deferred derivation | harness + RFC-0005 |

Every failure direction points to explicit annotation, never to silence: the
resolver refuses bare primitives by design, `explain` shows unbound fields,
and the derived-garbage class (a fixture violating the mixin its own carrier
declares) is unrepresentable under the drawn/stamped/pinned partition.

The `role` keyword set is the one place the contract genuinely leans on the
motivating interface — which is why the first vocabulary was argued from
three unrelated domains before freezing (the exercise and its result are
recorded under Open questions: five keywords, an entry rule, and a
registry policy).

### Raising the floor: better than today on every shape

"Never falls below today" is not the acceptance bar; **above it everywhere**
is. Three mechanisms deliver that, ranked by leverage. The first two are
vocabulary-free — they raise the barest interface as much as the richest.

**1 · The differential oracle.** One word on a subject:

```go
kvtest.Subject("in-memory", kv.NewInMemory).Oracle()
```

marks it the reference. Every *other* subject's model leg then runs
subject-vs-oracle through the existing engine (`model.WithReference` — the
wiring exists; only the derivation is new), instead of subject-vs-twin. The
in-memory implementation nearly every consumer writes anyway becomes the
oracle for the real backend, at zero additional code.

This retires the twin floor as the *default fate*: today 92 of the corpus's
114 model-emission references ride the twin (92 twin / 22 derived) — blind
to any deterministic bug, because both copies are broken identically. Under
the differential, the twin is the fallback only for single-subject runs, and
the report says which tier ran. Rules: at most one `Oracle()` per run (two
is a validation failure); the oracle subject itself still runs the full
deterministic suite and its own twin leg; differential legs are marked in
the report.

Pairing is never automatic — declaration-order pairing would give two
textually-adjacent subjects different meanings by position, the silent-
semantics class this RFC exists to kill. The consumer who never reads the
report is served instead by the red-with-fix philosophy in its green form:
a run with two or more subjects and no `Oracle` prints one line in the log
summary — *"2 subjects, no Oracle(): model legs ran on the twin floor; mark
your reference subject .Oracle()"* — a nudge at the exact moment of the
exact omission. Teams that want enforcement get a future additive
`RequireOracle` gate, never inference.

Honest hazard, with its policy: a live backend can diverge from in-memory on
legitimate nondeterminism (ordering, timing). A differential red therefore
carries the same seed/rerun line every model failure carries, and the leg's
flake policy is: **a divergence that does not reproduce under its own seed
is reported as a flake with both readings** — a real bug or an over-strong
claim to loosen (`eventually`, `unordered`) — never silently retried.

**2 · The equivalence family.** Two-paths-one-answer laws derived from
*shape pairs*, needing no oracle and no domain vocabulary:

| Pair | Law |
|---|---|
| batch / single (`AppendBatch` / `Append`) | the batch is observably equal to the loop; receipts correspond |
| ranged / point (`Range` / `Entry`) | the stream replays what point reads answer |
| iterator / re-iteration (pure readers) | two traversals agree; early stop does not poison the next |
| write / observe (any writer beside any reader) | read-back, generalized past keyed stores |

Pairing derives from role signatures and slice-of-request relationships;
`//testkit:equiv batch=AppendBatch single=Append` overrides where inference
cannot see it. These land in the index under `Checks.CrossRole` and are the
generalization of the worked example's cross-role row: the subject is its
own oracle across two paths, which sidesteps the reference problem entirely.

**3 · Property hand checks.** Consumer extensions get the engine's power,
not just its report:

```go
kvtest.PropOnAppend("no accepted append is ever lost",
 func(tb testing.TB, l kvtest.Ledger, req ledger.AppendRequest[…]) {
  // req is DRAWN from the request generators — valid by
  // construction, collision-dense, and SHRUNK on failure.
 })
```

`On<M>` writes an example; `PropOn<M>` writes a property over the same
fixture generators, with rapid's shrinking. A consumer's own claims stop
being example-weak the moment they matter.

**The vocabulary batch is committed scope, not deferred hope.** Byte
streams (`n <= len(p)`; short read implies error; zero-length in, `(0, nil)`
out), iterators, comparators (strict weak ordering), close-once (displacing
the poison misclassification on `Close() error`), and index domains ship as
a companion RFC-0005 on the same release train — detectors upstream in
eidos, derivation here, with the ledger and the twelve stdlib interfaces
from the audit as its acceptance corpus. RFC-0004 does not block on it;
consumers feel the two vocabulary-free mechanisms first.

### Extension proof: a simulation generator

The tiers audit proved a table by walking an input that does not exist yet
(classification #90). Same exercise here: pretend `generator/sim` exists —
FoundationDB-style deterministic simulation — and route every requirement
through this contract. The point is not to design the sim generator; it is
to demonstrate the contract carries a consumer it was not shaped around.

| Sim requirement | Contract mechanism | Verdict |
|---|---|---|
| A storm appears in the suite, droppable, manifested | a storm is a `Check[S]` (`Class: "sim/…"`, `RunWith`) | paved — checks-as-data |
| A third generator contributes checks and config | the same slots model and builder use; N contributors is the mechanism, not an extension of it | paved |
| Seeded workload of valid operations | request generators + the drawn/stamped/pinned partition — collision-dense, valid-by-construction op streams whose session fields stay coherent under the storm | paved |
| Simulated time | `Caps.Clock` + `OnClock`, red-with-fix unarmed | paved |
| Fault injection | `Caps.Induce` + fault stubs composed at subject construction; scheduled seed-driven faults are a richer `Caps` entry — additive struct field | additive |
| Post-storm invariants | the law registry, the equivalence family, and the differential oracle: recover, then diff against the `Oracle()` subject — the in-memory implementation is the specification the recovered state must match | paved |
| One seed reproduces the world | rapid choice streams + the canonical-draw provenance rule | paved |
| Deterministic scheduling | — | correctly absent: engine work, a sibling of `engine/model`. The contract carries the resulting checks and constrains nothing about how they execute inside |
| Crash / restart over durable state | `Subject.Recover` + `Check.RunWith` | closed by this RFC's amendments — birth-only lifecycle was the one genuine gap |

What the walkthrough decided elsewhere in this document: `RunWith` exists
because orchestrating checks need constructors, not an instance (clocked
laws are the precedent); `Recover` exists because crash-recovery is a
constructor variant like `OnClock`, and adding it is one field today versus
a breaking v2 conversation later — with its semantics explicitly reserved
for the sim RFC.

What it deliberately leaves to the sim RFC: the semantics of "crash" for an
in-process Go value. Abandoning an instance without `Close` only simulates
process death when durability is genuinely external; an in-memory subject
holding its state in pools "crashes" into garbage collection, not into a
recovery scenario. That is the sim generator's hard problem and it is
orthogonal to this contract. The existing simulation design note
(`docs/superpowers/sim.md`, May) predates this contract and must be
reconciled against this table when the sim RFC is written — a hook it
assumed that this contract does not carry is a finding against one of the
two documents.

### Migration

Breaking, one release, pre-v1. The mechanical mapping:

| Today | Under this RFC |
|---|---|
| `AssertXContract(t, opts…)` | `xtest.Run(t, opts…)` |
| `XSubject(name, func() X {…})` | `xtest.Subject(name, suite.Ctor(ctor))` — still closure-free at the call site |
| `XWithout("Method/check")` | `xtest.Without(xtest.Checks.Method.Check)` |
| `XOnMethod(name, fn)` | `xtest.OnMethod(name, fn)` — unchanged shape |
| `XWithFixture(fx)` / `XSeed(fn)` | `xtest.Config{…}` with request builders; seed deleted — checks arrange through their own fixtures |
| `XModel(XModelClocked(f), …)` | `SubjectBuilder.OnClock` + `Config` fields |
| standalone `AssertXMethodCheck` | unchanged — checks remain exported values |

Regeneration emits the new surface; consumer wiring files are rewritten by
hand (they are small — that is the point) or left on the last pre-contract
tag until ready. Rollback from the cutover is cheap and stated: revert the
release commit, re-tag the previous minor, regenerate back — pre-v1, no
consumer contract is broken by the round trip. `suite` itself is **held at
v0 until two consumers are wired against it** (the corpus and one external
repo); the v1 tag is a scheduling decision taken after that evidence, not
with this RFC.

## Alternatives

**Hardened options** — keep the funnel; add typed IDs, validate drops, error
on unarmed clocks. Cheapest, additive, and its strongest form fixes the
enumerated defects one by one (run-start drop validation, `b5c24be0`, is
this alternative's first installment, landed). **Why not:** it keeps the
private config that made write-only knobs and dropped opts possible, keeps
the per-interface vocabulary that makes every suite a relearning exercise,
and offers no data layer for the report, the manifest, or any future
tooling. It repairs the instances and preserves the class.

**Merge the suite and model plugins** — one generator, no slots, no seam.
The strongest case: the three slot contracts are this design's most novel
machinery, and deleting the plugin boundary deletes them. **Why not:** the
boundary is load-bearing three ways. `generator/builder` is a third
contributor regardless (the fixture surface), so slot composition exists
whether or not suite and model merge — the merger deletes one seam, not the
mechanism. `//testkit:suite` without `model` is a first-class configuration
today; a merged plugin either preserves that split internally (two
generators in a trenchcoat) or forces the rapid dependency graph onto
suite-only consumers. And ADR-0018's tier ownership — the gates census
which tier owns each classification — keys on the boundary; merging blurs
the very line the emission and conduct gates enforce. The slots stay, with
a mandate earned rather than inherited.

**A harness interface** (gocloud `drivertest` shape) as the *primary*
contract — implementors provide `Harness`; one exported `RunConformance`.
The right shape for third-party conformance publishing. **Why not now:** no
third-party implementor exists yet, and the harness is a thin additive layer
over the suite value when one does (the deferred sketch above). Designing
the public-conformance API before its first consumer is how the current
surface got its shape.

**A testify-style suite struct** — checks as methods on a generated struct,
composed by embedding. Familiar, IDE-navigable. **Why not:** method sets
compose by embedding only along one axis, two generators cannot co-author
one struct without slot-composing *source*, and checks-as-methods have no
data representation — no report, no manifest, no `Without` without
reflection over method names, which is the stringly problem wearing a
different coat.

**Struct tags for field roles** (`` `testkit:"stream"` ``) instead of
comment directives. Fair strengths: Go-native, runtime-readable, compact.
**Why not:** the directive pipeline is comment-based with axis qualifiers
(ADR-0008), tags cannot carry a qualified claim vocabulary, and a second
annotation syntax for one axis splits the language. Inference removes most
of the burden tags would have carried.

## Drawbacks

- **The vocabulary freezes at its least-proven moment.** Everything in the
  frozen inventory (What is decided) becomes semver-bound API on first tag.
  Structs keep fields additive, but a shape mistake is carried to v2. The
  mitigations: the inventory is enumerated rather than estimated, `suite`
  holds at v0 until two consumers are wired, and the hand-built spike
  (architecture, step 1) runs the data model against a real corpus package
  before anything else is funded.
- **Veneer volume scales with the interface.** Per method: ~5 ID constants,
  one index type, one hook, plus the subject builder and request builders.
  For the 18-method ledger that is roughly 90 IDs, 20+ generated types,
  18 hooks — order 1–2k generated lines beyond today's emission. Acceptable
  only because every symbol is data-shaped and thin; it is still real regen
  diff.
- **One variadic `Run` defers shape errors to run start.** Zero subjects,
  duplicate names, conflicting configs are runtime failures — loud and
  immediate, but not compile-time. The typed `suite.Run(t, s, subjects…)`
  form is co-exported and documented as the primary entry for anything
  beyond a wiring file.
- **Canonical-draw reproducibility couples to rapid.** A rapid upgrade that
  changes generator streams changes every "fixed" fixture value; the pinned
  seed and rapid version in the provenance header are load-bearing, the
  drift check compares them, and the bump runbook (Evolution) is the
  operator procedure. A cross-machine determinism check (two platforms, one
  seed, compared digests) gates the "reproducible across machines" claim
  before it ships in a header.
- **The manifest adds ceremony to every intentional check change** — a lock
  update per weakened or renamed check. That is the point, and it is still
  friction.
- **`role` inference can misfire** — a `types.Digest` field that is not a
  chain link, bound as one, is a red on correct code: the worst class the
  audit found. Containment is layered: inference binds only on type
  identity declared in the same module (hard), the inference evidence rides
  in the law failure message itself (M9 — the consumer sees the diagnosis
  at the red), and a directive always wins.
- **The `role` keyword set is new eidos surface.** The coupling the audit
  called a co-development loop grows by one vocabulary that must be
  versioned with everything else.

## What is decided

- A public data model in `go.thesmos.sh/testkit/suite`, in the root module,
  adding zero dependencies.
- **The frozen inventory, enumerated** — what the first tag binds:
  - types: `ID` (with its grammar and family registry), `Class` (with the
    axis-qualification rule), `Caps`, `Check[S]`, `Subject[S]`, `Suite[S]`,
    `Report`;
  - functions: `Run`, `Merge`, `Ctor`; methods: `Suite.With`,
    `Suite.Without`;
  - the field contracts of `Caps`, `Subject` (including `Oracle`, `Serial`,
    and `Recover`-with-reserved-semantics), and `Check`
    (`Run`/`RunWith` exclusivity);
  - the five `signature/…` `Class` constants;
  - the `checks.lock` format (`testkit-checks v1`, tab-separated, refused
    tabs/newlines in claims);
  - the `Report` JSON encoding (`testkit-report v1`, additive-only);
  - the conformance-statement schema (`testkit-conformance v1`,
    additive-only), owned by `suite`, emitted by the harness train.
- Deliberately absent from v1, additive later: `Wrap`, `Only`,
  `WithoutClass`, per-role runners (harness train).
- Generated packages emit a veneer of constructors over the data model:
  one tb-form `Subject` (plain constructors adapt via `suite.Ctor`,
  closure-free at the call site), the `SubjectBuilder[T]` chain lowering
  once across the erasure boundary, one variadic `Run` as sugar over the
  typed `suite.Run` (the documented primary), per-method `On`/`PropOn`
  hooks, a nested check index, `Config`, request builders, and a
  witnessed alias.
- Unknown drops fail compilation (generated) or fail `Run` naming the known
  set (hand-built); unmet `Caps` fail red with the arming instruction;
  skips never model missing capability; combinator misuse follows the
  validation table.
- The suite and model generators stay separate plugins slot-composing one
  veneer (argued in Alternatives); `ModelProperty` and the self-proof layer
  keep their roles.
- Fixture fields are rapid generators; deterministic checks take the
  canonical draw under a pinned seed recorded in provenance; request
  builders expose `Field(a, b, …)` / `FieldFrom(gen)` / `FieldPin(v)`;
  single-value pools on drawn role fields are refused at generation;
  stamped fields are absent from harness-position builders and present on
  `…Raw`.
- Claims live at the role interface; field roles infer from type identity
  with directive override; embedding lines carry nothing; `explain` renders
  every inference; inferred-field law failures carry the inference
  evidence; `advise` proposes and never applies.
- Regeneration maintains `checks.lock`; a dropped line without a lock
  change in the same run's write set fails `testkit check`; version-bump
  value drift reports as a per-package summary, distinct from line-level
  check-set drift.
- Classification-derived `Class` strings are the axis-qualified eidos
  names; their typed constants generate into the veneer; the strings are
  what freeze, and they are stable because the axis prefix namespaces them.
- The migration is breaking, one release, with the mapping table and the
  stated rollback; `suite` holds at v0 until two consumers are wired.

## What the design rests on

- eidos slots can compose plugins' contributions into one veneer file with
  stable ordering — the mechanism exists and is exercised, but not yet at
  this granularity. **De-risked before funding**: the architecture schedules
  a two-toy-plugin slot spike ahead of the step that depends on it.
- The subject constructor question is **settled by compile, not spiked**:
  the union-constraint form fails on go1.26.5 with `cannot infer T` at
  both call sites — constraint type inference unifies against the
  constraint's core type, and a union of two function shapes has none, so
  no toolchain bump changes it. The design is the tb-form single signature
  plus `suite.Ctor`, verified compiling.
- rapid's generator streams are deterministic under a pinned version and
  seed, making the canonical draw stable — verified for the pinned version,
  cross-checked on two platforms, re-verified by the drift check on every
  bump.
- Type-identity inference is decidable because role types (`types.StreamID`,
  `types.Seq`) are declared types, not aliases of primitives, in the
  consumer's own module.
- The engine's law catalogue already covers the worked example's families
  (chains, CAS, poison, lifecycle); this RFC adds derivation and surface,
  not laws — except the cross-role identity family, which is new derivation
  over existing runners.

## Open questions

None remain open. Every question the draft and review periods carried is
resolved below, with the reasoning recorded so the debate survives.

**The `role` keyword set — resolved by the three-domain exercise the RFC
demanded.** A *queue* maps stream→topic, seq→offset, payload→message: no
new keyword. A *cache* does not map — its identity is point-addressed, not
sequence-addressed — which forces **`key`** as a first-class keyword (the
corpus agrees: cas, cache, kv are keyed shapes, and `key` is what the
equivalence family's write/observe pairing needs to generalize past the
ledger). A *workflow engine* maps id→key, step→seq, state→payload: no new
keyword. The first vocabulary is therefore **`stream`, `seq`, `prev`,
`payload`, `key`** — the four the ledger forces plus the one the other
domains force. `fence`, `epoch`, `checkpoint` are held: one domain each,
and a keyword addition is a row change later while a wrong keyword now is
frozen inference surface. The entry rule, written down: *a keyword enters
when a second unrelated domain needs it or a law field cannot bind without
it.* Addition policy: **registry with census-or-red discipline** (the
`tiers` precedent), not RFC-per-keyword — a keyword only arms inference,
which directives override and evidence traces. Two carve-outs stay
RFC-gated: a new *kind* (beyond drawn/stamped/pinned) changes the partition
semantics, and the ID *family* registry is consumer-visible string surface
in lock files — a different bar.

**The optionality directive — resolved by the placement doctrine.**
`//testkit:optional` sits **on the role interface, beside the contract
claim**: optionality is a semantic of the role's claim, and "a role's
semantics: the role interface declaration" gives it exactly one home; a
separate feature file would be a second home for a role fact. It inherits
correctly — every embedder gets it for free. Granularity: roles and mixin
claims, never individual checks — the check is the unit of assertion, the
role is the unit of capability. The conformance statement renders a
**list, not a tier** (`"unsupported": ["shred"]`) — the additive-safe
primitive; a publisher can define named tiers on top of the list later
without freezing anything today. The harness train implements this; the
doctrine is decided now so the train cannot reopen it.

**Automatic differential pairing — resolved: never.** Declaration-order
pairing gives adjacent subjects different meanings by position — silent
semantics, the class this RFC exists to kill. The consumers-never-read-
the-report worry is answered by the green-form nudge specified in the
differential section; enforcement, if a team wants it, is a future
additive `RequireOracle` gate. Meaning stays explicit.

**What `Recover` recovers from — resolved as a bounded reservation.** The
name commits today to exactly one thing (acknowledged effects are
observed; the in-flight frontier is implementation-defined), recorded in
the field's doc. The sim RFC owes the rest **as a law pair, not prose** —
recover-after-clean-Close ≡ identity, recover-after-abandon ⊇ acknowledged
writes — because laws are testable and manifest-able. `Crash` stays out of
the contract: abandoning-without-Close is already expressible by an
orchestrating `RunWith` check through the constructors it holds, so a
`Crash` capability adds surface without power until the sim engine defines
real fault points — at which point it is an additive `Caps` entry the
extension-proof table already prices.

Resolved earlier in review: single-value pools on drawn role fields are
**refused**, with `FieldPin` explicit — soft guards go unread. The subject
constructor is the **tb-form single signature plus `suite.Ctor`** — the
union constraint was the first answer and does not infer (settled by
compile; see What the design rests on); one name survives, via adapter.
Classification `Class` strings are the **axis-qualified eidos names** with
veneer-generated constants — the strings freeze into lock files, so the
namespace prefix is what makes them safely stable.

## Amendments

2026-08-14, from the engine census
([engine-revision.md](../internal/engine-revision.md)). The RFC was accepted
before the census ran; three of its findings change consumer-visible
semantics and are recorded here rather than edited into the design above.

**A1 — the differential and the laws are separate legs.** The engine's
`LawsOnly` doc records the problem: the differential compares every call
against the reference and aborts on divergence, so a broken subject dies at
step 0 and the laws never run. As accepted, `.Oracle()` therefore masks
every law exactly when a real reference exists. The fix is structural: the
model slot emits **two checks** — a differential leg that carries no laws,
and a laws leg that drives its own sequences and takes no per-call
differential verdict. Both always run. The Report marks each leg's tier as
before. No flag; the separation is the check structure.

**A2 — `vacuous` joins the outcome set.** Outcomes are
`passed | failed | unmet | vacuous`. A check whose preconditions never
engaged (the engine's `law.Vacuous`: the poison that did not take, the
write that never errored) reports `vacuous`, not `passed`. This lands
before `testkit-report v1` freezes, because the encoding is additive-only
after that and an outcome consumers switch on is the worst place to add a
value late. The engine's `Registry` census (`ran`/`vacuous`) is the
producer for law legs; a hand check reports it through the same channel.

**A3 — the concurrent leg gets the flake policy the differential leg has.**
A concurrent failure's seed reproduces the drawn inputs, not the goroutine
interleaving, so it may not reproduce at all. Policy: a concurrent red that
does not reproduce under its own seed is reported as a flake with both
readings — a real interleaving bug or an over-strong claim — never silently
retried, and never presented as deterministic. Porcupine's `Unknown` stays
a failure, with load bounded against the timeout (engine-side fix).

Two mechanism notes, no design change: the canonical draw is implemented by
rapid's `Generator.Example(seed)` — the "What the design rests on" bullet
about stream determinism now names a shipping API rather than a property we
hoped for; and `rapid.MakeCustom{Fields: …}` is the intended mechanism for
deriving request-struct generators, with role fields overriding the
reflected defaults — a field only goes named-red when reflection cannot
terminate on its type and no generator was supplied.

2026-08-15, from the improvement programme's Tier 1 close. Five changes, all
pre-freeze, all inside the accepted design. Four are field shapes decided
before `testkit-report v1` and the ID grammar stop being free; the fifth
changes what the product claims.

**A4 — the report carries falsifiability.** `Report` as accepted says what
passed. It cannot say what was *able* to fail, and both tiers already compute
exactly that: the suite tier's falsification companion drives every generated
check against a stand-in built to break it, and the model tier's saturation
prover requires the kill to come from a defect of the law's own declared
class. Neither datum reaches the consumer, so `44 legs, 44 passed` does not
distinguish a suite that works from one that asserts nothing — which is the
premise of the tool, stopping at the repository boundary.

Each check reports `proven | argued | unproven`, with the argument where there
is one. The conformance statement then reads "64 checks, 61 proven able to
fail, 3 argued". Argued is not a weaker pass: the model tier's own registers
show three shapes of it — the wardrobe cannot produce the defect the law is
named for, no wear reaches the law, or the claim needs a value only the
consumer has.

**A5 — an ID's sub-segments are slugs, not sentences.** As drafted, a
generated ID is the claim in prose: `Put/reports a cancelled context`. The
typed constant protects the drop *site* from regeneration; the string is what
lands in `checks.lock` and the report, and prose is edited. Tier 1's
falsification work reworded five generated check messages in one change — the
same generator writes both, so a claim rewording and an ID rewording are one
edit, and one of them was going to be frozen.

The `label` production becomes `1*( lowercase / DIGIT / "-" )`. `Claim`
already carries the sentence and stays free to improve forever. Hand-written
IDs take the same grammar, which `Run` already validates.

**A6 — `Caps` is a keyed set, not a struct of two fields.** The accepted
`Caps{Clock bool; Induce error}` encodes two capabilities. The engine tracks
six kinds of door today — the conformance gate's unarmed-door register keys on
`<law-id>.<field>` and holds `clock`, an induction sentinel, and four
consumer-supplied values (`balanced`, `history`, `required`, `expected`). A law
needing an expected multiset cannot declare that need through two fields, so it
either declares nothing — the silent-green class this RFC exists to kill — or
`Caps` grows one field per door and is a registry with extra steps.

`Caps` becomes `map[Capability]any` with `CapClock` and `CapInduce` as the
first two entries and the registry open under the same census-or-red
discipline the role keywords take. The failure semantics are unchanged: a
capability the subject cannot meet fails the check with the arming
instruction.

**A7 — the outcome set is a shape, not an enum.** A2 added `vacuous` before
the freeze because an outcome consumers switch on is the worst place to add a
value late. It will not be the last: the model tier has since learned to
distinguish a law whose defect class no wear produces from one no wear
reaches, and A4 adds falsifiability.

So the frozen part is three **dispositions** — `passed`, `failed`, `notrun` —
and every finer answer is a `reason` in an open namespace: `vacuous`, `unmet`,
`unprovable`, `unreached`, whatever the engine learns next. Consumers switch on
three values forever and the richness accumulates where addition is free.
`unmet` and `vacuous` from A2 become reasons under `notrun`, which keeps the
distinction the POC learned the hard way — a capability the subject cannot
provide is a different finding from a claim whose preconditions never engaged.

**A8 — `checks.lock` records the classification vocabulary it was written
against.** `Class` strings are the axis-qualified eidos names, a namespace
consumed whole and not owned (ADR-0004). It grows — 72 classifications when
that ADR was accepted, 104 now — and growth is safe. A rename upstream is not:
it changes a frozen string in every consumer's lock file with nothing to
detect it. The lock file records the eidos version, so a rename surfaces as a
migration somebody can attribute.

2026-08-15, second pass. One change, from writing a second interface into the
worked example.

**A9 — the veneer is a namespace value per interface, not package
functions.** The design above names the veneer's entry points after the
concern — `Suite`, `Run`, `Checks`, `Config`, `Subject`, `Without` — and the
example proved that reads beautifully for one interface and does not compile
for two. The surface shipped today avoids that by prefixing every symbol with
the interface name, which is the "~30 exported symbols per generated package,
relearned per interface" this RFC's own Problem section calls a defect. The
example had quietly traded the defect for a compile error and neither document
said which happens at scale.

Everything callable becomes a method on one exported value per interface —
`StoreSuite.Run`, `StoreSuite.Checks.Put.Smoke()`, `StoreSuite.On.Put(...)`,
`StoreSuite.Assert.GetMiss(...)`. Four things stay outside it, each for a
reason that is not taste:

- the witnessed instantiation (`Store`), because a `RunWith` check names
  `suite.Subject[kvtest.Store]` in a signature;
- the subject constructor (`StoreSubject`), because it is generic in the
  concrete implementation type and Go has no generic methods — the same
  constraint that makes the subject builder exist;
- the config and fixture types, because a consumer overrides a pool field by
  name.

Measured on the example: **34 exported symbols for one four-method interface
became 5, and two interfaces cost 9 rather than 68.** Twenty cost roughly
ninety rather than roughly seven hundred — four or five names each, against
thirty-four each — and a reader learns the shape once instead of per
interface.

Two consequences worth stating. Generated *files* split by concern and not by
interface — Layout composes a filename from the source file, so twenty
interfaces declared in one `.go` file produce one checks file holding all
twenty, and per-interface files need a per-interface `//testkit:out`, which is
the consumer's call. And ID scope segments are method names, so two interfaces
in one package that share a method name would collide in the ID space; the
example asserts they do not rather than leaving it to be discovered, and the
qualification rule is owed by whichever RFC first needs it.

## Deferred

- `Wrap` — reintroduced with the fault-injection story that names its
  consumer; adding a method to `Subject` later is non-breaking.
- `Only` / `WithoutClass` — reintroduced with the pre-commit fast tier.
- Per-role runners — the harness train, where subset conformance lives.
- The pre-commit fast tier itself (`Only`-based profiles, budget-capped
  model runs).
- `FireRate` integration into the report — the vacuity instrument is its
  own program (the depth work), not part of this contract.
- `testkit advise` beyond proposal — auto-fix mode is explicitly rejected
  for now: a wrong claim reds correct code.
- Cross-package protocol derivation (the harness *bundles* the surface;
  deriving laws across it is RFC-0005's).

## Evidence

Every number in the Problem section, with the command that produced it,
runnable at the commit this RFC cites. The full measurement narrative is the
2026-08-13 platform audit (`docs/superpowers/platform-audit.md`, local
register; committing it alongside this RFC is recommended so the chain of
custody survives the branch).

| Claim | Derivation |
|---|---|
| 3 phantom drops (pre-`b5c24be0`) | audit P-04 census; `XWithout` call sites vs emitted check set |
| write-only `clock` field | `grep -n 'cfg.clock' <pkg>/iface_suite.gen.go` — assignment only, no read |
| `ContractModelConcurrent` ignores opts | `castest/iface_model.gen.go:138` — `opts` unused in body |
| 122 checks green with TTL deleted | audit probe P-06, executed in a scratch module |
| ~30 exported symbols per package | `go doc <pkg> \| grep -c '^func\|^type'` |
| 2.3% claim-derived on real interfaces | 87-check census over 12 transcribed interfaces; quartet vs semantic split |
| 92 twin / 22 derived of 114 model references | `grep -rh 'Reference:' --include='*_model.gen.go' conformance/corpus \| sort \| uniq -c` |
