# testkit

Test infrastructure for Go. testkit reads your interfaces and types, generates the test doubles, builders, fixtures, conformance suites, and benchmarks you would otherwise write by hand, and generates the tests that prove the generated code works. You write domain logic. testkit writes plumbing.

```go
//go:generate testkit stub      -o storetest/store_stub.gen.go    Store
//go:generate testkit builder   -o storetest/user_builder.gen.go  User
//go:generate testkit suite     -o storetest/store_spec.gen.go    Store
//go:generate testkit model     -o storetest/store_model.gen.go   Store
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
| 1    | `suite`                                    | Single-call contract per documented directive |
| 2-3  | `model`                                    | Property-based state-machine, differential, workload |
| 4    | `bench`                                    | Allocation and latency ceilings, complexity shape |
| 5    | `sim`, `chaos`, `differential-rollout`, `replay` | Subsystem simulation, continuous fault, shadow traffic, trace replay |

`stub` provides the runtime primitives the conformance tiers compose with — recording, fault injection, gating, virtual clocks, strict-mode dispatch. `sim` is the subsystem-level harness Tier 5 generators (`chaos`, `replay`) run on top of.

## Generators

Each generator emits both the artifact and the tests that exercise it. testkit ships 14 generators.

| Generator                  | Tier        | What it produces | Injection point |
|----------------------------|-------------|------------------|-----------------|
| `stub`                     | primitives  | Per-method test doubles with strategy-pattern fault injection (counted, probabilistic, time-windowed, predicate, retry), virtual clock, recording, gates, concurrent-test primitives. Modes: `BenchMode`, `DelegateTo`. Auto-detects `iter.Seq[T]` / `iter.Seq2[V, error]` and emits stream helpers. | Domain state for in-memory companions; per-method overrides via `WithIfaceMethod` constructor options. |
| `builder`                  | fixtures    | Fluent `With*` per exported field, `Append*` for slices, `WithEntry` / `WithEntries` for maps, `WithDataString` for `[]byte`, `Mutate`, `Clone`. Generic types, embedded structs, nested fields, sets-as-maps. | `<Type>Defaults() <Type>` function; optional per-field literal defaults via `//testkit:default`. |
| `sentinel`                 | static      | Prefix consistency, uniqueness, non-overlap (`errors.Is` asymmetry), unwrap-chain, format-string field round-trip, optional-method detection (`Is`, `Unwrap`). | None. |
| `enum`                     | static      | Exhaustiveness, all-values-distinct, stringer round-trip, out-of-range fallback, optional `ParseX` round-trip, optional `MarshalText` / `UnmarshalText` round-trip. | None. |
| `codec`                    | wire        | `codectest.Spec[T]` round-trip suite + benchmark + fuzz seeds + binary wire fixtures (`testdata/wire/*.bin`) regenerated when codec semantics change. Single source of truth across spec and wire. Modes: spec emission, `-update-wire` regeneration. | Sample value overrides via `<Type>Sample()` convention. |
| `suite`                    | 1           | `AssertContract(t, factory, opts...)` with one subtest per (method × directive). Single-call assertions, skip-with-diagnostic for missing options. Single output file. | `Factory func() Iface` closure plus domain-input options (`UnknownID`, `KnownID`, `SampleItem`, `Setup`). |
| `model`                    | 2-3         | rapid state-machine harness with three modes: `RunStateMachine` (oracle-based parity), `RunDifferential` (N-impl comparison), `RunWorkload` (random dispatch for sim drivers). Property-based input exploration, cross-method invariants, sequence assertions, atomicity rollback, partition isolation. | `Oracle` interface implementation; optional command generators, custom invariants, weights. |
| `bench`                    | 4           | `BenchmarkContract(b, factory, opts...)` with `AllocsMax(N)` / `LatencyMax(X)` gates per directive. Performance-shape benchmarks (`O(1)`, `O(log n)`, `O(n)`) via varying input size. Auto-enables stub `BenchMode`. | `Factory func() Iface` closure. |
| `sim`                      | 5           | Subsystem-shaped deterministic simulation harness (`sim.NewDispatcherSim(t, seed, cfg)`) wrapping the full production stack: stubs auto-wrapped with recording-stamped `OnRecord` hooks emitting into the engine trace; `Clock` / `RandSource` plumbed from engine seeds; completion-event sinks; capture-on-failure with minimal-reproducer seed extraction; `Workload[T]` and `Invariant[T]` registration verbs; cooperative-quiescence `AssertAll`. Per-subsystem composition (one Sim per top-level interface). Replaces hand-rolled per-package sim packages. | Top-level interface; `Workload[T]` and `Invariant[T]` registrations; optional seed and dispatcher config. |
| `chaos`                    | 5           | Continuous deterministic simulation harness driving randomized fault schedules, network partitions, clock skew, and process restarts across operation sequences. Seeded reproducible runs; on failure emits trace + minimal-reproducer seed. Integrates with `sim` via `OnRecord` hooks for trace correlation. | `Faults` configuration, `RunFault` / `PartitionSpec` declarations, soak-budget hints. |
| `differential-rollout`     | 5           | Shadow-traffic harness running an interface across N implementations in parallel with response comparison and divergence reporting. Migration-grade testing; pluggable equivalence relations for non-deterministic fields (timestamps, IDs). | Implementation list, equivalence-class declarations, divergence threshold. |
| `replay`                   | 5           | Trace-replay harness consuming captured production call traces (or sim-engine traces) and replaying them through impls to verify behavioral preservation across versions. | Trace source (file path or producer function), version-skew tolerance hints. |
| `smoke`                    | CLI surface | CLI command coverage: invokes each declared `cobra.Command` (or equivalent) with sampled flag combinations, asserts exit code and stdout/stderr shape per command. Auto-detects subcommand trees, flag types, required-flag validation. Captures golden output for stable commands; diffs on regression. | None for declared commands; optional flag-value distributions. |
| `pkgdoc`                   | compliance  | Compliance audit-doc skeleton (`docs/compliance/package-audit/<pkg>.md`) with REQ table, refactor-history banner, evidence section. Auto-fills mechanical parts; refreshes when source changes; validates REQ IDs against source directives. | Domain analysis (design notes, refactor narrative, exceptions). |

