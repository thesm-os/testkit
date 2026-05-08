# testkit

`go.thesmos.sh/testkit` is a standalone test infrastructure toolkit for Go projects. It provides:

1. **Primitives** — reusable test utilities (assertions, recording, fault injection, virtual clock, benchmarking contracts, golden files) every Go project needs.
2. **Generators** — a `//go:generate`-driven CLI that reads Go types and proto descriptors and emits mechanical test boilerplate (stubs, builders, models, suites, specs) plus the tests that prove the generated plumbing is correct.
3. **Validators** — CI-time structural checks that enforce invariants across the codebase (proto sync, depguard, migration chain, REQ traceability, parallel safety, and more).

## Principles

- **Zero domain knowledge.** testkit knows Go types and proto descriptors, not business logic. Domain semantics are always injected by the caller.
- **Standalone module.** No dependency on any application module. Adopt by importing, not by forking.
- **`//go:generate`-driven.** Generators are invoked via standard Go directives. Project-wide conventions live in `.testkit.yml`.
- **Generated plumbing has its own tests.** Every generator emits both the artifact and the test file that exercises it; coverage of the generated code is 100%.
- **Stubs are the substrate.** A single `MethodStub[Call]` primitive composes recording, fault injection, latency, virtual clock, strict mode, and call-count expectations. Every conformance tier (suite, model, bench, sim, chaos, replay) builds on it.

## Status

Pre-1.0. The runtime primitives (`MethodStub[T]`, `Recorder[T]`, `Fault` strategies, `Clock`/`TestClock`, `StartContract`, shape-typed assertion and bench contexts, golden-file helpers) and the generator engine in `gen/` are stable. Seven generators ship today: **`stub`**, **`builder`**, **`sentinel`**, **`enum`**, **`suite`**, **`model`**, **`bench`**. The remaining generators (`sim`, `chaos`, `differential-rollout`, `replay`, `codec`, `smoke`, `pkgdoc`) are designed but not yet implemented — their docs document the planned shape and are clearly marked.

## Documentation

