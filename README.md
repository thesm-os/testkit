# testkit

Test infrastructure for Go. testkit reads your interfaces and types, generates the test doubles, builders, fixtures, conformance suites, and benchmarks you would otherwise write by hand, and generates the tests that prove the generated code works. You write domain logic. testkit writes plumbing.

```go
//go:generate testkit stub      -o storetest/store_stub.gen.go    Store
//go:generate testkit builder   -o storetest/user_builder.gen.go  User
//go:generate testkit suite     -o storetest/store_spec.gen.go    Store
//go:generate testkit bench     -o storetest/store_bench.gen.go   Store
//go:generate testkit sentinel  -o errors.gen_test.go
//go:generate testkit enum      -o status_enum.gen_test.go        Status
```

```bash
go generate ./...   # produces *.gen.go and *.gen_test.go
go test ./...       # 100% branch coverage of generated plumbing
```

## Conformance tiers

Each conformance generator targets a tier. Lower tiers run in seconds; higher tiers run for minutes-to-hours and stand in for production load.

| Tier | Generator                                  | What it proves |
|------|--------------------------------------------|----------------|
| 1    | `suite`                                    | Single-call contract per documented directive across 21 method shapes |
| 2-3  | `model`                                    | Property-based state-machine, differential, workload (planned) |
| 4    | `bench`                                    | Allocation, mean latency, and per-percentile latency budgets across 21 shapes |
| 5    | `sim`, `chaos`, `differential-rollout`, `replay` | Subsystem simulation, continuous fault, shadow traffic, trace replay (planned) |

`stub` provides the runtime primitives the conformance tiers compose with — recording, fault injection, gating, virtual clocks, strict-mode dispatch. `sim` is the subsystem-level harness Tier 5 generators (`chaos`, `replay`) run on top of.

What each tier can and cannot see, and when to reach for the next one: [Conformance tiers](docs/explanation/conformance-tiers.md).

## Generators

Each generator emits both the artifact and the tests that exercise it. testkit's catalogue covers 14 generators. None are in the tree today — see [Status](#status); the table is what the eidos-based rebuild targets.

