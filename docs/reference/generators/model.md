# Model

> **Status: designed, not implemented — the runtime is shipped.**
> [RFC-0003](../../rfc/0003-the-projection-consumers.md) fixes this
> generator's design. The `engine/model` runtime it binds — the rapid-based
> runner, the law catalogue, the reference oracles, the Porcupine models —
> ships today and is usable by hand; the generator that derives the
> bindings is not implemented. Where this page and the RFC differ, the RFC
> is the authority.

The `model` generator binds the classifications
[ADR-0018](../../adr/0018-one-tier-owns-each-classification.md) assigns to
the model tier onto the shipped `engine/model` runtime: property-based
state-machine testing on [`pgregory.net/rapid`](https://pgregory.net/rapid),
linearizability checking on
[`anishathalye/porcupine`](https://github.com/anishathalye/porcupine),
bounded exhaustive search, and a mutation self-check that proves the bound
laws can kill injected bugs. Consumers get law-backed conformance for one
directive; everything below is derived from the shape stamps and the
[suite generator](suite.md)'s projection.

## The directive

```go
//testkit:model
type Store interface { ... }
```

Interface-scoped, no keys, negation denied — the tier exists where one is
declared, and deleting the line is the suppression, the same shape as
`//testkit:stub` and `//testkit:suite`. The generator emits where the
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
- **Generators, derived once** — keys from a small sampled set, the value
  type's key field pinned to the same generator, shared by the sequential,
  concurrent and exhaustive paths. Collisions are what make the laws fire.
- **A derived reference** — an adapter over the matching
  `engine/model/ref` oracle (`MapStore` for the CRUD family, `AtomicCell`
  for `cas`, `LeaseTracker` for `lease`, …), composed per state cluster
  where several match, armed per law. Where no oracle maps and none is
  supplied via `<Iface>ModelReference`, reference-needing laws **skip
  visibly**, one subtest per law ID naming the option that arms it.
- **A concurrent path** — where the shape matches a prebuilt
  `engine/model/linearize` Porcupine model, a `WithConcurrent` wiring with
  per-key partitioning and the non-linearizable remainder as `-race`
  stress.
- **Chain tracing** — `appender` shapes get the chain action family over a
  partition-keyed `engine/model/history` trace.
- **Derived leak checking** — `lifecycle` shapes wrap the run in
  `model.CheckGoroutineLeaks`.
- **A report header** — the generated docblock is a per-method table of
  what the run derived: actions, law IDs, the cluster map, what was skipped
  and the option that arms it.

Three probes guard soundness, each skipping with a diagnosis rather than
failing a correct implementation: a fresh-state probe (subject and
reference must start observably equal), a determinism probe before the
exhaustive search, and codegen feasibility for the action alphabet.

## The self-checks

`_model.gen_test.go` carries three residents:

- **`Fuzz<Iface>Model`** — the interface's *sequence space* as one fuzz
  target: `model.MakeFuzz` lets the coverage-guided engine drive rapid's
  choice stream, hunting for the action ordering that breaks a law. One
  target per interface.
- **`Test<Iface>ModelExhaustive`** — bounded model checking over the
  fixture-derived action alphabet with an observational state hash: proof
  of law absence within bounds, and the shortest counterexample sequence
  when there is none to prove.
- **The mutation kill test** — wraps the derived reference in each
  compatible `engine/model/mutation` operator (`DropWrites`,
  `ReturnWrongValue`, …) and asserts the law suite kills every one. An
  unkilled operator is a hole in the bindings, not in any consumer's
  implementation.

The last two are emitted only where the reference is derivable — both
certify subject-versus-reference agreement. All three, plus the concurrent
path, honour `testing.Short()`; the sequential law run does not skip,
because it is the assertion a declared classification owes.

## What a consumer writes

Nothing beyond the suite wiring. The laws run inside
`Assert<Iface>Contract` against every registered subject; the fuzz and
exhaustive entries read the same registry. The options are the escape
hatches: `<Iface>ModelReference(factory)` replaces the derived oracle,
`<Iface>ModelWith<Type>Gen(gen)` supplies an opaque type's generator, and
the existing `Without` paths drop by law ID.

## Layout conventions

| Tag | Suffix | Contents |
|---|---|---|
| *(primary)* | `_model.gen.go` | Report header, generators, adapter, law registry, action set, concurrent config, options, extension registration. |
| `test` | `_model.gen_test.go` | `Fuzz<Iface>Model`; the exhaustive and mutation tests where a reference is derivable. |

## See also

- [Suite](suite.md) — the projection and registry this generator reads, and
  the tier that owns the non-law checks.
- [Stub](stub.md) — the double; the model tier runs subjects plain, never
  wrapped.
- [RFC-0003](../../rfc/0003-the-projection-consumers.md) — the design
  record: the cluster rule, the law-field taxonomy, the probes, the
  integration matrix over every `engine/model` subpackage.