| Document | Description |
|----------|-------------|
| **Primitives** | |
| [Overview](primitives/README.md) | Index of all primitives + dependencies |
| [Assertions](primitives/assertions.md) | Positional + fluent helpers with go-cmp diffs |
| [Directive assertions](primitives/directive-assertions.md) | `AssertNilSafe`, `AssertCtxCancellation`, `AssertTimeout`, `AssertPure`, `AssertBounded` |
| [MethodStub](primitives/method-stub.md) | Generic per-method test double primitive |
| [Recording](primitives/recording.md) | `Recorder[T]` — filtering, waiting, hooks, gating, timestamping, bench mode |
| [Fault injection](primitives/fault-injection.md) | `Fault` interface + 5 strategies + `And`/`Or` composition |
| [Clock](primitives/clock.md) | `Clock` interface + `RealClock` + `TestClock` |
| [OrderTracker](primitives/order-tracker.md) | Cross-method call-order constraints |
| [RandSource](primitives/rand.md) | Pluggable RNG for probabilistic faults |
| [Concurrency](primitives/concurrency.md) | `ConcurrentStress`, `GoroutineLeak`, `Timeout` |
| [Benchmarking](primitives/benchmarking.md) | `Contract` for allocation and latency gates |
| [Golden files](primitives/golden-files.md) | `AssertGolden` + scrubbers |
| [Polling](primitives/polling.md) | `RetryUntil`, `AssertEventually` |
| [Context](primitives/context.md) | `t.Context()` + `Timeout` wrapper |
| [Helpers](primitives/helpers.md) | `TestError`, `RequireEnv`, `SeededRand`, `MustMarshal`, `Quiet`, `FailableTB`, `TempFile`, `FreePort`, `SortedKeys`, `TableTest`, `MapDiff`, rapid generators |
| **Guides** | |
| [Layout](layout.md) | Test package directory structure, file roles, where sample builders go |
| [Linter config](golangci.md) | Copy-pasteable `.golangci.yml` for testkit consumers |
| **Generators** | |
| [Overview](generators/README.md) | CLI interface, output conventions, tier framing |
| [`stub`](generators/stub.md) | Per-method test doubles — runtime substrate for tiers 1-5 (ready) |
| [`builder`](generators/builder.md) | Fluent fixture builders with `With*`, `Append*`, `Mutate`, `Clone` (ready) |
| [`sentinel`](generators/sentinel.md) | Prefix, uniqueness, non-overlap, unwrap-chain, custom-error round-trip (ready) |
| [`enum`](generators/enum.md) | Exhaustiveness, stringer, Parse, MarshalText/JSON round-trip (ready) |
| [`suite`](generators/suite.md) | Tier 1: `Assert<Iface>Contract` with shape-detected subtests + typed plug-in points (ready) |
| [`bench`](generators/bench.md) | Tier 4: `Benchmark<Iface>Contract` with shape-detected hot-paths + typed bench plug-ins (ready) |
| [`codec`](generators/codec.md) | `codectest.Spec[T]` + suite + bench + fuzz seeds + wire fixtures (planned) |
| [`model`](generators/model.md) | Tier 2-3: rapid property-based state-machine with differential testing, shape-specific laws, concurrent stress, trace combinators (ready) |
| [`sim`](generators/sim.md) | Tier 5: subsystem simulation harness (planned) |
| [`chaos`](generators/chaos.md) | Tier 5: continuous fault simulation on top of sim (planned) |
| [`differential-rollout`](generators/differential-rollout.md) | Tier 5: shadow-traffic comparison harness (planned) |
| [`replay`](generators/replay.md) | Tier 5: production-trace replay (planned) |
| [`smoke`](generators/smoke.md) | Cobra command coverage with golden output (planned) |
| [`pkgdoc`](generators/pkgdoc.md) | Compliance audit-doc skeleton with auto-fill + refresh (planned) |
| **Validators — Structural** | |
| [Overview](validators/README.md) | All 18 validators + CI integration guide |
| [Proto-Sync](validators/proto-sync.md) | Proto files match generated Go |
| [Migration](validators/migration.md) | SQL migration chain validity |
| [Depguard](validators/depguard.md) | Import graph layer rules |
| [Wire](validators/wire.md) | Wire golden file freshness |
| [Error Prefix](validators/error-prefix.md) | `errors.New` uses correct package prefix |
| [Skip Expiry](validators/skip-expiry.md) | `t.Skip` calls have valid expiry dates |
| **Validators — Test Quality** | |
| [Assertion-Free](validators/assertion-free.md) | Every `Test*` contains an assertion |
| [Test Naming](validators/test-naming.md) | No fragmented `Test<Type>_*`; subtests read as contracts |
| [time.Sleep](validators/time-sleep.md) | No `time.Sleep` in test files |
| [Orphaned Doubles](validators/orphaned-doubles.md) | No unused types in `*test/` packages |
| [Parallel Safety](validators/parallel-safety.md) | No `t.Setenv` + `t.Parallel`; no shared mutable state |
| [Contract Completeness](validators/contract-completeness.md) | Every documented contract has a gated benchmark |
| **Validators — Quality Gates** | |
| [Benchmarks](validators/benchmarks.md) | Every benchmark uses `StartContract` |
| [Bench Regression](validators/bench-regression.md) | No alloc/latency regression vs baseline |
| [Coverage](validators/coverage.md) | Per-layer line/branch coverage thresholds |
| [Mutation](validators/mutation.md) | Per-layer mutation efficacy + coverage |
| **Validators — Compliance** | |
| [Audit Completeness](validators/audit-completeness.md) | Every package has a complete audit doc |
| [REQ Traceability](validators/req-traceability.md) | Every REQ traces to a test and back |
| **Reference** | |
| [Configuration](configuration.md) | `.testkit.yml` reference |
| [Sub-packages](sub-packages.md) | container, httptest, oteltest, clitest |
| [Adoption](adoption.md) | How to adopt testkit in a new project |