| Generator                  | Tier        | What it produces | Injection point |
|----------------------------|-------------|------------------|-----------------|
| `stub`                     | primitives  | Per-method test doubles with strategy-pattern fault injection (counted, probabilistic, time-windowed, predicate, retry), virtual clock, recording, gates, concurrent-test primitives. Modes: `BenchMode`, `DelegateTo`. Auto-detects `iter.Seq[T]` / `iter.Seq2[V, error]` and emits stream helpers. | Domain state for in-memory companions; per-method overrides via `WithIfaceMethod` constructor options. |
| `builder`                  | fixtures    | Fluent `With*` per exported field, `Append*` for slices, `WithEntry` / `WithEntries` for maps, `WithDataString` for `[]byte`, `Mutate`, `Clone`. Generic types, embedded structs, nested fields, sets-as-maps. | `<Type>Defaults() <Type>` function; optional per-field literal defaults via `//testkit:default`. |
| `sentinel`                 | static      | Prefix consistency, uniqueness, non-overlap (`errors.Is` asymmetry), unwrap-chain, format-string field round-trip, optional-method detection (`Is`, `Unwrap`). | None. |
| `enum`                     | static      | Exhaustiveness, all-values-distinct, stringer round-trip, out-of-range fallback, optional `ParseX` round-trip, optional `MarshalText` / `UnmarshalText` round-trip. | None. |
| `codec`                    | wire        | `codectest.Spec[T]` round-trip suite + benchmark + fuzz seeds + binary wire fixtures (`testdata/wire/*.bin`) regenerated when codec semantics change. Single source of truth across spec and wire. Modes: spec emission, `-update-wire` regeneration. | Sample value overrides via `<Type>Sample()` convention. |
| `suite`                    | 1           | `Assert<Iface>Contract(t, factory, opts...)` with shape-detected subtests across 21 method shapes plus one subtest per applied directive. Skip-with-diagnostic for missing options. | `Factory func() Iface` closure plus typed `<Iface>On<Method>(...)` plug-in slots and `<Iface>PrePopulate` seeder. |
| `model`                    | 2-3         | rapid property-based state-machine: differential SUT vs reference testing, auto-derived shape-specific laws (ReadAfterWrite, DeleteReturnsNotFound, PureDeterminism, PredicateConsistency, StreamReentrancy), concurrent stress with Porcupine linearizability checking, trace combinators (AfterEvery, EventuallyAfter, Never), goroutine leak detection, `TestClock` integration for time-aware interfaces, fuzz target generation. *(planned)* | `Factory func() T`, optional `RefFactory func() T` for differential mode; per-method action helpers auto-emitted by shape; extension via `ExtraActions`, `ExtraLaws`, `WithConcurrent`. |
| `bench`                    | 4           | `Benchmark<Iface>Contract(b, factory, opts...)` with shape-detected `<Method>/hot-path` and `<Method>/concurrent-4` benchmarks across 21 shapes, plus opt-in `allocs-within-N` / `latency-within-D` / `percentiles` gates per `//testkit:allocs` / `//testkit:latency` / `//testkit:percentiles`. Auto-enables stub `BenchMode`. | `Factory func() Iface` closure plus typed `<Iface>BenchOn<Method>(...)` plug-in slots and `<Iface>BenchPrePopulate` seeder. |
| `sim`                      | 5           | Subsystem-shaped deterministic simulation harness (`sim.NewDispatcherSim(t, seed, cfg)`) wrapping the full production stack: stubs auto-wrapped with recording-stamped `OnRecord` hooks emitting into the engine trace; `Clock` / `RandSource` plumbed from engine seeds; completion-event sinks; capture-on-failure with minimal-reproducer seed extraction; `Workload[T]` and `Invariant[T]` registration verbs; cooperative-quiescence `AssertAll`. Per-subsystem composition (one Sim per top-level interface). Replaces hand-rolled per-package sim packages. | Top-level interface; `Workload[T]` and `Invariant[T]` registrations; optional seed and dispatcher config. |
| `chaos`                    | 5           | Continuous deterministic simulation harness driving randomized fault schedules, network partitions, clock skew, and process restarts across operation sequences. Seeded reproducible runs; on failure emits trace + minimal-reproducer seed. Integrates with `sim` via `OnRecord` hooks for trace correlation. | `Faults` configuration, `RunFault` / `PartitionSpec` declarations, soak-budget hints. |
| `differential-rollout`     | 5           | Shadow-traffic harness running an interface across N implementations in parallel with response comparison and divergence reporting. Migration-grade testing; pluggable equivalence relations for non-deterministic fields (timestamps, IDs). | Implementation list, equivalence-class declarations, divergence threshold. |
| `replay`                   | 5           | Trace-replay harness consuming captured production call traces (or sim-engine traces) and replaying them through impls to verify behavioral preservation across versions. | Trace source (file path or producer function), version-skew tolerance hints. |
| `smoke`                    | CLI surface | CLI command coverage: invokes each declared `cobra.Command` (or equivalent) with sampled flag combinations, asserts exit code and stdout/stderr shape per command. Auto-detects subcommand trees, flag types, required-flag validation. Captures golden output for stable commands; diffs on regression. | None for declared commands; optional flag-value distributions. |
| `pkgdoc`                   | compliance  | Compliance audit-doc skeleton (`docs/compliance/package-audit/<pkg>.md`) with REQ table, refactor-history banner, evidence section. Auto-fills mechanical parts; refreshes when source changes; validates REQ IDs against source directives. | Domain analysis (design notes, refactor narrative, exceptions). |

## Directives

Conformance generators are driven by `//testkit:` directives on interface methods. Directives are machine-readable, grep-able, and validated at codegen time: a declared directive's positional arguments, required keys, node kind and negation are all checked, and conflicting combinations error.

An unknown directive **name** is an error, with a suggestion:

```text
error store/iface.go:25:1: prescreen: no directive named "sutie" — did you mean "suite"?
```

Pre-screening names is the consuming tool's job by design ([eidos `Validate` docs](https://github.com/thesm-os/eidos/blob/main/core/directive/validator.go)), and testkit does it in an annotator over the whole node graph — so the typo is caught wherever it landed, not only on the kinds that directive applies to.

One gap remains, stated because a directive that is silently ignored costs you the check it was supposed to arm:

- **A directive documented as taking no keys accepts any.** The underlying schema cannot express "no KV pairs permitted" — tracked as [eidos#38](https://github.com/thesm-os/eidos/issues/38). A mistyped mixin parameter key has the same shape: [eidos#35](https://github.com/thesm-os/eidos/issues/35).

Until it closes, grep is the check for keys: if a directive you wrote produced no change in the generated header, a key is misspelled.

The matrix below is the directive vocabulary the rebuild targets, grouped by intent. Columns name the consuming generator; `enum` is directive-free. Per-generator directive surfaces are documented in the individual generator doc files.

### Error & return contract

| Directive | Args | stub | suite | bench |
|-----------|------|------|-------|-------|
| `errors` | `<ErrName>...` | ✓ `Fault<Sentinel>()` helpers | ✓ `<Method>/returns <ErrX>` | — |
| `wrapped-via` | `<ErrName>` | ✓ wraps via target in helpers | ✓ `<Method>/wrapped-via` | — |

### Behavioral properties

| Directive | Args | stub | suite | bench |
|-----------|------|------|-------|-------|
| `idempotent` | — | — | ✓ `<Method>/idempotent` | — |
| `pure` | — | — | ✓ `<Method>/pure` | — |
| `cacheable` | — | — | ✓ `<Method>/cacheable` (implies `pure`) | — |
| `monotonic` | — | — | ✓ `<Method>/monotonic` | — |

### Safety

| Directive | Args | stub | suite | bench |
|-----------|------|------|-------|-------|
| `concurrent` | — | — | ✓ `<Method>/concurrent` | — |
| `concurrent-readers` | — | — | ✓ `<Method>/concurrent-readers` | — |
| `nilsafe` | — | — | ✓ `<Method>/nilsafe` | — |
| `atomic` | — | — | ✓ `<Method>/atomic` | — |

### Context & lifecycle

| Directive | Args | stub | suite | bench |
|-----------|------|------|-------|-------|
| `ctx` | — | — | (documentation hint; auto-emitted ctx subtests cover the semantics) | — |
| `timeout` | `<duration>` | — | ✓ `<Method>/timeout` | — |
| `deprecated` | `<Replacement>` | ✓ `tb.Logf` in dispatch + `// Deprecated:` doc comment | ✓ `<Method>/deprecated` (skip with hint) | — |
| `lease` | `<ReleaseMethod>` | — | ✓ `<Method>/lease` | — |
| `integration-only` | — | ✓ skip dispatch | ✓ skip method block | ✓ skip method helper |

### Performance

| Directive | Args | stub | suite | bench |
|-----------|------|------|-------|-------|
| `allocs` | `<N>` | — | — | ✓ `<Method>/allocs-within-N` (gate) |
| `latency` | `<duration>` | — | — | ✓ `<Method>/latency-within-D` (gate) |
| `percentiles` | `p<N>=<duration>...` | — | — | ✓ `<Method>/percentiles` (per-percentile gate; reports `p50/p95/p99`) |

### Resilience

| Directive | Args | stub | suite | bench |
|-----------|------|------|-------|-------|
| `retryable` | — | ✓ companion marker for `retry-succeeds-on-attempt` | — | — |
| `retry-succeeds-on-attempt` | `<N>` | ✓ `RetrySchedule(err)` helper | ✓ `<Method>/retry-succeeds-on-attempt` | — |

### Causality, ordering, isolation

| Directive | Args | stub | suite | bench |
|-----------|------|------|-------|-------|
| `sideeffect` | `<Method>` | — | ✓ `<Method>/sideeffect` | — |
| `order-after` | `<Method>` | ✓ `AssertAfter` (strict mode) | ✓ `<Method>/order-after` | — |
| `partition` | `<Field>` | ✓ `FaultForPartition`, `FaultForOtherPartitions` | ✓ `<Method>/partition` | — |

### Input & validation

| Directive | Args | stub | suite | bench |
|-----------|------|------|-------|-------|
| `validates` | `<Field>` | — | ✓ `<Method>/validates` | — |
| `bounded` | `<min..max>` | — | ✓ `<Method>/bounded` | — |

### Properties, observability, traceability

| Directive | Args | stub | suite | bench |
|-----------|------|------|-------|-------|
| `invariant` | `<description>...` | — | (documentation hint) | — |
| `fuzz` | — | — | (planned) | — |
| `hooks` | `<HookName>...` | — | ✓ `<Method>/hooks` | — |
| `req` | `<REQ-ID>...` | — | ✓ name suffix on emitted subtests | ✓ name suffix on emitted subtests |

### Consistency, authorization, iteration

| Directive | Args | stub | suite | bench |
|-----------|------|------|-------|-------|
| `eventually` | `<timeout>` | — | ✓ `<Method>/eventually` | — |
| `scope` | `<ScopeName>` | — | ✓ `<Method>/scope` | — |
| `pagination` | `<CursorField>` | — | ✓ `<Method>/pagination` | — |

### Shape hints (signature-detection only; no subtest)

| Directive | Args | Effect |
|-----------|------|--------|
| `deleter` | — | Routes `func(ctx?, K) error` to `Deleter` shape (vs. `Writer`). |
| `mutator` | — | Marks `func(ctx?, V)` as `Mutator` (auto-detected from signature). |
| `not-mutator` | — | Opt-out of `Mutator` auto-detection (treat as `Writer`). |
| `keyfield` | `<FieldName>` | Reference-synthesis hint for the planned `model` generator. |

### Sample input replacement

| Directive | Args | stub | suite | bench |
|-----------|------|------|-------|-------|
| `sample` | `<Func>...` | — | ✓ replaces zero-value args in smoke / plug-in / hot-path call sites | ✓ replaces synthesized literals in hot-path / gate calls |

### Cross-method invariants

Each invariant directive is a first-class consumer; the suite emits a paired-method subtest at the carrier method's `t.Run` block. The `cross` directive remains as the escape hatch for invariants not yet shaped into a per-invariant directive.

| Directive | Args | suite |
|-----------|------|-------|
| `read-after-write` | `<Reader>` | ✓ `<Method>/read-after-write` |
| `delete-removes` | `<Reader>` | ✓ `<Method>/delete-removes` |
| `stream-reflects-mutations` | `<Stream>` | ✓ `<Method>/stream-reflects-mutations` |
| `lifecycle-after-close` | `<Reader>` | ✓ `<Method>/lifecycle-after-close` |
| `crdt-merge` | `<Other>` | ✓ `<Method>/crdt-merge` |
| `cross` | `<name> <Methods>...` | ✓ generic invariant escape hatch |

### Cross-package & per-field

| Directive | Args | Generator | Effect |
|-----------|------|-----------|--------|
| `sentinel-no-overlap-with` | `<ImportPath>...` | `sentinel` | Declare additional packages to verify sentinel non-overlap with. |
| `default` | `<Value>` | `builder` | Per-field literal default seeded into `Build()` when no `Defaults` factory exists. |

Composition is enforced: `pure` and `sideeffect` together is a codegen error; `pure` and `monotonic` together is a codegen error; `retry-succeeds-on-attempt` requires `retryable`; `cacheable` implies `pure` and inherits its conflicts; `concurrent` and `concurrent-readers` are mutually exclusive.

## Primitives

A focused utility set that earns its place. Every assertion uses [`go-cmp`](https://github.com/google/go-cmp) for structural diffs; not `reflect.DeepEqual`.

```go
testkit.Equal(t, got, want, "Get must return the stored item")
testkit.ErrorIs(t, err, store.ErrNotFound, "Get on missing key must return ErrNotFound")

testkit.Assert(t, user).
    IsNotNil("must exist").
    HasLen(3, "must have 3 fields populated")

c := testkit.StartContract(b).AllocsMax(0).LatencyMax(5 * time.Microsecond)
for c.Loop() {
    store.Get(b.Context(), key)
}
c.End()

rec := testkit.NewRecorder[PutCall]()
rec.OnRecord(func(c PutCall) { trace.Append(tick, c) })
rec.WaitForN(t, 3, 5*time.Second)
gate := rec.NewGate()
```

Full reference: [docs/reference/primitives/](docs/reference/primitives/README.md).

## Quick start

```bash
go get go.thesmos.sh/testkit@latest
go get go.thesmos.sh/testkit/engine@latest   # model-checking engine, optional
```

The runtime primitives are usable directly today. The generator workflow below
describes the target state and does not run yet — there is no `testkit` binary
in the tree. Add directives to the package that owns the types:

```go
// store/generate.go
package store

//go:generate testkit stub      -o storetest/store_stub.gen.go   Store
//go:generate testkit builder   -o storetest/user_builder.gen.go User
//go:generate testkit suite     -o storetest/store_spec.gen.go   Store
//go:generate testkit sentinel  -o errors.gen_test.go
```

Generate, scaffold the companion file, fill in domain logic, and run tests:

```bash
go generate ./...
testkit scaffold stub storetest Store
$EDITOR storetest/store_stub.go
go test ./...
```

## Output conventions

Generated files default to your existing layout. Override with `-o` for any non-default placement.

| Generator   | Default output                                     |
|-------------|----------------------------------------------------|
| `stub`      | `<pkg>test/<subject>_stub.gen.go`                  |
| `builder`   | `<pkg>test/<subject>_builder.gen.go`               |
| `suite`     | `<pkg>test/<subject>_spec.gen.go`                  |
| `model`     | `<pkg>test/<subject>_model.gen.go`                 |
| `bench`     | `<pkg>test/<subject>_bench.gen.go`                 |
| `sim`       | `<pkg>test/<subject>_sim.gen.go`                   |
| `chaos`     | `<pkg>test/<subject>_chaos.gen.go`                 |
| `replay`    | `<pkg>test/<subject>_replay.gen.go`                |
| `sentinel`  | `errors.gen_test.go`                               |
| `enum`      | `<subject>_enum.gen_test.go`                       |
| `codec`     | `<subject>_codec.gen_test.go`                      |
| `smoke`     | `cmd/<binary>/smoke.gen_test.go`                   |
| `pkgdoc`    | `docs/compliance/package-audit/<pkg>.md`           |

Combine multiple types into one file by passing several arguments:

```go
//go:generate testkit enum -o status_enum.gen_test.go OrderStatus PaymentStatus RefundStatus
```

## CI integration

```makefile
check: lint test check-structural check-test-quality check-quality check-compliance

check-structural:
 go generate ./... && git diff --exit-code
 testkit validate proto-sync migration depguard wire error-prefix skip-expiry

check-test-quality:
 testkit validate assertion-free test-naming time-sleep \
     orphaned-doubles parallel-safety contract-completeness

check-quality:
 testkit validate benchmarks bench-regression coverage mutation

check-compliance:
 testkit validate audit reqs
```

## Status

Pre-1.0, mid-restructure. Two modules ship today:

- **`go.thesmos.sh/testkit`** — the runtime primitives (`MethodStub[T]`, `Recorder[T]`, `FaultInjector`, `StartContract`, shape-typed assertion and bench contexts, golden-file helpers, virtual clocks, seeded randomness, polling). Depends on go-cmp alone.
- **`go.thesmos.sh/testkit/engine`** — the model-checking engine: property runner, law library, linearizability models, reference implementations, bounded model checking, failure classification and artifact emission.

Both are at 100% statement coverage.

The code generator is being rebuilt on [eidos](https://go.thesmos.sh/eidos) and is **not currently in the tree** — the hand-rolled `generator/`, `cmd/`, and `harness/` packages were removed rather than ported. The generator table above is the target catalogue, not shipped code. Until the tool module lands there is no `testkit` binary and no `//go:generate` workflow.

Directive vocabulary and generated-file layout may change until `v1.0.0`. Consumers should pin and regenerate on upgrade.

`v1.0.0` commits to ([ADR-0002](docs/adr/0002-support-external-consumers-under-semver.md)):

- Stable directive vocabulary and composition rules.
- Stable generated-file layout and naming conventions.
- Backward-compatible runtime primitives (additive only).
- Documented deprecation cycle for any directive removal.

## Dependencies

`go.thesmos.sh/testkit` — the runtime module, one dependency:

- [`github.com/google/go-cmp`](https://github.com/google/go-cmp) — structural diffs for assertions.

`go.thesmos.sh/testkit/engine` — the model-checking engine, which adds:

- [`pgregory.net/rapid`](https://pgregory.net/rapid) — property-based generation and shrinking.
- [`github.com/anishathalye/porcupine`](https://github.com/anishathalye/porcupine) — linearizability checking.

Consumers who only want assertions, stubs, and fault injection take the runtime
module and none of the engine's dependencies.

## Documentation

Start at [docs/](docs/README.md).

- [RFC-0001](docs/rfc/0001-testkit-as-a-generator-platform.md) — what the platform is and how it is shaped.
- [Architecture decisions](docs/adr/README.md) — why each piece is the way it is.
- [Primitives](docs/reference/primitives/README.md) — assertions, recording, fault injection, benchmarking, golden files, polling.
- [Generators](docs/reference/generators/README.md) — per-generator semantics, output, injection points.
- [Conformance tiers](docs/explanation/conformance-tiers.md) — what each tier proves and when to reach for it.
- [Configuration](docs/reference/configuration.md) — `.testkit.yaml` reference.
- [Layout](docs/reference/layout.md) — test package directory structure and file roles.
- [Linter config](docs/reference/golangci.md) — copy-pasteable `.golangci.yml` for testkit consumers.
- [Adoption](docs/how-to/adoption.md) — incremental adoption guide.

## License

[MIT](LICENSE)
