# Model

> **Status: shipped.**
> [RFC-0003](../../rfc/0003-the-projection-consumers.md) fixes this
> generator's design. What ships today: the directive, the differential
> property (actions, pools, derived references with their mixin
> refinements, law bindings), the isolated-law walk, the test clock and
> the clocked law family, generic interfaces at their witnesses, the
> concurrent Porcupine leg, the fuzz target, the generated companion that
> proves the emission — the reference's self-conformance, the inert-body
> probes, and the mutation kill matrix — and the consumer options. Where
> this page and the RFC differ, this page reflects what shipped: the RFC
> records the design, including the bounded exhaustive search that was
> since deleted as dead capability.

The `model` generator binds the classifications
[ADR-0018](../../adr/0018-one-tier-owns-each-classification.md) assigns to
the model tier onto the shipped `engine/model` runtime: property-based
state-machine testing on [`pgregory.net/rapid`](https://pgregory.net/rapid),
linearizability checking on
[`anishathalye/porcupine`](https://github.com/anishathalye/porcupine),
and a mutation self-check that proves the bound laws can kill injected
bugs. Consumers get law-backed conformance for one
directive; everything below is derived from the shape stamps and the
[suite generator](suite.md)'s projection.

## The directive

```go
//testkit:model
type Store interface { ... }
```

Interface-scoped, negation denied — the tier exists where one is declared,
and deleting the line is the suppression, the same shape as
`//testkit:stub` and `//testkit:suite`. Three keys: `ref=` names a
reference constructor where no shipped oracle models the shape, `gen=` a
generator constructor for a value type reflection cannot draw, and
`witness=` the concrete types a generic interface's property runs at —
comma-separated, one per type parameter, in declaration order. The witness
list is required exactly where the interface is generic: the pools, the
reference and every law assert *through* those types, so the source names
them or the generator refuses with the key that would fix it. Everything
the file emits then lands at the witnesses — the instantiated subject
type, the pools, the derived oracle — and the option plugs into the
consumer's matching instantiation of the harness. The generator emits where the
directive stands and `suite` queued a projection; a directive on an
interface `suite` never touched, or one where no classification maps to a
law, is a diagnostic at the directive.

The directive is what admits the dependency: the primary output imports the
`engine` module, and through it rapid and Porcupine — a requirement a
classification line alone must not impose. On an interface carrying
model-owned classifications and no `//testkit:model`, the suite harness
header names the tier as unarmed and the line that arms it, so an owed
assertion is visibly waiting rather than silently absent
([ADR-0017](../../adr/0017-every-classification-owes-an-assertion.md)).

Once armed, the laws run inside the ordinary `Assert<Iface>Contract` entry
against each subject's plain form; drop by path with
`<Iface>Without("model/AUTO-…")`, or delete the directive to shed the
emission and the dependency together.

A second directive is declared here: `//testkit:domain-gen <Type> <Func>`
registers a [`rapid.Generator`](https://pgregory.net/rapid) for an opaque
domain type the generator cannot synthesize from reflection. An opaque
parameter with no hint is a diagnostic at the parameter.

## What one interface gets

- **A law registry** — one binding per mapped classification, instantiated
  at concrete types. Law fields fill by a fixed taxonomy: role closures
  from the stamped method, generators from the shared derivation, constants
  from the classification's own parameter stamps, trace handles from the
  runner.
- **An action set** — one `engine/model/action` constructor call per
  method, matching its detector or contract; partner-role methods are
  excluded (the suite tier owns their checks).
- **Generators, derived once** — keys from a small sampled set, values
  blending the fixture pair with arbitrary `model.Make` draws, the value
  type's key field pinned to the key pool, shared by every path.
  Collisions are what make the laws fire; the wide bodies are what makes
  a same-key overwrite carry a body no fixture spells. A claim that
  narrows the accepted value domain (`validates`, `sample`) keeps the
  pool to the proven fixture pair, and the header says so.
- **A derived reference** — an adapter over the matching
  `engine/model/ref` oracle: the map for value-carries-key stores, the
  keyed store for key-beside-value writers, the collection for
  append-and-drain — each refined by claim (`noduplicates` and `crdtmerge`
  dedupe, `sticky` pins resolutions, `snapshotisolation` and `chain` force
  the log over the upsert inference). A contract claim outranks the
  shapes: where its role vocabulary resolves completely — the carrier's
  `role=`, the partner keys — the contract's own store stands in
  (`lease` → `LeaseTracker`), its constructor sentinels minted or
  lenified per the claims, and an oracle whose every sentinel lenifies
  away falls to the twin, because the kill matrix proved a
  never-disagreeing store checks nothing. Where no store models the shape, the
  **twin floor** stands in: a second instance from the subject's own
  factory, which catches nondeterminism and hidden shared state but not a
  subject wrong the same way twice — the header says why the floor was
  reached, and `ref=` raises it. The sequences drive only what the oracle
  models; a method the adapter holds inert is skipped by name.
- **A clock, where a claim reads time** — the `ttl`, `windowed`,
  `timeout` and `scheduled` families bind laws that age entries, roll
  windows and fire schedules. They arm only under
  `<Iface>ModelClocked(func(clk *clock.TestClock) T)`, which builds the
  subject on the run's own `TestClock`; the clocked laws advance it and
  nothing else does. A clocked run forces the twin reference even where a
  map oracle derives — under the clock the derived oracle lies, because
  mirrored writes age on the subject alone — so twins on one clock age
  and fire together. The scheduled law mirrors every accepted schedule
  onto the reference and asserts *at least* its batch fired after the
  advance: the ambient action stream schedules beside it, so exact-count
  is the quiescent claim the fixture's unit tests keep.
- **An isolated walk, where a law corrupts** — the close/poison/tamper
  family (`CursorCloseIdempotent`, `IdempotentLifecycle`,
  `PoisonConsistent`, `TamperEvident` and kin) runs once per iteration
  against a throwaway pair from the subject's factory; the shared pair
  never meets them. A law whose precondition the run's draws cannot
  satisfy reports vacuously, counted apart from a pass — a law vacuous on
  every check past the census floor is named in the run's log, because
  sixty vacuous returns are sixty times a binding asserted nothing.
- **A concurrent path** — where the unrefined map pair derives,
  `<Iface>ModelConcurrent` runs four workers interleaving the reader and
  writer over the same shared pools, Porcupine-checking the history
  against `linearize.KV` per key. It registers beside the sequential leg
  as `model/concurrent`; the laws stay sequential, whose step boundary
  they need, and the companion holds the leg to the derived reference.
- **A report header** — the generated docblock is a per-method table of
  what the run derived: actions, law IDs, the cluster map, what was skipped
  and the option that arms it.

## The self-checks

`_model.gen_test.go` carries the emission's own proofs:

- **`Fuzz<Iface>Model`** — the interface's *sequence space* as one fuzz
  target: `model.MakeFuzz` lets the coverage-guided engine drive rapid's
  choice stream, hunting for the action ordering that breaks a law. One
  target per interface.
- **The mutation kill matrix** — `Test<Iface>ModelKillsInertMutants`:
  one mutant per driven method, each a reference whose one method answers
  zeros and forwards nothing, and the property must fail every one. An
  unkilled mutant means that method's participation checks nothing — a
  hole in the derivation, named by method, asserted at a total kill rate.
  Each probe runs under a named failure surrogate and removes the
  failfiles and artifacts its expected failure provokes.

The last two are emitted only where the reference is derivable — both
certify subject-versus-reference agreement. All three, plus the concurrent
path, honour `testing.Short()`; the sequential law run does not skip,
because it is the assertion a declared classification owes.

## What a consumer writes

Nothing beyond the suite wiring: pass `<Iface>Model()` to the contract
entry. The options are the escape hatches: `<Iface>ModelReference(factory)`
replaces the derived oracle, `<Iface>ModelValues(gen)` replaces the values
pool wholesale, `<Iface>ModelClocked(build)` hands the subject the run's
test clock where a claim reads time, the `gen=` directive key names a
generator constructor in the routed package for a value type reflection
cannot draw (a pointer payload, an invariant-carrying domain type), the
`ref=` key names a reference constructor where no shipped oracle models
the shape, and `<Iface>Without("model")` declines the tier. `<Iface>ModelFuzz(f, factory)`
is the one-line fuzz wiring: the fuzzer's bytes replay as rapid's choice
stream over the subject's own branches.

## Layout conventions

| Tag | Suffix | Contents |
|---|---|---|
| *(primary)* | `_model.gen.go` | Report header, generators, adapter, law registry, action set, concurrent config, options, extension registration. |
| `test` | `_model.gen_test.go` | `Fuzz<Iface>Model`; the self-conformance and mutation tests where a reference is derivable. |

## See also

- [Suite](suite.md) — the projection and registry this generator reads, and
  the tier that owns the non-law checks.
- [Stub](stub.md) — the double; the model tier runs subjects plain, never
  wrapped.
- [RFC-0003](../../rfc/0003-the-projection-consumers.md) — the design
  record: the cluster rule, the law-field taxonomy, the probes, the
  integration matrix over every `engine/model` subpackage.