## Directive vocabulary

Conformance generators are driven by `//testkit:` directives on interface methods. Directives are machine-readable, grep-able, and validated at codegen time — unknown directives error, conflicting combinations error, redundant pairs warn.

The matrix below shows which generator consumes which directive and what it produces. `builder` consumes `default` (set defaults from struct-tag literals); `sentinel`, `enum`, `codec`, and `pkgdoc` are directive-free or consume their own narrow vocabulary.

| Directive | stub | suite | model | bench | chaos | differential-rollout | replay | sim | smoke |
|-----------|------|-------|-------|-------|-------|---------------------|--------|-----|-------|
| `errors`                   | ✓ helpers     | ✓ subtest       | ✓ failure paths   | —              | ✓ inject          | ✓ classify          | ✓ assert        | ✓ trace-classify   | ✓ exit-code         |
| `deprecated`               | ✓ log         | ✓ skip          | —                 | —              | —                 | —                   | —               | —                  | ✓ skip              |
| `integration-only`         | ✓ skip        | ✓ skip          | ✓ skip            | ✓ skip         | ✓ skip            | ✓ skip              | ✓ skip          | ✓ skip             | ✓ skip              |
| `retry-succeeds-on-attempt`| ✓ helper      | —               | —                 | —              | ✓ schedule        | —                   | —               | ✓ workload-retry   | —                   |
| `partition`                | ✓ helper      | —               | ✓ isolation       | —              | ✓ network         | ✓ shard-aware       | —               | ✓ trace-partition  | —                   |
| `order-after`              | ✓ tracker     | —               | ✓ sequence        | —              | ✓ schedule        | —                   | ✓ replay-order  | ✓ trace-order      | ✓ subcommand-order  |
| `nilsafe`                  | —             | ✓ subtest       | —                 | —              | —                 | —                   | —               | —                  | ✓ no-flag           |
| `ctx`                      | —             | ✓ subtest       | —                 | —              | ✓ cancellation    | —                   | —               | ✓ workload-cancel  | ✓ signal            |
| `timeout`                  | —             | ✓ subtest       | —                 | ✓ deadline     | ✓ slow-net        | —                   | —               | ✓ deadline         | ✓ deadline          |
| `pure`                     | —             | ✓ subtest       | —                 | —              | —                 | ✓ side-free         | ✓ deterministic | ✓ trace-stable     | ✓ no-side-effects   |
| `validates`                | —             | ✓ subtest       | —                 | —              | —                 | —                   | —               | —                  | ✓ flag-validation   |
| `bounded`                  | —             | ✓ subtest       | —                 | —              | —                 | ✓ range-equiv       | —               | ✓ trace-bounded    | —                   |
| `idempotent`               | —             | —               | ✓ property        | —              | ✓ retry-safe      | —                   | ✓ replay-safe   | ✓ workload-retry   | ✓ rerun             |
| `cacheable`                | —             | —               | ✓ property        | —              | —                 | ✓ memo-equiv        | —               | —                  | —                   |
| `monotonic`                | —             | —               | ✓ sequence        | —              | —                 | ✓ ordered-equiv     | ✓ ordering      | ✓ trace-monotonic  | —                   |
| `concurrent`               | —             | —               | ✓ stress          | ✓ parallel     | ✓ contention      | —                   | —               | ✓ workload-pool    | —                   |
| `concurrent-readers`       | —             | —               | ✓ stress          | ✓ parallel     | ✓ contention      | —                   | —               | ✓ workload-pool    | —                   |
| `atomic`                   | —             | —               | ✓ rollback        | —              | ✓ crash-mid       | —                   | ✓ no-partial    | ✓ trace-atomic     | ✓ all-or-nothing    |
| `sideeffect`               | —             | —               | ✓ relation        | —              | ✓ causality       | ✓ effect-equiv      | ✓ effect-replay | ✓ trace-causality  | ✓ command-effect    |
| `eventually`               | —             | —               | ✓ wait            | —              | ✓ convergence     | ✓ window-equiv      | ✓ window        | ✓ trace-eventually | —                   |
| `pagination`               | —             | —               | ✓ iterate         | —              | —                 | ✓ page-equiv        | —               | —                  | —                   |
| `invariant` (interface)    | —             | —               | ✓ global          | —              | ✓ continuous      | ✓ cross-impl        | ✓ replay-stable | ✓ post-tick        | —                   |
| `consistency`              | —             | —               | ✓ modifier        | —              | ✓ partition-relax | ✓ model-aware       | ✓ window        | ✓ trace-consistency| —                   |
| `lease`                    | —             | —               | ✓ lifecycle       | —              | ✓ holder-fault    | —                   | —               | ✓ trace-lease      | —                   |
| `allocs`                   | —             | —               | —                 | ✓ gate         | —                 | —                   | —               | —                  | —                   |
| `latency`                  | —             | —               | —                 | ✓ gate         | ✓ p99-target      | ✓ regression        | —               | —                  | ✓ command-budget    |
| `complexity`               | —             | —               | —                 | ✓ vary-input   | —                 | —                   | —               | —                  | —                   |
| `crash-safe`               | —             | —               | —                 | —              | ✓ kill-restore    | —                   | ✓ recovery      | ✓ workload-restart | —                   |
| `network-safe`             | —             | —               | —                 | —              | ✓ partition       | ✓ partition-equiv   | —               | ✓ workload-partition| —                  |
| `clock-skew`               | —             | —               | —                 | —              | ✓ skew            | —                   | ✓ drift         | ✓ trace-skew       | —                   |
| `exit-code`                | —             | —               | —                 | —              | —                 | —                   | —               | —                  | ✓ assert            |
| `golden-output`            | —             | —               | —                 | —              | —                 | —                   | —               | —                  | ✓ stdout/stderr     |
| `req`                      | —             | ✓ name          | ✓ name            | ✓ name         | ✓ name            | ✓ name              | ✓ name          | ✓ name             | ✓ name              |

