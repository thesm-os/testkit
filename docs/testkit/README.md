# testkit

`go.thesmos.sh/testkit` is a standalone test infrastructure
toolkit for Go projects. It provides two things:

1. **Primitives** — reusable test utilities (assertions,
   recording, fault injection, containers, benchmarking)
   that every Go project needs.

2. **Generators** — a `//go:generate`-driven CLI that
   reads Go types and proto descriptors and emits
   mechanical test boilerplate (stubs, builders, models,
   suites, specs) with 100% coverage of the generated
   plumbing.

## Principles

- **Zero domain knowledge.** The toolkit knows Go types,
  not business logic. Domain-specific behaviour is always
  injected by the caller.

- **Injectable.** Generated stubs, models, and suites
  accept domain logic via constructor arguments and oracle
  interfaces. The generator produces plumbing; the
  developer provides semantics.

- **Standalone module.** No dependency on any application
  module. Projects adopt testkit by importing it, not by
  forking it.

- **`//go:generate`-driven.** Generators are invoked via
  standard Go generate directives in source files — no
  central type registry. Project-wide conventions live in
  `.testkit.yml`.

## Documentation

| Document | Description |
|----------|-------------|
| **Primitives** | |
| [Overview](primitives/README.md) | Index of all primitives + dependencies |
| [Assertions](primitives/assertions.md) | Positional + fluent with go-cmp diffs |
| [Recording](primitives/recording.md) | Recorder[T] — filtering, waiting, hooks, gating |
| [Fault injection](primitives/fault-injection.md) | FaultInjector |
| [Concurrency](primitives/concurrency.md) | ConcurrentStress, GoroutineLeak |
| [Benchmarking](primitives/benchmarking.md) | Contract |
| [Golden files](primitives/golden-files.md) | AssertGolden + Scrubbers |
| [Context](primitives/context.md) | Timeout (Go 1.24+ `t.Context()` for the rest) |
| [Polling](primitives/polling.md) | RetryUntil, AssertEventually |
| [Helpers](primitives/helpers.md) | TestError, RequireEnv, SeededRand, FailableTB, etc. |
| **Generators** | |
| [Overview](generators/README.md) | CLI interface, output conventions, priority order |
| [In-Memory Stubs](generators/stub.md) | Three-tier stub + generated tests |
| [Recording Wrappers](generators/recording.md) | Call-capturing decorator + assertion helpers |
| [Fixture Builders](generators/builder.md) | Fluent `With*` builders + generated tests |
| [State Machine Models](generators/model.md) | rapid.StateMachine with oracle interface |
| [Conformance Suites](generators/suite.md) | Per-contract subtests from `//testkit:` directives |
| [Codec Test Specs](generators/codec.md) | `codectest.Spec[T]` from proto |
| [Error Sentinels](generators/sentinel.md) | Prefix, non-overlap, uniqueness |
| [Enum Tests](generators/enum.md) | Exhaustiveness, stringer, boundary |
| [Differential](generators/differential.md) | Fan-out comparing N implementations |
| [Wire Snapshots](generators/wire.md) | Regenerate golden `.bin` files |
| [Package Audit](generators/pkgdoc.md) | Compliance audit skeleton with auto-fill + refresh |
| [Integration](generators/integration.md) | TestMain + container + suite wiring |
| [Smoke Test](generators/smoke.md) | One-call-per-method baseline |
| [Sim Workload](generators/simworkload.md) | Random dispatch for sim drivers |
| [Scaffold](generators/scaffold.md) | One-time companion file generation |
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