Composition is enforced: `pure` and `sideeffect` together is a codegen error; `retry-succeeds-on-attempt` requires `retryable`; `cacheable` redundantly implies `pure`.

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

Full reference: [docs/testkit/primitives/](docs/testkit/primitives/README.md).

## Validators

Static CI checks. No test execution required; pure code and config analysis. testkit ships 18 validators across four categories.

- **Structural** — proto-sync, migration chain, depguard, wire freshness, error prefix, skip expiry.
- **Test quality** — assertion-free tests, test naming, `time.Sleep` detection, orphaned test doubles, parallel safety, contract-benchmark completeness.
- **Quality gates** — benchmark contracts, benchmark regression vs baseline, per-layer coverage thresholds, per-layer mutation thresholds.
- **Compliance** — audit-doc completeness, REQ-to-test traceability.

Full reference: [docs/testkit/validators/](docs/testkit/validators/README.md).

## Quick start

```bash
go install go.thesmos.sh/testkit/cmd/testkit@latest
go get go.thesmos.sh/testkit@latest
```

Add directives to the package that owns the types:

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

Pre-1.0. The runtime primitives (`MethodStub[T]`, `Recorder[T]`, `FaultInjector`, `StartContract`, shape-typed assertion contexts, golden-file helpers) and the generator engine (`gen/`) are stable. Five generators ship today: **`stub`**, **`builder`**, **`sentinel`**, **`enum`**, **`suite`**. The remaining nine are designed and documented but not yet implemented. Generator vocabulary and directive semantics may change in minor versions until the V1 cut. Consumers should pin and regenerate on upgrade.

V1 commits to:

- Stable directive vocabulary and composition rules.
- Stable generated-file layout and naming conventions.
- Backward-compatible runtime primitives (additive only).
- Documented deprecation cycle for any directive removal.

## Dependencies

Core `testkit` package:

- [`github.com/google/go-cmp`](https://github.com/google/go-cmp) — structural diffs for assertions.
- [`pgregory.net/rapid`](https://pgregory.net/rapid) — property-based generators for `model` and stub stream helpers.

Optional sub-packages with isolated dependencies:

| Package              | Adds                              |
|----------------------|-----------------------------------|
| `testkit/container`  | `testcontainers-go`               |
| `testkit/httptest`   | stdlib only                       |
| `testkit/oteltest`   | `go.opentelemetry.io/otel/sdk`    |
| `testkit/clitest`    | stdlib only                       |

## Documentation

- [Primitives](docs/testkit/primitives/README.md) — assertions, recording, fault injection, benchmarking, golden files, polling.
- [Generators](docs/testkit/generators/README.md) — per-generator semantics, output, injection points.
- [Validators](docs/testkit/validators/README.md) — 18 CI checks.
- [Configuration](docs/testkit/configuration.md) — `.testkit.yml` reference.
- [Adoption](docs/testkit/adoption.md) — incremental adoption guide.

## License

[MIT](LICENSE)
